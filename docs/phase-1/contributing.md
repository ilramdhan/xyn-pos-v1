# Contributing Guide — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Authoritative Standard | Read before writing your first line of code.

---

## 1. Prerequisites

Install all tools before starting. These versions are **exact** — do not use different versions.

```bash
# Go
go version  # must be 1.26.4
# Install: https://go.dev/dl/

# Buf (protobuf management)
buf --version  # must be 1.70.0
# Install: brew install bufbuild/buf/buf

# Docker & Docker Compose
docker --version          # 27+
docker compose version    # 2.27+

# golangci-lint
golangci-lint --version  # must be 2.12.2
# Install: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v2.12.2

# Node.js (for web app)
node --version  # 22 LTS
pnpm --version  # 9+
# Install pnpm: npm install -g pnpm

# Flutter
flutter --version  # must be 3.44.0
dart --version     # must be 3.12

# k6 (for stress tests)
k6 version  # must be 1.7.1

# mockery (for mock generation)
mockery --version  # v2.x
# Install: go install github.com/vektra/mockery/v2@latest

# swag (Swagger doc generation)
swag --version
# Install: go install github.com/swaggo/swag/cmd/swag@latest

# goose (migrations)
goose --version  # 3.27.1
# Install: go install github.com/pressly/goose/v3/cmd/goose@latest

# pre-commit
pre-commit --version  # 3.x
# Install: pip install pre-commit
```

---

## 2. First-Time Local Setup

### Step 1: Clone and initialize

```bash
git clone https://github.com/xyn/xyn-pos-v1.git
cd xyn-pos-v1

# Install pre-commit hooks
pre-commit install

# Initialize Go workspace
go work sync

# Install web dependencies
cd apps/web && pnpm install && cd ../..

# Install Flutter dependencies
cd apps/mobile && flutter pub get && cd ../..
```

### Step 2: Configure environment

```bash
# Copy environment templates
cp .env.example .env.local

# For each service:
cp services/tenant/.env.example  services/tenant/.env.local
cp services/pos/.env.example     services/pos/.env.local
cp services/payment/.env.example services/payment/.env.local
cp services/inventory/.env.example services/inventory/.env.local
```

Edit `.env.local` — required values are marked `# REQUIRED`.  
See `docs/phase-1/environment-variables.md` for full documentation.

### Step 3: Start local infrastructure

```bash
# Start all infrastructure services (PostgreSQL 18, Redis 8, Kafka 4.3 KRaft, etc.)
docker compose up -d

# Verify all services are healthy
docker compose ps
# All services should show "healthy" or "running"

# Wait for Kafka KRaft to be ready (first start takes ~30s)
docker compose logs -f kafka | grep "Kafka Server started"
```

### Step 4: Run database migrations

```bash
# Run all service migrations
make migrate-all

# Or for a specific service:
make migrate SERVICE=pos

# Verify migrations
make migrate-status SERVICE=pos
```

### Step 5: Generate Protobuf code

```bash
# Generate Go, TypeScript, and Dart code from .proto files
make proto

# Verify generated files exist
ls services/pos/internal/interfaces/grpc/pb/
ls apps/web/src/gen/
ls apps/mobile/lib/gen/
```

### Step 6: Run the stack

```bash
# Start all Go microservices
make services-up

# Start Next.js web app
cd apps/web && pnpm dev

# Start Flutter app (with a connected device/emulator)
cd apps/mobile && flutter run
```

### Step 7: Verify everything works

```bash
# Run all tests
make test

# Run linting
make lint

# Hit the health check endpoints
curl http://localhost:8080/health  # API Gateway (KrakenD)
curl http://localhost:9000/health  # POS service
```

---

## 3. Project Makefile Reference

```bash
# ── Development ───────────────────────────────────────────────────────
make dev            # Start full local dev stack (infra + services + web)
make services-up    # Start Go microservices only
make services-down  # Stop Go microservices

# ── Build ─────────────────────────────────────────────────────────────
make build              # Build all Go services
make build SERVICE=pos  # Build specific service
make docker-build       # Build all Docker images

# ── Testing ───────────────────────────────────────────────────────────
make test               # Run all unit tests
make test-integration   # Run integration tests (requires Docker)
make test-e2e           # Run E2E tests
make test-coverage      # Run tests + generate coverage report
make stress-test        # Run k6 stress tests

# ── Code Quality ──────────────────────────────────────────────────────
make lint           # Run golangci-lint 2.12.2
make lint-fix       # Run golangci-lint with --fix
make vet            # Run go vet
make fmt            # Run gofmt + goimports

# ── Protobuf ──────────────────────────────────────────────────────────
make proto          # Generate all proto code (Go + TS + Dart)
make proto-lint     # Lint .proto files with buf
make proto-breaking # Check for breaking changes against main

# ── Database ──────────────────────────────────────────────────────────
make migrate-all                    # Run all migrations
make migrate SERVICE=pos            # Migrate specific service
make migrate-down SERVICE=pos       # Rollback last migration
make migrate-status SERVICE=pos     # Show migration status
make migrate-create SERVICE=pos NAME=add_column_x  # Create new migration

# ── Code Generation ───────────────────────────────────────────────────
make mocks              # Regenerate all mocks with mockery
make swagger            # Generate Swagger docs for all services
make wire               # (DEPRECATED — use Manual DI in provider.go)

# ── Infrastructure ────────────────────────────────────────────────────
make infra-up       # docker compose up -d
make infra-down     # docker compose down
make infra-reset    # docker compose down -v (DESTROYS DATA)

# ── Utilities ─────────────────────────────────────────────────────────
make clean          # Clean build artifacts
make deps-update    # Update all Go dependencies
make audit          # Run security audit (gosec + govulncheck)
make docs           # Open docs in browser
```

---

## 4. Adding a New Microservice

Follow these steps exactly when creating a new bounded context service.

```bash
# Step 1: Create directory structure
mkdir -p services/{name}/{cmd/server,internal/{domain,application/{command,query},infrastructure/{postgres,kafka,redis},interfaces/grpc}}
mkdir -p services/{name}/tests/{unit,integration,e2e}

# Step 2: Initialize Go module
cd services/{name}
go mod init github.com/xyn/pos-v1/services/{name}

# Step 3: Add to Go workspace
cd ../..
go work use ./services/{name}

# Step 4: Create base files
touch services/{name}/cmd/server/main.go
touch services/{name}/provider.go          # Manual DI
touch services/{name}/Dockerfile
touch services/{name}/.env.example
touch services/{name}/CLAUDE.md            # Service-specific AI context

# Step 5: Create proto definition
mkdir -p proto/{name}/v1
touch proto/{name}/v1/{name}.proto

# Step 6: Add to docker-compose.yml
# Step 7: Add to Makefile targets
# Step 8: Add to ArgoCD app definition
```

**Template for `provider.go` (Manual DI):**

```go
// services/{name}/provider.go
package main

import (
    "fmt"

    "github.com/xyn/pos-v1/shared/go/pkg/database"
    "github.com/xyn/pos-v1/shared/go/pkg/events"
    "github.com/xyn/pos-v1/shared/go/pkg/telemetry"
    "github.com/xyn/pos-v1/services/{name}/internal/application"
    "github.com/xyn/pos-v1/services/{name}/internal/infrastructure/postgres"
    "github.com/xyn/pos-v1/services/{name}/internal/interfaces/grpc"
)

func NewServer(cfg *Config) (*Server, error) {
    // Infrastructure
    db, err := database.NewPool(cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("database pool: %w", err)
    }

    publisher, err := events.NewKafkaPublisher(cfg.KafkaBrokers)
    if err != nil {
        return nil, fmt.Errorf("kafka publisher: %w", err)
    }

    tp, err := telemetry.NewTracerProvider(cfg.OTELEndpoint, "{name}-service")
    if err != nil {
        return nil, fmt.Errorf("tracer provider: %w", err)
    }

    // Infrastructure adapters (implement domain ports)
    repo := postgres.New{Name}Repository(db)

    // Application layer
    svc := application.New{Name}Service(repo, publisher)

    // Interface layer
    handler := grpchandler.New{Name}Handler(svc)

    return &Server{
        grpc:   buildGRPCServer(handler),
        tracer: tp,
        db:     db,
    }, nil
}
```

---

## 5. Adding a New API Endpoint

### Step 1: Define in Proto

```protobuf
// proto/{service}/v1/{service}.proto
service {Service}Service {
    rpc {MethodName} ({MethodName}Request) returns ({MethodName}Response) {
        option (google.api.http) = {
            post: "/v1/{resource}"
            body: "*"
        };
    };
}
```

### Step 2: Generate Code

```bash
make proto
```

### Step 3: Implement in Domain

```go
// services/{svc}/internal/domain/{entity}/{entity}.go
// Add the behavior to the aggregate
func (e *{Entity}) {Action}(...) error {
    // Business rule validation
    // State transition
    // Emit domain event
}
```

### Step 4: Implement Use Case

```go
// services/{svc}/internal/application/command/{action}.go
type {Action}Command struct { ... }
type {Action}Handler struct { ... }
func (h *{Action}Handler) Handle(ctx context.Context, cmd {Action}Command) error { ... }
```

### Step 5: Implement Handler

```go
// services/{svc}/internal/interfaces/grpc/handler.go
func (h *{Service}Handler) {MethodName}(ctx context.Context, req *pb.{MethodName}Request) (*pb.{MethodName}Response, error) { ... }
```

### Step 6: Wire in `provider.go`

Add the new handler to the dependency graph in `provider.go`.

### Step 7: Write Tests

- Unit test: domain aggregate behavior
- Unit test: use case (with mocked repo)
- Integration test: repository + real DB
- Add to k6 scenario if high-traffic

### Step 8: Generate Swagger Docs

```bash
make swagger SERVICE={svc}
```

---

## 6. Creating a Database Migration

```bash
# Create migration file
make migrate-create SERVICE=pos NAME=add_receipt_number_to_orders

# Edit the generated file:
# services/pos/internal/infrastructure/postgres/migrations/
# {timestamp}_add_receipt_number_to_orders.sql
```

**Migration file template:**

```sql
-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders ADD COLUMN receipt_number VARCHAR(20);

-- Backfill existing rows (run in batches for large tables)
-- See database-strategy.md for zero-downtime migration pattern
UPDATE orders SET receipt_number = 'MIGRATED-' || id::varchar(8)
WHERE receipt_number IS NULL;

ALTER TABLE orders ALTER COLUMN receipt_number SET NOT NULL;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders DROP COLUMN receipt_number;
-- +goose StatementEnd
```

**Migration rules** (from database-strategy.md):
- Never edit a migration after it's been committed
- Always include a Down migration
- Use zero-downtime patterns for large tables (see `database-strategy.md §5.3`)
- Never drop a column in the same PR that removes its usage

---

## 7. Pre-commit Hooks

The following checks run automatically on every commit:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: go-lint
        name: golangci-lint
        language: system
        entry: golangci-lint run --fix
        types: [go]
        pass_filenames: false

      - id: go-test
        name: go test (changed packages only)
        language: system
        entry: bash -c 'go test $(git diff --cached --name-only | grep "\.go$" | xargs -I{} dirname {} | sort -u | tr "\n" " ")'
        types: [go]

      - id: buf-lint
        name: buf lint
        language: system
        entry: buf lint
        files: \.proto$
        pass_filenames: false

      - id: no-secrets
        name: detect secrets
        language: system
        entry: git secrets --scan

      - id: go-mod-tidy
        name: go mod tidy
        language: system
        entry: bash -c 'go work sync && go mod tidy'
        pass_filenames: false
```

If a hook fails, fix the issue before committing. **Never use `--no-verify`** unless you have a documented emergency reason.

---

## 8. Code Review Guidelines

### As an Author

- PR description explains **why**, not just **what**
- Link to the relevant issue/ticket
- Self-review your diff before requesting review
- Mark draft PRs clearly — don't request review on drafts
- Keep PRs focused — one concern per PR
- Respond to all review comments (resolve or discuss)

### As a Reviewer

Focus on:
1. **Correctness** — Does it do what it claims? Are edge cases handled?
2. **Security** — SQL injection? XSS? Token leakage? RLS bypass?
3. **Architecture** — Does it follow DDD layer rules? No domain imports in infra?
4. **Tests** — Are the right scenarios covered? Are tests meaningful?
5. **Performance** — N+1 queries? Missing indexes? Unbounded loops?

Do NOT nitpick:
- Variable naming style (unless it violates the naming rules in `rules.md`)
- Comment formatting (unless it contradicts the rules)
- Personal style preferences

**Blocking vs non-blocking comments:**
- Use `blocking:` prefix for issues that must be fixed before merge
- Use `nit:` prefix for suggestions that don't block merge
- Use `question:` for clarification requests (non-blocking unless answered)

---

## 9. Common Gotchas

| Gotcha | What Happens | Solution |
|---|---|---|
| Forgetting `SET app.current_tenant_id` | All RLS-protected queries return 0 rows — looks like empty data, not an error | Always use `withTenantCtx()` helper in repositories |
| Using `float64` for money | `0.1 + 0.2 = 0.30000000000000004` in payments | Use `int64` minor units or `github.com/shopspring/decimal` |
| Missing idempotency key | Duplicate payments on retry | Always generate UUID v4 on the client before the call |
| Kafka topic not existing | Consumer blocks forever | Pre-create topics in `docker-compose.yml` init container |
| Go workspace out of sync | `go: inconsistent vendoring` errors | Run `go work sync` |
| Buf cache stale | Old generated code | Run `buf generate --clean` |
| Riverpod 3.x breaking changes | Provider not found / type errors | See `mobile-architecture.md §3` — Riverpod 3 has new `@riverpod` syntax |
| Wire archived | `go get github.com/google/wire` fails | Use Manual DI in `provider.go` — Wire is never used in this project |
| Kafka 4.3 KRaft | ZooKeeper config doesn't work | See `docker-compose.yml` — uses `KAFKA_KRAFT_MODE=true` |
| golangci-lint v2 config | Old `.golangci.yml` format errors | v2 uses different config schema — see `rules.md §1.8` |

---

## 10. Getting Help

- **Architecture questions** → Read `docs/phase-1/architecture.md` first, then ask in `#backend` channel
- **Frontend questions** → Read `apps/web/CLAUDE.md` first
- **Mobile questions** → Read `apps/mobile/CLAUDE.md` and `docs/phase-1/mobile-architecture.md`
- **Deployment** → Read `docs/phase-1/deployment-guide.md`
- **Something broken in CI** → Check recent commits to `.github/workflows/` first
- **Performance issue** → Read `docs/phase-1/observability.md` — check Grafana dashboard first

When asking for help, always include:
1. What you were trying to do
2. What you expected to happen
3. What actually happened (error message, logs)
4. What you've already tried
