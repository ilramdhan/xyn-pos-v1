# Go Microservices — AI Assistant Context

> Scope: All services under services/  
> Parent context: ../../CLAUDE.md (read that first)

---

## Service Anatomy

Every service follows this exact structure. Deviating from it requires an ADR.

```
services/{name}/
├── cmd/server/main.go       ← Wire deps (provider.go) + start server. Nothing else.
├── provider.go              ← Manual DI: constructs the full dependency graph
├── internal/
│   ├── domain/
│   │   └── {entity}/
│   │       ├── {entity}.go      ← Aggregate root (the "noun" of this domain)
│   │       ├── errors.go        ← Sentinel errors: var ErrXxx = errors.New(...)
│   │       ├── events.go        ← Domain events (structs, no behavior)
│   │       ├── repository.go    ← Repository interface (port) — stdlib types only
│   │       └── service.go       ← Domain service (stateless logic spanning aggregates)
│   ├── application/
│   │   ├── command/             ← Write operations: Create, Update, Delete, etc.
│   │   │   └── {verb}_{entity}.go
│   │   └── query/               ← Read operations: Get, List, Search, etc.
│   │       └── {get/list}_{entity}.go
│   ├── infrastructure/
│   │   ├── postgres/
│   │   │   ├── {entity}_repo.go ← Implements domain/{entity}/repository.go
│   │   │   └── migrations/      ← Goose SQL migrations
│   │   ├── kafka/
│   │   │   └── event_publisher.go
│   │   └── redis/
│   │       └── cache.go
│   └── interfaces/
│       ├── grpc/
│       │   ├── server.go        ← gRPC server setup + interceptors
│       │   └── handler.go       ← Translates proto ↔ domain, calls application layer
│       └── http/                ← gRPC-Gateway generated REST (usually auto-generated)
├── Dockerfile
├── go.mod
└── CLAUDE.md                    ← This file's more specific version, per service
```

---

## DDD Rules (Non-Negotiable)

```
domain/     → ZERO non-stdlib imports. Pure Go. No pgx, no kafka, no Redis.
              If you see a non-stdlib import here, it's a bug, not a style choice.

application/ → imports: domain/ only.
              The use case knows WHAT to do but not HOW (no SQL, no HTTP, no Kafka).

infrastructure/ → imports: domain/ (to implement its interfaces).
                 Knows HOW to persist/publish. No business logic here.

interfaces/  → imports: application/ (to call use cases), proto/pb/ (for serialization).
              No business logic. No domain validation. Pure translation.
```

---

## Manual DI Pattern

```go
// provider.go — the ONLY file that knows about all layers
func NewServer(cfg *Config) (*Server, error) {
    // Build bottom-up: infra → domain ← app ← interface

    // 1. Infrastructure (pure I/O)
    db, err := database.NewPool(cfg.DatabaseURL)
    publisher, err := events.NewKafkaPublisher(cfg.KafkaBrokers)

    // 2. Repositories (implement domain ports)
    orderRepo := postgres.NewOrderRepository(db)

    // 3. Application layer (business orchestration)
    createOrderSvc := command.NewCreateOrderHandler(orderRepo, publisher)
    getOrderSvc := query.NewGetOrderHandler(orderRepo)

    // 4. Interface layer (delivery)
    handler := grpchandler.NewOrderHandler(createOrderSvc, getOrderSvc)

    return &Server{grpc: buildGRPCServer(handler)}, nil
}
```

---

## Aggregate Rules

```go
// The aggregate root controls ALL state changes.
// ✅ Correct: behavior on the aggregate
func (o *Order) AddItem(item OrderItem) error {
    if o.Status != StatusDraft {
        return ErrOrderAlreadyPaid
    }
    o.Items = append(o.Items, item)
    o.recalculateTotals()
    o.events = append(o.events, ItemAddedEvent{OrderID: o.ID, Item: item})
    return nil
}

// ❌ Wrong: logic in the application layer
func (h *AddItemHandler) Handle(...) {
    order.Items = append(order.Items, item)  // Bypasses aggregate rules
    order.Total += item.Price               // Direct field mutation
}
```

---

## Repository Contract

```go
// domain/{entity}/repository.go
// Interface only — no implementation details, no pgx types
type OrderRepository interface {
    FindByID(ctx context.Context, id uuid.UUID) (*Order, error)
    FindByTenant(ctx context.Context, tenantID uuid.UUID, filter Filter) ([]*Order, error)
    Save(ctx context.Context, order *Order) error
    Delete(ctx context.Context, id uuid.UUID) error
}
// Returns: *DomainType or error — never *pgx.Row, never sql.Rows
```

---

## Domain Events

```go
// Domain events are produced by aggregates, consumed by Kafka publishers
// Pattern: aggregate stores events, application layer publishes after successful save

// In handler:
if err := h.repo.Save(ctx, order); err != nil { return err }
if err := h.publisher.PublishAll(ctx, order.PopEvents()); err != nil {
    // Log but don't fail — eventual consistency, not strong consistency
    slog.WarnContext(ctx, "failed to publish events", "err", err)
}
```

---

## Error Chain Rule

```
infrastructure: fmt.Errorf("orderRepo.FindByID id=%s: %w", id, err)
               → maps pgx.ErrNoRows to domain.ErrOrderNotFound
application:   oops.Code("ORDER_NOT_FOUND").Public("Order not found").Wrapf(err, ...)
interface:     mapToGRPCError(ctx, err) → status.Error(codes.NotFound, publicMessage)
```

---

## PostgreSQL Rules

```go
// ✅ Always name columns explicitly
rows, err := pool.Query(ctx, "SELECT id, status, total FROM orders WHERE id = $1", id)

// ✅ Always set tenant context before queries
tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID.String())

// ✅ Use pgxpool, never sql.DB
// ✅ Use pgx/v5 named return scanning
// ❌ Never SELECT *
// ❌ Never string-concatenate user input into queries
```

---

## Testing Convention

```
domain/: No mocks needed. Pure functions. table-driven tests.
application/: Mock the repository interface. Test business logic.
infrastructure/: testcontainers-go. Real PostgreSQL 18.4.
interfaces/: Test mapping logic only. Mock the application service.
```
