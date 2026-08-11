.PHONY: up down logs psql tidy ingest-once inspect seed-meta resync

DOCKER_COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo "docker-compose"; fi)

up:
	$(DOCKER_COMPOSE) up --build

down:
	$(DOCKER_COMPOSE) down

logs:
	$(DOCKER_COMPOSE) logs -f api

psql:
	$(DOCKER_COMPOSE) exec postgres psql -U app -d pokemontcg

tidy:
	cd backend && go mod tidy

# Run one ingestion pass without waiting for the --interval loop.
ingest-once:
	$(DOCKER_COMPOSE) build ingest
	$(DOCKER_COMPOSE) run --rm ingest --interval=0
 
# Print the raw `decklist` JSON for one real player, to verify the shape
# assumed by internal/limitless/decklist.go. Usage: make inspect ID=<tournament-id>
inspect:
	$(DOCKER_COMPOSE) build ingest
	$(DOCKER_COMPOSE) run --rm --entrypoint inspect ingest --tournament=$(ID)
 
# Open the current Standard meta (idempotent) and backfill it onto any
# already-synced tournaments. See db/seed/001_current_standard_meta.sql.
seed-meta:
	$(DOCKER_COMPOSE) exec -T postgres psql -U app -d pokemontcg < db/seed/001_current_standard_meta.sql
 
# Force a full re-sync so already-synced tournaments/decklists pick up
# archetype_id now that a meta exists to scope archetypes to. Run this
# right after seed-meta the first time you open a meta.
resync:
	$(DOCKER_COMPOSE) build ingest
	$(DOCKER_COMPOSE) run --rm ingest --interval=0 --refresh=1s