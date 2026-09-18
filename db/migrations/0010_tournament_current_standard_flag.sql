-- db/migrations/0010_tournament_current_standard_flag.sql
--
-- Depends on 0009_meta_hierarchy.sql (meta_type / parent_meta_id).
--
-- is_current_standard marks whether a tournament's decks are still part
-- of the *currently open* Standard rotation, as opposed to a Standard
-- tournament that's since rotated out. It's denormalized onto
-- tournaments (rather than derived at query time via a join through
-- metas) so it can be filtered/indexed cheaply -- ingest always sets it
-- true when attaching a tournament to the currently open set meta
-- (which is necessarily under the currently open standard meta -- see
-- internal/ingest/sync.go), and a rotation flips it false in bulk for
-- the outgoing era's tournaments (see db/seed/003_rotate_standard.sql).
--
-- Not auto-run by docker-entrypoint-initdb.d -- apply with `make migrate`.
-- Written to be safe to run more than once.

ALTER TABLE tournaments
    ADD COLUMN IF NOT EXISTS is_current_standard BOOLEAN NOT NULL DEFAULT true;

CREATE INDEX IF NOT EXISTS idx_tournaments_current_standard
    ON tournaments (is_current_standard)
    WHERE is_current_standard;

-- Backfill: a tournament counts as current-standard if the set meta it's
-- attached to has a parent standard meta that's still open. Tournaments
-- with no meta_id at all (never attached to any meta) are left at the
-- column default (true) -- there's nothing to derive them from yet, and
-- ingest will correct them the next time they're synced.
UPDATE tournaments t
SET is_current_standard = (std.ends_at IS NULL)
FROM metas sm
JOIN metas std ON std.id = sm.parent_meta_id
WHERE t.meta_id = sm.id
  AND sm.meta_type = 'set';
