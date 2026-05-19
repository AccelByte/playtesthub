package window

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/anggorodewanto/playtesthub/pkg/repo"
)

const testNS = "test-ns"

// ---------- fakes ------------------------------------------------------------

type fakeLeaseStore struct {
	mu              sync.Mutex
	currentHolder   string
	expiresAt       time.Time
	tryAcquireCalls int
	refreshCalls    int
	releaseCalls    int
}

func (f *fakeLeaseStore) TryAcquire(_ context.Context, _, holder string, ttl time.Duration) (*repo.LeaderLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.tryAcquireCalls++
	if f.currentHolder != "" && f.currentHolder != holder && time.Now().Before(f.expiresAt) {
		return nil, repo.ErrLeaseHeld
	}
	f.currentHolder = holder
	f.expiresAt = time.Now().Add(ttl)
	return &repo.LeaderLease{Holder: holder, ExpiresAt: f.expiresAt}, nil
}

func (f *fakeLeaseStore) Refresh(_ context.Context, _, holder string, ttl time.Duration) (*repo.LeaderLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.refreshCalls++
	if f.currentHolder != holder {
		return nil, repo.ErrLeaseHeld
	}
	f.expiresAt = time.Now().Add(ttl)
	return &repo.LeaderLease{Holder: holder, ExpiresAt: f.expiresAt}, nil
}

func (f *fakeLeaseStore) Release(_ context.Context, _, holder string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.releaseCalls++
	if f.currentHolder == holder {
		f.currentHolder = ""
	}
	return nil
}

func (f *fakeLeaseStore) Get(_ context.Context, _ string) (*repo.LeaderLease, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.currentHolder == "" {
		return nil, repo.ErrNotFound
	}
	return &repo.LeaderLease{Holder: f.currentHolder, ExpiresAt: f.expiresAt}, nil
}

type fakePlaytestRepo struct {
	mu               sync.Mutex
	rows             []*repo.Playtest
	transitionCalls  int
	transitionErrors map[uuid.UUID]error
}

func (f *fakePlaytestRepo) ListDueForAutoTransition(_ context.Context, namespace string, now time.Time) ([]*repo.Playtest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]*repo.Playtest, 0)
	for _, r := range f.rows {
		if r.Namespace != namespace || r.DeletedAt != nil {
			continue
		}
		switch r.Status {
		case statusDraft:
			if r.StartsAt != nil && !r.StartsAt.After(now) {
				clone := *r
				out = append(out, &clone)
			}
		case statusOpen:
			if r.EndsAt != nil && !r.EndsAt.After(now) {
				clone := *r
				out = append(out, &clone)
			}
		}
	}
	return out, nil
}

func (f *fakePlaytestRepo) TransitionStatus(_ context.Context, namespace string, id uuid.UUID, from, to string) (*repo.Playtest, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.transitionCalls++
	if err, ok := f.transitionErrors[id]; ok {
		return nil, err
	}
	for _, r := range f.rows {
		if r.Namespace == namespace && r.ID == id && r.DeletedAt == nil {
			if r.Status != from {
				return nil, repo.ErrStatusCASMismatch
			}
			r.Status = to
			r.UpdatedAt = time.Now()
			clone := *r
			return &clone, nil
		}
	}
	return nil, repo.ErrStatusCASMismatch
}

type fakeAuditStore struct {
	mu      sync.Mutex
	entries []*repo.AuditLog
}

func (f *fakeAuditStore) Append(_ context.Context, row *repo.AuditLog) (*repo.AuditLog, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	clone := *row
	clone.ID = uuid.New()
	clone.CreatedAt = time.Now()
	f.entries = append(f.entries, &clone)
	ret := clone
	return &ret, nil
}

func (f *fakeAuditStore) List(_ context.Context, _ repo.AuditLogPageQuery) (*repo.AuditLogPage, error) {
	return &repo.AuditLogPage{}, nil
}

func (f *fakeAuditStore) ListByPlaytest(_ context.Context, _ uuid.UUID, _ int) ([]*repo.AuditLog, error) {
	return nil, nil
}

// ---------- harness ----------------------------------------------------------

type manualTicker struct {
	ch chan time.Time
}

func newManualTicker() *manualTicker {
	return &manualTicker{ch: make(chan time.Time, 8)}
}

func (m *manualTicker) factory() func(time.Duration) (<-chan time.Time, func()) {
	return func(_ time.Duration) (<-chan time.Time, func()) {
		return m.ch, func() {}
	}
}

func (m *manualTicker) tick() {
	m.ch <- time.Now()
}

type virtualClock struct {
	mu  sync.Mutex
	now time.Time
}

func newVirtualClock(start time.Time) *virtualClock {
	return &virtualClock{now: start}
}

func (c *virtualClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *virtualClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func newWorker(t *testing.T, cfg Config, leases repo.LeaderStore, plays PlaytestRepo, audit repo.AuditLogStore, mt *manualTicker, buf *bytes.Buffer, vc *virtualClock) *Worker {
	t.Helper()
	logger := slog.New(slog.NewJSONHandler(buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	w := New(cfg, leases, plays, audit, logger)
	w.ticker = mt.factory()
	if vc != nil {
		w.clock = vc.Now
	}
	return w
}

func defaultCfg() Config {
	return Config{
		Namespace:         testNS,
		HolderID:          "pod-a",
		LeaseTTL:          30 * time.Second,
		HeartbeatInterval: 10 * time.Second,
		TickInterval:      60 * time.Second,
	}
}

func runWorkerInBackground(t *testing.T, w *Worker) func() {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		_ = w.Run(ctx)
		close(done)
	}()
	return func() {
		cancel()
		select {
		case <-done:
		case <-time.After(2 * time.Second):
			t.Fatal("worker did not exit after cancel")
		}
	}
}

func waitFor(t *testing.T, cond func() bool, msg string) {
	t.Helper()
	for range 100 {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", msg)
}

// ---------- tests ------------------------------------------------------------

func TestWorker_AdvancesDraftToOpenWhenStartsAtReached(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(-time.Minute)
	id := uuid.New()
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: id, Namespace: testNS, Status: statusDraft, StartsAt: &startsAt,
		}},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()

	waitFor(t, func() bool {
		plays.mu.Lock()
		defer plays.mu.Unlock()
		return plays.transitionCalls >= 1
	}, "TransitionStatus to be called")

	plays.mu.Lock()
	if got := plays.rows[0].Status; got != statusOpen {
		t.Errorf("status = %q, want OPEN; log: %s", got, buf.String())
	}
	plays.mu.Unlock()

	audit.mu.Lock()
	defer audit.mu.Unlock()
	if len(audit.entries) != 1 {
		t.Fatalf("audit entries = %d, want 1", len(audit.entries))
	}
	if got := audit.entries[0].Action; got != repo.ActionPlaytestStatusTransition {
		t.Errorf("audit action = %q, want %q", got, repo.ActionPlaytestStatusTransition)
	}
	if audit.entries[0].ActorUserID != nil {
		t.Errorf("audit actor = %v, want nil (system-emitted)", audit.entries[0].ActorUserID)
	}
}

func TestWorker_AdvancesOpenToClosedWhenEndsAtReached(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	endsAt := now.Add(-time.Second)
	id := uuid.New()
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: id, Namespace: testNS, Status: statusOpen, EndsAt: &endsAt,
		}},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()

	waitFor(t, func() bool {
		plays.mu.Lock()
		defer plays.mu.Unlock()
		return plays.rows[0].Status == statusClosed
	}, "OPEN → CLOSED transition")
}

func TestWorker_DoesNotTransitionPlaytestWithoutDate(t *testing.T) {
	// DRAFT with no starts_at must stay DRAFT — pure manual mode.
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	id := uuid.New()
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: id, Namespace: testNS, Status: statusDraft, // no StartsAt
		}},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()
	// Allow the tick to settle — without sleeping, we can't prove a
	// no-op, so we wait long enough that any transition would have
	// happened, then assert it didn't.
	time.Sleep(150 * time.Millisecond)

	plays.mu.Lock()
	if got := plays.rows[0].Status; got != statusDraft {
		t.Errorf("status = %q, want DRAFT (no auto-flip without starts_at)", got)
	}
	if plays.transitionCalls != 0 {
		t.Errorf("TransitionStatus calls = %d, want 0", plays.transitionCalls)
	}
	plays.mu.Unlock()
}

func TestWorker_RespectsCASMismatchSilently(t *testing.T) {
	// Simulate the race where an admin's manual transition has already
	// moved the row past DRAFT before the worker's CAS runs. The CAS
	// should return ErrStatusCASMismatch, and the worker should swallow
	// it silently (no WARN log, no audit row).
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(-time.Minute)
	id := uuid.New()
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: id, Namespace: testNS, Status: statusDraft, StartsAt: &startsAt,
		}},
		transitionErrors: map[uuid.UUID]error{id: repo.ErrStatusCASMismatch},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()

	waitFor(t, func() bool {
		plays.mu.Lock()
		defer plays.mu.Unlock()
		return plays.transitionCalls >= 1
	}, "TransitionStatus attempt")

	audit.mu.Lock()
	defer audit.mu.Unlock()
	if len(audit.entries) != 0 {
		t.Errorf("audit entries = %d, want 0 on CAS miss", len(audit.entries))
	}
	if strings.Contains(buf.String(), "auto-transition failed") {
		t.Errorf("CAS mismatch should be silent, not WARN; log: %s", buf.String())
	}
}

func TestWorker_FollowerDoesNotSweep(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(-time.Minute)
	id := uuid.New()
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: id, Namespace: testNS, Status: statusDraft, StartsAt: &startsAt,
		}},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{
		currentHolder: "pod-other",
		expiresAt:     time.Now().Add(time.Hour),
	}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()
	time.Sleep(100 * time.Millisecond)

	plays.mu.Lock()
	if plays.transitionCalls != 0 {
		t.Errorf("follower ran sweep: calls=%d", plays.transitionCalls)
	}
	plays.mu.Unlock()
}

func TestWorker_LastTickAtAdvancesWhenLeading(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	plays := &fakePlaytestRepo{}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()

	waitFor(t, func() bool {
		return !w.LastTickAt().IsZero()
	}, "LastTickAt to advance from zero")

	if !w.LastTickAt().Equal(now) {
		t.Errorf("LastTickAt = %v, want %v", w.LastTickAt(), now)
	}
}

func TestWorker_LogsWindowTickEvent(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	plays := &fakePlaytestRepo{}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()

	waitFor(t, func() bool {
		return strings.Contains(buf.String(), `"event":"window_tick"`)
	}, "window_tick log line")

	if !strings.Contains(buf.String(), `"advanced":0`) {
		t.Errorf("expected advanced=0 in log; got %s", buf.String())
	}
}

func TestNextStatus_MonotonicForwardOnly(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{statusDraft, statusOpen, true},
		{statusOpen, statusClosed, true},
		{statusClosed, "", false},
		{"WEIRD", "", false},
	}
	for _, c := range cases {
		got, ok := nextStatus(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("nextStatus(%q) = (%q,%v), want (%q,%v)", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestWorker_TickIntervalGatesSweep(t *testing.T) {
	// Two ticks back-to-back with virtual clock not advancing past
	// TickInterval should result in exactly one sweep.
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(-time.Minute)
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: uuid.New(), Namespace: testNS, Status: statusDraft, StartsAt: &startsAt,
		}},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()
	waitFor(t, func() bool {
		plays.mu.Lock()
		defer plays.mu.Unlock()
		return plays.transitionCalls >= 1
	}, "first tick to advance")

	// Second tick within the same TickInterval: should be heartbeat-only
	// (no new transition call because the only DRAFT row already
	// advanced to OPEN; even if there were more, the gate would block).
	transitionCallsAfterFirst := plays.transitionCalls
	vc.Advance(time.Second) // < TickInterval (60s)
	mt.tick()
	time.Sleep(100 * time.Millisecond)

	plays.mu.Lock()
	defer plays.mu.Unlock()
	if plays.transitionCalls != transitionCallsAfterFirst {
		t.Errorf("transitionCalls = %d, want %d (sweep gate must hold)", plays.transitionCalls, transitionCallsAfterFirst)
	}
}

// Concurrent-access sanity: two Workers with different holders, only
// one acquires the lease. The other stays follower and never sweeps.
func TestTwoWorkers_OnlyOneSweeps(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(-time.Minute)
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: uuid.New(), Namespace: testNS, Status: statusDraft, StartsAt: &startsAt,
		}},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt1 := newManualTicker()
	mt2 := newManualTicker()
	vc := newVirtualClock(now)
	cfgA := defaultCfg()
	cfgA.HolderID = "pod-a"
	cfgB := defaultCfg()
	cfgB.HolderID = "pod-b"
	wA := newWorker(t, cfgA, leases, plays, audit, mt1, &bytes.Buffer{}, vc)
	wB := newWorker(t, cfgB, leases, plays, audit, mt2, &bytes.Buffer{}, vc)

	stopA := runWorkerInBackground(t, wA)
	defer stopA()
	stopB := runWorkerInBackground(t, wB)
	defer stopB()

	mt1.tick()
	mt2.tick()
	waitFor(t, func() bool {
		plays.mu.Lock()
		defer plays.mu.Unlock()
		return plays.rows[0].Status == statusOpen
	}, "exactly one worker to win the transition")

	plays.mu.Lock()
	defer plays.mu.Unlock()
	if plays.transitionCalls != 1 {
		t.Errorf("transitionCalls = %d, want exactly 1 (CAS dedup on race)", plays.transitionCalls)
	}
}

// Sanity: a TransitionStatus error other than CAS-mismatch logs a WARN
// but does not block subsequent rows. We can't easily prove "subsequent
// rows" without seeding two due rows, so this also exercises the WARN
// path.
func TestWorker_LogsWarnOnUnexpectedTransitionError(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(-time.Minute)
	id := uuid.New()
	plays := &fakePlaytestRepo{
		rows: []*repo.Playtest{{
			ID: id, Namespace: testNS, Status: statusDraft, StartsAt: &startsAt,
		}},
		transitionErrors: map[uuid.UUID]error{id: errors.New("boom")},
	}
	audit := &fakeAuditStore{}
	leases := &fakeLeaseStore{}
	mt := newManualTicker()
	vc := newVirtualClock(now)
	buf := &bytes.Buffer{}
	w := newWorker(t, defaultCfg(), leases, plays, audit, mt, buf, vc)

	stop := runWorkerInBackground(t, w)
	defer stop()
	mt.tick()

	waitFor(t, func() bool {
		return strings.Contains(buf.String(), "auto-transition failed")
	}, "WARN log on transient error")
}
