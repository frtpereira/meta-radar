DOCKER_COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo "docker-compose"; fi)

.PHONY: up upd down logs psql frontend-install frontend-dev frontend-dev-host frontend-build tidy migrate ingest-once seed-meta resync cluster inspect swagger-gen

up:
	$(DOCKER_COMPOSE) up --build

upd:
	$(DOCKER_COMPOSE) up --build -d

down:
	$(DOCKER_COMPOSE) down

logs:
	$(DOCKER_COMPOSE) logs -f api

psql:
	$(DOCKER_COMPOSE) exec postgres psql -U app -d pokemontcg

frontend:
	cd frontend && npm run build && npm run start

frontend-install:
	cd frontend && npm install

frontend-dev:
	cd frontend && npm run dev

tidy:
	cd backend && go mod tidy

# Regenerate docs/swagger.json + swagger.yaml from the @-annotations on
# handlers in internal/api and the general API info in cmd/api/main.go.
# Requires the swag CLI: `go install github.com/swaggo/swag/cmd/swag@latest`.
swagger-gen:
	cd backend && swag init -g cmd/api/main.go -o docs --outputTypes json,yaml --parseInternal

# Apply any migrations after 0001 (which is handled automatically by
# Postgres's init-on-empty-volume mechanism). Poor-man's migration runner --
# each file must be safe to re-run (IF NOT EXISTS etc). Replace with
# golang-migrate once this stops being enough. See README.
migrate:
	@for f in $$(ls db/migrations/*.sql | sort | tail -n +2); do \
		echo "applying $$f"; \
		$(DOCKER_COMPOSE) exec -T postgres psql -U app -d pokemontcg < $$f; \
	done

# Run one ingestion pass without waiting for the --interval loop.
ingest-once:
	$(DOCKER_COMPOSE) build ingest
	$(DOCKER_COMPOSE) run --rm ingest --interval=0

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

# Compute core cards + core_hash variant grouping for every archetype in
# every meta. Usage: make cluster [META=<meta-id>] [THRESHOLD=0.7]
cluster:
	$(DOCKER_COMPOSE) build ingest
	$(DOCKER_COMPOSE) run --rm --entrypoint cluster ingest --meta=$(META) --threshold=$(or $(THRESHOLD),0.7)

# Print the raw `decklist` JSON for one real player (or, with PAIRINGS=1,
# the raw /pairings response) to verify the shapes assumed by
# internal/limitless. Usage:
#   make inspect ID=<tournament-id>
#   make inspect ID=<tournament-id> PAIRINGS=1
inspect:
	$(DOCKER_COMPOSE) build ingest
	$(DOCKER_COMPOSE) run --rm --entrypoint inspect ingest --tournament=$(ID) $(if $(PAIRINGS),--pairings)