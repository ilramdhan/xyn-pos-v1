# ADR-0001: Manual Dependency Injection over Google Wire

- **Status**: Accepted
- **Date**: 2026-06-05
- **Deciders**: Principal Engineer

---

## Context

Go microservices require a mechanism to wire dependencies (database pool → repository → use case handler → gRPC server) at startup. Google Wire was the standard codegen approach for this. On August 25, 2025, **Google Wire v0.7.0 was archived** — the project is no longer maintained.

We need a replacement approach that:
- Is production-safe with no maintenance risk
- Is understandable to engineers and AI assistants
- Adds zero runtime overhead
- Supports future testability (easy to inject mocks)

---

## Decision

Use **Manual Dependency Injection** via a `provider.go` file in each service root. No external DI library. Constructor chaining in pure Go.

---

## Options Considered

| Option | Pros | Cons |
|---|---|---|
| **Manual DI** (chosen) | Zero dependencies, full control, what Wire generated anyway, easy to read and debug | More boilerplate per service (mitigated by template) |
| `samber/do v2.0.0` | Type-safe DI container with generics, scope support, health checks | Adds a dependency, different mental model, overhead from reflection-free but still abstracted |
| `uber/fx` | Production-proven at Uber, lifecycle management | Heavy framework, steep learning curve, complex for small services |
| `wire` fork (community) | Drop-in replacement | Unknown maintenance quality, single contributor risk |

---

## Consequences

**Easier:**
- Every developer can read `provider.go` and understand the full dependency graph in one file
- No magic — if a dependency is missing, the compiler tells you immediately
- AI assistants (Claude Code) can generate correct `provider.go` without knowing a framework
- Zero risk of upstream abandonment

**Harder:**
- Each new service needs a `provider.go` written manually (mitigated by the template in `contributing.md`)
- Adding a new dependency requires editing two places: the constructor and `provider.go`

**Template:**

```go
// services/{name}/provider.go
package main

func NewServer(cfg *Config) (*Server, error) {
    // 1. Infrastructure
    db, err := database.NewPool(cfg.DatabaseURL)
    if err != nil {
        return nil, fmt.Errorf("database: %w", err)
    }
    publisher, err := events.NewKafkaPublisher(cfg.KafkaBrokers)
    if err != nil {
        return nil, fmt.Errorf("kafka publisher: %w", err)
    }

    // 2. Repositories
    orderRepo := postgres.NewOrderRepository(db)

    // 3. Application handlers
    createOrder := command.NewCreateOrderHandler(orderRepo, publisher)
    getOrder    := query.NewGetOrderHandler(orderRepo)

    // 4. Interface layer
    handler := grpc.NewOrderHandler(createOrder, getOrder)

    return &Server{grpc: buildGRPCServer(handler)}, nil
}
```
