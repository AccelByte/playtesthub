-- Rollback for 0006_m5b_adt. No IF EXISTS — a clean rollback fails
-- loudly on schema drift.

DROP INDEX adt_link_pending_expires_at_idx;
DROP TABLE adt_link_pending;

DROP INDEX adt_linkage_studio_adt_uniq;
DROP TABLE adt_linkage;

ALTER TABLE playtest
    DROP CONSTRAINT playtest_adt_namespace_model,
    DROP COLUMN adt_fallback_download_url,
    DROP COLUMN adt_build_id,
    DROP COLUMN adt_game_id,
    DROP COLUMN adt_namespace;

ALTER TABLE playtest
    DROP CONSTRAINT playtest_initial_code_quantity_model,
    ADD CONSTRAINT playtest_initial_code_quantity_model
        CHECK (
            (distribution_model = 'STEAM_KEYS'   AND initial_code_quantity IS NULL) OR
            (distribution_model = 'AGS_CAMPAIGN' AND initial_code_quantity BETWEEN 1 AND 50000)
        );

ALTER TABLE playtest
    DROP CONSTRAINT playtest_distribution_model_enum,
    ADD CONSTRAINT playtest_distribution_model_enum
        CHECK (distribution_model IN ('STEAM_KEYS', 'AGS_CAMPAIGN'));
