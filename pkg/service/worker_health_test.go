package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

type fakeLeaderStore struct {
	leases map[string]*repo.LeaderLease
	getErr error
}

func (f *fakeLeaderStore) TryAcquire(_ context.Context, _, _ string, _ time.Duration) (*repo.LeaderLease, error) {
	return nil, errors.New("TryAcquire not implemented in fake")
}
func (f *fakeLeaderStore) Refresh(_ context.Context, _, _ string, _ time.Duration) (*repo.LeaderLease, error) {
	return nil, errors.New("Refresh not implemented in fake")
}
func (f *fakeLeaderStore) Release(_ context.Context, _, _ string) error { return nil }
func (f *fakeLeaderStore) Get(_ context.Context, name string) (*repo.LeaderLease, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	l, ok := f.leases[name]
	if !ok {
		return nil, repo.ErrNotFound
	}
	return l, nil
}

func TestGetWorkerHealth_RequiresAuth(t *testing.T) {
	srv, _, _ := newTestServer()
	srv.WithWorkerHealth(&fakeLeaderStore{}, []WorkerInfo{{Name: "reclaim-job", TickInterval: time.Second, LeaseTTL: 30 * time.Second}})
	_, err := srv.GetWorkerHealth(context.Background(), &pb.GetWorkerHealthRequest{Namespace: testNamespace})
	requireStatus(t, err, codes.Unauthenticated)
}

func TestGetWorkerHealth_NamespaceMismatchPermissionDenied(t *testing.T) {
	srv, _, _ := newTestServer()
	srv.WithWorkerHealth(&fakeLeaderStore{}, nil)
	_, err := srv.GetWorkerHealth(authCtx(uuid.New()), &pb.GetWorkerHealthRequest{Namespace: "not-the-right-one"})
	requireStatus(t, err, codes.PermissionDenied)
}

func TestGetWorkerHealth_NoLeasesAllStale(t *testing.T) {
	srv, _, _ := newTestServer()
	srv.WithWorkerHealth(&fakeLeaderStore{leases: map[string]*repo.LeaderLease{}}, []WorkerInfo{
		{Name: "reclaim-job", TickInterval: 30 * time.Second, LeaseTTL: 30 * time.Second},
		{Name: "window-worker", TickInterval: 60 * time.Second, LeaseTTL: 30 * time.Second},
	})
	resp, err := srv.GetWorkerHealth(authCtx(uuid.New()), &pb.GetWorkerHealthRequest{Namespace: testNamespace})
	if err != nil {
		t.Fatalf("GetWorkerHealth: %v", err)
	}
	if got, want := len(resp.Workers), 2; got != want {
		t.Fatalf("got %d entries, want %d", got, want)
	}
	for _, w := range resp.Workers {
		if !w.Stale {
			t.Fatalf("worker %s expected stale=true, got false", w.Name)
		}
		if w.LeaseHolder != "" {
			t.Fatalf("worker %s expected empty lease_holder, got %q", w.Name, w.LeaseHolder)
		}
	}
}

func TestGetWorkerHealth_FreshLeaseNotStale(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	leases := &fakeLeaderStore{leases: map[string]*repo.LeaderLease{
		"reclaim-job": {
			Name:       "reclaim-job",
			Holder:     "pod-a",
			AcquiredAt: now.Add(-5 * time.Second),
			ExpiresAt:  now.Add(25 * time.Second),
		},
	}}
	srv, _, _ := newTestServer()
	srv.WithWorkerHealth(leases, []WorkerInfo{
		{Name: "reclaim-job", TickInterval: 30 * time.Second, LeaseTTL: 30 * time.Second},
	})
	srv.clockForWorkerHealth = func() time.Time { return now }

	resp, err := srv.GetWorkerHealth(authCtx(uuid.New()), &pb.GetWorkerHealthRequest{Namespace: testNamespace})
	if err != nil {
		t.Fatalf("GetWorkerHealth: %v", err)
	}
	if len(resp.Workers) != 1 {
		t.Fatalf("got %d workers, want 1", len(resp.Workers))
	}
	w := resp.Workers[0]
	if w.Stale {
		t.Fatalf("fresh lease should not be stale")
	}
	if w.LeaseHolder != "pod-a" {
		t.Fatalf("got lease_holder=%q, want pod-a", w.LeaseHolder)
	}
	// last_tick_at derived as expires_at - LeaseTTL.
	if got := w.LastTickAt.AsTime(); !got.Equal(now.Add(-5 * time.Second)) {
		t.Fatalf("last_tick_at=%s, want %s", got, now.Add(-5*time.Second))
	}
}

func TestGetWorkerHealth_ExpiredLeasePastStaleThresholdMarksStale(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	// expires_at + 2*TickInterval = expires_at + 120s. now is past that.
	leases := &fakeLeaderStore{leases: map[string]*repo.LeaderLease{
		"window-worker": {
			Name:       "window-worker",
			Holder:     "pod-b",
			AcquiredAt: now.Add(-10 * time.Minute),
			ExpiresAt:  now.Add(-3 * time.Minute),
		},
	}}
	srv, _, _ := newTestServer()
	srv.WithWorkerHealth(leases, []WorkerInfo{
		{Name: "window-worker", TickInterval: 60 * time.Second, LeaseTTL: 30 * time.Second},
	})
	srv.clockForWorkerHealth = func() time.Time { return now }

	resp, err := srv.GetWorkerHealth(authCtx(uuid.New()), &pb.GetWorkerHealthRequest{Namespace: testNamespace})
	if err != nil {
		t.Fatalf("GetWorkerHealth: %v", err)
	}
	if !resp.Workers[0].Stale {
		t.Fatalf("expected stale=true for lease past the threshold")
	}
}

func TestGetWorkerHealth_ExpiredLeaseStillInsideStaleGraceNotStale(t *testing.T) {
	now := time.Date(2026, 5, 19, 12, 0, 0, 0, time.UTC)
	// expires_at + 2*TickInterval = expires_at + 120s. Lease is 30s past
	// expiry — leader may have just lost the lease but cluster is still
	// within the grace window before we mark the banner red.
	leases := &fakeLeaderStore{leases: map[string]*repo.LeaderLease{
		"window-worker": {
			Name:       "window-worker",
			Holder:     "pod-b",
			AcquiredAt: now.Add(-2 * time.Minute),
			ExpiresAt:  now.Add(-30 * time.Second),
		},
	}}
	srv, _, _ := newTestServer()
	srv.WithWorkerHealth(leases, []WorkerInfo{
		{Name: "window-worker", TickInterval: 60 * time.Second, LeaseTTL: 30 * time.Second},
	})
	srv.clockForWorkerHealth = func() time.Time { return now }

	resp, err := srv.GetWorkerHealth(authCtx(uuid.New()), &pb.GetWorkerHealthRequest{Namespace: testNamespace})
	if err != nil {
		t.Fatalf("GetWorkerHealth: %v", err)
	}
	if resp.Workers[0].Stale {
		t.Fatalf("expected stale=false inside grace window")
	}
}
