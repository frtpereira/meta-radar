-- 0013_official_tournaments.sql
-- Supports ingesting official (offline) Regionals/Worlds tournaments from
-- limitlesstcg.com's labs API, alongside the existing
-- play.limitlesstcg.com-sourced online tournaments.
--
-- Official tournaments run three divisions (Masters/Seniors/Juniors) that
-- Limitless's labs API treats as entirely separate result sets (own
-- standings/pairings/decklists each). We store each division as its own
-- row in `tournaments` -- so every existing per-tournament join/foreign
-- key (standings, decklists, pairings, sync_log) keeps working
-- unmodified -- and group divisions of the same real-world event with a
-- shared `event_id`, so a future "single event page with a division
-- switcher" can query `WHERE event_id = $1`.
--
-- For tournaments synced from the existing Play API, event_id is just the
-- tournament's own id (a "group" of one), so every row has a usable
-- event_id regardless of source.

ALTER TABLE tournaments
    ADD COLUMN ingest_source TEXT NOT NULL DEFAULT 'play_api'
        CHECK (ingest_source IN ('play_api', 'labs')),
    ADD COLUMN division TEXT
        CHECK (division IN ('MA', 'SR', 'JR')),
    ADD COLUMN event_id TEXT;

UPDATE tournaments SET event_id = id WHERE event_id IS NULL;

ALTER TABLE tournaments ALTER COLUMN event_id SET NOT NULL;

CREATE INDEX idx_tournaments_event_id ON tournaments (event_id);

-- Completeness of a labs-sourced tournament's decklists is derived, not
-- stored: `standings.decklist_id IS NULL` (excluding standing = 0, i.e.
-- dropped/DQ'd players) already means "no decklist yet", the same as it
-- does for Play API tournaments. See internal/ingest/labs_sync.go's
-- staleLabsEventIDs for the query that drives rechecking.
