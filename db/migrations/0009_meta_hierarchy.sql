-- db/migrations/0009_meta_hierarchy.sql
--
-- Splits `metas` into two kinds of row instead of one:
--
--   * meta_type = 'standard' -- a permanent, format-level container
--     (e.g. "Standard") that spans the whole current rotation. This is
--     the default lens the website shows: it never closes on a set
--     release, only on an actual Standard rotation.
--   * meta_type = 'set'      -- the existing behaviour: a finer-grained
--     era that opens whenever a new set shakes up the format, closing
--     the previous one. Every set meta belongs to exactly one standard
--     meta via parent_meta_id.
--
-- Not auto-run by docker-entrypoint-initdb.d -- apply with `make migrate`.
-- Written to be safe to run more than once.

ALTER TABLE metas
    ADD COLUMN IF NOT EXISTS meta_type      TEXT NOT NULL DEFAULT 'set'
        CHECK (meta_type IN ('standard', 'set')),
    ADD COLUMN IF NOT EXISTS parent_meta_id UUID REFERENCES metas(id);

-- A set meta's parent must itself be a standard meta -- enforced here
-- rather than relying on application code to get it right.
ALTER TABLE metas DROP CONSTRAINT IF EXISTS parent_meta_must_be_standard;
ALTER TABLE metas
    ADD CONSTRAINT parent_meta_must_be_standard
    CHECK (
        parent_meta_id IS NULL
        OR meta_type = 'set'
    );

-- Replace the old "one open meta per format" index: a format can now
-- have one open standard meta *and* one open set meta at the same time
-- (they're different meta_types), but still only one of each.
DROP INDEX IF EXISTS one_open_meta_per_format;
CREATE UNIQUE INDEX IF NOT EXISTS one_open_meta_per_format_type
    ON metas (format_code, meta_type)
    WHERE ends_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_metas_parent ON metas (parent_meta_id);

-- Backfill: every meta that already exists predates this migration and
-- is therefore a "set" meta in the new model (meta_type defaults to
-- 'set' above). Give each distinct format_code a permanent standard
-- parent, backdated to that format's earliest known meta, and point
-- all of its set metas at it.
DO $$
DECLARE
    fmt RECORD;
    v_standard_id UUID;
BEGIN
    FOR fmt IN
        SELECT DISTINCT format_code FROM metas WHERE meta_type = 'set'
    LOOP
        -- Idempotent: reuse an existing open standard meta for this
        -- format if this migration (or a previous partial run) already
        -- created one.
        SELECT id INTO v_standard_id
        FROM metas
        WHERE format_code = fmt.format_code AND meta_type = 'standard' AND ends_at IS NULL;

        IF v_standard_id IS NULL THEN
            INSERT INTO metas (name, format_code, meta_type, starts_at)
            SELECT INITCAP(LOWER(fmt.format_code)), fmt.format_code, 'standard', MIN(starts_at)
            FROM metas
            WHERE format_code = fmt.format_code AND meta_type = 'set'
            RETURNING id INTO v_standard_id;
        END IF;

        UPDATE metas
        SET parent_meta_id = v_standard_id
        WHERE format_code = fmt.format_code
          AND meta_type = 'set'
          AND parent_meta_id IS NULL;
    END LOOP;
END $$;
