COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; else echo "docker-compose"; fi)
# Name of the current compose project (usually the folder name) — needed so `make down`
# only removes this project's volumes and doesn't touch same-named volumes of other
# projects on this machine.
PROJECT_NAME := $(shell $(COMPOSE) config --format json 2>/dev/null | grep -o '"name": *"[^"]*"' | head -1 | sed -E 's/.*"([^"]+)"$$/\1/')

.PHONY: up down down-all logs db-logs run test test-e2e e2e-up e2e-down loadtest swagger

# Bring up the whole project with one command: DB + migrate + SeaweedFS + TEI + backend + frontend
up:
	$(COMPOSE) up -d --build
	@echo ""
	@echo "Готово:"
	@echo "  UI:      http://localhost:$${FRONTEND_PORT:-3000}"
	@echo "  API:     http://localhost:$${SERVER_PORT:-8080}"
	@echo "  Swagger: http://localhost:$${SERVER_PORT:-8080}/swagger/index.html"
	@echo "  S3 (SeaweedFS): http://localhost:$${S3_PORT:-9000}"
	@echo ""

# Stop and remove all containers and volumes EXCEPT the TEI model cache (teidata) —
# it is downloaded from the internet on each tei container start, so we keep it across
# `make down`/`make up` to avoid re-downloading it.
down:
	$(COMPOSE) down
	@for v in pgdata seaweedfs_data; do \
		ids=$$(docker volume ls -q --filter "label=com.docker.compose.project=$(PROJECT_NAME)" --filter "label=com.docker.compose.volume=$$v"); \
		if [ -n "$$ids" ]; then docker volume rm $$ids; fi; \
	done

# Full reset, including the TEI model cache (the next `make up` re-downloads it)
down-all:
	$(COMPOSE) down -v

logs:
	$(COMPOSE) logs -f app frontend

db-logs:
	$(COMPOSE) logs -f db

# Local backend run without Docker (needs a running DB, see `make up`)
run:
	cd backend && go run ./cmd/api

test:
	cd backend && go test ./...

# Regenerate the OpenAPI spec from handler annotations (requires the swag CLI)
swagger:
	cd backend && swag init -g cmd/api/main.go -o docs --parseInternal

# Lightweight stack without TEI for E2E/K6
e2e-up:
	$(COMPOSE) -f docker-compose.e2e.yml up -d --build
	@for i in $$(seq 1 60); do \
		if curl -fsS http://localhost:8080/healthz >/dev/null 2>&1; then echo "API is up"; exit 0; fi; \
		sleep 2; \
	done; echo "API failed to become healthy"; $(COMPOSE) -f docker-compose.e2e.yml logs; exit 1

e2e-down:
	$(COMPOSE) -f docker-compose.e2e.yml down -v

test-e2e: e2e-up
	cd backend && BASE_URL=http://localhost:8080 go test -tags=e2e ./tests/e2e/ -count=1 -v

# k6 load testing (needs a running API, e.g. `make e2e-up`)
loadtest:
	BASE_URL=$${BASE_URL:-http://localhost:8080} k6 run loadtest/k6/auth_item_flow.js
