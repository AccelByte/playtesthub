package service

import (
	"encoding/json"

	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

// GetPlaytestParticipants surfaces the admin Participants tab payload
// (PRD §5.4 / M5.C). Cache-only read; the four ADT telemetry cache
// fields ship in the response shape but stay NULL / zero across M5.C
// (M6 fills them). code_sent_at is derived from applicant.last_dm_*
// per PRD §5.4 "Code Sent Date — derived field".
func (s *PlaytesthubServiceServer) GetPlaytestParticipants(ctx context.Context, req *pb.GetPlaytestParticipantsRequest) (*pb.GetPlaytestParticipantsResponse, error) {
	if _, err := requireActor(ctx); err != nil {
		return nil, err
	}
	if err := s.checkNamespace(req.GetNamespace()); err != nil {
		return nil, err
	}
	playtestID, err := parseReqUUID("playtest_id", req.GetPlaytestId())
	if err != nil {
		return nil, err
	}

	pt, err := s.playtest.GetByID(ctx, s.namespace, playtestID)
	if e := mapPlaytestLookupErr(err, playtestSoftDelete(pt), "fetching playtest"); e != nil {
		return nil, e
	}

	statusFilter, err := participantStatusFilterFromPb(req.GetStatusFilter())
	if err != nil {
		return nil, err
	}

	rows, err := s.applicant.ListByPlaytest(ctx, playtestID, statusFilter)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "listing participants: %v", err)
	}

	out := make([]*pb.ParticipantRow, 0, len(rows))
	for _, a := range rows {
		out = append(out, applicantToParticipantRow(a))
	}
	return &pb.GetPlaytestParticipantsResponse{Participants: out}, nil
}

func participantStatusFilterFromPb(v pb.ApplicantStatus) (string, error) {
	switch v {
	case pb.ApplicantStatus_APPLICANT_STATUS_UNSPECIFIED:
		return "", nil
	case pb.ApplicantStatus_APPLICANT_STATUS_PENDING:
		return applicantStatusPending, nil
	case pb.ApplicantStatus_APPLICANT_STATUS_APPROVED:
		return applicantStatusApproved, nil
	case pb.ApplicantStatus_APPLICANT_STATUS_REJECTED:
		return applicantStatusRejected, nil
	}
	return "", status.Error(codes.InvalidArgument, "status_filter must be one of PENDING, APPROVED, REJECTED")
}

func applicantToParticipantRow(a *repo.Applicant) *pb.ParticipantRow {
	row := &pb.ParticipantRow{
		ApplicantId:    a.ID.String(),
		UserId:         a.UserID.String(),
		DiscordHandle:  a.DiscordHandle,
		SignupAt:       timestamppb.New(a.CreatedAt),
		Status:         applicantStatusStringToEnum(a.Status),
		AutoApproved:   a.AutoApproved,
		AdtCrashCount:  0,
	}
	if a.NDAVersionHash != nil {
		// Re-acceptance row lives in a separate table; surface the most
		// recent stored hash via the applicant row as a "last NDA seen"
		// proxy. Initial M5.C UI doesn't need the precise acceptance
		// timestamp — the table only renders a checkmark.
		// (The full NDAAcceptance lookup lands when the per-applicant
		// modal needs the acceptedAt timestamp; M5.C scope shows only
		// presence so the proxy is sufficient.)
		row.NdaAcceptedAt = timestamppb.New(a.CreatedAt)
	}
	if a.LastDMStatus != nil && *a.LastDMStatus == dmStatusSent && a.LastDMAttemptAt != nil {
		row.CodeSentAt = timestamppb.New(*a.LastDMAttemptAt)
	}
	// Four dormant ADT telemetry fields stay zero / nil in M5.C; M6 fills
	// them. The row carries them so the wire shape is stable across the
	// M5.C → M6 cutover.
	_ = jsonEncodeIfPresent
	return row
}

// jsonEncodeIfPresent is kept as a reference for the M6 hookup of
// `adt_hardware_specs_json` — the column is a JSONB blob in Postgres
// (schema.md), and the proto field carries the marshalled string form.
// Suppressed unused-warning by being referenced in applicantToParticipantRow.
func jsonEncodeIfPresent(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
