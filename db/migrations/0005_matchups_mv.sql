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
), directed AS (
    SELECT
        meta_id,
        archetype1_id AS archetype_id,
        archetype2_id AS opponent_archetype_id,
        CASE WHEN winner_player_id = player1_id THEN 1 ELSE 0 END AS wins,
        CASE WHEN winner_player_id = player2_id THEN 1 ELSE 0 END AS losses,
        CASE WHEN winner_player_id IS NULL THEN 1 ELSE 0 END AS ties
    FROM base

    UNION ALL

    SELECT
        meta_id,
        archetype2_id AS archetype_id,
        archetype1_id AS opponent_archetype_id,
        CASE WHEN winner_player_id = player2_id THEN 1 ELSE 0 END AS wins,
        CASE WHEN winner_player_id = player1_id THEN 1 ELSE 0 END AS losses,
        CASE WHEN winner_player_id IS NULL THEN 1 ELSE 0 END AS ties
    FROM base
)
SELECT
    meta_id,
    a.id AS archetype_id,
    a.name AS archetype_name,
    a.slug AS archetype_slug,
    o.id AS opponent_archetype_id,
    o.name AS opponent_name,
    o.slug AS opponent_slug,
    COUNT(*)::int AS matches,
    SUM(d.wins)::int AS wins,
    SUM(d.losses)::int AS losses,
    SUM(d.ties)::int AS ties,
    CASE WHEN a.id = o.id THEN NULL
         ELSE (SUM(d.wins) + 0.5 * SUM(d.ties)) / COUNT(*)::float8 END AS score_rate,
    CASE WHEN a.id = o.id THEN NULL
         WHEN (SUM(d.wins) + SUM(d.losses)) = 0 THEN NULL
         ELSE SUM(d.wins)::float8 / (SUM(d.wins) + SUM(d.losses))::float8 END AS win_rate
FROM directed d
JOIN archetypes a ON a.id = d.archetype_id
JOIN archetypes o ON o.id = d.opponent_archetype_id
GROUP BY meta_id, a.id, a.name, a.slug, o.id, o.name, o.slug
ORDER BY matches DESC;

-- Unique index required to support CONCURRENTLY refreshes
CREATE UNIQUE INDEX IF NOT EXISTS idx_matchups_mv_unique
    ON matchups_mv (meta_id, archetype_id, opponent_archetype_id);

CREATE INDEX IF NOT EXISTS idx_matchups_mv_meta_matches
    ON matchups_mv (meta_id, matches DESC);

-- Populate materialized view now
REFRESH MATERIALIZED VIEW matchups_mv;
