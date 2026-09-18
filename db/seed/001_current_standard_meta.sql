-- db/seed/001_current_standard_meta.sql
--
-- Not run automatically (unlike db/migrations, this isn't mounted into
-- docker-entrypoint-initdb.d) -- run it deliberately to bootstrap a
-- format for the first time. See `make seed-meta`.
--
-- Requires db/migrations/0009_meta_hierarchy.sql to have been applied.
--
-- Opens two rows, not one:
--   1. A permanent 'standard' meta -- the format-level container the
--      website defaults to (see /api/metas/current). This only ever
--      closes on an actual Standard rotation (db/seed/003_rotate_standard.sql).
--   2. Its first 'set' meta, a child of (1) -- the granular era that
--      tournaments/archetypes actually attach to. Later set releases
--      open a new one of these without touching (1) -- see
--      db/seed/002_open_set_meta.sql.
--
-- Edit FORMAT_CODE / SET_NAME / STARTS_AT below before running if
-- you're opening anything other than "today, current Standard".

DO $$
DECLARE
    v_format_code    TEXT := 'STANDARD';
    v_standard_name  TEXT := 'Standard';
    v_set_name       TEXT := 'Current Standard';
    v_starts_at      TIMESTAMPTZ := now();
    v_standard_id    UUID;
    v_set_id         UUID;
BEGIN
    -- Idempotent: if an open standard meta already exists for this
    -- format, reuse it instead of erroring on the one-open-meta-
    -- per-format-and-type unique index.
    SELECT id INTO v_standard_id FROM metas
        WHERE format_code = v_format_code AND meta_type = 'standard' AND ends_at IS NULL;

    IF v_standard_id IS NULL THEN
        INSERT INTO metas (name, format_code, meta_type, starts_at)
        VALUES (v_standard_name, v_format_code, 'standard', v_starts_at)
        RETURNING id INTO v_standard_id;

        RAISE NOTICE 'created standard meta % (%) for format %', v_standard_id, v_standard_name, v_format_code;
    ELSE
        RAISE NOTICE 'reusing existing open standard meta % for format %', v_standard_id, v_format_code;
    END IF;

    SELECT id INTO v_set_id FROM metas
        WHERE format_code = v_format_code AND meta_type = 'set' AND ends_at IS NULL;

    IF v_set_id IS NULL THEN
        INSERT INTO metas (name, format_code, meta_type, starts_at, parent_meta_id)
        VALUES (v_set_name, v_format_code, 'set', v_starts_at, v_standard_id)
        RETURNING id INTO v_set_id;

        RAISE NOTICE 'created set meta % (%) under standard %', v_set_id, v_set_name, v_standard_id;
    ELSE
        RAISE NOTICE 'reusing existing open set meta % for format %', v_set_id, v_format_code;
    END IF;

    -- Backfill: attach any already-synced tournaments for this format
    -- that predate this meta existing. The ingest worker only attaches
    -- meta_id for tournaments it syncs *after* an open set meta is
    -- present, so this catches everything synced before you ran this
    -- script.
    UPDATE tournaments
    SET meta_id = v_set_id, is_current_standard = true
    WHERE format_code = v_format_code
      AND meta_id IS NULL;
END $$;
