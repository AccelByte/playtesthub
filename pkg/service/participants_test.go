package service

import (
	"testing"
	"time"

	"github.com/google/uuid"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

// TestGetPlaytestParticipants_CodeSentDateFromLastDMAttempt asserts the
// PRD §5.4 "Code Sent Date — derived field" rule: code_sent_at is read
// off applicant.last_dm_attempt_at when last_dm_status='sent'.
func TestGetPlaytestParticipants_CodeSentDateFromLastDMAttempt(t *testing.T) {
	svr, pt, ap := newTestServer()
	p := seedOpenPlaytest(t, pt, "participants-codes")

	sent := "sent"
	failed := "failed"
	attemptAt := time.Now().UTC().Add(-2 * time.Hour)
	a1 := seedApplicant(t, ap, p.ID, applicantStatusApproved, "111")
	a1.LastDMStatus = &sent
	a1.LastDMAttemptAt = &attemptAt
	a2 := seedApplicant(t, ap, p.ID, applicantStatusApproved, "222")
	a2.LastDMStatus = &failed
	a2.LastDMAttemptAt = &attemptAt // failed → code_sent_at NULL despite attempt

	resp, err := svr.GetPlaytestParticipants(authCtx(uuid.New()), &pb.GetPlaytestParticipantsRequest{
		Namespace:  testNamespace,
		PlaytestId: p.ID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.GetParticipants()) != 2 {
		t.Fatalf("rows = %d, want 2", len(resp.GetParticipants()))
	}
	var sentRow, failedRow *pb.ParticipantRow
	for _, r := range resp.GetParticipants() {
		switch r.GetApplicantId() {
		case a1.ID.String():
			sentRow = r
		case a2.ID.String():
			failedRow = r
		}
	}
	if sentRow.GetCodeSentAt() == nil {
		t.Errorf("sent-row code_sent_at = nil, want %v", attemptAt)
	}
	if failedRow.GetCodeSentAt() != nil {
		t.Errorf("failed-row code_sent_at = %v, want nil (last_dm_status != 'sent')", failedRow.GetCodeSentAt())
	}
}

// TestGetPlaytestParticipants_ADTRowsHaveNoCodeSentAt confirms ADT
// applicants — no code grant, so no DM-attribution-driven code_sent_at —
// surface NULL Code Sent Date in M5.C.
func TestGetPlaytestParticipants_ADTRowsHaveNoCodeSentAt(t *testing.T) {
	svr, pt, ap := newTestServer()
	p := seedOpenPlaytest(t, pt, "participants-adt")
	p.DistributionModel = distModelADT
	// Approved ADT applicant with no DM history.
	_ = seedApplicant(t, ap, p.ID, applicantStatusApproved, "adt-1")

	resp, err := svr.GetPlaytestParticipants(authCtx(uuid.New()), &pb.GetPlaytestParticipantsRequest{
		Namespace:  testNamespace,
		PlaytestId: p.ID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.GetParticipants()) != 1 {
		t.Fatalf("rows = %d, want 1", len(resp.GetParticipants()))
	}
	if resp.GetParticipants()[0].GetCodeSentAt() != nil {
		t.Errorf("ADT row code_sent_at = %v, want nil", resp.GetParticipants()[0].GetCodeSentAt())
	}
	// Four dormant ADT telemetry fields all NULL/zero in M5.C.
	if resp.GetParticipants()[0].GetAdtDownloadAt() != nil {
		t.Errorf("adt_download_at = %v, want nil (M6-deferred)", resp.GetParticipants()[0].GetAdtDownloadAt())
	}
	if resp.GetParticipants()[0].GetAdtCrashCount() != 0 {
		t.Errorf("adt_crash_count = %d, want 0", resp.GetParticipants()[0].GetAdtCrashCount())
	}
}

// TestGetPlaytestParticipants_StatusFilter restricts to the requested
// status enum.
func TestGetPlaytestParticipants_StatusFilter(t *testing.T) {
	svr, pt, ap := newTestServer()
	p := seedOpenPlaytest(t, pt, "participants-filter")
	_ = seedApplicant(t, ap, p.ID, applicantStatusApproved, "ok")
	_ = seedApplicant(t, ap, p.ID, applicantStatusPending, "wait")
	_ = seedApplicant(t, ap, p.ID, applicantStatusRejected, "no")

	resp, err := svr.GetPlaytestParticipants(authCtx(uuid.New()), &pb.GetPlaytestParticipantsRequest{
		Namespace:    testNamespace,
		PlaytestId:   p.ID.String(),
		StatusFilter: pb.ApplicantStatus_APPLICANT_STATUS_PENDING,
	})
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if len(resp.GetParticipants()) != 1 {
		t.Fatalf("rows = %d, want 1 (PENDING-only filter)", len(resp.GetParticipants()))
	}
	if resp.GetParticipants()[0].GetStatus() != pb.ApplicantStatus_APPLICANT_STATUS_PENDING {
		t.Errorf("status = %s, want PENDING", resp.GetParticipants()[0].GetStatus())
	}
}

// Tiny smoke against applicantToParticipantRow's auto-approved propagation.
func TestApplicantToParticipantRow_AutoApproved(t *testing.T) {
	a := &repo.Applicant{
		ID:           uuid.New(),
		UserID:       uuid.New(),
		Status:       applicantStatusApproved,
		AutoApproved: true,
		CreatedAt:    time.Now(),
	}
	row := applicantToParticipantRow(a)
	if !row.GetAutoApproved() {
		t.Error("auto_approved did not propagate")
	}
}
