# Architecture Blueprint — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Draft | Owner: Principal Engineer

---

## 1. Tech Stack Evaluation & Architectural Insights

### 1.1 Backend: Golang + gRPC + DDD

**Why Golang?**
- Compiled, statically typed — catches class of bugs at compile time that dynamic languages miss
- Goroutines + channels give you true concurrency without the overhead of OS threads (1 goroutine ≈ 2–8 KB stack vs 1 MB for OS thread)
- Single binary deployment — no runtime dependency hell
- Excellent ecosystem for networking, gRPC, and observability tooling

**Architectural Decision: Manual DI (Replaces Wire)**
> ⚠️ `google/wire` was **archived on Aug 25, 2025** — read-only, no more updates.

Use **Manual DI** with constructor chaining in `provider.go`. This is actually the Go community's preferred approach:
- Zero dependencies — pure Go
- Fully readable — the dependency graph is a normal function you can read and debug
- No magic — no reflection, no code generation, zero runtime overhead
- Exactly as performant as Wire's generated code (because it IS what Wire would generate)

```go
// services/pos/provider.go
func NewServer(cfg *Config) (*Server, error) {
    db, _ := database.NewPool(cfg.DatabaseURL)
    repo := postgres.NewOrderRepository(db)
    svc := application.NewOrderService(repo)
    handler := grpchandler.NewOrderHandler(svc)
    return &Server{grpc: grpc.NewServer(handler)}, nil
}
```

**Why gRPC as Internal Protocol?**
- Binary serialization (Protobuf) is ~5–10x smaller payload than JSON
- Strict interface contracts enforced at compile time via generated code
- Built-in streaming for KDS (Kitchen Display System) and real-time inventory
- HTTP/2 multiplexing eliminates head-of-line blocking between microservices
- REST/JSON can be served to external clients via gRPC-Gateway — one source of truth (`.proto`) for both

**CQRS Implementation Strategy**

```
Command Side (Write)                Query Side (Read)
─────────────────                   ─────────────────
gRPC Service Handler                gRPC Service Handler
       │                                    │
  Command Bus                          Query Bus
       │                                    │
  Domain Aggregate                    Read Model (DTO)
       │                                    │
  Domain Events                       PostgreSQL View /
       │                               Redis Cache /
  Event Store                         ClickHouse
       │
  Event Handler → Read Model Projection
```

Commands mutate state; they return only an ID or acknowledgement — never a full entity.  
Queries read from optimized read models — never from aggregate state directly.  
This lets you scale reads and writes independently and evolve them separately.

**Idempotency for Financial Transactions**
Every write operation that touches money (checkout, refund, void) must:
1. Accept a client-generated `idempotency_key` (UUID v4)
2. Store the key + result in Redis with TTL = 24h
3. On duplicate request: return the cached result, skip re-execution
4. This makes retries safe across network failures

---

### 1.2 API Gateway

**Recommended: KrakenD (self-hosted) for MVP, migrate to Kong if needed**

| | KrakenD | Kong |
|---|---|---|
| Performance | ~140k req/s (stateless, no DB) | ~40k req/s (needs PostgreSQL) |
| Config | Declarative JSON — GitOps-friendly | DB-backed or declarative (deck) |
| Complexity | Low — single binary | Higher — ecosystem of plugins |
| JWT Validation | Native | Plugin |
| Rate Limiting | Native | Plugin |

KrakenD compiles its config into a binary. This is GitOps-native and aligns with your IaC philosophy. Use Kong when you need a plugin ecosystem that KrakenD cannot match (custom auth plugins, complex ACL).

---

### 1.3 Identity & Auth: Keycloak

**Why Keycloak over SuperTokens?**
- Enterprise-grade RBAC, fine-grained permissions, and group hierarchies — needed for multi-tenant, multi-branch RBAC
- Well-supported OIDC/OAuth2 flows
- Built-in multi-realm support maps cleanly to your tenant model
- Large community, battle-tested in finance/healthcare

Use **PASETO v4 local tokens** for session tokens between internal services (not JWT). PASETO avoids JWT's algorithm confusion vulnerabilities. Use JWT only when external clients (mobile, third-party integrations) require it.

---

### 1.4 Database Layer

| Role | Technology | Reasoning |
|---|---|---|
| OLTP | PostgreSQL **18.4** | ACID, JSONB for flexible schemas, Row-level security, new JSON functions |
| Analytics/OLAP | ClickHouse **26.5** | Columnar storage, 100-1000x faster aggregation than PostgreSQL for analytics |
| Cache / Session | Redis **8.8** (Cluster) | Sub-millisecond reads, new data types, pub/sub for real-time events |
| Object Storage | MinIO | S3-compatible, self-hostable, good for receipts, product images |

**Migrations: Goose**
Preferred over golang-migrate because:
- Go-based migrations (not just SQL) for complex data transforms
- Rollback support
- Version tracking via `goose_db_version` table
- Works well with embed.FS for packaging migrations into the binary

---

### 1.5 Message Broker: Kafka

Use **Kafka 4.3.0** over RabbitMQ for this system:

> ⚠️ **Kafka 4.3 removes ZooKeeper entirely.** Runs KRaft mode natively. Docker Compose config is significantly different — see `deployment-guide.md §2.2`.

- Kafka retains messages (log-based) → replay events for read model projection, audit trail, analytics pipeline
- ClickHouse Kafka engine can consume directly from Kafka topics
- Partition-based scaling aligns with your multi-tenant model (partition by tenant_id)
- RabbitMQ is a message queue — once consumed, gone. Bad for event sourcing and analytics pipelines.

Use **Kafka Connect** to stream PostgreSQL changes (via Debezium CDC) to ClickHouse without custom ETL code.

---

### 1.6 Frontend

**Web: Next.js 16.2.7 (App Router) + TypeScript 5.8+ + React 19.2**
- React 19 Compiler enabled — automatic memoization, no more manual `useMemo`/`useCallback`
- App Router Server Components → minimal JavaScript shipped to browser
- shadcn/ui (June 2026 registry) + Tailwind CSS 4.3 — components owned in-repo
- TanStack Query **5.80+** for server state, Zustand **5.0.14** for local/UI state
- Connect-Web (gRPC) for real-time KDS streaming

**Mobile: Flutter 3.44.0 / Dart 3.12 + Riverpod 3.2.1**
- Riverpod **3.x** has breaking changes from v2 — `Ref` is no longer generic
- Drift **2.33.0** (SQLite ORM) for offline-first local database — type-safe, reactive streams
- Dart 3.12 features: records, pattern matching, sealed classes, macros
- Background sync via `WorkManager` (Android) / `BGTaskScheduler` (iOS)

---

### 1.7 Infrastructure

| Component | Tool | Notes |
|---|---|---|
| IaC | Terraform **1.15.5** | Mature, provider ecosystem, HCL readable |
| Container Runtime | Docker / containerd | |
| Local K8s | K3s **1.36.1** | Lightweight, works on home server / Raspberry Pi |
| GitOps | ArgoCD **3.4.3** | Pulls from Git, self-heals drift |
| CI/CD | GitHub Actions | Triggers → build → test → push image → ArgoCD sync |
| Observability | OpenTelemetry Go **1.44.0** + Grafana **13.0.2** + Prometheus **3.12.0** | |
| Load Testing | k6 **1.7.1** | |

---

## 2. Monorepo vs Multi-Repo Decision

**Decision: Monorepo (using Go Workspaces + Turborepo)**

### Reasoning

| Concern | Monorepo | Multi-Repo |
|---|---|---|
| Atomic cross-service changes | ✅ One PR touches proto + service A + service B | ❌ Coordinated PRs across repos |
| Shared types / proto | ✅ Single source, no versioning drift | ❌ Proto version mismatch bugs |
| CI complexity | Moderate (affected-only builds) | High (cross-repo pipeline orchestration) |
| Team scaling | Fine until ~50+ engineers | Needed at ~100+ engineers |
| Code discovery | ✅ `grep` works everywhere | ❌ Need service catalog |

### Repository Structure

```
xyn-pos-v1/                          ← Monorepo root
├── apps/
│   ├── web/                         ← Next.js web app
│   ├── mobile/                      ← Flutter app
│   └── storybook/                   ← UI component docs
├── services/
│   ├── tenant/                      ← Tenant & Identity BC
│   ├── pos/                         ← POS Core & Cart BC
│   ├── payment/                     ← Payment & Checkout BC
│   ├── inventory/                   ← Inventory & Supply Chain BC
│   ├── kitchen/                     ← Kitchen & Staff BC
│   ├── marketing/                   ← Marketing & CRM BC
│   ├── analytics/                   ← Analytics & AI BC
│   └── hardware/                    ← Hardware Integration BC
├── proto/                           ← All .proto files (Buf managed)
│   ├── tenant/v1/
│   ├── pos/v1/
│   ├── payment/v1/
│   ├── inventory/v1/
│   └── ...
├── shared/
│   ├── go/
│   │   ├── pkg/                     ← Shared Go packages
│   │   │   ├── auth/                ← JWT/PASETO utilities
│   │   │   ├── database/            ← DB connection pool helpers
│   │   │   ├── errors/              ← Domain error types
│   │   │   ├── events/              ← Kafka event publishing
│   │   │   ├── logger/              ← Structured logging (slog)
│   │   │   ├── middleware/          ← gRPC interceptors
│   │   │   ├── telemetry/           ← OpenTelemetry setup
│   │   │   └── validator/           ← Input validation
│   │   └── go.work                  ← Go workspace file
│   └── ui/                          ← Shared React components
├── infra/
│   ├── terraform/                   ← IaC
│   ├── k8s/                         ← Kubernetes manifests
│   ├── docker/                      ← Dockerfiles
│   └── scripts/                     ← Dev scripts
├── docs/
│   ├── phase-1/                     ← This document lives here
│   ├── adr/                         ← Architecture Decision Records
│   └── runbooks/
├── tools/
│   ├── buf.gen.yaml                 ← Buf code generation config
│   └── buf.yaml                     ← Buf workspace config
├── .github/
│   └── workflows/
├── docker-compose.yml               ← Local dev environment
├── turbo.json                       ← Turborepo config (affected builds)
└── CLAUDE.md                        ← AI assistant instructions
```

---

## 3. DDD Bounded Contexts Design

### 3.1 Bounded Context Map

```
┌──────────────────────────────────────────────────────────────────────┐
│                        xyn-pos Platform                              │
│                                                                      │
│  ┌─────────────────┐     ┌─────────────────┐                        │
│  │  Tenant &        │────▶│  POS Core &     │                        │
│  │  Identity        │     │  Cart           │                        │
│  │  Context         │     │  Context        │                        │
│  └─────────────────┘     └────────┬────────┘                        │
│           │                       │                                  │
│           │                       ▼                                  │
│           │              ┌─────────────────┐                        │
│           │              │  Payment &       │                        │
│           └─────────────▶│  Checkout        │                        │
│                          │  Context         │                        │
│                          └────────┬────────┘                        │
│                                   │                                  │
│           ┌───────────────────────┼──────────────────┐              │
│           ▼                       ▼                   ▼              │
│  ┌──────────────┐      ┌──────────────────┐  ┌────────────────┐    │
│  │  Inventory & │      │  Kitchen &        │  │  Marketing &   │   │
│  │  Supply      │      │  Staff           │  │  CRM           │   │
│  │  Chain       │      │  Context         │  │  Context       │   │
│  └──────────────┘      └──────────────────┘  └────────────────┘   │
│           │                                            │             │
│           └────────────────────┬───────────────────────┘            │
│                                ▼                                     │
│                      ┌──────────────────┐                           │
│                      │  Analytics &     │                           │
│                      │  AI Context      │                           │
│                      └──────────────────┘                           │
│                                                                      │
│  ┌──────────────────────────────────────────────────────────┐       │
│  │           Hardware Integration Context (Cross-cutting)    │       │
│  └──────────────────────────────────────────────────────────┘       │
└──────────────────────────────────────────────────────────────────────┘
```

### 3.2 Context Boundaries & Relationships

| Context | Upstream | Downstream | Integration Pattern |
|---|---|---|---|
| Tenant & Identity | — | All others | Shared Kernel (TenantID, UserID types) |
| POS Core & Cart | Tenant, Inventory | Payment, Kitchen, Analytics | Events (OrderCreated, OrderUpdated) |
| Payment & Checkout | Tenant, POS Core | Analytics, Marketing | Events (PaymentCompleted, RefundProcessed) |
| Inventory & Supply Chain | Tenant | POS Core, Analytics, Marketing | Events (StockUpdated, LowStockAlert) |
| Kitchen & Staff | Tenant, POS Core | Analytics | Events (OrderDispatched, ItemReady) |
| Marketing & CRM | Tenant, Payment | Analytics | Events (MemberPointAdded, PromoApplied) |
| Analytics & AI | All others (consumer) | — | Event Consumer (all topics) |
| Hardware Integration | POS Core, Payment | — | Anti-Corruption Layer (driver adapters) |

### 3.3 Domain Model per Context

#### Tenant & Identity Context
```
Aggregates:   Tenant, User, Role, Branch, Subscription
Entities:     TenantConfig, SubscriptionPlan, Permission
Value Objects: TenantID, UserID, BranchID, Email, Phone, PIN
Domain Events: TenantProvisioned, UserRegistered, SubscriptionUpgraded, BranchCreated
```

#### POS Core & Cart Context
```
Aggregates:   Order, Shift, Menu
Entities:     OrderItem, OrderAddon, ShiftCashMovement, MenuItem, MenuCategory
Value Objects: OrderID, ShiftID, CartItem, Money, Quantity, TaxRate
Domain Events: OrderCreated, OrderItemAdded, OrderCancelled, ShiftOpened, ShiftClosed
```

#### Payment & Checkout Context
```
Aggregates:   Payment, Refund, Void
Entities:     PaymentMethod, PaymentGatewayTransaction, SplitPaymentEntry
Value Objects: PaymentID, Money, IdempotencyKey, ReceiptNumber
Domain Events: PaymentCompleted, PaymentFailed, RefundProcessed, VoidProcessed
```

#### Inventory & Supply Chain Context
```
Aggregates:   Product, StockLedger, PurchaseOrder, Warehouse
Entities:     StockMutation, BOMComponent, Supplier, StockOpname
Value Objects: ProductID, SKU, WarehouseID, Quantity, ReorderPoint
Domain Events: StockDeducted, StockReceived, LowStockAlert, POCreated
```

---

## 4. Internal Service Folder Structure (DDD)

Every microservice follows this structure:

```
services/pos/
├── cmd/
│   └── server/
│       └── main.go              ← Entry point (wire injection here)
├── internal/
│   ├── domain/                  ← Pure domain — NO external dependencies
│   │   ├── order/
│   │   │   ├── order.go         ← Aggregate root
│   │   │   ├── order_item.go    ← Entity
│   │   │   ├── events.go        ← Domain events
│   │   │   ├── repository.go    ← Repository interface (port)
│   │   │   └── service.go       ← Domain service (business rules)
│   │   └── shift/
│   ├── application/             ← Use cases — orchestrates domain
│   │   ├── command/
│   │   │   ├── create_order.go  ← Command + Handler
│   │   │   └── add_item.go
│   │   └── query/
│   │       ├── get_order.go     ← Query + Handler
│   │       └── list_orders.go
│   ├── infrastructure/          ← Adapters — implements ports
│   │   ├── postgres/
│   │   │   ├── order_repo.go    ← Implements domain/order/repository.go
│   │   │   └── migrations/
│   │   ├── kafka/
│   │   │   └── event_publisher.go
│   │   └── redis/
│   │       └── cache.go
│   └── interfaces/              ← Delivery layer
│       ├── grpc/
│       │   ├── server.go
│       │   └── handler.go       ← Calls application layer
│       └── http/                ← REST fallback (gRPC-Gateway generated)
├── wire.go                      ← Wire providers
├── wire_gen.go                  ← Wire generated file
├── Dockerfile
└── go.mod
```

**Dependency Rule (enforced by linter):**
```
interfaces → application → domain ← infrastructure
                                  ↑
                             (implements ports)
```
Infrastructure depends on domain (via ports/interfaces). Domain never depends on infrastructure.

---

## 5. High-Level ERD

### Core Entity Relationships

```
TENANT
├── id (UUID)
├── slug (unique)
├── name
├── plan_id → SUBSCRIPTION_PLAN
└── status

BRANCH (outlet)
├── id (UUID)
├── tenant_id → TENANT
├── name
├── address
└── timezone

USER
├── id (UUID)
├── tenant_id → TENANT
├── email (unique per tenant)
├── role_id → ROLE
└── branch_id → BRANCH (nullable, for branch-scoped users)

ORDER
├── id (UUID)
├── tenant_id → TENANT
├── branch_id → BRANCH
├── cashier_id → USER
├── shift_id → SHIFT
├── status (ENUM: draft, pending, paid, cancelled, void)
├── subtotal, tax, discount, total (DECIMAL)
├── idempotency_key (UNIQUE)
└── created_at, updated_at

ORDER_ITEM
├── id (UUID)
├── order_id → ORDER
├── product_id → PRODUCT
├── quantity, unit_price, subtotal
└── notes

PAYMENT
├── id (UUID)
├── order_id → ORDER (unique for single payment)
├── idempotency_key (UNIQUE)
├── method (ENUM: cash, card, qris, ewallet, etc.)
├── amount, change_amount
├── status (ENUM: pending, completed, failed, voided, refunded)
└── gateway_reference

PRODUCT
├── id (UUID)
├── tenant_id → TENANT
├── sku (unique per tenant)
├── name, description
├── category_id → CATEGORY
├── unit_price
├── track_inventory (BOOL)
└── is_active

STOCK_LEDGER
├── id (UUID)
├── product_id → PRODUCT
├── warehouse_id → WAREHOUSE
├── quantity_on_hand
└── updated_at

STOCK_MUTATION
├── id (UUID)
├── product_id → PRODUCT
├── warehouse_id → WAREHOUSE
├── type (ENUM: sale, purchase, adjustment, transfer_in, transfer_out)
├── quantity (signed — negative for deductions)
├── reference_id (order_id, po_id, etc.)
└── created_at

AUDIT_LOG
├── id (UUID)
├── tenant_id → TENANT
├── actor_id → USER
├── entity_type (VARCHAR)
├── entity_id (UUID)
├── action (ENUM: create, update, delete)
├── old_values (JSONB)
├── new_values (JSONB)
└── created_at
```

### Multi-Tenant Notes
- Every table that holds tenant data has a `tenant_id` column
- PostgreSQL Row-Level Security (RLS) policies enforce isolation at the database layer
- See `database-strategy.md` for full multi-tenancy strategy

---

## 6. Event Flow: Order-to-Payment

```
Client (Web/Mobile POS)
        │
        ▼
  API Gateway (KrakenD)
        │  validates JWT, rate limits
        ▼
  POS Service (gRPC)
        │
        ├── CreateOrder command
        │       │
        │       ▼
        │   Order aggregate created
        │   OrderCreated event → Kafka [pos.order.created]
        │
        ├── AddItems command
        │       │
        │       ▼
        │   Order aggregate updated
        │   OrderItemAdded events → Kafka [pos.order.updated]
        │
        └── Checkout command
                │
                ▼
        Payment Service (gRPC)
                │
                ├── Validates idempotency_key
                ├── Creates Payment aggregate
                ├── Calls Payment Gateway (Midtrans/Xendit)
                ├── PaymentCompleted event → Kafka [payment.completed]
                │
                └── Kafka Consumers:
                        ├── Inventory Service → deducts stock
                        ├── Kitchen Service → creates KDS ticket
                        ├── Analytics Service → updates ClickHouse
                        └── Marketing Service → awards loyalty points
```

---

## 7. Architecture Decision Records (ADR) Index

| ID | Decision | Status |
|---|---|---|
| ADR-001 | Use Monorepo with Go Workspaces | Accepted |
| ADR-002 | KrakenD as API Gateway for MVP | Accepted |
| ADR-003 | Keycloak for IAM | Accepted |
| ADR-004 | PASETO v4 for internal service tokens | Accepted |
| ADR-005 | Goose for DB migrations | Accepted |
| ADR-006 | Kafka over RabbitMQ | Accepted |
| ADR-007 | Wire over Uber FX for DI | Accepted |
| ADR-008 | Riverpod over BLoC for Flutter | Accepted |
| ADR-009 | Row-Level Security for multi-tenancy | Accepted (see database-strategy.md) |
| ADR-010 | KDS via gRPC Server Streaming | Proposed |
