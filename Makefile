.DEFAULT_GOAL := help

GO_TAGS        := sqlite_fts5
SERVER_BIN     := server
CONFIG_PATH    ?= build/dev/config.yaml
FRONTEND_DIR   := frontend
IMAGE_LOCAL    := warehouse06:local
COMPOSE        := docker compose
COMPOSE_PROD   := docker compose -f build/prod/docker.compose.example.yaml
COMPOSE_RUN    := $(COMPOSE) run --rm --no-deps
BACKEND_RUN    := $(COMPOSE_RUN) --entrypoint sh backend -c
FRONTEND_RUN   := $(COMPOSE_RUN) --entrypoint sh frontend -c

export CONFIG_PATH

# ── Help ──────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@printf "Usage: make <target>\n\nTargets:\n"
	@grep -E '^[a-zA-Z0-9_.-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  %-22s %s\n", $$1, $$2}'

# ── Setup ─────────────────────────────────────────────────────────────────────

.PHONY: setup
setup: ## Copy dev .env if missing
	@test -f build/dev/.env || cp build/dev/.env-example build/dev/.env
	@test -f build/dev/config.yaml || cp build/dev/config.yaml.example build/dev/config.yaml

# ── Development ───────────────────────────────────────────────────────────────

.PHONY: dev
dev: up-build ## Start dev stack (docker compose up --build)

.PHONY: up
up: setup ## docker compose up (dev backend + frontend)
	$(COMPOSE) up

.PHONY: docker-up
docker-up: up

.PHONY: up-build
up-build: setup ## docker compose up --build
	$(COMPOSE) up --build

.PHONY: docker-up-build
docker-up-build: up-build

.PHONY: down
down: ## docker compose down
	$(COMPOSE) down

.PHONY: docker-down
docker-down: down

# ── Build ─────────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build production Docker image (warehouse06:local)
	docker build -f build/prod/Containerfile -t $(IMAGE_LOCAL) .

IMAGE ?= $(IMAGE_LOCAL)

.PHONY: smoke
smoke: ## Run the built image (IMAGE=<tag>) and wait until its healthcheck passes
	@cid=$$(docker run -d --rm $(IMAGE)); \
	trap 'docker stop $$cid >/dev/null 2>&1 || true' EXIT; \
	for i in $$(seq 1 30); do \
		s=$$(docker inspect --format '{{.State.Health.Status}}' $$cid 2>/dev/null); \
		[ "$$s" = healthy ] && exit 0; \
		[ "$$s" = unhealthy ] && { docker logs $$cid; exit 1; }; \
		sleep 2; \
	done; \
	docker logs $$cid; exit 1

.PHONY: prod
prod: setup-prod ## Production compose up --build
	$(COMPOSE_PROD) up --build

.PHONY: setup-prod
setup-prod:
	@test -f build/prod/config.yaml || cp build/prod/config.yaml.example build/prod/config.yaml

.PHONY: docker-prod
docker-prod: prod

# ── Emulator vendor ───────────────────────────────────────────────────────────

.PHONY: vendor-emulator
vendor-emulator: ensure-built ## Refresh frontend/emulator-src/ from vendor.lock.json (REF=<sha> to bump vector06js)
	$(FRONTEND_RUN) 'npm ci && npm run vendor:emulator $(if $(REF),-- --ref $(REF),)'

.PHONY: vendor-emulator-check
vendor-emulator-check: ensure-built ## Verify vendor.lock.json matches upstream
	$(FRONTEND_RUN) 'npm ci && npm run vendor:emulator -- --check'

# ── Go checks (plain commands; run inside the dev container or in CI) ────────

GOLANGCI_LINT_VERSION := v2.12.2

.PHONY: fmt
fmt: ## Check gofmt formatting
	test -z "$$(gofmt -l cmd internal)"

.PHONY: vet
vet: ## Run go vet
	go vet -tags $(GO_TAGS) ./...

.PHONY: go-lint
go-lint: ## Run golangci-lint
	golangci-lint run

.PHONY: lint-install
lint-install: ## Install pinned golangci-lint
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: go-test
go-test: ## Run Go tests with the race detector
	CGO_ENABLED=1 go test -race -tags $(GO_TAGS) -count=1 ./...

.PHONY: vuln
vuln: ## Scan Go dependencies for known vulnerabilities
	go run golang.org/x/vuln/cmd/govulncheck@latest -tags $(GO_TAGS) ./...

# ── Test / CI (dev flow: everything runs in containers) ──────────────────────

.PHONY: ensure-built
ensure-built:
	$(COMPOSE) build backend

.PHONY: test
test: ensure-built ## Run Go tests (race) in dev backend container
	$(BACKEND_RUN) 'make go-test'

.PHONY: lint
lint: ensure-built ## Go fmt/vet/golangci-lint in dev backend container
	$(BACKEND_RUN) 'make fmt vet go-lint'

.PHONY: fe-test
fe-test: ## Frontend unit tests in container
	$(FRONTEND_RUN) 'npm ci && npm test'

.PHONY: fe-lint
fe-lint: ## Frontend eslint in container
	$(FRONTEND_RUN) 'npm ci && npm run lint'

.PHONY: typecheck
typecheck: ## Frontend TypeScript typecheck in container
	$(FRONTEND_RUN) 'npm ci && npm run typecheck'

.PHONY: fe-build
fe-build: ## Frontend production build in container
	$(FRONTEND_RUN) 'npm ci && npm run build'

.PHONY: e2e
e2e: ## Playwright e2e tests in container (mocked API, no backend needed)
	$(COMPOSE) --profile e2e run --rm e2e

.PHONY: ci
ci: ensure-built ## Full CI pipeline in containers
	$(FRONTEND_RUN) 'npm ci && npm run typecheck && npm run lint && npm test && npm run build'
	$(BACKEND_RUN) 'make fmt vet go-lint go-test'
	$(MAKE) e2e
	$(MAKE) vendor-emulator-check
	docker build -f build/prod/Containerfile -t $(IMAGE_LOCAL) .
	$(MAKE) smoke

# ── Clean ─────────────────────────────────────────────────────────────────────

.PHONY: clean
clean: ## Remove server binary and frontend build artifacts
	rm -f $(SERVER_BIN)
	rm -rf $(FRONTEND_DIR)/dist $(FRONTEND_DIR)/public/emulator
