package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	"github.com/anggorodewanto/playtesthub/pkg/adt"
	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
)

// TestListADTGames_HappyPath verifies the service handler proxies through
// to adt.Client.ListGames scoped to the resolved studio_namespace.
func TestListADTGames_HappyPath(t *testing.T) {
	h := newADTTestServer(t)
	linkage := h.linkage.live[testStudioNamespace+"|"+testADTNamespace]
	h.mem.SeedGames(testStudioNamespace, testADTNamespace, []adt.Game{
		{ID: "game-1", Name: "Aces", CreatedAt: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)},
		{ID: "game-2", Name: "Bombers", CreatedAt: time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)},
	})

	resp, err := h.svr.ListADTGames(authCtx(uuid.New()), &pb.ListADTGamesRequest{
		Namespace:    testNamespace,
		AdtLinkageId: linkage.ID.String(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := len(resp.GetGames()); got != 2 {
		t.Fatalf("len(games) = %d, want 2", got)
	}
	if resp.GetGames()[0].GetId() != "game-2" {
		t.Errorf("first game id = %q, want game-2 (newest-first sort)", resp.GetGames()[0].GetId())
	}
	if resp.GetGames()[0].GetName() != "Bombers" {
		t.Errorf("first game name = %q, want Bombers", resp.GetGames()[0].GetName())
	}
	if resp.GetGames()[0].GetCreatedAt() == nil {
		t.Errorf("created_at is nil, want populated")
	}
}

// TestListADTGames_LinkageNotFound_FailedPrecondition pins the byte-exact
// message mirroring ListADTBuilds (errors.md row for ListADTGames).
func TestListADTGames_LinkageNotFound_FailedPrecondition(t *testing.T) {
	h := newADTTestServer(t)
	_, err := h.svr.ListADTGames(authCtx(uuid.New()), &pb.ListADTGamesRequest{
		Namespace:    testNamespace,
		AdtLinkageId: uuid.New().String(),
	})
	requireStatus(t, err, codes.FailedPrecondition)
	requireMsgContains(t, err, "no ADT linkage matches this id for the caller's studio; link an ADT namespace first")
}

// TestListADTGames_ADTLinkageMissing_MapsTo401 verifies the live-adapter
// 401 → ErrLinkageMissing path is mapped to byte-exact FailedPrecondition
// "adt linkage no longer exists or service token rejected, re-link required".
func TestListADTGames_ADTLinkageMissing_MapsTo401(t *testing.T) {
	h := newADTTestServer(t)
	linkage := h.linkage.live[testStudioNamespace+"|"+testADTNamespace]
	h.mem.ClearLinkage(testStudioNamespace, testADTNamespace)

	_, err := h.svr.ListADTGames(authCtx(uuid.New()), &pb.ListADTGamesRequest{
		Namespace:    testNamespace,
		AdtLinkageId: linkage.ID.String(),
	})
	requireStatus(t, err, codes.FailedPrecondition)
	requireMsgContains(t, err, "adt linkage no longer exists or service token rejected, re-link required")
}

// TestListADTGames_TransportError_MapsToUnavailable verifies non-401
// errors from the client surface as gRPC Unavailable.
func TestListADTGames_TransportError_MapsToUnavailable(t *testing.T) {
	h := newADTTestServer(t)
	linkage := h.linkage.live[testStudioNamespace+"|"+testADTNamespace]
	h.mem.ListGamesErr = []error{errors.New("boom")}

	_, err := h.svr.ListADTGames(authCtx(uuid.New()), &pb.ListADTGamesRequest{
		Namespace:    testNamespace,
		AdtLinkageId: linkage.ID.String(),
	})
	requireStatus(t, err, codes.Unavailable)
}

// TestListADTGames_InvalidLinkageID_InvalidArgument pins the standard
// UUID validation surfacing the consistent "must be a valid UUID" mode.
func TestListADTGames_InvalidLinkageID_InvalidArgument(t *testing.T) {
	h := newADTTestServer(t)
	_, err := h.svr.ListADTGames(authCtx(uuid.New()), &pb.ListADTGamesRequest{
		Namespace:    testNamespace,
		AdtLinkageId: "not-a-uuid",
	})
	requireStatus(t, err, codes.InvalidArgument)
}

// TestListADTGames_RequireActor pins that unauthenticated calls fail
// before the studio-resolver runs (PRD §4.7).
func TestListADTGames_RequireActor(t *testing.T) {
	h := newADTTestServer(t)
	_, err := h.svr.ListADTGames(context.Background(), &pb.ListADTGamesRequest{
		Namespace:    testNamespace,
		AdtLinkageId: uuid.New().String(),
	})
	requireStatus(t, err, codes.Unauthenticated)
}
