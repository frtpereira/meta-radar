-- db/migrations/0002_archetype_core.sql
--
-- Not auto-run by docker-entrypoint-initdb.d (only 0001 is, and only on an
-- empty volume). Apply with `make migrate` -- see Makefile / README.
-- Written with IF NOT EXISTS so it's safe to run more than once.

ALTER TABLE archetypes
    ADD COLUMN IF NOT EXISTS core_cards       JSONB,
    ADD COLUMN IF NOT EXISTS core_threshold   REAL,
    ADD COLUMN IF NOT EXISTS core_computed_at TIMESTAMPTZ;