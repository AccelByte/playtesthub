# DM queue mechanics

Detailed Discord DM queue behavior. Referenced from PRD §5.4.

## Bounded in-memory FIFO

- The DM send queue is an **in-memory bounded FIFO**.
- **Default max depth: 10,000** pending sends (tunable via backend configuration).
- Worker drains at a configurable safe rate (default ≈5 DMs/sec; see PRD §5.9) to stay within Discord's DM rate limits.
- Approval RPCs return immediately; DM delivery is asynchronous.

## Throttling

- Discord DM sends are internally throttled.
- Approve RPC enqueues a DM task; a worker drains the queue at a safe rate.
- A DM that fails inside the worker follows the standard `lastDmStatus='failed'` + `applicant.dm_failed` path (PRD §4.1 step 6d), surfaced via the "DM failed" filter.

## Overflow behavior

- When an enqueue would exceed the max depth, the approve flow does **not** block.
- The enqueue **fails immediately**.
- The applicant is marked `lastDmStatus='failed'` with `lastDmError='dm_queue_overflow'`.
- An `applicant.dm_failed` audit row is written (same path as any other DM failure).
- The admin retriages via the "DM failed" filter and can use "Retry DM" once the queue drains.

## Restart behavior (in-memory loss)

- The queue is in-memory and **not persisted**.
- On backend restart or crash, any pending (un-sent) DM tasks are **lost**.

### Startup sweep

On process restart the backend scans all `APPROVED` applicants and re-marks lost DMs:

- **Idempotency guard**: the sweep only re-marks applicants where `lastDmStatus IS NULL` or `'pending'`. Applicants already at `lastDmStatus='failed'` are **not** touched, preserving the original error reason (e.g. `dm_queue_overflow`).
- Affected applicants are transitioned to `lastDmStatus='failed'` with `lastDmError='lost_on_restart'`.
- Standard `applicant.dm_failed` audit row is written per §4.1 step 6d.
- The standard "DM failed" filter and Retry-DM button surface them for re-send.
- No pending-state applicants are hidden from admins.
- The Retry-DM gate (`lastDmStatus='failed' AND status=APPROVED`) is unchanged.

## Circuit breaker

- **Trip condition**: 50 consecutive DM send failures within 60s.
- **Action on trip**: pause queue draining for 5 minutes; auto-resume.
- **Admin work not blocked**: while tripped, new approves still enqueue (so admin work isn't blocked) but DMs don't drain.
- **Surface**: any DM attempted while tripped is marked `lastDmStatus='failed'` with `lastDmError='dm_circuit_open'`.
- **Audit events**: `dm.circuit_opened` and `dm.circuit_closed` (system-attributed).

## Bulk retry RPC

- `RetryFailedDms(playtestId)` retries every applicant with `lastDmStatus='failed'` for the given playtest.
- Reuses per-applicant retry semantics (PRD §5.4 — RetryDM).
- Admin UI exposes a **"Retry all failed DMs"** button on the Applicants page.

## Missing recipient

- A Job whose `discordUserId` is empty (the applicant row has `discord_user_id IS NULL`) is **short-circuited** before the Discord client is invoked.
- Surface: `lastDmStatus='failed'` with `lastDmError='missing_recipient'` plus the standard `applicant.dm_failed` audit row.
- Why nullable: `discord_user_id` lands in migration 0004; rows persisted before that (or signups from non-Discord-federated IAM tokens) carry NULL. The queue treats this as a permanent per-applicant failure rather than an outage. An operator backfilling `discord_user_id` can then run RetryDM to re-enqueue.

## DM body shape — ADT distribution

ADT-distribution playtests substitute the standard "your code is here" copy with a build-download body. ADT returns a list of URLs (one per build asset — single-file builds → one element, multi-asset builds → many) and the DM surfaces every URL in ADT's original order.

- **Single-URL build** (the common case):

  ```text
  Download your playtest build for "Acme Closed Beta": https://cdn.example/build.zip
  ```

- **Multi-URL build** (multi-asset releases — game binary + patcher + manifest, etc.) renders a numbered list, one tappable URL per line so Discord renders each as a separate link:

  ```text
  Download your playtest build for "Acme Closed Beta":
  1) https://cdn.example/build.zip
  2) https://cdn.example/manifest.bin
  3) https://cdn.example/patcher.exe
  ```

`RetryDM` re-mints fresh URLs via `adt.Client.IssueDownloadURL` on every call because the previous URLs may have expired (ADT bounds them with a 24h CDN TTL — PRD §4.8.3). The same numbered-list rendering applies to the re-sent body. The audit row's `adtUrls` array (schema.md `applicant.approve`) records the URL list as minted at approve time; RetryDM does NOT rewrite that audit row even when a fresh resolution returns different URLs.

## DM body shape — survey-link append (Track D phase 2)

When the playtest has a survey configured (`pt.SurveyID` non-nil) the approval DM gains a trailing survey-link line so approved players see the feedback channel in the same message that delivers their code (or ADT build). The append applies to both branches (STEAM_KEYS / AGS_CAMPAIGN and ADT) — every approval DM carries the same nudge. With `PLAYER_BASE_URL` configured the line is a tappable URL anchored at the same hash-router shape as the pending link, with the slug passed through `url.PathEscape` so future grammar relaxations stay safe; without it, the line falls back to a non-clickable nudge.

- **Configured `PLAYER_BASE_URL` + survey set** (the production shape):

  ```text
  You're approved for "Acme Closed Beta". View your code: https://x/#/playtest/acme-beta/pending
  After you've played, share feedback: https://x/#/playtest/acme-beta/survey
  ```

- **Empty `PLAYER_BASE_URL` + survey set** (smoke / offline boots): the second line reads `Share feedback in the playtest hub after you play.` — non-clickable but still surfaces the survey channel. ADT bodies follow the same pattern: download line first, then either the tappable survey URL or the fallback nudge.

Playtests with no survey see the historical single-line body unchanged — the append is a no-op when `pt.SurveyID` is nil.

## DM body shape — survey-publish (Track D phase 3)

`CreateSurvey` fans out a standalone survey-publish DM to every APPROVED + NDA-current applicant whose `applicant.last_survey_dm_id IS DISTINCT FROM` the new `survey.id`. Distinct from the approval DM (which already carries the survey nudge for applicants approved *after* the survey is created — see "survey-link append" above), this body is the discovery channel for the cohort approved *before* the survey was authored. `EditSurvey` is silent — only `CreateSurvey` triggers the fan-out, so iterating on prompt copy never re-DMs the cohort.

- **Configured `PLAYER_BASE_URL`** (the production shape):

  ```text
  Survey is live for "Acme Closed Beta" — share your feedback: https://x/#/playtest/acme-beta/survey
  ```

- **Empty `PLAYER_BASE_URL`** (smoke / offline boots):

  ```text
  Survey is live for "Acme Closed Beta" — open the playtest hub to share your feedback.
  ```

The slug uses `url.PathEscape` matching the approval DM's survey-link append. Idempotency rides on `applicant.last_survey_dm_id`: every successful `Enqueue` is followed by `MarkSurveyDMSent(applicantId, surveyId, now)`. The column tracks "we attempted delivery for this surveyId", not "delivery confirmed" — the queue's retry / circuit breaker handles the latter. Enqueue overflow or sender errors leave `last_survey_dm_id` nil so the **survey-publish restart sweep** (run once at boot before the DM worker starts, alongside the existing `Sweep` for `lost_on_restart`) catches them on the next process boot. The sweep walks every live playtest with `surveyId IS NOT NULL` and re-runs the same `ListApprovedNeedingSurveyDM` predicate — re-running it is a no-op for applicants already stamped for the current survey id. Jobs are enqueued with `Manual=false` so the queue does **not** emit the `applicant.dm_sent` audit row (that's reserved for admin-triggered Retry DM successes per PRD §5.4).

## `lastDmError` truncation

- `lastDmError` is byte-truncated to **500 chars** (PRD §5.2 — Applicant entity).
- Truncation preserves a **valid UTF-8 boundary**: multi-byte codepoints are not cut mid-codepoint. If the truncation point falls inside a multi-byte sequence, the truncation is shifted backward to the nearest codepoint boundary.
