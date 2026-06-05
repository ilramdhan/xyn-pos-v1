# ADR-0003: Hybrid Multi-Tenancy Strategy

- **Status**: Accepted
- **Date**: 2026-06-05
- **Deciders**: Principal Engineer

---

## Context

xyn-pos-v1 is a multi-tenant SaaS POS platform. Tenants (merchants) need data isolation for compliance, security, and white-label requirements. We need a strategy that balances:

- **Cost**: Shared infra is cheaper; dedicated is expensive
- **Isolation**: Enterprise customers demand full data isolation
- **Compliance**: Some regions require data residency
- **Operational complexity**: Fewer databases = easier migrations and monitoring

---

## Decision

**Hybrid multi-tenancy:**

| Tier | Strategy | Implementation |
|---|---|---|
| Starter / Growth | Shared PostgreSQL with Row-Level Security (RLS) | `SET LOCAL app.current_tenant_id = $1` on every transaction |
| Enterprise | Dedicated PostgreSQL instance | Separate connection string per tenant, provisioned on signup |

---

## Options Considered

| Option | Pros | Cons |
|---|---|---|
| Schema-per-tenant | Easy migration, natural isolation | Postgres limit ~10k schemas, hard to query across tenants |
| **Hybrid RLS + dedicated** (chosen) | Cost-efficient for small tenants, strong isolation for Enterprise | Two code paths for some operations |
| Full dedicated per tenant | Maximum isolation | Cost-prohibitive at scale for SMB customers |
| Single shared DB, application-level filtering | Simplest | SQL injection risk bypasses isolation, compliance non-starter |

---

## Consequences

**Application code rule:**
Every database transaction **must** begin with:
```go
tx.Exec(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID.String())
```

The RLS policy then filters all rows automatically.

**Never bypass RLS** by using a superuser connection for application queries. The `app_user` role has RLS enabled; the `migration_user` role bypasses RLS only for schema migrations.

See `docs/phase-1/database-strategy.md` for full RLS policy templates.
