// Package window runs the PRD §5.1 "Window-driven auto-transition"
// background job: a single elected leader periodically advances playtest
// status (DRAFT → OPEN, OPEN → CLOSED) when the configured
// `starts_at` / `ends_at` boundary has been reached.
//
// Election uses the leader_lease table — same mechanics as
// internal/reclaim. Every replica runs Run; only the lease holder
// performs the sweep on each tick.
package window

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

// LeaseName is the leader_lease.name value used by the window worker.
const LeaseName = "window-worker"

// Statuses match the string column values used elsewhere in the codebase
// (pkg/service/playtest.go statusDraft/statusOpen/statusClosed). Kept
// as untyped constants to avoid a cross-package import for two strings.
const (
	statusDraft  = "DRAFT"
	statusOpen   = "OPEN"
	statusClosed = "CLOSED"
)

// Config holds the runtime knobs for Run. Every field is required —
// callers populate from pkg/config (which enforces PRD §5.9 defaults).
type Config struct {
	// Namespace bounds the worker to a single AGS namespace. The lease
	// is also per-namespace via the holder identifier (production wires
	// `pod-name + namespace` to keep the election sharp).
	Namespace string

	// HolderID identifies this replica.
	HolderID string

	// LeaseTTL bounds how long a crashed leader can starve the
	// auto-transition sweep. PRD §5.9 default 30s.
	LeaseTTL time.Duration

	// HeartbeatInterval is how often the active leader refreshes the
	// lease. Must be < LeaseTTL with safety margin. PRD §5.9 default 10s.
	HeartbeatInterval time.Duration

	// TickInterval is the cadence at which the leader scans for due
	// transitions. PRD §5.9 `WINDOW_TICK_SECONDS` default 60s.
	TickInterval time.Duration
}

// PlaytestRepo is the slice of repo.PlaytestStore the worker needs.
// Narrowed to keep the unit-test fake minimal.
type PlaytestRepo interface {
	ListDueForAutoTransition(ctx context.Context, namespace string, now time.Time) ([]*repo.Playtest, error)
	TransitionStatus(ctx context.Context, namespace string, id uuid.UUID, from, to string) (*repo.Playtest, error)
}

// Worker is the long-running goroutine driving the auto-transition loop.
// Construct with New, then call Run inside a goroutine; cancel the
// passed context to stop. Worker is safe to construct (and Run) on
// every replica — only the elected leader actually sweeps.
type Worker struct {
	cfg    Config
	leases repo.LeaderStore
	plays  PlaytestRepo
	audit  repo.AuditLogStore
	logger *slog.Logger

	// clock is used by the leader to decide which playtests are due.
	// Production wires time.Now; tests inject a virtual clock.
	clock func() time.Time

	// ticker creates the loop driver. Production uses time.NewTicker;
	// tests inject a deterministic channel they can fire by hand.
	ticker func(d time.Duration) (<-chan time.Time, func())

	// lastTickAt is the last time tick() ran a sweep on this replica.
	// Exposed via LastTickAt for the future GetWorkerHealth RPC
	// (STATUS_M4.md phase 5). Reading it concurrently is fine — only
	// the worker goroutine writes, and stale-by-one-tick reads are
	// acceptable.
	lastTickAt time.Time

	leading bool
}

// New constructs a Worker. logger may be nil — the loop falls back to
// slog.Default(). The audit store is required: every successful
// transition writes a system-attributed playtest.status_transition row.
func New(cfg Config, leases repo.LeaderStore, plays PlaytestRepo, audit repo.AuditLogStore, logger *slog.Logger) *Worker {
	if logger == nil {
		logger = slog.Default()
	}
	return &Worker{
		cfg:    cfg,
		leases: leases,
		plays:  plays,
		audit:  audit,
		logger: logger,
		clock:  time.Now,
		ticker: defaultTicker,
	}
}

// Run drives the loop until ctx is cancelled. Returns nil on clean
// shutdown. Errors during a sweep or heartbeat are logged but never
// abort the loop — the next tick retries.
func (w *Worker) Run(ctx context.Context) error {
	tickCh, stop := w.ticker(w.tickPeriod())
	defer stop()

	w.tryAcquire(ctx)

	for {
		select {
		case <-ctx.Done():
			w.releaseIfLeading(context.Background())
			return nil
		case <-tickCh:
			w.tick(ctx)
		}
	}
}

// LastTickAt returns the last time this replica executed a sweep tick.
// Zero time means "never ticked". Reserved for the future
// GetWorkerHealth admin RPC (STATUS_M4.md phase 5).
func (w *Worker) LastTickAt() time.Time {
	return w.lastTickAt
}

func (w *Worker) tick(ctx context.Context) {
	if !w.leading {
		w.tryAcquire(ctx)
		if !w.leading {
			return
		}
	}
	if !w.heartbeat(ctx) {
		return
	}
	now := w.clock()
	if !w.lastTickAt.IsZero() && now.Sub(w.lastTickAt) < w.cfg.TickInterval {
		// Heartbeat-only tick: lease is fresh, but sweep cadence has
		// not elapsed yet. Mirrors internal/reclaim's pattern so a
		// single ticker drives both heartbeats and sweeps.
		return
	}
	w.lastTickAt = now
	due, err := w.plays.ListDueForAutoTransition(ctx, w.cfg.Namespace, now)
	if err != nil {
		w.logger.LogAttrs(ctx, slog.LevelWarn, "window sweep failed",
			slog.String("event", "window_tick"),
			slog.String("leaseHolder", w.cfg.HolderID),
			slog.String("error", err.Error()),
		)
		return
	}
	advanced := 0
	for _, p := range due {
		next, ok := nextStatus(p.Status)
		if !ok {
			continue
		}
		got, err := w.plays.TransitionStatus(ctx, w.cfg.Namespace, p.ID, p.Status, next)
		if err != nil {
			if errors.Is(err, repo.ErrStatusCASMismatch) {
				// Another writer (admin manual transition, or another
				// worker replica that beat us to it) won the race. The
				// monotonic forward-only invariant still holds; we just
				// skip this row this tick.
				continue
			}
			w.logger.LogAttrs(ctx, slog.LevelWarn, "auto-transition failed",
				slog.String("event", "window_tick"),
				slog.String("leaseHolder", w.cfg.HolderID),
				slog.String("playtestId", p.ID.String()),
				slog.String("from", p.Status),
				slog.String("to", next),
				slog.String("error", err.Error()),
			)
			continue
		}
		// System-emitted audit row: actor=nil per schema.md.
		if auditErr := repo.AppendStatusTransition(ctx, w.audit, w.cfg.Namespace, got.ID, nil, p.Status, got.Status); auditErr != nil {
			w.logger.LogAttrs(ctx, slog.LevelWarn, "auto-transition audit write failed",
				slog.String("event", "window_tick"),
				slog.String("leaseHolder", w.cfg.HolderID),
				slog.String("playtestId", got.ID.String()),
				slog.String("error", auditErr.Error()),
			)
		}
		advanced++
	}
	w.logger.LogAttrs(ctx, slog.LevelInfo, "window tick",
		slog.String("event", "window_tick"),
		slog.String("leaseHolder", w.cfg.HolderID),
		slog.Int("advanced", advanced),
		slog.Time("lastTickAt", now),
	)
}

// nextStatus returns the auto-advance target for a given current status,
// honouring the PRD §5.1 monotonic forward-only invariant. CLOSED has
// no next state — auto-transitions stop there.
func nextStatus(current string) (string, bool) {
	switch current {
	case statusDraft:
		return statusOpen, true
	case statusOpen:
		return statusClosed, true
	default:
		return "", false
	}
}

func (w *Worker) tryAcquire(ctx context.Context) {
	_, err := w.leases.TryAcquire(ctx, LeaseName, w.cfg.HolderID, w.cfg.LeaseTTL)
	if err == nil {
		w.leading = true
		w.logger.LogAttrs(ctx, slog.LevelInfo, "window worker acquired leader lease",
			slog.String("event", "window_lease_acquired"),
			slog.String("leaseHolder", w.cfg.HolderID),
		)
		return
	}
	if !errors.Is(err, repo.ErrLeaseHeld) {
		w.logger.LogAttrs(ctx, slog.LevelWarn, "window lease acquire failed",
			slog.String("event", "window_lease_acquire_failed"),
			slog.String("leaseHolder", w.cfg.HolderID),
			slog.String("error", err.Error()),
		)
	}
	w.leading = false
}

func (w *Worker) heartbeat(ctx context.Context) bool {
	_, err := w.leases.Refresh(ctx, LeaseName, w.cfg.HolderID, w.cfg.LeaseTTL)
	if err == nil {
		return true
	}
	w.leading = false
	if !errors.Is(err, repo.ErrLeaseHeld) {
		w.logger.LogAttrs(ctx, slog.LevelWarn, "window heartbeat failed",
			slog.String("event", "window_heartbeat_failed"),
			slog.String("leaseHolder", w.cfg.HolderID),
			slog.String("error", err.Error()),
		)
		return false
	}
	w.logger.LogAttrs(ctx, slog.LevelInfo, "window worker lost leader lease",
		slog.String("event", "window_lease_lost"),
		slog.String("leaseHolder", w.cfg.HolderID),
	)
	return false
}

func (w *Worker) releaseIfLeading(ctx context.Context) {
	if !w.leading {
		return
	}
	if err := w.leases.Release(ctx, LeaseName, w.cfg.HolderID); err != nil {
		w.logger.LogAttrs(ctx, slog.LevelWarn, "window worker release failed",
			slog.String("event", "window_lease_release_failed"),
			slog.String("error", err.Error()),
		)
	}
	w.leading = false
}

func (w *Worker) tickPeriod() time.Duration {
	if w.cfg.HeartbeatInterval > 0 && w.cfg.HeartbeatInterval < w.cfg.TickInterval {
		return w.cfg.HeartbeatInterval
	}
	return w.cfg.TickInterval
}

func defaultTicker(d time.Duration) (<-chan time.Time, func()) {
	t := time.NewTicker(d)
	return t.C, t.Stop
}
