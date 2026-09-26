-- db/migrations/0014_sync_log_relax_tournament_fk.sql
--
-- sync_log is an audit trail of sync *attempts*, not just successful
-- ones: internal/ingest needs to log a skip or failure for a tournament
-- before any tournaments row for it exists -- or, for official
-- tournaments predating labs_sync.go's MinEventDate cutoff, before one
-- will *ever* exist. The original FK required tournament_id to already
-- be present in tournaments, so every such log write failed with
-- "violates foreign key constraint sync_log_tournament_id_fkey" instead
-- of recording the attempt.
--
-- Not auto-run by docker-entrypoint-initdb.d -- apply with `make migrate`.
-- Written to be safe to run more than once.

ALTER TABLE sync_log DROP CONSTRAINT IF EXISTS sync_log_tournament_id_fkey;
