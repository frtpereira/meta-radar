-- db/migrations/0011_meta_snapshots.sql
--
-- Foundation for tracking how a meta's archetype stats move over time.
-- cmd/snapshot (see backend/cmd/snapshot) computes one meta_snapshots
-- header row plus one meta_snapshot_archetypes row per archetype, on a
-- daily and/or weekly cadence, for every currently-open meta (both
-- 'standard' and 'set' -- see 0009_meta_hierarchy.sql). Nothing reads
-- these yet; they're written now so winrate/usage/performance-vs-
-- popularity graphs have real history to draw on once that UI exists,
-- instead of only ever being able to show the current moment.
--
-- Not auto-run by docker-entrypoint-initdb.d -- apply with `make migrate`.
-- Written to be safe to run more than once.

CREATE TABLE IF NOT EXISTS meta_snapshots (
    id            BIGSERIAL PRIMARY KEY,
    meta_id       UUID NOT NULL REFERENCES metas(id) ON DELETE CASCADE,
    snapshot_type TEXT NOT NULL CHECK (snapshot_type IN ('daily', 'weekly')),
    -- The day the snapshot represents. For 'weekly' this is the first
    -- day (Monday) of the ISO week the snapshot covers, so a week's
    -- data always lands on one predictable row regardless of which day
    -- of the week the job actually runs.
    snapshot_date DATE NOT NULL,
    generated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- How many decklists this snapshot's shares/rates were computed
    -- over, so a near-empty early snapshot can be told apart from a
    -- well-sampled one later.
    total_decks   INT NOT NULL DEFAULT 0,
    UNIQUE (meta_id, snapshot_type, snapshot_date)
);

CREATE INDEX IF NOT EXISTS idx_meta_snapshots_meta_date
    ON meta_snapshots (meta_id, snapshot_type, snapshot_date DESC);

-- Re-running cmd/snapshot for a date that already has a snapshot
-- replaces that snapshot's archetype rows (see the ON CONFLICT in
-- cmd/snapshot) rather than accumulating duplicates -- ON DELETE
-- CASCADE here is what makes "delete and reinsert" cheap to do from Go
-- without a separate DELETE statement.
CREATE TABLE IF NOT EXISTS meta_snapshot_archetypes (
    id             BIGSERIAL PRIMARY KEY,
    snapshot_id    BIGINT NOT NULL REFERENCES meta_snapshots(id) ON DELETE CASCADE,
    archetype_id   BIGINT NOT NULL REFERENCES archetypes(id),
    deck_count     INT NOT NULL DEFAULT 0,
    -- Usage share within the snapshot's meta at that point in time
    -- (deck_count / meta_snapshots.total_decks), stored rather than
    -- computed on read so a graph can plot it directly.
    share_pct      REAL,
    matches        INT NOT NULL DEFAULT 0,
    wins           INT NOT NULL DEFAULT 0,
    losses         INT NOT NULL DEFAULT 0,
    ties           INT NOT NULL DEFAULT 0,
    win_rate       REAL,
    avg_standing   REAL,
    UNIQUE (snapshot_id, archetype_id)
);

CREATE INDEX IF NOT EXISTS idx_meta_snapshot_archetypes_snapshot
    ON meta_snapshot_archetypes (snapshot_id);

-- Supports "usage/winrate history for this archetype over time" without
-- going through the snapshot header row first.
CREATE INDEX IF NOT EXISTS idx_meta_snapshot_archetypes_archetype
    ON meta_snapshot_archetypes (archetype_id);
