# API → frontend consumer map

Every frontend request goes through the single client in
`frontend/lib/api.ts` — there are no direct `fetch()` calls to the
backend anywhere else in `frontend/app`. So "who calls this endpoint"
is really "who calls this `lib/api.ts` function," which is what the
table below tracks.

Swagger (`@Router` annotations in `backend/internal/api/handlers.go`,
served at `/swagger/*`) documents each endpoint's request/response
shape; it has no notion of frontend callers, so that mapping lives
here instead, kept next to the router it describes. Re-check this
table whenever a page starts/stops calling an endpoint, or when a new
page is added — it goes stale silently otherwise.

| Endpoint | `lib/api.ts` function | Consumed by |
|---|---|---|
| `GET /health` | — (not called from the frontend) | *None.* Not wired into any page or infra healthcheck — the `docker-compose.yml` Postgres healthcheck is separate. Intended for manual/ops checks only. |
| `GET /api/tournaments` | `getTournaments` | `app/page.tsx` (home — "live tournaments" table, `min_players` default), `app/tournaments/page.tsx` (full tournaments table with all filters: meta, format, source, date range, winner archetype, event/organizer name, sort, pagination) |
| `GET /api/tournaments/{id}` | `getTournament` | `app/tournaments/[id]/page.tsx` (tournament detail: metadata + standings) |
| `GET /api/metas` | `getMetas` | `app/page.tsx`, `app/tournaments/page.tsx`, `app/matchups/page.tsx`, `app/archetypes/page.tsx` — all four use it only to populate the meta selector dropdown |
| `GET /api/archetypes/stats` | `getArchetypeStats` | `app/page.tsx` (home — "top archetypes" table), `app/tournaments/page.tsx` (winner-archetype filter options), `app/matchups/page.tsx` (archetype filter options), `app/archetypes/page.tsx` (main archetype explorer table), `app/archetypes/[id]/page.tsx` (re-fetched for the archetype's own current meta stat row) |
| `GET /api/archetypes/{id}` | `getArchetypeDetail` | `app/archetypes/[id]/page.tsx` (archetype detail: name, slug, core cards, threshold) |
| `GET /api/archetypes/{id}/variants` | `getArchetypeVariants` | *None.* Defined in the client and covered by `lib/api.test.ts`, but no page calls it — the variants view described in `gpt.md` hasn't been built yet. |
| `GET /api/archetypes/{id}/card-stats` | `getArchetypeCardStats` | `app/archetypes/[id]/page.tsx` (splits into core "skeleton" vs. optional/tech cards) |
| `GET /api/matchups/stats` | `getMatchupStats` | `app/matchups/page.tsx` (full matchup table, client-side sorted/paginated), `app/archetypes/[id]/page.tsx` (mini matchup table scoped to one archetype, `min_matches: 1`) |
| `GET /api/players/{nickname}` | `getPlayer` | `app/players/[nickname]/page.tsx` (player detail) |
| `GET /api/decklists/{id}` | `getDecklist` | `app/decklists/[id]/page.tsx` (decklist detail) |
| `POST /api/webhooks/limitless` | — (not called from the frontend) | *None.* Called by Limitless's own webhook delivery, not the app — see `cmd/ingest` / the webhook receiver in the ingest worker. |

## Pages with no API calls

Static/informational pages that render without hitting the backend:
`app/about`, `app/contact`, `app/disclaimers`, `app/privacy`,
`app/tos`, `app/players/page.tsx` (players index — no list endpoint
exists yet; only `/api/players/{nickname}` for a single lookup).
