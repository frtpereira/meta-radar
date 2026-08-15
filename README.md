# Pokemon TCG Deck Tracker

Tracks top-performing Pokemon TCG decks from tournaments listed on
[limitlesstcg.com](https://limitlesstcg.com), using their official API
(`https://play.limitlesstcg.com/api` — no key required except for the
`/games/{id}/decks` endpoint).

## Stack

- **Postgres 16** (Docker) — source of truth
- **Go** (`chi` + `pgx`) — REST API + ingestion worker
- **Next.js** (React + TypeScript) — frontend

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

## Frontend

The frontend now lives in `frontend/` and uses Next.js App Router.

```bash
cd frontend
npm install
npm run dev
```

By default it talks to the API at `http://localhost:8080/api`. If you need a different base URL, set `NEXT_PUBLIC_API_BASE_URL` in `frontend/.env.local`.

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

## API Reference

### Health

- `GET /health`
    - Returns `{ "status": "ok" }`.

### Tournaments

- `GET /api/tournaments?min_players=64&format=STANDARD&meta_id=...`
    - Lists tournaments stored in Postgres.
    - `min_players` filters by field size.
    - `format` filters by Limitless format code.
    - `meta_id` filters to one internal meta.

### Metas

- `GET /api/metas`
    - Lists all metas, newest first.

### Archetypes

- `GET /api/archetypes/stats?meta_id=...`
    - Returns one row per archetype in that meta.
    - Fields: `deck_count`, `avg_standing`, `drop_count`, `matches`, `wins`,
      `losses`, `ties`, `score_rate`, and `win_rate`.
    - `score_rate` comes from pairings and counts draws as half a win.
    - `win_rate` also comes from pairings, but ignores draws in the denominator.

- `GET /api/archetypes/{id}`
    - Returns the archetype metadata plus its computed `core_cards`, if the
      clustering job has already been run.

- `GET /api/archetypes/{id}/variants`
    - Groups decklists by `core_hash` so you can see distinct builds within one
      archetype.
    - Returns `deck_count`, `avg_standing`, `drop_count`, and a sample
      `decklist_id` for each build.

### Matchups

- `GET /api/matchups/stats?meta_id=...&archetype_id=...&min_matches=5&include_mirrors=false`
    - Returns directional archetype-vs-archetype results from stored pairings.
    - `meta_id` is required.
    - `archetype_id` is optional; when present it narrows results to one
      archetype.
    - `min_matches` defaults to `1` and filters out sparse pairings.
    - `include_mirrors=true` includes archetype-vs-itself rows.
    - Mirror rows are intentionally returned with `score_rate = null` and
      `win_rate = null`, because the directed aggregation makes them a
      mathematical dead end at 0.5.

### Webhooks

- `POST /api/webhooks/limitless`
    - Accepts a JSON body shaped like:

        ```json
        {
            "secret": "shared-secret",
            "event": {
                "name": "tournament_finished",
                "tournamentId": "abc123",
                "game": "PTCG"
            }
        }
        ```

    - Verifies the shared secret when `WEBHOOK_SECRET` is configured.
    - Responds `202 Accepted` immediately and syncs that tournament in the
      background.

### Ingestion worker

- `cmd/ingest`
    - Walks `GET /tournaments` in pages.
    - Skips tournaments under `--min-players` (default `64`).
    - For each tournament it syncs `/details`, `/standings`, and `/pairings`,
      then upserts tournament, player, archetype, decklist, standing, and
      pairing rows in one transaction.
    - Run it once with `make ingest-once`.
    - Keep it running with the `ingest` service in `docker-compose.yml`.

### Data model

- Schema tables currently in use: `metas`, `tournaments`, `players`,
  `archetypes`, `decklists`, `standings`, `pairings`, and `sync_log`.

Try it after `make up`:

```bash
make ingest-once
curl http://localhost:8080/api/tournaments?min_players=64
```

## Operator playbook

When you create or open a new meta you should seed it, re-sync so
existing tournaments/decklists pick up `archetype_id`, then run the
batch clustering job to compute `core_cards` and `core_hash`.

```bash
# one-time / schema step
make migrate

# seed the current Standard meta (idempotent)
make seed-meta

# force a full re-sync so decklists/archetypes are backfilled
make resync

# compute cores & variants; omit META to run all metas
make cluster META=<meta-id> THRESHOLD=0.7
```

Notes:

- The clustering logic is a separate batch job (see `cmd/cluster`) and
  is not run during per-tournament ingestion. Run it after bulk or
  one-off ingests when you need archetype cores/variants refreshed.
- The `cluster` binary is built into the backend image by the
  `backend/Dockerfile` and the Makefile target runs it inside the
  `ingest` image.

(Won't return anything meaningful until a `metas` row exists for the
format you're syncing — see below.)

## How to read archetype stats

`GET /api/archetypes/stats?meta_id=...` combines two kinds of signal:

- placement-based volume data from `decklists` and `standings`
- pairings-based match results from the `pairings` table

That means the same response tells you both how popular a deck was and how
it actually performed in matches.

Field breakdown:

- `deck_count`: how many decklists in the meta were tagged with that archetype
- `avg_standing`: average final placing across all non-dropped players; lower
  is better
- `drop_count`: how many entries ended with `standing = 0`
- `matches`: total recorded match appearances for that archetype across all
  pairings
- `wins`: match wins credited to that archetype
- `losses`: match losses credited to that archetype
- `ties`: draws / non-decisive results
- `score_rate`: `(wins + 0.5 * ties) / matches`.
  This treats a draw as half a win, which is a good compact summary of raw
  match performance.
- `win_rate`: `wins / (wins + losses)`.
  This ignores ties entirely and answers a narrower question: when the match
  actually produced a winner, how often did the archetype win?

Why the two rates are close but not identical in your example:

- `score_rate` counts ties as half-credit.
- `win_rate` removes ties from the denominator.
- If an archetype has many draws, `score_rate` will usually sit slightly below
  or above `win_rate` depending on how those draws are distributed.

Using your sample:

- `Dragapult` shows `deck_count = 886`, so it was one of the most popular
  archetypes in the meta.
- Its `avg_standing = 99.93` suggests middling-to-strong finishes relative to
  the field.
- Its `score_rate = 0.547` and `win_rate = 0.549` both point to a positive
  match performance, not just a good placement profile.

In short: `deck_count` tells you popularity, `avg_standing` tells you finishing
position, and `score_rate` / `win_rate` tell you how the deck actually fared in
matches.

Annotated example:

```json
{
    "name": "Dragapult",
    "deck_count": 886,
    "avg_standing": 99.9320241691843,
    "drop_count": 224,
    "matches": 4419,
    "wins": 2322,
    "losses": 1906,
    "ties": 191,
    "score_rate": 0.547069472731387,
    "win_rate": 0.549195837275308
}
```

- `deck_count = 886`: this archetype appeared in 886 published decklists.
- `avg_standing = 99.93`: across non-dropped players, the average finish was
  around 100th place.
- `drop_count = 224`: 224 players with this archetype dropped before
  completing the event.
- `matches = 4419`: there were 4419 recorded pairings involving this
  archetype.
- `wins = 2322`, `losses = 1906`, `ties = 191`: the match record behind the
  rate numbers.
- `score_rate = 0.547...`: this is the cleanest overall match-performance
  number; draws count as half a win.
- `win_rate = 0.549...`: this asks a slightly different question by ignoring
  draws in the denominator.

## Progress and next steps

1. **Verify the decklist payload shape.** Done. The raw `/standings`
   decklist payload has been checked against a real response and
   `internal/limitless/decklist.go` no longer depends on a guessed shape.

2. **Variant clustering on top of Limitless's archetypes.** Done, as of
   `internal/archetype` + `cmd/cluster`. For each archetype (scoped to a
   meta), it finds cards played in ≥70% of that archetype's decklists
   (the "core"), stores that core on the archetype row, and hashes each
   decklist's core-card subset into `core_hash` -- so two decklists with
   the same skeleton but different 1-of tech choices land on the same
   hash, and genuinely different builds don't.

    This is a **batch job**, run after ingestion, not part of the sync
    itself -- it needs to see the whole population of an archetype's
    decklists to know what counts as "core":

    ```bash
    make migrate       # one-time: adds archetypes.core_cards/core_threshold/core_computed_at
    make cluster        # all metas, default threshold 0.7
    make cluster META=<meta-id> THRESHOLD=0.6
    ```

    New endpoints:
    - `GET /api/archetypes/{id}` — archetype metadata + its computed core
      card list
    - `GET /api/archetypes/{id}/variants` — decklists grouped by
      `core_hash`, each with `deck_count`, `avg_standing` (drops excluded),
      and `drop_count`

    Known simplification: cards are keyed by name+set+number, so two
    different prints of a functionally identical card (e.g. an older
    reprint of a staple Trainer) count as separate cards rather than being
    merged -- worth revisiting if it visibly fragments cores in practice.

3. **Meta lifecycle automation.** Basic meta support exists now: metas can
   be seeded, listed via `/api/metas`, and attached during ingest when an
   open meta already exists. The remaining gap is making meta changes an
   explicit workflow instead of a manual SQL step.

    Concrete next steps:
    - add an admin path (API, CLI, or both) to open a meta, close the
      previous one for the same format, and record the exact transition
      date
    - bundle the follow-up work that is easy to miss today: backfill
      `tournaments.meta_id`, force a re-sync for `archetype_id`, and rerun
      clustering for the newly scoped archetypes
    - decide whether the long-term source of truth should stay manual or
      become a rules-based trigger tied to set releases

4. **Pairings ingestion.** Done. `cmd/ingest` fetches
   `/tournaments/{id}/pairings`, stores them in `pairings`, and uses that
   data to power `/api/matchups/stats` plus win-rate fields on
   `/api/archetypes/stats`.

5. **Analytics and operator workflow.** The core data model is in place;
   the remaining backend work is mainly about making the outputs easier to
   trust and the maintenance steps easier to run.

    Concrete next steps:
    - expand matchup and archetype rollups with filters that matter in
      practice, such as Swiss-only vs top-cut, per-round slices, and
      minimum sample thresholds
    - add uncertainty-aware metrics so tiny samples do not look as strong
      as well-supported matchup records
    - decide whether clustering should remain a separate batch job or be
      wrapped in a single post-ingest maintenance command alongside resync
      and meta backfill steps
    - add a short operator playbook in this README for the common sequence:
      migrate, seed/open meta, resync, cluster, then query the API

6. Frontend.
