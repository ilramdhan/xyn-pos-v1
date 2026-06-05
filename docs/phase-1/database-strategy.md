# Database Strategy — xyn-pos-v1

> Phase 1 — Architecture & Blueprinting  
> Status: Draft | Owner: Principal Engineer

---

## 1. Multi-Tenant Data Isolation Strategy

### 1.1 Comparison of Approaches

| Strategy | Isolation Level | Cost | Complexity | Best For |
|---|---|---|---|---|
| Shared DB, Row-level (RLS) | Low–Medium | Lowest | Low | SaaS MVP, cost-sensitive |
| Shared DB, Schema-per-tenant | Medium | Low | Medium | ~100–1000 tenants |
| Database-per-tenant | High | High | High | Enterprise, strict compliance |
| Hybrid | High | Medium | High | Tiered SaaS |

### 1.2 Decision: Hybrid Strategy

**For MVP → Scale:** Start with **Row-Level Security (RLS)** on a shared database. Offer **Database-per-tenant** as a paid tier for enterprise customers who require strict compliance (HIPAA, PCI-DSS).

**Rationale:**
- RLS is built into PostgreSQL — zero application code needed for isolation enforcement
- As you grow, tenants can be migrated to dedicated databases without changing application code (the abstraction is the same)
- The hybrid model is a proven pattern at Notion, Linear, and Stripe

---

### 1.3 PostgreSQL Row-Level Security Implementation

#### Step 1: Tenant Context in Every Connection

Every database connection must set the tenant context before executing queries:

```sql
-- Set by the application layer before every transaction
SET app.current_tenant_id = '018e1234-5678-7abc-def0-123456789abc';
```

In Go, this is done in the repository layer:

```go
// In infrastructure/postgres/db.go
func (r *orderRepo) withTenantCtx(ctx context.Context, tenantID uuid.UUID) (*sql.Tx, error) {
    tx, err := r.db.BeginTx(ctx, nil)
    if err != nil {
        return nil, err
    }
    _, err = tx.ExecContext(ctx, "SET LOCAL app.current_tenant_id = $1", tenantID.String())
    if err != nil {
        tx.Rollback()
        return nil, err
    }
    return tx, nil
}
```

#### Step 2: RLS Policies on Every Tenant-Scoped Table

```sql
-- Enable RLS on the table
ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE orders FORCE ROW LEVEL SECURITY;  -- applies to table owner too

-- Policy: tenant can only see their own rows
CREATE POLICY tenant_isolation ON orders
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- For service accounts that need cross-tenant access (admin, analytics)
-- Use a separate database role that bypasses RLS
CREATE ROLE xyn_service_account BYPASSRLS;
```

#### Step 3: Application User vs Service Account

```
Application queries     → role: xyn_app       → RLS enforced
Admin/migration queries → role: xyn_admin      → RLS bypassed
Analytics reads         → role: xyn_analytics  → RLS bypassed, read-only
```

Never use a BYPASSRLS role for regular application queries. This is a defense-in-depth measure — if application code has a bug that fails to set `app.current_tenant_id`, the RLS policy will return 0 rows instead of leaking data.

#### Step 4: Audit Table (Cross-tenant, RLS bypassed)

```sql
-- audit_log is owned by xyn_admin, not subject to tenant RLS
-- but still filtered by tenant in application queries
CREATE TABLE audit_log (
    id          UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    tenant_id   UUID NOT NULL,
    actor_id    UUID NOT NULL,
    entity_type VARCHAR(100) NOT NULL,
    entity_id   UUID NOT NULL,
    action      VARCHAR(20) NOT NULL,  -- create, update, delete
    old_values  JSONB,
    new_values  JSONB,
    ip_address  INET,
    user_agent  TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_log_tenant ON audit_log (tenant_id, created_at DESC);
CREATE INDEX idx_audit_log_entity ON audit_log (entity_type, entity_id);
```

---

### 1.4 Database-per-Tenant (Enterprise Tier)

For tenants on the Enterprise plan, provision a dedicated PostgreSQL instance:

```
Tenant Provisioning Flow:
1. Subscription upgraded to Enterprise
2. TenantProvisioned event emitted
3. Infrastructure service receives event
4. Terraform applies new PostgreSQL instance (or Kubernetes job runs)
5. Connection string encrypted and stored in Vault/Secret Manager
6. DNS record created: {tenant_slug}.db.xyn.internal
7. Goose migrations run on new instance
8. Tenant config updated: data_isolation = "dedicated"
```

The application connection pool manager routes by tenant:

```go
type ConnectionRouter struct {
    shared    *pgxpool.Pool          // Default: shared DB
    dedicated map[uuid.UUID]*pgxpool.Pool // Enterprise: per-tenant DB
}

func (r *ConnectionRouter) For(tenantID uuid.UUID) *pgxpool.Pool {
    if pool, ok := r.dedicated[tenantID]; ok {
        return pool  // Enterprise tenant
    }
    return r.shared  // Standard tenant
}
```

---

## 2. Connection Pool Strategy

### 2.1 PgBouncer for Connection Pooling

PostgreSQL's connection limit is a hard resource constraint. With microservices, each service instance creates its own pool. Without pooling, 10 services × 10 pods × 10 connections = 1000 connections (PostgreSQL max is typically 100–300).

```
[Service Pods] → [PgBouncer] → [PostgreSQL]
                 (transaction-mode pooling)
                 max_client_conn = 1000
                 default_pool_size = 20 per DB
```

**PgBouncer transaction mode** is required for RLS with `SET LOCAL` — session mode would leak settings across clients.

### 2.2 Connection Pool Configuration (Go)

```go
// shared/go/pkg/database/pool.go
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
    poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
    if err != nil {
        return nil, fmt.Errorf("parse DSN: %w", err)
    }

    poolCfg.MaxConns = int32(cfg.MaxConns)           // default: 10
    poolCfg.MinConns = int32(cfg.MinConns)           // default: 2
    poolCfg.MaxConnLifetime = 30 * time.Minute
    poolCfg.MaxConnIdleTime = 5 * time.Minute
    poolCfg.HealthCheckPeriod = 1 * time.Minute

    return pgxpool.NewWithConfig(ctx, poolCfg)
}
```

---

## 3. Offline-to-Online Data Synchronization

### 3.1 The Offline-First Problem

Mobile POS operators must be able to:
- Take orders and process payments **without internet**
- Have changes sync **reliably** when connectivity is restored
- Handle **conflicts** when the same data was modified both locally and on the server

### 3.2 Sync Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Mobile Device (Flutter)                  │
│                                                              │
│  ┌────────────────┐     ┌────────────────┐                  │
│  │  UI Layer      │     │  Sync Engine   │                  │
│  │  (Riverpod)    │────▶│                │                  │
│  └────────────────┘     │ - Outbox table │                  │
│                          │ - Conflict     │                  │
│                          │   resolver     │                  │
│                          │ - Retry queue  │                  │
│                          └───────┬────────┘                  │
│                                  │                           │
│                    ┌─────────────▼──────────────┐           │
│                    │  Drift (SQLite) Local DB    │           │
│                    │  - orders                   │           │
│                    │  - outbox_events            │           │
│                    │  - sync_state               │           │
│                    └─────────────────────────────┘           │
└───────────────────────────────┬─────────────────────────────┘
                                │ (when online)
                                ▼
                    ┌─────────────────────┐
                    │  Sync Service (Go)   │
                    │  gRPC bidirectional  │
                    │  streaming           │
                    └──────────┬──────────┘
                               │
                    ┌──────────▼──────────┐
                    │  PostgreSQL          │
                    │  (Source of Truth)   │
                    └─────────────────────┘
```

### 3.3 Outbox Pattern for Offline Operations

Every operation done offline is written to a local `outbox_events` table in Drift (SQLite):

```dart
// Drift table definition
class OutboxEvents extends Table {
  IntColumn get id => integer().autoIncrement()();
  TextColumn get eventType => text()();           // 'order.created', 'payment.completed'
  TextColumn get aggregateId => text()();         // UUID of the affected entity
  TextColumn get payload => text()();             // JSON payload
  TextColumn get idempotencyKey => text().unique()();  // UUID v4 generated offline
  IntColumn get retryCount => integer().withDefault(const Constant(0))();
  TextColumn get status => text().withDefault(const Constant('pending'))();
  DateTimeColumn get createdAt => dateTime()();
  DateTimeColumn get processedAt => dateTime().nullable()();
}
```

The sync engine processes the outbox in order when connectivity is restored:

```dart
class SyncEngine {
  Future<void> processPendingOutbox() async {
    final events = await db.outboxEvents.getPending();
    for (final event in events) {
      try {
        await _sendToServer(event);
        await db.outboxEvents.markProcessed(event.id);
      } on IdempotencyConflictException {
        // Server already has this — mark as processed, move on
        await db.outboxEvents.markProcessed(event.id);
      } on ConflictException catch (e) {
        await _resolveConflict(event, e.serverVersion);
      }
    }
  }
}
```

### 3.4 Conflict Resolution Strategy

**Last-Writer-Wins (LWW)** with timestamp vector clocks for most entities.  
**Server-Wins** for financial data (payments, stock deductions) — never allow offline data to override a server-committed financial record.

```
Conflict Resolution Rules by Entity:
┌─────────────────────────────────────────────────────────┐
│ Entity        │ Strategy       │ Reasoning                │
├─────────────────────────────────────────────────────────┤
│ Order         │ Server-wins    │ POS might not know of    │
│               │ (open orders)  │ server-side voids        │
│ Order         │ Local-wins     │ Offline orders are new;  │
│               │ (new, draft)   │ no server version exists │
│ Payment       │ Server-wins    │ Financial integrity       │
│ Product price │ Server-wins    │ Price changes are admin  │
│ Stock qty     │ CRDT (counter) │ Decrement can merge      │
│ Customer data │ Last-write-wins│ Low risk                 │
└─────────────────────────────────────────────────────────┘
```

**CRDT for Stock:**  
Stock deductions from multiple offline devices are tracked as delta counters:

```
Server stock: 50 units
Device A (offline): sold 3 → delta = -3
Device B (offline): sold 2 → delta = -2
Sync result: 50 + (-3) + (-2) = 45 ✅
(Not: both set to 47 ✗)
```

### 3.5 Sync Protocol (gRPC Bidirectional Streaming)

```protobuf
// proto/sync/v1/sync.proto
service SyncService {
  rpc SyncStream (stream SyncMessage) returns (stream SyncMessage);
}

message SyncMessage {
  string message_id = 1;
  oneof payload {
    OutboxEvent event = 2;
    SyncAck ack = 3;
    SyncConflict conflict = 4;
    ServerPush push = 5;      // Server pushes downstream updates
  }
}
```

**Sync Session Flow:**
1. Client connects, sends `sync_checkpoint` (last known server sequence number)
2. Server sends all events since checkpoint (ServerPush messages)
3. Client sends pending outbox events
4. Server processes, responds with ACK or CONFLICT per event
5. Connection stays open for real-time downstream pushes (price changes, menu updates)

### 3.6 Idempotency Keys for Offline Operations

Every offline operation generates a UUID v4 `idempotency_key` at the point of creation:

```dart
final order = Order(
  id: Uuid().v4(),
  idempotencyKey: Uuid().v4(),  // Generated locally, sent with sync
  // ...
);
```

The server's `POST /orders` endpoint (or gRPC `CreateOrder`) checks:
1. Does this idempotency_key exist in the `idempotency_store` (Redis)?
2. Yes → return the stored result (no side effects)
3. No → execute, store result in Redis with 24h TTL

This makes retries safe: even if the network drops mid-sync, re-sending is harmless.

---

## 4. Data Flow: OLTP to OLAP (Analytics Pipeline)

```
PostgreSQL (OLTP)
      │
      │  Debezium CDC (Change Data Capture)
      ▼
Kafka Topics (raw events)
  ├── xyn.orders.cdc
  ├── xyn.payments.cdc
  ├── xyn.stock_mutations.cdc
  └── ...
      │
      │  ClickHouse Kafka Engine
      ▼
ClickHouse (OLAP)
  ├── orders_mv (Materialized View)
  ├── payments_mv
  ├── stock_history_mv
  └── sales_analytics_mv
      │
      ▼
Grafana / Analytics API
```

**Why CDC instead of dual-writes?**
Dual-writes (write to PostgreSQL AND publish to Kafka) have a consistency problem: the Kafka write might fail after PostgreSQL commits. Debezium reads the PostgreSQL WAL (write-ahead log) — you get guaranteed-consistent events with no application code changes.

---

## 5. Migration Strategy

### 5.1 Goose Setup

```go
// tools/migrate/main.go
//go:embed migrations/*.sql
var embedMigrations embed.FS

func main() {
    goose.SetBaseFS(embedMigrations)
    if err := goose.SetDialect("postgres"); err != nil {
        log.Fatal(err)
    }
    if err := goose.Up(db, "migrations"); err != nil {
        log.Fatal(err)
    }
}
```

### 5.2 Migration Rules

1. **Never drop a column in the same migration that removes its usage** — deploy the code change first, then drop in a subsequent migration after verifying no reads/writes
2. **Always add columns as nullable first** — backfill, then add NOT NULL constraint
3. **Name migrations with context**: `20240101_001_add_idempotency_key_to_orders.sql`
4. **Never edit a committed migration** — add a new one to fix mistakes
5. **Multi-tenant migrations**: Run against shared DB via admin role; run against each dedicated tenant DB via provisioning service

### 5.3 Zero-Downtime Schema Changes

Pattern for adding a NOT NULL column to a large table:
```sql
-- Step 1: Add nullable (instant)
ALTER TABLE orders ADD COLUMN receipt_number VARCHAR(20);

-- Step 2: Backfill in batches (no table lock)
UPDATE orders SET receipt_number = generate_receipt(id)
WHERE receipt_number IS NULL
LIMIT 10000;
-- Repeat until 0 rows updated

-- Step 3: Add constraint (check existing data, fast)
ALTER TABLE orders ADD CONSTRAINT orders_receipt_not_null
    CHECK (receipt_number IS NOT NULL) NOT VALID;

-- Step 4: Validate (online, no lock)
ALTER TABLE orders VALIDATE CONSTRAINT orders_receipt_not_null;

-- Step 5: Set NOT NULL (uses validated constraint, instant)
ALTER TABLE orders ALTER COLUMN receipt_number SET NOT NULL;
```

---

## 6. Redis Usage Patterns

| Use Case | Key Pattern | TTL | Notes |
|---|---|---|---|
| Session tokens | `session:{token_hash}` | 24h | Invalidated on logout |
| Idempotency keys | `idem:{key}` | 24h | JSON-encoded result |
| Rate limiting | `rl:{tenant}:{endpoint}:{window}` | 60s | Sliding window counter |
| Menu cache | `menu:{tenant_id}:{branch_id}` | 5min | Invalidated on menu update |
| Stock cache | `stock:{product_id}:{warehouse_id}` | 30s | Short TTL, stock changes often |
| Distributed lock | `lock:{resource}:{id}` | 30s | Redlock algorithm |
| PubSub: KDS | `kds:{branch_id}` | — | Real-time kitchen updates |

---

## 7. Backup & Recovery Strategy

| Data Tier | Tool | Frequency | Retention | RTO | RPO |
|---|---|---|---|---|---|
| PostgreSQL (shared) | pg_basebackup + WAL archiving | Continuous | 30 days | <1h | <5min |
| PostgreSQL (dedicated) | Per-tenant pg_dump | Daily | 90 days | <4h | <24h |
| Redis | RDB snapshot | Hourly | 7 days | <15min | <1h |
| ClickHouse | clickhouse-backup | Daily | 30 days | <4h | <24h |
| MinIO | Replication to secondary | Real-time | 180 days | <1h | Near-zero |

**Point-in-Time Recovery (PITR)** is essential for financial data. PostgreSQL WAL archiving to S3-compatible storage (MinIO) enables recovery to any second within the retention window.
