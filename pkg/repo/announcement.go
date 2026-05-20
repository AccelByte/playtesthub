package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Announcement send-to filter values — keep in sync with the
// announcement_send_to_filter_enum CHECK in migration 0007.
const (
	AnnouncementSendToAll          = "ALL"
	AnnouncementSendToApprovedOnly = "APPROVED_ONLY"
	AnnouncementSendToPendingOnly  = "PENDING_ONLY"
)

// Announcement aggregate status values — keep in sync with the
// announcement_status_enum CHECK in migration 0007.
const (
	AnnouncementStatusSending = "SENDING"
	AnnouncementStatusSent    = "SENT"
	AnnouncementStatusPartial = "PARTIAL"
	AnnouncementStatusFailed  = "FAILED"
)

// AnnouncementRecipient dm_status values — mirror the per-row enum.
const (
	AnnouncementRecipientStatusQueued = "QUEUED"
	AnnouncementRecipientStatusSent   = "SENT"
	AnnouncementRecipientStatusFailed = "FAILED"
)

// Announcement mirrors the announcement table in migration 0007. Subject +
// message are PII-sensitive and never logged / audited (PRD §5.4
// "Bulk announcements"; docs/schema.md §"PII guarantee").
type Announcement struct {
	ID              uuid.UUID
	PlaytestID      uuid.UUID
	SendToFilter    string
	Subject         string
	Message         string
	Status          string
	RecipientsTotal int32
	RecipientsSent  int32
	CreatedByUserID uuid.UUID
	CreatedAt       time.Time
}

// AnnouncementRecipient mirrors the announcement_recipient join.
type AnnouncementRecipient struct {
	AnnouncementID uuid.UUID
	ApplicantID    uuid.UUID
	DMStatus       string
	DMSentAt       *time.Time
	DMFailedAt     *time.Time
	DMErrorCode    *string
}

// AnnouncementStore is the data access surface for the M5.C announcement
// tables (PRD §5.4 "Bulk announcements"). Service-layer handlers depend
// on the interface; tests mock it.
type AnnouncementStore interface {
	// Insert writes the announcement row. recipients_total is the
	// fan-out cardinality known at call time; recipients_sent starts at
	// 0 and is incremented by MarkRecipientSent as the queue drains.
	Insert(ctx context.Context, a *Announcement) (*Announcement, error)

	// InsertRecipients bulk-inserts the per-applicant join rows (one per
	// resolved recipient). All rows start at dm_status='QUEUED'. The
	// total is asserted equal to announcement.recipients_total on the
	// service side; the repo layer trusts the caller.
	InsertRecipients(ctx context.Context, announcementID uuid.UUID, applicantIDs []uuid.UUID) error

	// ListByPlaytest returns announcements for the playtest ordered by
	// created_at DESC. status is recomputed at read time via aggregate
	// over announcement_recipient.dm_status (PRD §5.4 / schema.md).
	ListByPlaytest(ctx context.Context, playtestID uuid.UUID) ([]*Announcement, error)

	// MarkRecipientSent updates the per-recipient row + increments
	// announcement.recipients_sent atomically. Called by the fan-out
	// path on Sender success.
	MarkRecipientSent(ctx context.Context, announcementID, applicantID uuid.UUID, sentAt time.Time) error

	// MarkRecipientFailed updates the per-recipient row with the error
	// reason. announcement.recipients_sent is NOT incremented.
	MarkRecipientFailed(ctx context.Context, announcementID, applicantID uuid.UUID, failedAt time.Time, errorCode string) error

	// FinaliseStatus recomputes the aggregate status on the announcement
	// row from the per-recipient dm_status spread. Called once the
	// inline fan-out drains.
	FinaliseStatus(ctx context.Context, announcementID uuid.UUID) error

	// ListRecipients returns the per-recipient join rows for the
	// announcement; powers the per-row detail modal in M5.C.
	ListRecipients(ctx context.Context, announcementID uuid.UUID) ([]*AnnouncementRecipient, error)
}

// PgAnnouncementStore is the Postgres-backed AnnouncementStore.
type PgAnnouncementStore struct {
	pool *pgxpool.Pool
}

func NewPgAnnouncementStore(pool *pgxpool.Pool) *PgAnnouncementStore {
	return &PgAnnouncementStore{pool: pool}
}

const announcementColumns = `
	id, playtest_id, send_to_filter, subject, message, status,
	recipients_total, recipients_sent, created_by_user_id, created_at`

func (s *PgAnnouncementStore) Insert(ctx context.Context, a *Announcement) (*Announcement, error) {
	const sql = `
		INSERT INTO announcement (
			playtest_id, send_to_filter, subject, message, status,
			recipients_total, recipients_sent, created_by_user_id
		)
		VALUES ($1, $2, $3, $4, $5, $6, 0, $7)
		RETURNING ` + announcementColumns

	if a.Status == "" {
		a.Status = AnnouncementStatusSending
	}

	row := s.pool.QueryRow(ctx, sql,
		a.PlaytestID, a.SendToFilter, a.Subject, a.Message,
		a.Status, a.RecipientsTotal, a.CreatedByUserID)
	got, err := scanAnnouncement(row)
	if err != nil {
		return nil, fmt.Errorf("inserting announcement: %w", classifyPgError(err))
	}
	return got, nil
}

func (s *PgAnnouncementStore) InsertRecipients(ctx context.Context, announcementID uuid.UUID, applicantIDs []uuid.UUID) error {
	if len(applicantIDs) == 0 {
		return nil
	}
	const sql = `
		INSERT INTO announcement_recipient (announcement_id, applicant_id)
		SELECT $1, unnest($2::uuid[])`
	if _, err := s.pool.Exec(ctx, sql, announcementID, applicantIDs); err != nil {
		return fmt.Errorf("inserting announcement recipients: %w", classifyPgError(err))
	}
	return nil
}

func (s *PgAnnouncementStore) ListByPlaytest(ctx context.Context, playtestID uuid.UUID) ([]*Announcement, error) {
	const sql = `
		SELECT ` + announcementColumns + `
		  FROM announcement
		 WHERE playtest_id = $1
		 ORDER BY created_at DESC, id DESC`

	rows, err := s.pool.Query(ctx, sql, playtestID)
	if err != nil {
		return nil, fmt.Errorf("listing announcements: %w", err)
	}
	defer rows.Close()

	var out []*Announcement
	for rows.Next() {
		got, scanErr := scanAnnouncement(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scanning announcement: %w", scanErr)
		}
		out = append(out, got)
	}
	return out, rows.Err()
}

func (s *PgAnnouncementStore) MarkRecipientSent(ctx context.Context, announcementID, applicantID uuid.UUID, sentAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx,
		`UPDATE announcement_recipient
		    SET dm_status = $3, dm_sent_at = $4
		  WHERE announcement_id = $1 AND applicant_id = $2`,
		announcementID, applicantID, AnnouncementRecipientStatusSent, sentAt); err != nil {
		return fmt.Errorf("updating recipient: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE announcement
		    SET recipients_sent = recipients_sent + 1
		  WHERE id = $1`,
		announcementID); err != nil {
		return fmt.Errorf("incrementing recipients_sent: %w", err)
	}
	return tx.Commit(ctx)
}

func (s *PgAnnouncementStore) MarkRecipientFailed(ctx context.Context, announcementID, applicantID uuid.UUID, failedAt time.Time, errorCode string) error {
	if _, err := s.pool.Exec(ctx,
		`UPDATE announcement_recipient
		    SET dm_status = $3, dm_failed_at = $4, dm_error_code = $5
		  WHERE announcement_id = $1 AND applicant_id = $2`,
		announcementID, applicantID, AnnouncementRecipientStatusFailed, failedAt, errorCode); err != nil {
		return fmt.Errorf("updating recipient failed: %w", err)
	}
	return nil
}

func (s *PgAnnouncementStore) FinaliseStatus(ctx context.Context, announcementID uuid.UUID) error {
	// The aggregate matches schema.md §"Status aggregation":
	//   SENT     — every recipient SENT.
	//   SENDING  — any recipient QUEUED.
	//   FAILED   — every recipient FAILED (no SENT, no QUEUED).
	//   PARTIAL  — mix of SENT + FAILED, no QUEUED.
	const sql = `
		UPDATE announcement
		   SET status = sub.status
		  FROM (
			SELECT
				CASE
					WHEN SUM(CASE WHEN dm_status = 'QUEUED' THEN 1 ELSE 0 END) > 0 THEN 'SENDING'
					WHEN SUM(CASE WHEN dm_status = 'FAILED' THEN 1 ELSE 0 END) = COUNT(*) THEN 'FAILED'
					WHEN SUM(CASE WHEN dm_status = 'SENT'   THEN 1 ELSE 0 END) = COUNT(*) THEN 'SENT'
					ELSE 'PARTIAL'
				END AS status
			  FROM announcement_recipient
			 WHERE announcement_id = $1
		  ) AS sub
		 WHERE announcement.id = $1`
	if _, err := s.pool.Exec(ctx, sql, announcementID); err != nil {
		return fmt.Errorf("finalising announcement status: %w", err)
	}
	return nil
}

func (s *PgAnnouncementStore) ListRecipients(ctx context.Context, announcementID uuid.UUID) ([]*AnnouncementRecipient, error) {
	const sql = `
		SELECT announcement_id, applicant_id, dm_status, dm_sent_at, dm_failed_at, dm_error_code
		  FROM announcement_recipient
		 WHERE announcement_id = $1
		 ORDER BY applicant_id`
	rows, err := s.pool.Query(ctx, sql, announcementID)
	if err != nil {
		return nil, fmt.Errorf("listing recipients: %w", err)
	}
	defer rows.Close()
	var out []*AnnouncementRecipient
	for rows.Next() {
		r := &AnnouncementRecipient{}
		if scanErr := rows.Scan(&r.AnnouncementID, &r.ApplicantID, &r.DMStatus, &r.DMSentAt, &r.DMFailedAt, &r.DMErrorCode); scanErr != nil {
			return nil, fmt.Errorf("scanning recipient: %w", scanErr)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// scanAnnouncement maps a pgx row to *Announcement.
func scanAnnouncement(row interface{ Scan(...any) error }) (*Announcement, error) {
	a := &Announcement{}
	err := row.Scan(
		&a.ID, &a.PlaytestID, &a.SendToFilter, &a.Subject, &a.Message,
		&a.Status, &a.RecipientsTotal, &a.RecipientsSent,
		&a.CreatedByUserID, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return a, nil
}
