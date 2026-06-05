# xyn-pos-v1 — Makefile
# Run `make help` to see all available targets

.PHONY: help dev-up dev-down dev-logs dev-ps \
        proto-lint proto-gen proto-breaking \
        lint lint-fix \
        test test-unit test-integration test-coverage \
        build build-all \
        migrate-up migrate-down migrate-status \
        k6-smoke k6-load k6-spike \
        clean

# ─────────────────────────────────────────────────────────
# Help
# ─────────────────────────────────────────────────────────

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2}' | sort

# ─────────────────────────────────────────────────────────
# Docker Compose — Local dev stack
# ─────────────────────────────────────────────────────────

dev-up: ## Start all local services (Postgres, Redis, Kafka, MinIO, Keycloak, Observability)
	docker compose up -d
	@echo "\n✅ Stack running:"
	@echo "  PostgreSQL:   localhost:5432 (app_user / xyn_app_password)"
	@echo "  PgBouncer:    localhost:6432"
	@echo "  Redis:        localhost:6379"
	@echo "  Kafka:        localhost:9094 (external)"
	@echo "  Kafka UI:     http://localhost:8090"
	@echo "  MinIO:        http://localhost:9001 (console)"
	@echo "  Keycloak:     http://localhost:8080 (admin/xyn_keycloak_password)"
	@echo "  API Gateway:  http://localhost:8000"
	@echo "  Jaeger UI:    http://localhost:16686"
	@echo "  Prometheus:   http://localhost:9090"
	@echo "  Grafana:      http://localhost:3001 (admin/xyn_grafana_password)"

dev-down: ## Stop and remove all containers + volumes
	docker compose down -v

dev-logs: ## Tail all container logs
	docker compose logs -f

dev-ps: ## Show running containers
	docker compose ps

dev-restart: ## Restart a specific service: make dev-restart svc=kafka
	docker compose restart $(svc)

# ─────────────────────────────────────────────────────────
# Protobuf / Buf
# ─────────────────────────────────────────────────────────

proto-lint: ## Lint all .proto files
	buf lint proto/

proto-gen: ## Generate Go + TypeScript + Dart code from protos
	buf generate proto/
	@echo "Generated code in gen/"

proto-breaking: ## Check for breaking proto changes against main
	buf breaking proto/ --against '.git#branch=main,subdir=proto'

proto-format: ## Format all .proto files
	buf format -w proto/

# ─────────────────────────────────────────────────────────
# Linting
# ─────────────────────────────────────────────────────────

lint: ## Run golangci-lint on all Go code
	golangci-lint run ./...

lint-fix: ## Run golangci-lint with auto-fix
	golangci-lint run --fix ./...

# ─────────────────────────────────────────────────────────
# Testing
# ─────────────────────────────────────────────────────────

test: test-unit ## Run unit tests (default)

test-unit: ## Run unit tests (no external deps — domain layer only)
	go test -v -race -count=1 ./services/.../internal/domain/...

test-integration: ## Run integration tests (requires Docker)
	go test -v -race -count=1 -tags=integration -timeout=120s ./services/.../internal/infrastructure/...

test-coverage: ## Run all tests with coverage report
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

test-service: ## Test a single service: make test-service svc=pos
	go test -v -race -count=1 ./services/$(svc)/...

# ─────────────────────────────────────────────────────────
# Build
# ─────────────────────────────────────────────────────────

build: ## Build a single service: make build svc=pos
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/$(svc) ./services/$(svc)/cmd/server/

build-all: ## Build all services
	@for svc in tenant pos payment inventory kitchen analytics; do \
		echo "Building $$svc..."; \
		CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o bin/$$svc ./services/$$svc/cmd/server/; \
	done

# ─────────────────────────────────────────────────────────
# Database Migrations (Goose 3.27.1)
# ─────────────────────────────────────────────────────────

migrate-up: ## Run all pending migrations: make migrate-up svc=pos
	goose -dir ./services/$(svc)/internal/infrastructure/postgres/migrations postgres \
		"$$DATABASE_URL" up

migrate-down: ## Roll back last migration: make migrate-down svc=pos
	goose -dir ./services/$(svc)/internal/infrastructure/postgres/migrations postgres \
		"$$DATABASE_URL" down

migrate-status: ## Show migration status: make migrate-status svc=pos
	goose -dir ./services/$(svc)/internal/infrastructure/postgres/migrations postgres \
		"$$DATABASE_URL" status

migrate-create: ## Create new migration: make migrate-create svc=pos name=add_orders_table
	goose -dir ./services/$(svc)/internal/infrastructure/postgres/migrations create $(name) sql

# ─────────────────────────────────────────────────────────
# Stress / Load Testing (k6 1.7.1)
# ─────────────────────────────────────────────────────────

k6-smoke: ## Quick smoke test (5 VUs, 1 min)
	k6 run --env SCENARIO=smoke tests/k6/checkout.js

k6-load: ## Normal load test (50 req/s, 10 min)
	k6 run --env SCENARIO=normal_load tests/k6/checkout.js

k6-spike: ## Spike test (500 req/s, 2 min burst)
	k6 run --env SCENARIO=spike tests/k6/checkout.js

# ─────────────────────────────────────────────────────────
# Utilities
# ─────────────────────────────────────────────────────────

clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html

tidy: ## Run go mod tidy on all modules
	go work sync
	@for svc in shared/go services/tenant services/pos services/payment services/inventory services/kitchen services/analytics; do \
		echo "Tidying $$svc..."; \
		(cd $$svc && go mod tidy); \
	done

gen-mocks: ## Generate mocks with mockery v2
	go generate ./...

seed-dev: ## Seed development data (tenant, products, users)
	go run ./scripts/seed/main.go

check-versions: ## Print installed tool versions
	@echo "Go:              $$(go version)"
	@echo "Buf:             $$(buf --version)"
	@echo "golangci-lint:   $$(golangci-lint --version)"
	@echo "goose:           $$(goose --version)"
	@echo "k6:              $$(k6 version)"
	@echo "Docker:          $$(docker --version)"
	@echo "Docker Compose:  $$(docker compose version)"
