package service

import (
	"context"
	"errors"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	pb "github.com/anggorodewanto/playtesthub/pkg/pb/playtesthub/v1"
	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

// WorkerInfo describes one registered background worker so
// GetWorkerHealth can read its lease state and decide whether the
// admin banner should fire (STATUS_M4.md phase 5). Production
// populates this list in main.go after each worker goroutine starts.
type WorkerInfo struct {
	// Name matches the leader_lease.name value used by the worker —
	// e.g. reclaim.LeaseName ("reclaim-job"), window.LeaseName
	// ("window-worker").
	Name string

	// TickInterval is how often the worker advances its sweep cadence.
	// stale := now > expires_at + 2*TickInterval. Always > 0.
	TickInterval time.Duration

	// LeaseTTL matches the worker's leader-lease TTL. Combined with
	// the lease row's expires_at this yields the time of the most
	// recent heartbeat (= expires_at - LeaseTTL) which is the closest
	// globally-observable proxy for "last tick". Always > 0.
	LeaseTTL time.Duration
}

// WithWorkerHealth wires the LeaderStore + the list of workers
// GetWorkerHealth reports on. Required for the admin health banner; an
// empty list returns an empty response. A nil LeaderStore short-circuits
// GetWorkerHealth to Unimplemented (a deployment that did not opt into
// the health banner stays honest about it).
func (s *PlaytesthubServiceServer) WithWorkerHealth(leases repo.LeaderStore, workers []WorkerInfo) *PlaytesthubServiceServer {
	s.leaderLease = leases
	s.workers = append(s.workers[:0], workers...)
	// clockForWorkerHealth is overridable for unit tests; default is
	// the wall clock. Setting it here keeps NewPlaytesthubServiceServer
	// minimal (callers that never wire workers never pay the field).
	if s.clockForWorkerHealth == nil {
		s.clockForWorkerHealth = time.Now
	}
	return s
}

// GetWorkerHealth returns one entry per registered worker. A worker
// with no leader_lease row (no replica has ever acquired it) surfaces
// as `lease_holder=""` with `stale=true` so a never-ticked worker is
// unmissable in the admin banner.
func (s *PlaytesthubServiceServer) GetWorkerHealth(ctx context.Context, req *pb.GetWorkerHealthRequest) (*pb.GetWorkerHealthResponse, error) {
	if _, err := requireActor(ctx); err != nil {
		return nil, err
	}
	if err := s.checkNamespace(req.GetNamespace()); err != nil {
		return nil, err
	}
	now := s.clockForWorkerHealth()
	entries := make([]*pb.WorkerHealthEntry, 0, len(s.workers))
	for _, w := range s.workers {
		entries = append(entries, s.workerHealthEntry(ctx, w, now))
	}
	return &pb.GetWorkerHealthResponse{Workers: entries}, nil
}

func (s *PlaytesthubServiceServer) workerHealthEntry(ctx context.Context, w WorkerInfo, now time.Time) *pb.WorkerHealthEntry {
	if s.leaderLease == nil {
		return &pb.WorkerHealthEntry{Name: w.Name, Stale: true}
	}
	lease, err := s.leaderLease.Get(ctx, w.Name)
	if err != nil {
		if errors.Is(err, repo.ErrNotFound) {
			return &pb.WorkerHealthEntry{Name: w.Name, Stale: true}
		}
		// Surface store-level errors as stale so the banner fires
		// rather than silently masking the problem.
		return &pb.WorkerHealthEntry{Name: w.Name, Stale: true}
	}
	lastTickAt := lease.AcquiredAt
	if w.LeaseTTL > 0 {
		lastTickAt = lease.ExpiresAt.Add(-w.LeaseTTL)
	}
	return &pb.WorkerHealthEntry{
		Name:        w.Name,
		LeaseHolder: lease.Holder,
		LastTickAt:  timestamppb.New(lastTickAt),
		ExpiresAt:   timestamppb.New(lease.ExpiresAt),
		Stale:       now.After(lease.ExpiresAt.Add(2 * w.TickInterval)),
	}
}
