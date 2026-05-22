-- Rollback for 0008_m5c_survey_dm. Idempotent with IF EXISTS so re-running
-- against a partially-rolled-back schema does not fail.

DROP INDEX IF EXISTS applicant_pending_survey_dm_idx;

ALTER TABLE applicant
    DROP COLUMN IF EXISTS last_survey_dm_at,
    DROP COLUMN IF EXISTS last_survey_dm_id;
