-- db/migrations/0016_labs_scoped_player_ids.sql
--
-- Official (labs) players used to be stored as "labs-<tp_id>", where tp_id
-- restarts at 1 for every event/division. Players from different
-- tournaments therefore collided on the same row and overwrote each
-- other's names. Player ids are now scoped to the tournament
-- ("labs-<event>-<division>-<tp_id>"). The old rows can't be repaired
-- (names were overwritten), so drop the labs tournaments (standings,
-- decklists and pairings cascade) and the orphaned old-format players;
-- the next ingest pass re-syncs them.
--
-- Not auto-run by docker-entrypoint-initdb.d -- apply with `make migrate`.
-- Written to be safe to run more than once.

DELETE FROM tournaments WHERE ingest_source = 'labs';
DELETE FROM players WHERE id ~ '^labs-[0-9]+$';
