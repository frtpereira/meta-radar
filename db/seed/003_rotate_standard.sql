-- db/seed/003_rotate_standard.sql
--
-- Run this only on an actual Standard rotation (cards leaving the
-- legal pool), not on an ordinary set release -- see
-- db/seed/002_open_set_meta.sql for that, much more common case. See
-- `make rotate-standard`.
--
-- Closes the current standard meta, flips is_current_standard to false
-- for every tournament under it (its decks are no longer legal in the
-- new rotation), then opens a brand-new standard meta plus its first
-- set meta so the format has somewhere to attach going forward.
--
-- Edit FORMAT_CODE / SET_NAME / STARTS_AT below before running.

DO $$
DECLARE
    v_format_code    TEXT := 'STANDARD';
    v_standard_name  TEXT := 'Standard';
    v_set_name       TEXT := 'New Standard Meta';
    v_starts_at      TIMESTAMPTZ := now();
    v_old_standard_id UUID;
    v_new_standard_id UUID;
    v_new_set_id      UUID;
BEGIN
    SELECT id INTO v_old_standard_id FROM metas
        WHERE format_code = v_format_code AND meta_type = 'standard' AND ends_at IS NULL;

    IF v_old_standard_id IS NULL THEN
        RAISE EXCEPTION 'no open standard meta for format % to rotate', v_format_code;
    END IF;

    -- Close the outgoing standard era and every set meta still open
    -- under it (there should be at most one).
    UPDATE metas SET ends_at = v_starts_at
        WHERE id = v_old_standard_id
           OR (parent_meta_id = v_old_standard_id AND ends_at IS NULL);

    -- The outgoing era's tournaments are no longer part of current
    -- Standard -- their decks may include cards that just rotated out.
    UPDATE tournaments t
    SET is_current_standard = false
    FROM metas sm
    WHERE t.meta_id = sm.id
      AND sm.parent_meta_id = v_old_standard_id;

    INSERT INTO metas (name, format_code, meta_type, starts_at)
    VALUES (v_standard_name, v_format_code, 'standard', v_starts_at)
    RETURNING id INTO v_new_standard_id;

    INSERT INTO metas (name, format_code, meta_type, starts_at, parent_meta_id)
    VALUES (v_set_name, v_format_code, 'set', v_starts_at, v_new_standard_id)
    RETURNING id INTO v_new_set_id;

    RAISE NOTICE 'rotated format %: closed standard %, opened standard % with set %',
        v_format_code, v_old_standard_id, v_new_standard_id, v_new_set_id;
END $$;
