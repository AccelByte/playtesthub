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
| **M4 — Playtest window enforcement** | PRD §5.1 revision dropping "display-only", validation rules (`endsAt > startsAt`), background `internal/window/` worker reusing `leader_lease` to auto-flip status at boundaries, admin UI relabel + validation + worker-health banner, `pth playtest schedule-info` surface, e2e + smoke coverage with a stub clock. One new read-only admin RPC (`GetWorkerHealth`); no new tables, no new migrations. | in progress — phases 1–3 shipped (backend); phases 4–9 queued |

---

## Phase tracker

Phase status legend: `[ ]` not started · `[~]` in progress · `[x]` shipped.

Each shipped phase carries tests + smoke-harness extension + verification per the TDD + smoke rule in [`engineering.md`](engineering.md) §4–5.

1. `[x]` **PRD §5.1 revision + [`schema.md`](schema.md) note + [`errors.md`](errors.md) rows** — PRD §5.1 `startsAt`/`endsAt` paragraph rewritten with the auto-transition rule + nullable-date matrix; explicit "auto-close does not gate survey submit" carve-out preserved per M3 phase 4. New env var `WINDOW_TICK_SECONDS` documented in PRD §5.9 (default 60; 0 disables the worker). [`schema.md`](schema.md) note added to `playtest.status_transition` describing `actorUserId=NULL` as the M4 system-emitted signal. [`errors.md`](errors.md) row added: `CreatePlaytest`/`EditPlaytest` with `endsAt <= startsAt` → `InvalidArgument` byte-exact `ends_at must be after starts_at`. [`CHANGELOG.md`](CHANGELOG.md) v2.3 entry calls out the backwards-compatibility note (existing playtests with dates populated start auto-transitioning on deploy).
2. `[x]` **Validation at `CreatePlaytest` / `EditPlaytest`** — `validateWindow(startsAt, endsAt)` added in [`pkg/service/validation.go`](../pkg/service/validation.go); wired into both create and edit handlers. Five service-layer tests pin happy paths (starts-only, ends-only) + rejection paths (inverted window create, equal window create, inverted window edit). [`errors.md`](errors.md) byte-exact message asserted in two of them. No proto changes (fields already exist).
3. `[x]` **Window worker (`internal/window/`)** — new `internal/window/window.go` worker mirrors `internal/reclaim/`. Leader-lease election (lease name `window-worker`), `TickInterval = WINDOW_TICK_SECONDS` (default 60s; 0 disables — main.go logs `window_disabled`), virtual-clock injectable for tests. Each tick reads playtests due via new repo method `PlaytestStore.ListDueForAutoTransition(ctx, ns, now)` (DRAFT past `starts_at` + OPEN past `ends_at`, soft-deleted excluded), runs the transition through the existing `TransitionStatus` CAS, and writes a system-attributed `playtest.status_transition` audit row via new helper `repo.AppendStatusTransition(ctx, store, ns, id, actor *uuid.UUID, from, to)`. INFO log on each tick: `{event:"window_tick", advanced, leaseHolder, lastTickAt}`. CAS mismatches are swallowed silently (admin manual transition wins). `LastTickAt()` accessor exposed for phase 5's `GetWorkerHealth`. **Pre-existing bug fixed along the way**: manual `TransitionPlaytestStatus` (M1 phase 6) never wrote its `playtest.status_transition` audit row despite [`schema.md`](schema.md) claiming so — now writes via the same helper with the admin actor id. Smoke harness asserts `"event":"window_started"` boot line in [`scripts/smoke/boot.sh`](../scripts/smoke/boot.sh). Repo integration test `TestPlaytestListDueForAutoTransition` proves the SQL filter against testcontainers. Worker unit tests (12 cases) cover DRAFT→OPEN, OPEN→CLOSED, no-flip without date, CAS-mismatch silence, follower no-sweep, two-workers-one-sweep race, tick-interval gating, log shape, monotonic-forward `nextStatus`.
4. `[ ]` **Admin UI — drop "display-only" label + UTC rendering + cross-field validation + auto-transition preview** — [`admin/src/federated-element.tsx`](../admin/src/federated-element.tsx) lines 375 + 511: relabel `Starts / Ends (display-only in MVP)` → `Starts / Ends (UTC)`. Switch the date display formatter from local `dayjs(value).format(...)` to `dayjs.utc(value).format('YYYY-MM-DD HH:mm')` for these fields, and configure the RangePicker to emit/consume UTC (`showTime` + `dayjs.utc` adapter). Add help text under the field describing the auto-transition rule (one sentence + matrix link). Client-side `endsAt > startsAt` validation matches the server. Playtest detail row gains a tooltip on the status badge: when DRAFT + `startsAt` set, show `Auto-opens <relative time>`; when OPEN + `endsAt` set, show `Auto-closes <relative time>`. Vitest covers UTC formatting, validation rejection of inverted window, and tooltip render. Codegen-fresh gate (M3 phase 15) unaffected — no proto change.
5. `[ ]` **`GetWorkerHealth` RPC + admin worker-health banner** — one new read-only admin RPC `GetWorkerHealth` returning `{name, leaseHolder, lastTickAt, expiresAt, stale}` for each registered worker (today: `reclaim_worker`, `window_worker`). `stale := now > expiresAt + 2*tickInterval`. Implementation reads existing `leader_lease` rows directly — no new table. Admin UI ([`admin/src/federated-element.tsx`](../admin/src/federated-element.tsx)) shows a red Alert banner at the top of every page when any worker is stale: `Window worker hasn't ticked since <relative time>. Auto-transitions are paused — flip status manually via the Publish/Close buttons until ops investigates.` Polled via react-query with a 30s refetch. The banner is purely informational; manual `TransitionPlaytestStatus` continues to work. New row in [`errors.md`](errors.md) is not needed — the RPC has no error conditions beyond auth. Smoke harness probes the route returns 401 unauth; `cloud.sh` extended.
6. `[ ]` **Player Svelte — opens-soon banner (cut-if-behind)** — `Landing.svelte` shows a yellow `role="status"` banner on DRAFT-with-`startsAt` playtests: `Signups open <relative time>`. Pure cosmetic; the existing DRAFT→NotFound visibility rule (PRD §5.1) still applies, so this banner is only visible if the playtest is OPEN and surfaced via a future "preview" mode. **Likely cut**: DRAFT visibility is non-trivial to land here, and the auto-open worker covers the operator concern without it. Tracked as cut-if-behind below.
7. `[ ]` **`pth` CLI — `pth playtest schedule-info`** — new subcommand prints `{slug, status, startsAt, endsAt, nextAutoTransition: {at, to}}` for a single playtest. Read-only; reuses `GetPlaytestForAdmin`. Registry entry, [`cli.md`](cli.md) §6 update, `describe.golden.json` regenerated, `pth.sh` dry-run probe added. Nothing else in the CLI needs M4 changes — `pth playtest create/edit` already accept dates.
8. `[ ]` **E2E + smoke (`pth flow golden-m4`)** — `e2e/golden_m4_test.go` reuses `suiteHarness` from M1/M2/M3. `pth flow golden-m4` extends `golden-m3` with: create-playtest-with-window (`startsAt=now+2s`, `endsAt=now+4s`, leave status=DRAFT) → poll `schedule-info` until status flips to OPEN → poll until status flips to CLOSED → assert audit log contains two `playtest.status_transition` rows with `actor=system`. Worker tick interval overridden to 1s for the test. NDJSON, stop-on-first-failure. `cloud.sh` adds the `GetWorkerHealth` 401 probe. Bounded total runtime ~10s.
9. `[ ]` **CI + README walkthrough (M4 cadence)** — README "Reproduce the golden flow" updated to mention the window-enforcement step (one sentence + link to this doc). No new CI jobs — `e2e/` already in scope from M1 phase 12. Doc + minor copy edits.

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
