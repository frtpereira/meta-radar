-- db/migrations/0004_matchup_indexes.sql
-- Additional indexes to speed up matchup aggregations (applied with `make migrate`).

-- Help the planner find pairings for a tournament filtered by result quickly.
CREATE INDEX IF NOT EXISTS idx_pairings_tournament_result
    ON pairings (tournament_id, result);

-- Help joins from pairings -> decklists when we need both the tournament and
-- archetype for a given player (used heavily by the matchup aggregation).
CREATE INDEX IF NOT EXISTS idx_decklists_tournament_archetype
    ON decklists (tournament_id, archetype_id);

-- Optional: index on winner_player_id can help some aggregations that
-- reference the winner directly.
CREATE INDEX IF NOT EXISTS idx_pairings_winner_player
    ON pairings (winner_player_id);
