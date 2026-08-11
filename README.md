# Pokemon TCG Deck Tracker

Tracks top-performing Pokemon TCG decks from tournaments listed on
[limitlesstcg.com](https://limitlesstcg.com), using their official API
(`https://play.limitlesstcg.com/api` — no key required except for the
`/games/{id}/decks` endpoint).

## Stack

- **Postgres 16** (Docker) — source of truth
- **Go** (`chi` + `pgx`) — REST API + ingestion worker
- **React** (TanStack, to be added) — frontend

## Getting started

```bash
cp .env.example .env
make up
```

This builds the Go API image, starts Postgres, and runs the migration in
`db/migrations/0001_init.sql` automatically on first boot (via Postgres's
`docker-entrypoint-initdb.d` mechanism — it only runs once against an empty
data volume, so edit-and-restart won't re-apply it; see "Adding a new
migration" below).

Check it's alive:

```bash
curl http://localhost:8080/health
curl http://localhost:8080/api/tournaments
```

## First real step: generate go.sum

The `go.mod` here lists dependencies but has no `go.sum` yet (this
environment has no network access to fetch them). Before `make up` will
build, run:

```bash
make tidy
```

with network access available locally, which will download `chi`, `cors`,
and `pgx`, and write `backend/go.sum`.

## Adding a new migration

The `docker-entrypoint-initdb.d` mechanism only runs on an empty data
volume, so it's fine for the very first migration but won't work for
ongoing schema changes. Once you're iterating, switch to a real migration
tool — `golang-migrate` is a natural fit with this layout:

```bash
migrate create -ext sql -dir db/migrations -seq add_pairings_table
```

and run migrations explicitly (e.g. from `cmd/api/main.go` on startup, or a
separate `make migrate` target) instead of relying on container init.

## What's implemented so far

- Schema: `metas`, `tournaments`, `players`, `archetypes`, `decklists`,
  `standings`, `sync_log`
- REST endpoints:
    - `GET /api/tournaments?min_players=64&format=STANDARD&meta_id=...`
    - `GET /api/metas`
    - `GET /api/archetypes/stats?meta_id=...`
    - `GET /api/matchups/stats?meta_id=...&archetype_id=...&min_matches=5&include_mirrors=false`
        - `GET /api/archetypes/{id}` — metadata + computed core cards
        - `GET /api/archetypes/{id}/variants` — decklists grouped by build (`core_hash`)
        - `POST /api/webhooks/limitless` — verifies the shared secret, then syncs
          that one tournament in the background
- **Ingestion worker** (`cmd/ingest`): walks `GET /tournaments` (paged),
  skips anything under `--min-players` (default 64), and for everything
  else fetches `/details` + `/standings` + `/pairings` and upserts
  tournament, player, archetype, decklist, standing, and pairing rows in one
  transaction per tournament.
    - Dedup is a plain upsert on Limitless's own tournament id, so re-running
      the worker is always safe.
    - By default it **never re-syncs a tournament it's already stored**
      (`Options.Refresh == 0`) — a completed tournament's results don't
      change. Pass `--interval` to keep the process running and pick up
      newly-finished tournaments on each pass.
    - Archetype rows are seeded from Limitless's own auto-categorization
      (the `deck.id` / `deck.name` fields on each standings entry) rather
      than reimplementing that classification — see "Not yet implemented"
      below for where custom variant-clustering fits in on top of that.
    - Run it once ad hoc: `make ingest-once`. Runs continuously alongside
      the API via the `ingest` service in `docker-compose.yml`
      (`--interval=15m` by default).

Try it after `make up`:

```bash
make ingest-once
curl http://localhost:8080/api/tournaments?min_players=64
```

(Won't return anything meaningful until a `metas` row exists for the
format you're syncing — see below.)

## Not yet implemented (next steps)

1. **Meta management.** `metas` is an empty table you populate by hand
   (one row per format with `ends_at IS NULL` for the currently active
   one) — `syncTournament` only _attaches_ tournaments to an existing open
   meta, it never creates one. Needs an admin flow or a rule eventually
   (e.g. "new meta whenever a set with X new archetype-defining cards
   releases").

    To open the current Standard meta and backfill it onto anything
    already synced:

    ```bash
    make seed-meta
    make resync
    ```

    `seed-meta` is idempotent (safe to re-run; reuses the existing open
    meta for that format if there is one) and backfills `tournaments.meta_id`
    directly in SQL. `resync` is the step that's easy to miss: the ingest
    worker only assigns `archetype_id` on a decklist when an open meta
    exists _at sync time_ to scope the archetype to, so anything synced
    before you ran `seed-meta` has `archetype_id = NULL` and won't show up
    in `/api/archetypes/stats` until you force a re-sync with
    `--refresh=1s` (which `make resync` does for you).

2. **Pairings-driven analytics expansion.** Pairings are now stored and
   exposed through `/api/matchups/stats`, but you'll likely want additional
   rollups (for example: per-round conversion, Swiss-only vs top-cut splits,
   and confidence intervals) before using these stats as the sole ranking
   signal.

3. **Frontend.**
