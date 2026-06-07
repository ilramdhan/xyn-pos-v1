# Phase 3 Design Spec — Core Domain & Microservices Boilerplate

> Date: 2026-06-07  
> Status: Approved  
> Owner: Principal Engineer  
> Branch: `feat/phase3-core-domain-boilerplate`

---

## 1. Scope

Phase 3 delivers the **foundational layer** that all 7 microservices will depend on. No business logic is shipped here — this is the skeleton, toolbox, and first working template.

### What Is (and Is NOT) in Phase 3

| In Scope | Out of Scope |
|---|---|
| Shared Go packages (errors, logger, auth, database, telemetry, events, middleware) | Registration / login flows → Phase 4 |
| Common Proto definitions + codegen (Go + TS + Dart) | Freemium / subscription billing → Phase 4 |
| Tenant Service: full DDD boilerplate template | POS, Payment, Inventory service logic → Phase 4 |
| PostgreSQL schema + Goose migrations for Tenant BC | Redis Cluster config → Phase 3.4 (partial) |
| RLS policy template function | KrakenD API Gateway config → Phase 4 |
| OpenTelemetry instrumentation on all layers | gRPC-Gateway REST transcoding → Phase 4 |
| Unit + integration tests for all layers | E2E / k6 stress tests → Phase 4+ |

---

## 2. Execution Order (Approach A — Bottom-Up)

```
Epic 3.2 → Epic 3.1 → Epic 3.4 → Epic 3.3
  Proto       Shared    DB Infra   Tenant Svc
  codegen     pkgs      + RLS      (template)
```

Each epic is a merge-ready unit. Later epics depend on earlier ones.

---

## 3. Epic 3.2 — Common Proto Definitions

### 3.2.1 Existing Files (Do Not Modify)

- `proto/common/v1/money.proto` — `Money{amount_minor int64, currency_code string}`
- `proto/common/v1/pagination.proto` — `Pagination`, `PaginationMeta`
- `proto/common/v1/error.proto` — Standard error detail

### 3.2.2 New Files

#### `proto/common/v1/address.proto`

```proto
message Address {
  string street    = 1;
  string city      = 2;
  string province  = 3;
  string postal_code = 4;
  string country   = 5; // ISO 3166-1 alpha-2, e.g. "ID"
}
```

#### `proto/tenant/v1/tenant.proto`

RPCs:
- `CreateTenant(CreateTenantRequest) → CreateTenantResponse`
- `GetTenant(GetTenantRequest) → GetTenantResponse`
- `CreateBranch(CreateBranchRequest) → CreateBranchResponse`
- `ListBranches(ListBranchesRequest) → ListBranchesResponse`

Key message fields:
- `Tenant`: id, name, slug, plan (enum: FREE/GROWTH/ENTERPRISE), status, created_at, updated_at
- `Branch`: id, tenant_id, name, address (Address), timezone, is_active, created_at
- `SubscriptionPlan`: max_branches, max_users, features (repeated string)

All write RPCs include `string idempotency_key = 1` as first field.

### 3.2.3 buf.gen.yaml Outputs

```
proto/ → gen/go/          (Go pb + grpc stubs)
proto/ → gen/ts/          (TypeScript via connect-es)
proto/ → gen/dart/        (Dart via protoc-gen-dart)
```

Generated code is committed to the repo. Never edit generated files.

---

## 4. Epic 3.1 — Shared Go Packages

All packages live under `shared/go/pkg/`. Each has its own `_test.go`. Zero circular dependencies between packages.

### 4.1 `errors/`

```
shared/go/pkg/errors/
├── codes.go        ← Error code constants (string enum)
├── domain.go       ← Base domain error type
├── grpc.go         ← MapToGRPCStatus(err) → *status.Status
└── http.go         ← MapToHTTPStatus(err) → int
```

Key design:
- `MapToGRPCStatus` wraps oops error codes → gRPC status codes. Single source of truth — all services call this.
- Error code string constants (e.g., `CodeNotFound = "NOT_FOUND"`) prevent typo divergence.
- `IsNotFound(err)`, `IsConflict(err)` — convenience helpers used in application layer.

### 4.2 `logger/`

```
shared/go/pkg/logger/
├── logger.go       ← NewLogger(cfg) → *slog.Logger using slog-zap handler
├── context.go      ← FromContext(ctx), WithTraceID(ctx, id), WithTenantID(ctx, id)
└── middleware.go   ← SlogFields(ctx) → []any — extracts trace_id, tenant_id, user_id
```

Key design:
- Logger is always extracted from context — no global logger variable.
- `WithTraceID` injects the OTEL trace ID into the logger so every log line is correlatable.
- Log fields: `trace_id`, `span_id`, `tenant_id`, `user_id`, `service`, `env`.

### 4.3 `auth/`

```
shared/go/pkg/auth/
├── claims.go       ← Claims struct: TenantID, UserID, BranchID, Roles, ExpiresAt
├── paseto.go       ← GenerateToken(claims, key) → string, VerifyToken(token, key) → *Claims
└── context.go      ← ClaimsFromContext(ctx) → *Claims, WithClaims(ctx, claims) → ctx
```

Key design:
- PASETO v4 local (symmetric) for inter-service tokens.
- `ClaimsFromContext` panics if claims not in context — forces interceptor to always inject them.
- `TenantID` and `UserID` are `uuid.UUID`, never strings — prevents confusion.

### 4.4 `database/`

```
shared/go/pkg/database/
├── pool.go         ← NewPool(dsn string) → (*pgxpool.Pool, error)
├── tenant.go       ← WithTenantContext(ctx, pool, tenantID) — executes SET LOCAL
└── tx.go           ← WithTransaction(ctx, pool, fn func(pgx.Tx) error) error
```

Key design:
- `WithTenantContext` wraps every query execution with `SET LOCAL app.current_tenant_id = $1` — RLS is automatically enforced.
- `WithTransaction` rolls back on error automatically.
- Pool config: `MaxConns=25`, `MinConns=5`, `MaxConnLifetime=1h`, `HealthCheckPeriod=30s`.

### 4.5 `telemetry/`

```
shared/go/pkg/telemetry/
├── setup.go        ← Setup(cfg TelemetryConfig) (shutdown func(), err error)
├── tracer.go       ← Tracer(serviceName) → otel.Tracer — thin wrapper
├── meter.go        ← Meter(serviceName) → otel.Meter — thin wrapper
└── span.go         ← RecordError(span, err), SetStatus(span, err)
```

Key design:
- `Setup()` configures OTLP gRPC exporter → OTEL Collector. Returns a `shutdown func()` called in `defer` in `main()`.
- `TelemetryConfig`: `ServiceName`, `ServiceVersion`, `OTLPEndpoint`, `Environment`.
- `RecordError(span, err)` sets span status to Error + records exception — used at every layer boundary.

### 4.6 `events/`

```
shared/go/pkg/events/
├── publisher.go    ← KafkaPublisher: Publish(ctx, topic, key, payload) error
├── consumer.go     ← KafkaConsumer: Subscribe(ctx, topic, group, handler func([]byte) error)
└── envelope.go     ← EventEnvelope{EventID, EventType, TenantID, OccurredAt, Payload}
```

Key design:
- All events are wrapped in `EventEnvelope` — includes `tenant_id` for consumer-side RLS isolation.
- Publisher uses `franz-go` with idempotent producer enabled.
- `EventEnvelope` is JSON-serialized — Protobuf for events is Phase 4+.

### 4.7 `middleware/`

```
shared/go/pkg/middleware/
├── chain.go        ← ChainUnary(...grpc.UnaryServerInterceptor) grpc.ServerOption
├── recovery.go     ← RecoveryInterceptor — panic → gRPC Internal error, never crash
├── auth.go         ← AuthInterceptor(verifyFn) — validates PASETO, injects Claims to ctx
├── logging.go      ← LoggingInterceptor — logs method, duration, status, trace_id
├── tracing.go      ← TracingInterceptor — creates span per RPC, propagates trace context
└── tenant.go       ← TenantInterceptor — extracts tenant_id from claims, validates non-zero
```

Interceptor chain order (outermost → innermost):
```
recovery → tracing → logging → auth → tenant → handler
```

Rationale: recovery must be outermost (catch panics from all others). Tracing second so all inner interceptors appear as children. Auth before tenant (tenant requires valid claims). Logging after tracing so trace_id is available in log fields.

---

## 5. Epic 3.4 — Database Infrastructure

### 5.1 PostgreSQL Schema for Tenant BC

Migration file: `services/tenant/internal/infrastructure/postgres/migrations/001_create_tenant_schema.sql`

```sql
-- Tenants table
CREATE TABLE tenants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL UNIQUE,
    plan        TEXT        NOT NULL DEFAULT 'FREE'
                            CHECK (plan IN ('FREE', 'GROWTH', 'ENTERPRISE')),
    status      TEXT        NOT NULL DEFAULT 'ACTIVE'
                            CHECK (status IN ('ACTIVE', 'SUSPENDED', 'DELETED')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Branches table
CREATE TABLE branches (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL REFERENCES tenants(id),
    name            TEXT        NOT NULL,
    street          TEXT,
    city            TEXT,
    province        TEXT,
    postal_code     TEXT,
    country         TEXT        NOT NULL DEFAULT 'ID',
    timezone        TEXT        NOT NULL DEFAULT 'Asia/Jakarta',
    is_active       BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Idempotency keys for write operations
CREATE TABLE idempotency_keys (
    key         TEXT        PRIMARY KEY,
    tenant_id   UUID        NOT NULL,
    operation   TEXT        NOT NULL,
    result_json JSONB,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_branches_tenant_id ON branches(tenant_id);
CREATE INDEX idx_idempotency_keys_expires ON idempotency_keys(expires_at);
```

### 5.2 RLS Policies

```sql
-- Enable RLS on both tables
ALTER TABLE tenants  ENABLE ROW LEVEL SECURITY;
ALTER TABLE branches ENABLE ROW LEVEL SECURITY;

-- Tenants: can only see their own row
CREATE POLICY tenant_isolation ON tenants
    USING (id::text = current_setting('app.current_tenant_id', true));

-- Branches: can only see branches belonging to their tenant
CREATE POLICY branch_isolation ON branches
    USING (tenant_id::text = current_setting('app.current_tenant_id', true));
```

### 5.3 Goose Embed Pattern

```go
// services/tenant/internal/infrastructure/postgres/migrations.go
//go:embed migrations/*.sql
var migrationsFS embed.FS
```

Migration runs at service startup via `goose.SetBaseFS(migrationsFS)`.

### 5.4 Redis Setup

`shared/go/pkg/cache/` (simple client factory for idempotency key storage):
- `NewClient(cfg RedisConfig) (*redis.Client, error)`
- Used by Tenant Service to check/store idempotency keys in Phase 4.
- Redis 8.8, TLS in production, `redis.UniversalClient` to support single/cluster.

---

## 6. Epic 3.3 — Tenant Service (Template Boilerplate)

### 6.1 Directory Structure

```
services/tenant/
├── cmd/server/main.go          ← Bootstrap only: config, telemetry, provider, graceful shutdown
├── provider.go                 ← Manual DI: all layers wired here
├── config.go                   ← Viper config struct + Load() function
├── Dockerfile                  ← Multi-stage, distroless, non-root
├── go.mod                      ← Module: github.com/xyn-pos/services/tenant
├── internal/
│   ├── domain/
│   │   └── tenant/
│   │       ├── tenant.go       ← Tenant aggregate root
│   │       ├── branch.go       ← Branch entity
│   │       ├── plan.go         ← SubscriptionPlan value object
│   │       ├── errors.go       ← Sentinel errors
│   │       ├── events.go       ← TenantCreated, BranchAdded
│   │       ├── repository.go   ← TenantRepository interface
│   │       └── testdata.go     ← NewTestTenant(), NewTestBranch() builders
│   ├── application/
│   │   ├── command/
│   │   │   ├── create_tenant.go
│   │   │   └── create_branch.go
│   │   └── query/
│   │       ├── get_tenant.go
│   │       └── list_branches.go
│   ├── infrastructure/
│   │   └── postgres/
│   │       ├── tenant_repo.go
│   │       ├── migrations.go   ← embed.FS declaration
│   │       └── migrations/
│   │           └── 001_create_tenant_schema.sql
│   └── interfaces/
│       └── grpc/
│           ├── server.go       ← grpc.NewServer + interceptor chain
│           └── tenant_handler.go ← proto ↔ domain mapping
```

### 6.2 Domain Model

#### `Tenant` Aggregate

```go
type Tenant struct {
    ID        uuid.UUID
    Name      string
    Slug      string          // URL-safe, unique across all tenants
    Plan      SubscriptionPlan
    Status    TenantStatus
    Branches  []Branch
    events    []DomainEvent   // unexported, via PopEvents()
    CreatedAt time.Time
    UpdatedAt time.Time
}

func NewTenant(name, slug string, plan SubscriptionPlan) (*Tenant, error)
func (t *Tenant) AddBranch(b Branch) error       // validates plan limits
func (t *Tenant) Suspend(reason string) error
func (t *Tenant) PopEvents() []DomainEvent
```

`AddBranch` enforces `Plan.MaxBranches` limit → returns `ErrBranchLimitReached` if exceeded.

#### `SubscriptionPlan` Value Object

```go
type SubscriptionPlan struct {
    Tier        PlanTier   // FREE | GROWTH | ENTERPRISE
    MaxBranches int
    MaxUsers    int
    Features    []string
}

var (
    FreePlan       = SubscriptionPlan{Tier: PlanTierFree,       MaxBranches: 1,  MaxUsers: 3}
    GrowthPlan     = SubscriptionPlan{Tier: PlanTierGrowth,     MaxBranches: 5,  MaxUsers: 20}
    EnterprisePlan = SubscriptionPlan{Tier: PlanTierEnterprise, MaxBranches: -1, MaxUsers: -1} // unlimited
)
```

#### Sentinel Errors (`errors.go`)

```go
var (
    ErrTenantNotFound      = errors.New("tenant not found")
    ErrSlugAlreadyTaken    = errors.New("tenant slug is already taken")
    ErrBranchLimitReached  = errors.New("branch limit reached for subscription plan")
    ErrTenantSuspended     = errors.New("tenant is suspended")
    ErrInvalidSlug         = errors.New("slug must be lowercase alphanumeric with hyphens only")
)
```

### 6.3 Application Layer

#### `command/create_tenant.go`

```go
type CreateTenantCommand struct {
    IdempotencyKey string
    Name           string
    Slug           string
    Plan           domain.PlanTier
}

type CreateTenantHandler struct {
    repo      domain.TenantRepository
    publisher events.Publisher
}

func (h *CreateTenantHandler) Handle(ctx context.Context, cmd CreateTenantCommand) (*domain.Tenant, error)
```

Flow: validate slug format → check slug uniqueness → `domain.NewTenant()` → `repo.Save()` → publish `TenantCreated` event (non-fatal if Kafka is down).

#### `query/get_tenant.go`

```go
type GetTenantQuery struct {
    TenantID uuid.UUID
}

type GetTenantHandler struct {
    repo domain.TenantRepository
}

func (h *GetTenantHandler) Handle(ctx context.Context, q GetTenantQuery) (*domain.Tenant, error)
```

### 6.4 Repository Interface

```go
// internal/domain/tenant/repository.go
type TenantRepository interface {
    Save(ctx context.Context, t *Tenant) error
    FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
    FindBySlug(ctx context.Context, slug string) (*Tenant, error)
    ExistsBySlug(ctx context.Context, slug string) (bool, error)
    ListBranchesByTenant(ctx context.Context, tenantID uuid.UUID) ([]Branch, error)
}
```

### 6.5 Infrastructure — Postgres Repository

Key patterns:
- `Save()` uses `INSERT ... ON CONFLICT (id) DO UPDATE` (upsert) — idempotent writes.
- `FindByID()` maps `pgx.ErrNoRows` → `domain.ErrTenantNotFound` — never leaks DB errors.
- All queries use named columns explicitly — no `SELECT *`.
- RLS enforced via `database.WithTenantContext(ctx, pool, tenantID)` before every query block.

### 6.6 gRPC Interface Layer

`tenant_handler.go` responsibilities:
1. Extract `Claims` from context via `auth.ClaimsFromContext(ctx)`
2. Map proto request → application command/query (no business logic here)
3. Call application handler
4. Map domain result → proto response
5. Map errors via `errors.MapToGRPCStatus(err)`

```go
func (h *TenantHandler) CreateTenant(ctx context.Context, req *pb.CreateTenantRequest) (*pb.CreateTenantResponse, error) {
    span := otel.Tracer("tenant-service").Start(ctx, "CreateTenant")
    defer span.End()

    cmd := command.CreateTenantCommand{
        IdempotencyKey: req.IdempotencyKey,
        Name:           req.Name,
        Slug:           req.Slug,
        Plan:           domain.PlanTier(req.Plan.String()),
    }

    tenant, err := h.createTenant.Handle(ctx, cmd)
    if err != nil {
        telemetry.RecordError(span, err)
        return nil, errors.MapToGRPCStatus(err).Err()
    }

    return &pb.CreateTenantResponse{Tenant: toProto(tenant)}, nil
}
```

### 6.7 OpenTelemetry Per Layer

| Layer | Instrumentation |
|---|---|
| `main.go` | `telemetry.Setup()` → OTLP gRPC exporter |
| `interfaces/grpc` | `middleware.TracingInterceptor` auto-spans every RPC |
| `application/command` | Manual span: `tracer.Start(ctx, "CreateTenantHandler")` |
| `infrastructure/postgres` | `otelpgx.NewTracer()` wraps pool — auto DB spans |
| Kafka publisher | Manual span on `events.Publish()` |
| Logger | `trace_id` + `span_id` injected from context on every log line |

### 6.8 `main.go` Pattern

```go
func main() {
    cfg := config.Load()

    // Telemetry must be first
    shutdown, err := telemetry.Setup(telemetry.Config{
        ServiceName:    "tenant-service",
        ServiceVersion: cfg.Version,
        OTLPEndpoint:   cfg.OTLPEndpoint,
        Environment:    cfg.Env,
    })
    if err != nil { log.Fatalf("telemetry: %v", err) }
    defer shutdown()

    // Build dependency graph
    srv, err := provider.New(cfg)
    if err != nil { log.Fatalf("provider: %v", err) }

    // Graceful shutdown
    ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
    defer stop()

    go srv.Start()

    <-ctx.Done()
    srv.GracefulStop()
}
```

### 6.9 Testing Plan

| Layer | Type | Tool | Coverage Target |
|---|---|---|---|
| `domain/tenant` | Unit, table-driven | testify/suite | ≥ 90% |
| `application/command` | Unit with mocks | testify/mock + mockery | ≥ 80% |
| `application/query` | Unit with mocks | testify/mock + mockery | ≥ 80% |
| `infrastructure/postgres` | Integration | testcontainers-go PostgreSQL 18.4 | ≥ 60% |
| `interfaces/grpc` | Unit (mapping only) | testify | ≥ 75% |

**Critical test cases:**
- `TestAddBranch_WhenAtPlanLimit_ShouldReturnErrBranchLimitReached`
- `TestCreateTenant_DuplicateSlug_ShouldReturnErrSlugAlreadyTaken`
- `TestFindByID_DifferentTenant_ShouldReturnNotFound` (RLS isolation)
- `TestCreateTenant_Idempotent_ShouldReturnSameResult` (idempotency)

---

## 7. Dockerfile Pattern (All Services)

```dockerfile
# Build stage
FROM golang:1.26.4-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /service ./cmd/server

# Runtime stage — distroless, non-root
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /service /service
USER nonroot:nonroot
ENTRYPOINT ["/service"]
```

---

## 8. Branch & Commit Strategy

- Branch: `feat/phase3-core-domain-boilerplate`
- Commit order follows epic order: `feat(proto)` → `feat(shared)` → `feat(db)` → `feat(tenant)`
- Each epic is a separate commit with clear scope
- PR to `main` when all epics done, CI green

---

## 9. Definition of Done — Phase 3

- [ ] `buf generate` produces Go + TS + Dart code with zero errors
- [ ] All 7 shared packages have `_test.go` with ≥ 80% coverage
- [ ] Tenant Service compiles: `go build ./...` zero errors
- [ ] `make lint` passes with zero issues (golangci-lint 2.12.2)
- [ ] Unit tests pass: `go test -race ./...`
- [ ] Integration test passes: `TestTenantRepository_Integration` with testcontainers
- [ ] RLS test passes: `TestFindByID_DifferentTenant_ShouldReturnNotFound`
- [ ] OTEL trace appears in Jaeger for a CreateTenant call (manual smoke test)
- [ ] Dockerfile builds and runs without root privileges
