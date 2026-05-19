# playtesthub — M4 Build Status (scoping)

Phase tracker for **M4 — Playtest window enforcement**: turn `startsAt` / `endsAt` from display-only fields into the drivers of automatic `DRAFT → OPEN → CLOSED` status transitions. Conventions match [`STATUS.md`](STATUS.md) (one-line outcome per phase, pointer to the canonical doc for the *what* and *how*). This is a **scoping document** — every phase is `[ ]` not started until the PRD revision in phase 1 lands.

## Why M4

Today `Playtest.startsAt` / `Playtest.endsAt` are stored and surfaced but never consulted (PRD §5.1: *"display-only in MVP — do NOT gate signup, NDA accept, approve, reject, or survey submit; only OPEN/CLOSED status gates lifecycle"*). The admin UI labels them honestly (`Starts / Ends (display-only in MVP)` at `admin/src/federated-element.tsx:375` + `:511`), but the field is a foot-gun: operators set dates expecting them to take effect, then watch nothing happen at the boundary.

M4 closes the gap by making **status the gate it already is, but driven by the dates** — a single source of truth, not two competing ones.

## Design choice (resolved before phase 1)

Three options were on the table:

| Option | Summary | Verdict |
| --- | --- | --- |
| **A. Auto-transition status from dates** | Background worker flips `status` at boundary; existing OPEN/CLOSED gating handles the rest. | **Chosen.** Single source of truth. No double-gate divergence. Reuses existing `TransitionPlaytestStatus` audit + every RPC-level OPEN/CLOSED check landed in M1/M2/M3. |
| B. Reject RPCs outside the window | Keep status; reject `Signup` / etc. when `now < startsAt OR now > endsAt`. | Rejected. Creates two independent gates that can disagree, doubles the error-path surface in [`errors.md`](errors.md), and forces every existing OPEN/CLOSED test to re-prove "but what if dates also disagree". |
| C. Hybrid | Auto-transition + RPC-time check. | Rejected. Strict superset of (A)'s complexity for no win. |

**Manual override semantics**: the worker only advances `DRAFT → OPEN` (when `startsAt` set + reached + status=DRAFT) and `OPEN → CLOSED` (when `endsAt` set + reached + status=OPEN). Admin manual transitions stay forward-only and take precedence — the worker never reverts or fights an admin choice. Status remains a monotonic forward state machine.

**Nullable date semantics**:

| `startsAt` | `endsAt` | Behavior |
| --- | --- | --- |
| set | set | full auto: open at start, close at end |
| set | NULL | auto-open at start, manual-only close |
| NULL | set | manual-only open, auto-close at end |
| NULL | NULL | full manual (current behavior — backwards-compat for every M1/M2/M3 playtest) |

## Status legend

- `not started` — no code written yet
- `in progress` — some code landed; milestone not yet complete
- `shipped` — milestone deliverables merged and demoed

## Milestone

| Milestone | Scope (summary) | Status |
| --- | --- | ------ |
| **M4 — Playtest window enforcement** | PRD §5.1 revision dropping "display-only", validation rules (`endsAt > startsAt`), background `internal/window/` worker reusing `leader_lease` to auto-flip status at boundaries, admin UI relabel + validation + worker-health banner, `pth playtest schedule-info` surface, e2e + smoke coverage with a stub clock. One new read-only admin RPC (`GetWorkerHealth`); no new tables, no new migrations. | shipped — phases 1–5 + 7–9 done; phase 6 deferred as cosmetic cut-if-behind |

---

## Phase tracker

Phase status legend: `[ ]` not started · `[~]` in progress · `[x]` shipped.

Each shipped phase carries tests + smoke-harness extension + verification per the TDD + smoke rule in [`engineering.md`](engineering.md) §4–5.

1. `[x]` **PRD §5.1 revision + [`schema.md`](schema.md) note + [`errors.md`](errors.md) rows** — PRD §5.1 `startsAt`/`endsAt` paragraph rewritten with the auto-transition rule + nullable-date matrix; explicit "auto-close does not gate survey submit" carve-out preserved per M3 phase 4. New env var `WINDOW_TICK_SECONDS` documented in PRD §5.9 (default 60; 0 disables the worker). [`schema.md`](schema.md) note added to `playtest.status_transition` describing `actorUserId=NULL` as the M4 system-emitted signal. [`errors.md`](errors.md) row added: `CreatePlaytest`/`EditPlaytest` with `endsAt <= startsAt` → `InvalidArgument` byte-exact `ends_at must be after starts_at`. [`CHANGELOG.md`](CHANGELOG.md) v2.3 entry calls out the backwards-compatibility note (existing playtests with dates populated start auto-transitioning on deploy).
2. `[x]` **Validation at `CreatePlaytest` / `EditPlaytest`** — `validateWindow(startsAt, endsAt)` added in [`pkg/service/validation.go`](../pkg/service/validation.go); wired into both create and edit handlers. Five service-layer tests pin happy paths (starts-only, ends-only) + rejection paths (inverted window create, equal window create, inverted window edit). [`errors.md`](errors.md) byte-exact message asserted in two of them. No proto changes (fields already exist).
3. `[x]` **Window worker (`internal/window/`)** — new `internal/window/window.go` worker mirrors `internal/reclaim/`. Leader-lease election (lease name `window-worker`), `TickInterval = WINDOW_TICK_SECONDS` (default 60s; 0 disables — main.go logs `window_disabled`), virtual-clock injectable for tests. Each tick reads playtests due via new repo method `PlaytestStore.ListDueForAutoTransition(ctx, ns, now)` (DRAFT past `starts_at` + OPEN past `ends_at`, soft-deleted excluded), runs the transition through the existing `TransitionStatus` CAS, and writes a system-attributed `playtest.status_transition` audit row via new helper `repo.AppendStatusTransition(ctx, store, ns, id, actor *uuid.UUID, from, to)`. INFO log on each tick: `{event:"window_tick", advanced, leaseHolder, lastTickAt}`. CAS mismatches are swallowed silently (admin manual transition wins). `LastTickAt()` accessor exposed for phase 5's `GetWorkerHealth`. **Pre-existing bug fixed along the way**: manual `TransitionPlaytestStatus` (M1 phase 6) never wrote its `playtest.status_transition` audit row despite [`schema.md`](schema.md) claiming so — now writes via the same helper with the admin actor id. Smoke harness asserts `"event":"window_started"` boot line in [`scripts/smoke/boot.sh`](../scripts/smoke/boot.sh). Repo integration test `TestPlaytestListDueForAutoTransition` proves the SQL filter against testcontainers. Worker unit tests (12 cases) cover DRAFT→OPEN, OPEN→CLOSED, no-flip without date, CAS-mismatch silence, follower no-sweep, two-workers-one-sweep race, tick-interval gating, log shape, monotonic-forward `nextStatus`.
4. `[x]` **Admin UI — drop "display-only" label + UTC rendering + cross-field validation + auto-transition preview** — extracted to [`admin/src/window.ts`](../admin/src/window.ts) so the file-level export rule (`react-refresh/only-export-components`) stays clean: `dayjs.extend(utc) + extend(relativeTime)`, `DATE_RANGE_LABEL = 'Starts / Ends (UTC)'`, the cross-field validator emitting the byte-exact server error `ends_at must be after starts_at`, a `getValueFromEvent` adapter that pins user-typed wall-clock values as UTC via `v.utc(true)`, and `autoTransitionPreview`. [`admin/src/federated-element.tsx`](../admin/src/federated-element.tsx) consumes them: both create and edit forms relabel the RangePicker (no more "display-only in MVP"), wire the validator + UTC adapter + help text (`Auto-opens at start, auto-closes at end. Leave either side empty to control that boundary manually.`), and the playtests list status column wraps the existing `<Tag>` in an antd `<Tooltip>` showing `Auto-opens <relative>` on DRAFT-with-`startsAt` or `Auto-closes <relative>` on OPEN-with-`endsAt`. Edit preload swaps `dayjs(...)` for `dayjs.utc(...)` so prefilled times render in UTC. Vitest gains 7 cases (total admin suite 50): UTC label on create + edit, byte-exact validator rejection for inverted + equal windows + acceptance of half-open ones, tooltip render on DRAFT + OPEN, and the negative case on DRAFT-without-`startsAt`. Codegen-fresh gate unaffected (no proto change).
5. `[x]` **`GetWorkerHealth` RPC + admin worker-health banner** — proto: `GetWorkerHealth(GetWorkerHealthRequest) → GetWorkerHealthResponse` with `repeated WorkerHealthEntry workers` (each `{name, lease_holder, last_tick_at, expires_at, stale}`); HTTP `GET /v1/admin/namespaces/{namespace}/workers/health`, admin-permissioned. Handler in [`pkg/service/worker_health.go`](../pkg/service/worker_health.go) reads existing `leader_lease` rows directly via `repo.LeaderStore.Get`; `stale := now > expires_at + 2*tick_interval`. A missing lease row (never elected) surfaces as `lease_holder=""` + `stale=true` so a never-ticked worker is unmissable. Service gains `WithWorkerHealth(leases, []WorkerInfo)` + an overridable clock for tests; bootapp constructs the registry from cfg (always `reclaim-job`; `window-worker` only when `WINDOW_TICK_SECONDS > 0`). 6 unit tests cover auth + namespace check + no-leases-all-stale + fresh + past-threshold + inside-grace boundaries. Admin UI: `WorkerHealthBanner` polls every 30s and renders a red `data-testid="worker-health-banner"` Alert listing every stale worker name with the load-bearing copy *"Auto-transitions are paused — flip status manually via the Publish/Close buttons until ops investigates."* Vitest adds 2 cases (banner shows on stale, hides on all-fresh). Smoke: `cloud.sh` adds the `GET …/workers/health` 401 probe under a new M4 RPC array. No new [`errors.md`](errors.md) rows.
6. `[ ]` **Player Svelte — opens-soon banner (cut-if-behind)** — `Landing.svelte` shows a yellow `role="status"` banner on DRAFT-with-`startsAt` playtests: `Signups open <relative time>`. Pure cosmetic; the existing DRAFT→NotFound visibility rule (PRD §5.1) still applies, so this banner is only visible if the playtest is OPEN and surfaced via a future "preview" mode. **Cut**: DRAFT visibility is non-trivial to land here, and the auto-open worker covers the operator concern without it. Tracked as cut-if-behind below.
7. `[x]` **`pth` CLI — `pth playtest schedule-info`** — [`cmd/pth/playtest.go`](../cmd/pth/playtest.go) adds the subcommand. Reads `AdminGetPlaytest` under the hood; emits a single JSON object `{slug, status, startsAt, endsAt, nextAutoTransition}` where `nextAutoTransition` is `{at, to}` for DRAFT-with-`startsAt` (→ OPEN) and OPEN-with-`endsAt` (→ CLOSED), `null` otherwise. `--dry-run` echoes the underlying `AdminGetPlaytestRequest`. Registry + golden file regenerated; `pth.sh` adds a dry-run probe; 4 new unit tests cover dry-run + DRAFT-with-startsAt + OPEN-with-endsAt + the no-dates-no-next path.
8. `[x]` **E2E + smoke (`pth flow golden-m4`)** — [`cmd/pth/flow.go`](../cmd/pth/flow.go) gains `runFlowGoldenM4`: four NDJSON steps — `create-playtest` (DRAFT + `startsAt=now+start-offset` + `endsAt=now+end-offset`) → `await-auto-open` (polls `AdminGetPlaytest` until `status=OPEN`) → `await-auto-close` → `assert-system-transitions` (lists audit log filtered to `actor=system, action=playtest.status_transition`; asserts ≥2 entries). The flow is admin-only by design (window enforcement is server-driven). [`e2e/main_test.go`](../e2e/main_test.go) starts an in-process `window.Worker` (TickInterval=1s, LeaseTTL=30s) so the bounded ~10s budget holds for the harness; [`e2e/golden_m4_test.go`](../e2e/golden_m4_test.go) reuses `suiteHarness` and drives admin login → `pth flow golden-m4`. 3 CLI unit tests pin dry-run shape, the polling happy path with stubbed AdminGet calls, and the timeout-renders-`DeadlineExceeded`-FAILED-line failure path. `pth.sh` extends the dry-run probe (4 NDJSON lines, `status=DRY_RUN`). `cloud.sh` GetWorkerHealth probe already landed in phase 5.
9. `[x]` **README walkthrough (M4 cadence)** — README "Reproduce the golden flow" now lists `e2e/golden_m4_test.go` alongside the M1/M2/M3 entries and adds a "Window enforcement (M4)" paragraph explaining that `Playtest.startsAt`/`endsAt` are no longer display-only and pointing operators at `pth flow golden-m4`. No new CI jobs — `e2e/` was already in scope from M1 phase 12.

---

## Cross-cutting rules

- **Minimal new surface.** One new read-only admin RPC (`GetWorkerHealth`, phase 5) for the health banner. No new mutating RPCs, no new tables, no new migrations, no new fields on existing playtest/applicant/code entities.
- **Smoke harness lands with the code that introduces it** ([CLAUDE.md](../CLAUDE.md), [`engineering.md`](engineering.md) §5.1). Phase 3's window worker boot-wiring assertion is the load-bearing one.
- **Clock injection over `time.Sleep` in tests.** The window worker takes a `Clock` interface; tests run a stub clock that fast-forwards instantly. No `time.Sleep` in unit/integration tests; the e2e test in phase 7 is the only place real seconds elapse (and it stays under ~10s by setting the tick interval to 1s).
- **Forward-only status remains invariant.** The worker only advances; it never reverts. Admin manual transitions (`TransitionPlaytestStatus`) keep their existing semantics — they win against any worker tick because the CAS predicate matches the prior status only.
- **Backwards compatibility for existing playtests.** Every M1/M2/M3 playtest with `startsAt`/`endsAt` already populated will start auto-transitioning the moment phase 3 deploys. Operators who relied on "dates are decorative" need a heads-up — the PRD §5.1 revision in phase 1 is the place that surfaces this, and the v2.2 [`CHANGELOG.md`](CHANGELOG.md) entry calls it out as a behavior change.

## Cut-if-behind tracking

#### M4

- **Player opens-soon banner (phase 6)** — cosmetic; the auto-open worker is the operator-facing deliverable. Cut without loss of M4's core value.
- **Admin auto-transition preview tooltip (phase 4 sub-bullet)** — relabel + UTC rendering + validation are the load-bearing parts of phase 4; the tooltip is informational. Ship the load-bearing pieces, defer the tooltip if behind.
- **`pth playtest schedule-info` (phase 7)** — `pth playtest get` already returns the dates + status. A dedicated subcommand is ergonomics, not capability. Defer if behind.

## Resolved decisions (2026-05-19 scoping interview)

1. **Time-zone display in admin → UTC.** Admin renders `startsAt` / `endsAt` in UTC with an explicit `(UTC)` field-label suffix. Server already stores UTC (timestamptz); current admin code renders in browser-local via `dayjs(value).format(...)` — phase 4 swaps to `dayjs.utc(value)` and configures the RangePicker UTC adapter. Removes the foot-gun where operators set "8pm" in their local timezone and the auto-transition fires at a different wall-clock moment than they expected.
2. **`endsAt` auto-close does NOT gate survey submit.** Confirmed the M3 phase 4 invariant: APPROVED applicants can submit surveys post-CLOSED. Phase 1's PRD §5.1 revision spells this out explicitly so future readers don't infer "auto-close = hard cutoff for everything".
3. **Worker-down mode → soft-degrade + admin warning banner.** Status sticks at whatever value it has; manual `TransitionPlaytestStatus` still works. Phase 5 lands the `GetWorkerHealth` admin RPC + red Alert banner that surfaces when any worker is stale (`now > expiresAt + 2*tickInterval`). Operators get an unmissable signal something's wrong without coupling backend lifecycle to worker health.
