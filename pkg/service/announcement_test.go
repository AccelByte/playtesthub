package service

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

// --- fakeAnnouncementStore ---------------------------------------------------

type fakeAnnouncementStore struct {
	mu         sync.Mutex
	rows       []*repo.Announcement
	recipients map[uuid.UUID][]*repo.AnnouncementRecipient
}

func newFakeAnnouncementStore() *fakeAnnouncementStore {
	return &fakeAnnouncementStore{recipients: map[uuid.UUID][]*repo.AnnouncementRecipient{}}
}

func (f *fakeAnnouncementStore) Insert(_ context.Context, a *repo.Announcement) (*repo.Announcement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	clone := *a
	clone.ID = uuid.New()
	clone.CreatedAt = time.Now()
	if clone.Status == "" {
		clone.Status = repo.AnnouncementStatusSending
	}
	f.rows = append(f.rows, &clone)
	ret := clone
	return &ret, nil
}

func (f *fakeAnnouncementStore) InsertRecipients(_ context.Context, announcementID uuid.UUID, applicantIDs []uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, id := range applicantIDs {
		f.recipients[announcementID] = append(f.recipients[announcementID], &repo.AnnouncementRecipient{
			AnnouncementID: announcementID,
			ApplicantID:    id,
			DMStatus:       repo.AnnouncementRecipientStatusQueued,
		})
	}
	return nil
}

func (f *fakeAnnouncementStore) GetByID(_ context.Context, announcementID uuid.UUID) (*repo.Announcement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.rows {
		if r.ID == announcementID {
			clone := *r
			return &clone, nil
		}
	}
	return nil, repo.ErrNotFound
}

func (f *fakeAnnouncementStore) ListByPlaytest(_ context.Context, playtestID uuid.UUID) ([]*repo.Announcement, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*repo.Announcement, 0)
	for _, r := range f.rows {
		if r.PlaytestID == playtestID {
			clone := *r
			out = append(out, &clone)
		}
	}
	return out, nil
}

func (f *fakeAnnouncementStore) MarkRecipientSent(_ context.Context, announcementID, applicantID uuid.UUID, sentAt time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.recipients[announcementID] {
		if r.ApplicantID == applicantID {
			r.DMStatus = repo.AnnouncementRecipientStatusSent
			t := sentAt
			r.DMSentAt = &t
		}
	}
	for _, ann := range f.rows {
		if ann.ID == announcementID {
			ann.RecipientsSent++
		}
	}
	return nil
}

func (f *fakeAnnouncementStore) MarkRecipientFailed(_ context.Context, announcementID, applicantID uuid.UUID, failedAt time.Time, errorCode string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, r := range f.recipients[announcementID] {
		if r.ApplicantID == applicantID {
			r.DMStatus = repo.AnnouncementRecipientStatusFailed
			t := failedAt
			r.DMFailedAt = &t
			code := errorCode
			r.DMErrorCode = &code
		}
	}
	return nil
}

func (f *fakeAnnouncementStore) FinaliseStatus(_ context.Context, announcementID uuid.UUID) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	rec := f.recipients[announcementID]
	var sent, failed, queued int
	for _, r := range rec {
		switch r.DMStatus {
		case repo.AnnouncementRecipientStatusSent:
			sent++
		case repo.AnnouncementRecipientStatusFailed:
			failed++
		default:
			queued++
		}
	}
	finalStatus := repo.AnnouncementStatusPartial
	switch {
	case queued > 0:
		finalStatus = repo.AnnouncementStatusSending
	case failed == len(rec):
		finalStatus = repo.AnnouncementStatusFailed
	case sent == len(rec):
		finalStatus = repo.AnnouncementStatusSent
	}
	for _, ann := range f.rows {
		if ann.ID == announcementID {
			ann.Status = finalStatus
		}
	}
	return nil
}

func (f *fakeAnnouncementStore) ListRecipients(_ context.Context, announcementID uuid.UUID) ([]*repo.AnnouncementRecipient, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*repo.AnnouncementRecipient, 0)
	for _, r := range f.recipients[announcementID] {
		clone := *r
		out = append(out, &clone)
	}
	return out, nil
}

// --- fakeAnnouncementSender --------------------------------------------------

type fakeAnnouncementSender struct {
	mu          sync.Mutex
	deliveries  []deliveryRecord
	failHandles map[string]error
}

type deliveryRecord struct {
	Recipient string
	Body      string
}

func (f *fakeAnnouncementSender) SendDM(_ context.Context, recipient, message string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if err, ok := f.failHandles[recipient]; ok {
		return err
	}
	f.deliveries = append(f.deliveries, deliveryRecord{Recipient: recipient, Body: message})
	return nil
}

func (f *fakeAnnouncementSender) snapshot() []deliveryRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]deliveryRecord, len(f.deliveries))
	copy(out, f.deliveries)
	return out
}

// --- tests -------------------------------------------------------------------

type announcementRig struct {
	svr    *PlaytesthubServiceServer
	pt     *fakePlaytestStore
	ap     *fakeApplicantStore
	ann    *fakeAnnouncementStore
	sender *fakeAnnouncementSender
	audit  *fakeAuditLogStore
}

func seedAnnouncementRig(t *testing.T) announcementRig {
	t.Helper()
	svr, pt, ap := newTestServer()
	annStore := newFakeAnnouncementStore()
	sender := &fakeAnnouncementSender{}
	audit := &fakeAuditLogStore{}
	svr = svr.
		WithAuditLogStore(audit).
		WithAnnouncementStore(annStore, sender)
	return announcementRig{svr: svr, pt: pt, ap: ap, ann: annStore, sender: sender, audit: audit}
}

func seedOpenPlaytest(t *testing.T, store *fakePlaytestStore, slug string) *repo.Playtest {
	t.Helper()
	p := &repo.Playtest{
		ID:                uuid.New(),
		Namespace:         testNamespace,
		Slug:              slug,
		Title:             "playtest " + slug,
		Status:            statusOpen,
		DistributionModel: distModelSteamKeys,
		CreatedAt:         time.Now(),
		UpdatedAt:         time.Now(),
	}
	store.rows = append(store.rows, p)
	return p
}

func seedApplicant(t *testing.T, ap *fakeApplicantStore, playtestID uuid.UUID, status, discordID string) *repo.Applicant {
	t.Helper()
	id := discordID
	a := &repo.Applicant{
		ID:            uuid.New(),
		PlaytestID:    playtestID,
		UserID:        uuid.New(),
		DiscordHandle: discordID,
		DiscordUserID: &id,
		Status:        status,
		CreatedAt:     time.Now(),
	}
	ap.rows = append(ap.rows, a)
	return a
}

// TestCreateAnnouncement_HappyPath_ResolvesAtCallTime asserts that the
// recipient set resolves against the applicant set as it stood at call
// time (PRD §5.4 "Bulk announcements"). One applicant added after the
// broadcast is NOT included.
func TestCreateAnnouncement_HappyPath_ResolvesAtCallTime(t *testing.T) {
	rig := seedAnnouncementRig(t)
	p := seedOpenPlaytest(t, rig.pt, "ann-happy")
	a1 := seedApplicant(t, rig.ap, p.ID, applicantStatusApproved, "111")
	_ = seedApplicant(t, rig.ap, p.ID, applicantStatusApproved, "222")

	resp, err := rig.svr.CreateAnnouncement(authCtx(uuid.New()), &pb.CreateAnnouncementRequest{
		Namespace:    testNamespace,
		PlaytestId:   p.ID.String(),
		SendToFilter: pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_APPROVED_ONLY,
		Subject:      "build update",
		Message:      "new patch live",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if resp.GetAnnouncement().GetRecipientsTotal() != 2 {
		t.Fatalf("recipients_total = %d, want 2", resp.GetAnnouncement().GetRecipientsTotal())
	}

	// Adding a third applicant after the call must NOT auto-include.
	_ = seedApplicant(t, rig.ap, p.ID, applicantStatusApproved, "333")
	if got := len(rig.sender.snapshot()); got != 2 {
		t.Errorf("sender deliveries = %d, want 2 (resolution is at call time)", got)
	}

	// Per-recipient row created for the original cohort only.
	recs, _ := rig.ann.ListRecipients(context.Background(), uuid.MustParse(resp.GetAnnouncement().GetId()))
	if len(recs) != 2 {
		t.Errorf("recipients = %d, want 2", len(recs))
	}
	for _, r := range recs {
		if r.DMStatus != repo.AnnouncementRecipientStatusSent {
			t.Errorf("recipient %s dm_status = %s, want SENT", r.ApplicantID, r.DMStatus)
		}
	}

	// Final aggregate status is SENT (all delivered).
	if resp.GetAnnouncement().GetStatus() != pb.AnnouncementStatus_ANNOUNCEMENT_STATUS_SENT {
		t.Errorf("status = %s, want SENT", resp.GetAnnouncement().GetStatus())
	}

	// Audit row carries no subject / message (PII canary).
	if got := len(rig.audit.rows); got != 1 {
		t.Fatalf("audit row count = %d, want 1", got)
	}
	row := rig.audit.rows[0]
	if row.Action != repo.ActionAnnouncementCreate {
		t.Errorf("action = %q, want %q", row.Action, repo.ActionAnnouncementCreate)
	}
	payload := string(row.After)
	if strings.Contains(payload, "build update") || strings.Contains(payload, "new patch live") {
		t.Errorf("audit payload leaks subject/message: %s", payload)
	}
	// Sanity: known IDs DO appear in payload.
	if !strings.Contains(payload, resp.GetAnnouncement().GetId()) {
		t.Errorf("audit payload missing announcementId: %s", payload)
	}
	_ = a1
}

func TestCreateAnnouncement_RejectsEmptySubject(t *testing.T) {
	rig := seedAnnouncementRig(t)
	p := seedOpenPlaytest(t, rig.pt, "ann-empty-subject")

	_, err := rig.svr.CreateAnnouncement(authCtx(uuid.New()), &pb.CreateAnnouncementRequest{
		Namespace:    testNamespace,
		PlaytestId:   p.ID.String(),
		SendToFilter: pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_ALL,
		Subject:      "",
		Message:      "x",
	})
	requireStatus(t, err, codes.InvalidArgument)
	if !strings.Contains(err.Error(), "announcement subject must not be empty") {
		t.Errorf("byte-exact message missing: %v", err)
	}
}

func TestCreateAnnouncement_RejectsOverlongMessage(t *testing.T) {
	rig := seedAnnouncementRig(t)
	p := seedOpenPlaytest(t, rig.pt, "ann-long-msg")

	_, err := rig.svr.CreateAnnouncement(authCtx(uuid.New()), &pb.CreateAnnouncementRequest{
		Namespace:    testNamespace,
		PlaytestId:   p.ID.String(),
		SendToFilter: pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_ALL,
		Subject:      "subj",
		Message:      strings.Repeat("a", 4001),
	})
	requireStatus(t, err, codes.InvalidArgument)
	if !strings.Contains(err.Error(), "announcement message must be at most 4000 characters") {
		t.Errorf("byte-exact message missing: %v", err)
	}
}

func TestCreateAnnouncement_RejectsClosedPlaytest(t *testing.T) {
	rig := seedAnnouncementRig(t)
	p := seedOpenPlaytest(t, rig.pt, "ann-closed")
	p.Status = statusClosed

	_, err := rig.svr.CreateAnnouncement(authCtx(uuid.New()), &pb.CreateAnnouncementRequest{
		Namespace:    testNamespace,
		PlaytestId:   p.ID.String(),
		SendToFilter: pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_ALL,
		Subject:      "subj",
		Message:      "msg",
	})
	requireStatus(t, err, codes.FailedPrecondition)
	if !strings.Contains(err.Error(), "playtest is closed; announcements can no longer be sent") {
		t.Errorf("byte-exact message missing: %v", err)
	}
}

// TestCreateAnnouncement_FilterAppliesAtFanOut covers the Send-To filter
// semantics: APPROVED_ONLY against a mixed cohort picks the APPROVED
// rows only.
func TestCreateAnnouncement_FilterAppliesAtFanOut(t *testing.T) {
	rig := seedAnnouncementRig(t)
	p := seedOpenPlaytest(t, rig.pt, "ann-filter")
	_ = seedApplicant(t, rig.ap, p.ID, applicantStatusApproved, "ok-1")
	_ = seedApplicant(t, rig.ap, p.ID, applicantStatusPending, "pending-2")
	_ = seedApplicant(t, rig.ap, p.ID, applicantStatusApproved, "ok-3")

	resp, err := rig.svr.CreateAnnouncement(authCtx(uuid.New()), &pb.CreateAnnouncementRequest{
		Namespace:    testNamespace,
		PlaytestId:   p.ID.String(),
		SendToFilter: pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_APPROVED_ONLY,
		Subject:      "x",
		Message:      "y",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if resp.GetAnnouncement().GetRecipientsTotal() != 2 {
		t.Errorf("recipients_total = %d, want 2 (approved only)", resp.GetAnnouncement().GetRecipientsTotal())
	}
	delivered := rig.sender.snapshot()
	if len(delivered) != 2 {
		t.Errorf("sender deliveries = %d, want 2", len(delivered))
	}
	for _, d := range delivered {
		if d.Recipient == "pending-2" {
			t.Errorf("PENDING recipient %q should not have been included", d.Recipient)
		}
	}
}

// TestListAnnouncements_OrdersDescAndStatusAggregates checks that
// ListAnnouncements returns the per-playtest history with the aggregate
// status from the last broadcast.
func TestListAnnouncements_OrdersDescAndStatusAggregates(t *testing.T) {
	rig := seedAnnouncementRig(t)
	p := seedOpenPlaytest(t, rig.pt, "ann-list")
	_ = seedApplicant(t, rig.ap, p.ID, applicantStatusApproved, "rec-a")
	_ = seedApplicant(t, rig.ap, p.ID, applicantStatusApproved, "rec-b")
	// Make one recipient fail to exercise PARTIAL aggregate.
	rig.sender.failHandles = map[string]error{"rec-b": context.DeadlineExceeded}

	_, err := rig.svr.CreateAnnouncement(authCtx(uuid.New()), &pb.CreateAnnouncementRequest{
		Namespace:    testNamespace,
		PlaytestId:   p.ID.String(),
		SendToFilter: pb.AnnouncementSendToFilter_ANNOUNCEMENT_SEND_TO_FILTER_ALL,
		Subject:      "x",
		Message:      "y",
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}

	resp, err := rig.svr.ListAnnouncements(authCtx(uuid.New()), &pb.ListAnnouncementsRequest{
		Namespace:  testNamespace,
		PlaytestId: p.ID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.GetAnnouncements()) != 1 {
		t.Fatalf("announcements = %d, want 1", len(resp.GetAnnouncements()))
	}
	if got := resp.GetAnnouncements()[0].GetStatus(); got != pb.AnnouncementStatus_ANNOUNCEMENT_STATUS_PARTIAL {
		t.Errorf("status = %s, want PARTIAL (one delivery failed)", got)
	}
}
