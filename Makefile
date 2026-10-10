# =============================================================================
# FounderOS (on BusinessOS) — Project Makefile
# Usage: make <target>
# Run `make help` to see all available targets.
# =============================================================================

.DEFAULT_GOAL := help
SHELL         := /bin/bash

# Colours for terminal output
BOLD  := \033[1m
RESET := \033[0m
GREEN := \033[32m
CYAN  := \033[36m

.PHONY: help
help: ## Show this help message
	@printf '$(BOLD)FounderOS — available targets:$(RESET)\n\n'
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ { printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2 }' $(MAKEFILE_LIST)
	@printf '\n'

# =============================================================================
# Setup
# =============================================================================

.PHONY: onboard
onboard: ## First local source setup: initialize, start, and verify everything
	@bash scripts/onboard.sh

.PHONY: setup
setup: ## First Docker setup: auto-generate secrets, pull images, start stack
	@bash scripts/setup.sh

.PHONY: dev
dev: ## Start all services (build if needed), follow logs
	@docker compose up -d --build
	@docker compose logs -f

# =============================================================================
# Local dev (no Docker) — backend + frontend + Electron on host
# Ports come from frontend/.env.development.local (BACKEND_PORT, FRONTEND_PORT).
# =============================================================================

.PHONY: dev-local
dev-local: ## Start backend + frontend + Electron locally (no Docker)
	@bash scripts/dev-local.sh start

# =============================================================================
# OptimalEngine (Elixir) — the live engine BusinessOS proxies to
# =============================================================================
# Starts the canonical Elixir OptimalEngine with an empty local runtime data
# root. Listens on http://localhost:4200 — match the URL set in
# desktop/backend-go/.env (OPTIMAL_ENGINE_URL).
#
# First run installs Elixir deps (mix deps.get) which can take a minute.
# Subsequent runs are instant.

.PHONY: optimal-engine
optimal-engine: ## Start the Elixir OptimalEngine on http://localhost:4200
	@cd optimal-engine && \
		(test -d deps || mix deps.get) && \
		mkdir -p ../.run/optimal-engine-data/.optimal && \
		OPTIMAL_ENGINE_ROOT=$$(pwd)/../.run/optimal-engine-data \
		OPTIMAL_ENGINE_DB=$$(pwd)/../.run/optimal-engine-data/.optimal/index.db \
		OPTIMAL_API_ENABLED=true \
		mix run --no-halt

.PHONY: optimal-reindex
optimal-reindex: ## Reindex the Elixir engine's nodes after manual MD edits
	@cd optimal-engine && \
		OPTIMAL_ENGINE_ROOT=$$(pwd)/../.run/optimal-engine-data \
		OPTIMAL_ENGINE_DB=$$(pwd)/../.run/optimal-engine-data/.optimal/index.db \
		mix optimal.index

.PHONY: optimal-engine-stop
optimal-engine-stop: ## Stop any Elixir engine listening on :4200
	@PID=$$(lsof -ti :4200); \
		if [ -n "$$PID" ]; then echo "stopping $$PID"; kill $$PID; \
		else echo "no engine on :4200"; fi

# --- OptimalEngine source sync (git subtree) ----------------------------------
# The optimal-engine/ directory is a git subtree of Miosa-osa/OptimalEngine.
# Use these targets to keep it in sync with the standalone Elixir repo when
# you work on it in ~/code/OptimalEngine.
.PHONY: optimal-engine-pull
optimal-engine-pull: ## Pull upstream Elixir engine changes into optimal-engine/
	@echo "→ fetching optimal-engine-elixir/main…"
	@git fetch optimal-engine-elixir main
	@echo "→ merging into optimal-engine/ (squashed)…"
	@git subtree pull --prefix=optimal-engine optimal-engine-elixir main --squash

.PHONY: optimal-engine-push
optimal-engine-push: ## Push optimal-engine/ fixes back upstream to Miosa-osa/OptimalEngine
	@echo "→ pushing optimal-engine/ → optimal-engine-elixir/main"
	@git subtree push --prefix=optimal-engine optimal-engine-elixir main

.PHONY: dev-local-stop
dev-local-stop: ## Stop the local dev stack
	@bash scripts/dev-local.sh stop

.PHONY: dev-local-restart
dev-local-restart: ## Restart the local dev stack
	@bash scripts/dev-local.sh restart

.PHONY: dev-local-status
dev-local-status: ## Show local dev stack status
	@bash scripts/dev-local.sh status

.PHONY: dev-local-verify
dev-local-verify: ## Prove the local stack is ready, including database and schema
	@bash scripts/dev-local.sh verify

.PHONY: dev-local-logs
dev-local-logs: ## Tail logs from local dev stack (Ctrl-C to exit)
	@bash scripts/dev-local.sh logs

.PHONY: test-dev-local-launcher
test-dev-local-launcher: ## Test clean local infrastructure bootstrap and env preservation
	@/bin/bash scripts/tests/dev-local-bootstrap-test.sh

.PHONY: verify-oss-projection
verify-oss-projection: ## Verify an already-projected public working tree
	@bash scripts/verify-oss-projection.sh

.PHONY: up
up: ## Start all services in the background
	@docker compose up -d

.PHONY: down
down: ## Stop all services (preserves volumes)
	@docker compose down

.PHONY: restart
restart: ## Restart all services
	@docker compose restart

# =============================================================================
# Build
# =============================================================================

.PHONY: build
build: ## Build all Docker images
	@docker compose build

.PHONY: build-backend
build-backend: ## Build only the backend image
	@docker compose build backend

.PHONY: build-frontend
build-frontend: ## Build only the frontend image
	@docker compose build frontend

.PHONY: rebuild
rebuild: ## Force rebuild all images (no cache)
	@docker compose build --no-cache

# =============================================================================
# Logs & Status
# =============================================================================

.PHONY: logs
logs: ## Follow logs from all services
	@docker compose logs -f

.PHONY: logs-backend
logs-backend: ## Follow backend logs only
	@docker compose logs -f backend

.PHONY: logs-frontend
logs-frontend: ## Follow frontend logs only
	@docker compose logs -f frontend

.PHONY: logs-db
logs-db: ## Follow postgres logs only
	@docker compose logs -f postgres

.PHONY: status
status: ## Show service health status
	@docker compose ps

# =============================================================================
# Testing
# =============================================================================

.PHONY: test
test: test-backend test-frontend ## Run all tests

.PHONY: test-backend
test-backend: ## Run Go backend tests
	@echo ""
	@printf '$(BOLD)Running Go tests...$(RESET)\n'
	@cd desktop/backend-go && go test ./... -count=1

.PHONY: test-frontend
test-frontend: ## Run SvelteKit frontend tests
	@echo ""
	@printf '$(BOLD)Running frontend tests...$(RESET)\n'
	@cd frontend && pnpm test

.PHONY: test-backend-verbose
test-backend-verbose: ## Run Go tests with verbose output
	@cd desktop/backend-go && go test ./... -count=1 -v

.PHONY: doctor
doctor: ## Run backend production-readiness preflight checks
	@cd desktop/backend-go && go run ./cmd/doctor

# =============================================================================
# Database
# =============================================================================

.PHONY: db-shell
db-shell: ## Open a psql shell inside the postgres container
	@docker compose exec postgres psql -U $${POSTGRES_USER:-postgres} -d $${POSTGRES_DB:-business_os}

.PHONY: db-schema-check
db-schema-check: ## Verify the database has the runtime schema required by the app
	@cd desktop/backend-go && go run ./scripts/db/verify_schema_contract.go

.PHONY: check-schema
check-schema: ## Verify schema.sql builds clean from empty AND is idempotent (drift guard)
	@./scripts/check-schema.sh

# =============================================================================
# Cleanup
# =============================================================================

.PHONY: clean
clean: ## Stop containers and remove volumes (DESTROYS all local data)
	@printf '$(BOLD)Removing containers and volumes...$(RESET)\n'
	@docker compose down -v
	@printf '$(GREEN)Done$(RESET)\n'

.PHONY: clean-images
clean-images: ## Remove locally built BusinessOS images
	@docker rmi businessos-backend:local businessos-frontend:local 2>/dev/null || true
	@printf '$(GREEN)Local images removed$(RESET)\n'

# =============================================================================
# Shortcuts
# =============================================================================

.PHONY: shell-backend
shell-backend: ## Open a shell inside the running backend container
	@docker compose exec backend sh

.PHONY: shell-frontend
shell-frontend: ## Open a shell inside the running frontend container
	@docker compose exec frontend sh

.PHONY: urls
urls: ## Print service URLs
	@bash scripts/print-urls.sh

# ── FounderOS demo: workspaces + dummy data ─────────────────────────────────
.PHONY: demo
demo: ## Create the FounderOS workspaces and load the demo data (OWNER=<email you registered with>)
	@test -n "$(OWNER)" || { echo "usage: make demo OWNER=<the email you registered with at /register>"; exit 2; }
	@cd desktop/backend-go && go run ./cmd/founderos-bootstrap -owner-email "$(OWNER)" && go run ./cmd/founderos-seed
