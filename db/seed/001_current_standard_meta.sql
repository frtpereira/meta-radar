-- db/seed/001_current_standard_meta.sql
--
-- Not run automatically (unlike db/migrations, this isn't mounted into
-- docker-entrypoint-initdb.d) -- run it deliberately whenever you need to
-- open a new meta. See `make seed-meta`.
--
-- Edit META_NAME / FORMAT_CODE / STARTS_AT below before running if you're
-- opening anything other than "today, current Standard".

DO $$
DECLARE
    v_format_code TEXT := 'STANDARD';
    v_name        TEXT := 'Current Standard';
    v_starts_at   TIMESTAMPTZ := now();
    v_meta_id     UUID;
BEGIN
    -- Idempotent: if an open meta already exists for this format, reuse it
    -- instead of erroring on the one-open-meta-per-format unique index.
    SELECT id INTO v_meta_id FROM metas WHERE format_code = v_format_code AND ends_at IS NULL;

    IF v_meta_id IS NULL THEN
        INSERT INTO metas (name, format_code, starts_at)
        VALUES (v_name, v_format_code, v_starts_at)
        RETURNING id INTO v_meta_id;

        RAISE NOTICE 'created meta % (%) for format %', v_meta_id, v_name, v_format_code;
    ELSE
        RAISE NOTICE 'reusing existing open meta % for format %', v_meta_id, v_format_code;
    END IF;

    -- Backfill: attach any already-synced tournaments for this format that
    -- predate this meta existing. The ingest worker only attaches meta_id
    -- for tournaments it syncs *after* an open meta is present, so this
    -- catches everything synced before you ran this script.
    UPDATE tournaments
    SET meta_id = v_meta_id
    WHERE format_code = v_format_code
      AND meta_id IS NULL;
END $$;