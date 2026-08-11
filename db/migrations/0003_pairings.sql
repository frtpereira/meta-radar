-- db/migrations/0003_pairings.sql
--
-- Stores round-by-round pairings and outcomes from
-- GET /tournaments/{id}/pairings.

CREATE TABLE IF NOT EXISTS pairings (
    id               BIGSERIAL PRIMARY KEY,
    tournament_id    TEXT NOT NULL REFERENCES tournaments(id) ON DELETE CASCADE,
    phase            INT NOT NULL,
    round            INT NOT NULL,
    table_number     INT NOT NULL,
    player1_id       TEXT REFERENCES players(id),
    player2_id       TEXT REFERENCES players(id),
    winner_player_id TEXT REFERENCES players(id),
    result           TEXT NOT NULL, -- 'win' | 'draw' | 'bye' | 'unknown'
    raw_pairing      JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_pairings_tournament_round
    ON pairings (tournament_id, phase, round, table_number);

CREATE INDEX IF NOT EXISTS idx_pairings_players
    ON pairings (player1_id, player2_id);
