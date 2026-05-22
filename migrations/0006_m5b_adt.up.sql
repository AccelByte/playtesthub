-- 0006_m5b_adt — M5.B schema: ADT distribution model + linkage tables.
-- Shape reference: docs/schema.md §"adt_linkage table" / §"adt_link_pending
-- table" and PRD §4.8 / §5.1 (ADT fields on Playtest).
--
-- Adds:
--   playtest.adt_namespace, .adt_game_id, .adt_build_id,
--     .adt_fallback_download_url — model-specific identifiers + static
--     fallback URL (PRD §5.1).
--   adt_linkage              — identity-only row for a successful
--                              studio ↔ ADT-namespace link. NO credential
--                              columns by design (PRD §4.8.2 — auth to
--                              ADT on every API call is the AGS service
--                              IAM JWT).
--   adt_link_pending         — short-lived nonce store for the linking
--                              redirect round-trip; rows older than
--                              ADT_LINKAGE_PENDING_TTL_SECONDS swept
--                              inline by CompleteADTLink.
--
-- Migrations are append-only (CLAUDE.md) — never edit 0001–0005; fix
-- forward by dropping and re-adding the affected CHECK constraints below.

-- Extend distribution_model enum to allow 'ADT'.
ALTER TABLE playtest
    DROP CONSTRAINT playtest_distribution_model_enum,
    ADD CONSTRAINT playtest_distribution_model_enum
        CHECK (distribution_model IN ('STEAM_KEYS', 'AGS_CAMPAIGN', 'ADT'));

-- Relax initial_code_quantity_model: STEAM_KEYS / AGS_CAMPAIGN rules
-- unchanged; ADT carries no code pool (PRD §5.5) so initial_code_quantity
-- must be NULL.
ALTER TABLE playtest
    DROP CONSTRAINT playtest_initial_code_quantity_model,
    ADD CONSTRAINT playtest_initial_code_quantity_model
        CHECK (
            (distribution_model = 'STEAM_KEYS'   AND initial_code_quantity IS NULL) OR
            (distribution_model = 'AGS_CAMPAIGN' AND initial_code_quantity BETWEEN 1 AND 50000) OR
            (distribution_model = 'ADT'          AND initial_code_quantity IS NULL)
        );

-- ADT identifiers + static fallback URL. Three identifiers are immutable
-- post-create (mirrors distribution_model / ags_item_id — see PRD §5.1
-- EditPlaytest whitelist); adt_fallback_download_url is mutable via
-- EditPlaytest so operators can repoint without recreating the playtest.
ALTER TABLE playtest
    ADD COLUMN adt_namespace             TEXT,
    ADD COLUMN adt_game_id               TEXT,
    ADD COLUMN adt_build_id              TEXT,
    ADD COLUMN adt_fallback_download_url TEXT,
    -- PRD §5.1 / STATUS_M5.md B2: model↔fields invariant. Anchored on
    -- adt_namespace alone per the B2 contract; service-layer
    -- validateADTFields covers the full (namespace, game_id, build_id)
    -- triple at create time.
    ADD CONSTRAINT playtest_adt_namespace_model
        CHECK ((distribution_model = 'ADT') = (adt_namespace IS NOT NULL));

-- adt_linkage ------------------------------------------------------------
-- Identity row for a successful studio ↔ ADT-namespace link. NO
-- adt_credential_* columns: every ADT API call is authed by a freshly
-- minted AGS service IAM JWT (PRD §4.8.2 / schema.md §"adt_linkage
-- table"). Migration test asserts this absence as a regression canary.

CREATE TABLE adt_linkage (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    studio_namespace   TEXT        NOT NULL,
    adt_namespace      TEXT        NOT NULL,
    linked_by_user_id  UUID        NOT NULL,
    linked_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at         TIMESTAMPTZ
);

-- Partial unique so a studio can re-link the same adt_namespace after
-- soft-deleting an earlier linkage (the old row stays for audit chain
-- integrity — schema.md §"adt_linkage table").
CREATE UNIQUE INDEX adt_linkage_studio_adt_uniq
    ON adt_linkage (studio_namespace, adt_namespace)
    WHERE deleted_at IS NULL;

-- adt_link_pending -------------------------------------------------------
-- Short-lived nonce store for the linking redirect round-trip
-- (PRD §4.8.2). `state` is the 32-byte CSRF nonce; CompleteADTLink
-- consumes the row on success and runs an inline
-- DELETE WHERE expires_at < now() to sweep stale rows (schema.md
-- §"adt_link_pending table").

CREATE TABLE adt_link_pending (
    state                TEXT        PRIMARY KEY,
    studio_namespace     TEXT        NOT NULL,
    started_by_user_id   UUID        NOT NULL,
    expires_at           TIMESTAMPTZ NOT NULL
);

-- Supports the inline sweep predicate.
CREATE INDEX adt_link_pending_expires_at_idx
    ON adt_link_pending (expires_at);
