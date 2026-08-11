-- 0001_init.sql
-- Core schema for the Pokemon TCG deck tracker.

CREATE EXTENSION IF NOT EXISTS pgcrypto; -- gen_random_uuid()

-- A "meta" is our own grouping of tournaments into a competitive season,
-- independent of Limitless's coarser `format` field (e.g. "STANDARD").
-- This lets us split a format into sub-metas when a new set shakes things up
-- without waiting for an actual rotation.
CREATE TABLE metas (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    format_code   TEXT NOT NULL,          -- Limitless's format id, e.g. "STANDARD"
    starts_at     TIMESTAMPTZ NOT NULL,
    ends_at       TIMESTAMPTZ,            -- NULL = currently active meta
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Only one meta per format_code may be "open" (ends_at IS NULL) at a time.
CREATE UNIQUE INDEX one_open_meta_per_format
    ON metas (format_code)
    WHERE ends_at IS NULL;

CREATE TABLE players (
    id            TEXT PRIMARY KEY,       -- Limitless player id
    name          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Source of truth for tournament metadata. Limitless's own tournament id is
-- our primary key, which makes dedup a plain upsert -- no fuzzy matching on
-- name/date needed.
CREATE TABLE tournaments (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    game            TEXT NOT NULL,
    format_code     TEXT NOT NULL,
    meta_id         UUID REFERENCES metas(id),
    date            TIMESTAMPTZ NOT NULL,
    players         INT NOT NULL,
    is_online       BOOLEAN NOT NULL DEFAULT false,
    is_public       BOOLEAN NOT NULL DEFAULT true,
    has_decklists   BOOLEAN NOT NULL DEFAULT false,
    organizer_name  TEXT,
    raw_details     JSONB,                -- full /details payload, for fields we haven't modeled yet
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_checked_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tournaments_format_date ON tournaments (format_code, date DESC);
CREATE INDEX idx_tournaments_players ON tournaments (players);
CREATE INDEX idx_tournaments_meta ON tournaments (meta_id);

-- Archetypes are scoped per-meta: "Charizard ex / Pidgeot ex" in one meta is
-- a distinct row from the "same" archetype in the next meta, since the
-- optimal build (and thus its core) can shift completely.
CREATE TABLE archetypes (
    id            BIGSERIAL PRIMARY KEY,
    meta_id       UUID NOT NULL REFERENCES metas(id),
    name          TEXT NOT NULL,
    slug          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (meta_id, slug)
);

-- One decklist per player-tournament entry (that had a published list).
CREATE TABLE decklists (
    id            BIGSERIAL PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    player_id     TEXT REFERENCES players(id),
    archetype_id  BIGINT REFERENCES archetypes(id),
    cards         JSONB NOT NULL,         -- [{name, set, number, count, category}]
    core_hash     TEXT,                   -- hash of the archetype's "core" card subset
    raw_list      JSONB,                  -- untouched payload from Limitless, for reprocessing
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tournament_id, player_id)
);

CREATE INDEX idx_decklists_archetype_core ON decklists (archetype_id, core_hash);
CREATE INDEX idx_decklists_cards_gin ON decklists USING GIN (cards);

CREATE TABLE standings (
    id            BIGSERIAL PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    player_id     TEXT REFERENCES players(id),
    standing       INT NOT NULL,
    wins          INT NOT NULL DEFAULT 0,
    losses        INT NOT NULL DEFAULT 0,
    ties          INT NOT NULL DEFAULT 0,
    decklist_id   BIGINT REFERENCES decklists(id),
    UNIQUE (tournament_id, player_id)
);

CREATE INDEX idx_standings_tournament ON standings (tournament_id, standing);

-- Tracks ingestion so the sync worker can skip unchanged tournaments and
-- webhooks/polling don't race each other.
CREATE TABLE sync_log (
    id            BIGSERIAL PRIMARY KEY,
    tournament_id TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    source        TEXT NOT NULL,          -- 'poll' | 'webhook'
    status        TEXT NOT NULL,          -- 'success' | 'error'
    detail        TEXT,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sync_log_tournament ON sync_log (tournament_id, created_at DESC);
