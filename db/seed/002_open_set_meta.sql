-- db/seed/002_open_set_meta.sql
--
-- Run this whenever a new set releases and shakes up what's good/bad
-- enough to warrant its own meta -- see `make open-set-meta`. Closes
-- the currently open 'set' meta for the format and opens a new one
-- under the same standard parent, so /api/metas keeps a clean per-set
-- history while /api/metas/current's standard id doesn't change.
--
-- Requires 001_current_standard_meta.sql to have been run at least
-- once for this format (there must be an open standard meta to attach
-- the new set meta to).
--
-- Edit FORMAT_CODE / SET_NAME / STARTS_AT below before running.

DO $$
DECLARE
    v_format_code TEXT := 'STANDARD';
    v_set_name    TEXT := '30th Celebration'; -- e.g. 'Delta Reign'
    v_starts_at   TIMESTAMPTZ := now();
    v_standard_id UUID;
    v_old_set_id  UUID;
    v_new_set_id  UUID;
BEGIN
    SELECT id INTO v_standard_id FROM metas
        WHERE format_code = v_format_code AND meta_type = 'standard' AND ends_at IS NULL;

    IF v_standard_id IS NULL THEN
        RAISE EXCEPTION 'no open standard meta for format % -- run 001_current_standard_meta.sql first', v_format_code;
    END IF;

    SELECT id INTO v_old_set_id FROM metas
        WHERE format_code = v_format_code AND meta_type = 'set' AND ends_at IS NULL;

    IF v_old_set_id IS NOT NULL THEN
        UPDATE metas SET ends_at = v_starts_at WHERE id = v_old_set_id;
        RAISE NOTICE 'closed set meta %', v_old_set_id;
    END IF;

    INSERT INTO metas (name, format_code, meta_type, starts_at, parent_meta_id)
    VALUES (v_set_name, v_format_code, 'set', v_starts_at, v_standard_id)
    RETURNING id INTO v_new_set_id;

    RAISE NOTICE 'opened set meta % (%) under standard %', v_new_set_id, v_set_name, v_standard_id;

    -- Tournaments continue to attach to whichever set meta is open at
    -- sync time (see internal/ingest/sync.go) -- nothing to backfill
    -- here for tournaments that haven't synced yet. Existing
    -- tournaments already attached to the old set meta are left as-is;
    -- they belong to that era.
END $$;
