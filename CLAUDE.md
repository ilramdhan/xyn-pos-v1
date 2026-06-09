# xyn-pos-v1 — AI Assistant Context

> Read this file completely before making any changes to this codebase.  
> Last updated: 2026-06-10 | Stack versions pinned below are **authoritative**.

---

## What This Project Is

**xyn-pos-v1** is a multi-tenant SaaS Point of Sales & ERP Mini platform built for enterprise scale from day one. It serves F&B, Retail, Sports, and AYCE business types across web, mobile (iOS/Android offline-first), and hardware peripherals (thermal printers, barcode scanners, cash drawers).

**Business model:** Multi-tenant, multi-branch, multi-warehouse, subscription tiers (Freemium / Growth / Enterprise).  
**Architecture:** Go microservices + DDD + gRPC (internal) + REST (external via gRPC-Gateway). Monorepo.  
**Learning goal:** This is also a deliberate learning project — always explain the WHY behind decisions, guide step-by-step, never dump everything at once.

---

## Authoritative Tech Stack Versions

> These are the ONLY versions to reference. Do not use older versions from training data.

### Backend
| Package | Version | Notes |
|---|---|---|
| Go | **1.26.4** | Use new stdlib features (range-over-func, etc.) |
| gRPC-Go | **1.81.1** | |
| pgx (PostgreSQL driver) | **5.10.0** | Use pgxpool, not database/sql |
| Goose (migrations) | **3.27.1** | Embed migrations with embed.FS |
| OpenTelemetry Go SDK | **1.44.0** | |
| golangci-lint | **2.12.2** | v2 config format — see rules.md |
| testify | **1.11.1** | |
| testcontainers-go | **0.42.0** | |
| samber/oops | **latest** | Rich error context |
| samber/lo | **latest** | Functional helpers |
| samber/slog-zap | **latest** | slog handler |
| **Wire** | ~~0.7.0~~ **ARCHIVED** | Use Manual DI — see docs/phase-1/architecture.md |

### Databases & Infrastructure
| Component | Version | Notes |
|---|---|---|
| PostgreSQL | **18.4** | RLS enabled, new JSON functions |
| Redis | **8.8** | New data structures |
| Kafka | **4.3.0** | **KRaft mode — ZooKeeper REMOVED** |
| ClickHouse | **26.5** | |
| Keycloak | **26.6.3** | |
| KrakenD | **2.13.7** | |
| Terraform | **1.15.5** | |
| K3s | **1.36.1** | |
| ArgoCD | **3.4.3** | |
| Grafana | **13.0.2** | |
| Prometheus | **3.12.0** | New TSDB format |
| k6 | **1.7.1** | |

### Frontend Web
| Package | Version | Notes |
|---|---|---|
| Next.js | **16.2.7** | App Router only |
| React | **19.2** | React Compiler enabled |
| TypeScript | **5.8+** | strict: true always |
| Tailwind CSS | **4.3** | CSS-first config |
| shadcn/ui | **June 2026 registry** | Components owned in-repo |
| TanStack Query | **5.80+** | v5 API only |
| Zustand | **5.0.14** | v5 API — breaking change from v4 |
| Buf | **1.70.0** | |

### Mobile
| Package | Version | Notes |
|---|---|---|
| Flutter | **3.44.0** | |
| Dart | **3.12** | Records, patterns, macros |
| Riverpod | **3.2.1** | Breaking changes from v2 — see mobile-architecture.md |
| Drift | **2.33.0** | |

---

## Repository Structure

```
xyn-pos-v1/
├── CLAUDE.md                    ← YOU ARE HERE
├── services/
│   ├── CLAUDE.md                ← Go service rules
│   ├── tenant/                  ← Tenant & Identity BC
│   ├── pos/                     ← POS Core & Cart BC
│   ├── payment/                 ← Payment & Checkout BC
│   ├── inventory/               ← Inventory & Supply Chain BC
│   ├── kitchen/                 ← Kitchen & Staff BC
│   ├── marketing/               ← Marketing & CRM BC
│   └── analytics/               ← Analytics & AI BC
├── proto/                       ← All .proto files (Buf managed)
├── shared/go/pkg/               ← Shared Go packages
├── apps/
│   ├── web/                     ← Next.js 16 web app
│   │   └── CLAUDE.md            ← Web-specific rules
│   └── mobile/                  ← Flutter app
│       └── CLAUDE.md            ← Mobile-specific rules
├── infra/
│   ├── argocd/                  ← ArgoCD App-of-Apps + ApplicationSet
│   ├── clickhouse/              ← ClickHouse init SQL + Debezium CDC connector
│   ├── grafana/                 ← Dashboard provisioning (service-health, business-metrics)
│   ├── k8s/
│   │   ├── base/                ← Base K8s manifests (all services + infra + network-policies + quotas)
│   │   └── overlays/            ← Kustomize overlays: dev / staging / prod
│   ├── krakend/                 ← KrakenD API gateway config
│   ├── otel/                    ← OTEL Collector config (tail-sampling, multi-backend)
│   ├── prometheus/              ← Prometheus scrape config
│   ├── promtail/                ← Promtail → Loki log shipping
│   └── terraform/               ← Hetzner Cloud + Cloudflare DNS
├── scripts/
│   ├── backup/                  ← pg_dump 3×/day, 7d retention, Backblaze B2
│   ├── k3s/                     ← K3s install + post-install (Helm, Sealed Secrets, ArgoCD)
│   ├── kafka/                   ← Kafka topic init (idempotent)
│   ├── postgres/                ← init.sql (databases, RLS roles)
│   └── templates/               ← Dockerfile.service template
└── docs/phase-1/                ← All architecture documents
```

---

## Critical Rules — NEVER Violate These

```
❌ NEVER import infrastructure packages into the domain layer
❌ NEVER put tenant_id in request body for auth — always extract from verified JWT claims
❌ NEVER use float32/float64 for money — use int64 minor units or decimal.Decimal
❌ NEVER skip idempotency key on financial operations (payment, refund, void)
❌ NEVER use SELECT * in production queries — always name columns explicitly
❌ NEVER log sensitive data (passwords, PINs, card numbers, tokens)
❌ NEVER use Wire (archived Aug 2025) — use Manual DI with constructor chaining
❌ NEVER use ZooKeeper — Kafka 4.3 runs KRaft mode only
❌ NEVER hardcode secrets — use environment variables + Vault
❌ NEVER create a goroutine without a documented exit condition
❌ NEVER use context.Background() inside a request handler — propagate the request context
❌ NEVER use math/rand for security — use crypto/rand
```

---

## Always Do

```
✅ ALWAYS set app.current_tenant_id in DB session before queries (RLS enforcement)
✅ ALWAYS accept context.Context as first parameter in any I/O function
✅ ALWAYS wrap errors with fmt.Errorf("operation: %w", err) at layer boundaries
✅ ALWAYS use the standard BaseResponse[T] wrapper for REST responses (see response-standards.md)
✅ ALWAYS run make lint before suggesting code changes
✅ ALWAYS use pgxpool (not sql.DB) for database connections
✅ ALWAYS use structured logging with slog (not fmt.Println or log.Printf)
✅ ALWAYS check if an operation needs an idempotency key (financial writes = yes)
✅ ALWAYS use UUID v4 for entity IDs (gen_random_uuid() in PostgreSQL 18)
✅ ALWAYS add OpenTelemetry span at service boundaries
```

---

## DDD Layer Dependency Rule

```
interfaces/     → calls application/
application/    → calls domain/
infrastructure/ → implements domain/ interfaces (ports)
domain/         → ZERO external imports

Direction: interfaces → application → domain ← infrastructure
```

If you see an import violation, flag it immediately. This is a blocking issue.

---

## Manual DI Pattern (Replaces Wire)

```go
// services/{name}/provider.go — construct dependency graph here
func NewServer(cfg *config.Config) (*grpc.Server, error) {
    db, err := database.NewPool(cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("database: %w", err)
    }

    repo := postgres.NewOrderRepository(db)
    publisher := kafka.NewEventPublisher(cfg.KafkaBrokers)
    svc := application.NewOrderService(repo, publisher)
    handler := grpchandler.NewOrderHandler(svc)

    return grpc.NewServer(handler), nil
}
```

No magic. No reflection. The dependency graph is readable in one file.

---

## Response Standard (REST)

All REST endpoints return `BaseResponse[T]`. See `docs/phase-1/response-standards.md`.

```json
{
  "request_id": "trace-abc123",
  "status_code": "200",
  "is_success": true,
  "message": "Order created successfully",
  "data": { ... },
  "timestamp": "2026-06-05T14:00:00Z"
}
```

---

## Error Handling Chain

```
Domain Error (ErrOrderNotFound)
    → Application wraps: fmt.Errorf("GetOrder: %w", err)
    → Infrastructure wraps: fmt.Errorf("orderRepo.FindByID: %w", err)
    → gRPC handler maps to: status.Error(codes.NotFound, ...)
    → REST gateway maps to: HTTP 404 + ErrorResponse
```

See `docs/phase-1/error-handling.md` for full taxonomy.

---

## Key Architectural Decisions

| Decision | Choice | Reason |
|---|---|---|
| DI Framework | **Manual DI** | Wire archived; explicit > magic |
| API Gateway | **KrakenD 2.13.7** | Stateless, 140k+ req/s, GitOps config |
| Auth | **Keycloak 26.6.3 + PASETO v4** | Enterprise RBAC, no JWT algo confusion |
| Message Broker | **Kafka 4.3 KRaft** | Log retention for event replay, CDC to ClickHouse |
| Multi-tenancy | **PostgreSQL RLS + Hybrid** | Shared DB + dedicated for Enterprise tier |
| Mobile sync | **Outbox + CRDT** | Safe offline ops, conflict-free stock merges |
| Money type | **int64 minor units** | No floating point errors |

---

## Docs Index

| Document | What's Inside |
|---|---|
| `docs/phase-1/architecture.md` | Bounded contexts, ERD, event flows, ADR index |
| `docs/phase-1/database-strategy.md` | RLS, multi-tenancy, offline sync, migrations |
| `docs/phase-1/hardware-integration.md` | ESC/POS, WebUSB, barcode, cash drawer |
| `docs/phase-1/rules.md` | Golang rules, testing rules, git workflow |
| `docs/phase-1/frontend-rules.md` | **Frontend rules — DRY, shared components, testing, naming** |
| `docs/phase-1/design.md` | UI/UX design system, shadcn/ui, Flutter theme |
| `docs/phase-1/api-contracts.md` | gRPC proto standards, idempotency, streaming |
| `docs/phase-1/roadmap.md` | Epic/Story breakdown Phase 1–6 |
| `docs/phase-1/response-standards.md` | **BaseResponse[T] pattern, error catalog** |
| `docs/phase-1/error-handling.md` | **Error taxonomy, wrapping strategy** |
| `docs/phase-1/testing-strategy.md` | **Unit, integration, e2e, k6 stress tests** |
| `docs/phase-1/contributing.md` | **Onboarding, local setup, first PR** |
| `docs/phase-1/environment-variables.md` | **All env vars, secret management** |
| `docs/phase-1/observability.md` | **OTEL, Grafana 13, Prometheus 3, Loki** |
| `docs/phase-1/mobile-architecture.md` | **Flutter deep-dive, Riverpod 3, Drift** |
| `docs/phase-1/deployment-guide.md` | **Docker → K3s → K8s, ArgoCD GitOps** |

---

## Phase Status

| Phase | Name | Status |
|---|---|---|
| Phase 1 | Architecture & Blueprinting | ✅ Done |
| Phase 2 | Environment & Infrastructure | ✅ Done |
| Phase 3 | Core Domain & Boilerplate | ✅ Done |
| Phase 4 | Backend MVP | ✅ Done (PR #3 merged) |
| Phase 5 | Frontend Web & Mobile | 🔲 In Progress |
| Phase 6 | Advanced Features & DevOps | ⬜ Planned |

Phase 5 plans: `docs/superpowers/plans/2026-06-10-phase5{a,b,c}-*.md`

---

## When Working in This Codebase

### Adding a new backend feature (Go microservice):
1. Start in `domain/` — define the aggregate and its behavior
2. Add repository interface (port) to `domain/`
3. Write use case in `application/command/` or `application/query/`
4. Implement repository in `infrastructure/postgres/`
5. Expose via `interfaces/grpc/handler.go`
6. Add to `provider.go` (Manual DI)
7. Write proto in `proto/{service}/v1/`
8. Run `buf generate`

**Never skip the domain layer.** Business logic in handlers is an architecture violation.

### Adding a new frontend feature (Next.js):
1. Check if a shared component exists first (`components/ui/` or `components/shared/`)
2. If new shared component needed → create in `components/shared/` with a `.test.tsx`
3. Build feature component in `components/features/{domain}/`
4. Add TanStack Query hook to `services/{domain}.ts`
5. Wire into the page in `app/(layout-group)/route/page.tsx`
6. All data states: loading → skeleton, error → ErrorState, empty → EmptyState
7. `use client` only when you need interactivity

See `docs/phase-1/frontend-rules.md` for full frontend rules.  
See `apps/web/CLAUDE.md` for web-specific patterns.

**Before every backend PR:**
```bash
make lint      # golangci-lint 2.12.2
make test      # go test ./...
make proto     # buf generate (if .proto changed)
```

**Before every frontend PR:**
```bash
cd apps/web
pnpm typecheck   # zero TypeScript errors
pnpm lint        # zero ESLint errors
pnpm test        # all Vitest tests green
pnpm build       # production build succeeds
```
