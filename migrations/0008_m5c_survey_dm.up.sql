-- 0008_m5c_survey_dm — M5 Track D phase 3 schema additions per
-- docs/PRD.md §4.1 step 8 + §5.6 + docs/STATUS_M5.md "Track D — Survey
-- discovery" D3.
--
-- Adds two nullable columns to applicant for the survey-publish DM
-- fan-out:
--
--   last_survey_dm_id  UUID  — the survey.id we last queued a
--                              survey-publish DM for. Tracks "we
--                              attempted delivery for this surveyId",
--                              not "delivery confirmed" — the DM
--                              queue's retry / circuit breaker handles
--                              the latter. Skipping applicants where
--                              last_survey_dm_id == survey.id is what
--                              makes CreateSurvey + the boot-time
--                              restart sweep idempotent.
--   last_survey_dm_at  TIMESTAMPTZ — wall-clock UTC of the most recent
--                              MarkSurveyDMSent for forensics. Not
--                              read by any production code path.
--
-- No FOREIGN KEY on last_survey_dm_id: surveys are versioned (every
-- EditSurvey inserts a new row) and old rows are kept forever per
-- PRD §5.6, so the FK would only fire on a manual delete. The column
-- is informational, not relational.
--
-- A partial index covers the restart-sweep predicate
-- (APPROVED applicants who have never received a survey-publish DM
-- for the current playtest survey) so the sweep stays cheap as the
-- applicant table grows.
--
-- Migrations are append-only (CLAUDE.md) — never edit 0001–0007; fix
-- forward.

ALTER TABLE applicant
    ADD COLUMN last_survey_dm_id UUID,
    ADD COLUMN last_survey_dm_at TIMESTAMPTZ;

CREATE INDEX applicant_pending_survey_dm_idx
    ON applicant (playtest_id)
    WHERE last_survey_dm_id IS NULL AND status = 'APPROVED';
