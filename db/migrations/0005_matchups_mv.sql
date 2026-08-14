-- db/migrations/0005_matchups_mv.sql
-- Create a materialized view to pre-aggregate matchup stats per meta.
-- Apply with `make migrate`.

DROP MATERIALIZED VIEW IF EXISTS matchups_mv;

CREATE MATERIALIZED VIEW matchups_mv AS
WITH base AS (
    SELECT
        t.meta_id::uuid AS meta_id,
        p.player1_id,
        p.player2_id,
        p.winner_player_id,
        d1.archetype_id AS archetype1_id,
        d2.archetype_id AS archetype2_id
    FROM pairings p
    JOIN tournaments t ON t.id = p.tournament_id
    JOIN decklists d1 ON d1.tournament_id = p.tournament_id AND d1.player_id = p.player1_id
    JOIN decklists d2 ON d2.tournament_id = p.tournament_id AND d2.player_id = p.player2_id
    WHERE p.result IN ('win', 'draw')
),
pairs AS (
    SELECT
        meta_id,
        LEAST(archetype1_id, archetype2_id) AS archetype_a_id,
        GREATEST(archetype1_id, archetype2_id) AS archetype_b_id,
        CASE WHEN LEAST(archetype1_id, archetype2_id) = archetype1_id
             THEN CASE WHEN winner_player_id = player1_id THEN 1 ELSE 0 END
             ELSE CASE WHEN winner_player_id = player2_id THEN 1 ELSE 0 END END AS win_a,
        CASE WHEN LEAST(archetype1_id, archetype2_id) = archetype1_id
             THEN CASE WHEN winner_player_id = player2_id THEN 1 ELSE 0 END
             ELSE CASE WHEN winner_player_id = player1_id THEN 1 ELSE 0 END END AS win_b,
        CASE WHEN winner_player_id IS NULL THEN 1 ELSE 0 END AS ties
    FROM base
)
SELECT
    pairs.meta_id,
    a.id AS archetype_id,
    a.name AS archetype_name,
    a.slug AS archetype_slug,
    b.id AS opponent_archetype_id,
    b.name AS opponent_name,
    b.slug AS opponent_slug,
    COUNT(*)::int AS matches,
    SUM(pairs.win_a)::int AS wins,
    SUM(pairs.win_b)::int AS losses,
    SUM(pairs.ties)::int AS ties,
    CASE WHEN a.id = b.id THEN NULL
         ELSE (SUM(pairs.win_a) + 0.5 * SUM(pairs.ties)) / COUNT(*)::float8 END AS score_rate,
    CASE WHEN a.id = b.id THEN NULL
         WHEN (SUM(pairs.win_a) + SUM(pairs.win_b)) = 0 THEN NULL
         ELSE SUM(pairs.win_a)::float8 / (SUM(pairs.win_a) + SUM(pairs.win_b))::float8 END AS win_rate
FROM pairs
JOIN archetypes a ON a.id = pairs.archetype_a_id
JOIN archetypes b ON b.id = pairs.archetype_b_id
GROUP BY pairs.meta_id, a.id, a.name, a.slug, b.id, b.name, b.slug
ORDER BY matches DESC;

-- Unique index required to support CONCURRENTLY refreshes
CREATE UNIQUE INDEX IF NOT EXISTS idx_matchups_mv_unique
    ON matchups_mv (meta_id, archetype_id, opponent_archetype_id);

CREATE INDEX IF NOT EXISTS idx_matchups_mv_meta_matches
    ON matchups_mv (meta_id, matches DESC);

-- Populate materialized view now
REFRESH MATERIALIZED VIEW matchups_mv;
