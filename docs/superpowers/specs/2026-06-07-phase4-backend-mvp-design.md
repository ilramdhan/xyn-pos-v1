# Phase 4 — Backend MVP Implementation: Master Design Spec

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this spec epic-by-epic. Each epic produces its own implementation plan via superpowers:writing-plans before implementation begins.

**Goal:** Implement all five backend bounded contexts (Identity & Auth, Product & Menu, POS Core, Payment, Inventory) as production-ready Go microservices with full DDD architecture, gRPC APIs, Kafka event integration, Swagger docs, distributed tracing, and comprehensive tests through E2E.

**Architecture:** Hybrid communication — gRPC for direct command/query RPCs between services; Kafka for domain events crossing service boundaries (OrderPaid → Inventory, PaymentCompleted → POS). All services follow the DDD pattern established in Phase 3 (Tenant Service as template).

**Tech Stack:** Go 1.26.4, gRPC-Go 1.81.1, pgx/v5 10.0, goose 3.27.1, testcontainers-go 0.42.0, go-redis/v9, Kafka 4.3 KRaft, Midtrans SDK, OpenTelemetry 1.44.0, buf 1.70.0, golangci-lint 2.12.2.

**Branch:** `feat/phase4-backend-mvp`

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Pending from Phase 3 — Redis Setup](#2-pending-from-phase-3--redis-setup)
3. [Kafka Topic Map](#3-kafka-topic-map)
4. [Epic 4.1 — Identity & Auth (tenant service)](#4-epic-41--identity--auth-tenant-service)
5. [Epic 4.2 — Product & Menu (pos service)](#5-epic-42--product--menu-pos-service)
6. [Epic 4.3 — POS Core (pos service)](#6-epic-43--pos-core-pos-service)
7. [Epic 4.4 — Payment Service](#7-epic-44--payment-service)
8. [Epic 4.5 — Inventory Service](#8-epic-45--inventory-service)
9. [Cross-Cutting Concerns](#9-cross-cutting-concerns)
10. [Testing Strategy](#10-testing-strategy)
11. [Proto Contract Map](#11-proto-contract-map)
12. [Implementation Order](#12-implementation-order)

---

## 1. Architecture Overview

### Service Map

```
┌──────────────────────────────────────────────────────────────────┐
│                      API Gateway (KrakenD 2.13.7)                │
│          REST endpoints · JWT validation · Rate limiting          │
└────┬─────────────────┬────────────────┬──────────────┬───────────┘
     │ gRPC            │ gRPC           │ gRPC         │ gRPC
     ▼                 ▼                ▼              ▼
┌──────────┐    ┌──────────────┐  ┌──────────┐  ┌──────────────┐
│  tenant  │    │     pos      │  │ payment  │  │  inventory   │
│ service  │    │   service    │  │ service  │  │   service    │
│          │    │              │  │          │  │              │
│ Tenants  │    │ Products     │  │Payments  │  │ Stock ledger │
│ Branches │    │ Categories   │  │QRIS/Cash │  │ Mutations    │
│ Users    │    │ Variants     │  │Midtrans  │  │ Warehouses   │
│ RBAC     │    │ Add-ons      │  │Void/     │  │ BOM/Recipe   │
│ Auth     │    │ Orders       │  │Refund    │  │ Low-stock    │
│ PIN      │    │ Shifts       │  │Webhook   │  │ alerts       │
│ Keycloak │    │ Discounts    │  │Idempoten.│  │              │
└──────────┘    └──────────────┘  └──────────┘  └──────────────┘
     │                 │                │              │
     └─────────────────┴────────────────┴──────────────┘
                               │
               ┌───────────────▼──────────────┐
               │      Apache Kafka 4.3 KRaft   │
               │                               │
               │  pos.order.paid ─────────────►│──► inventory (deduct stock)
               │  payment.completed ──────────►│──► pos (mark order PAID)
               │  payment.voided ─────────────►│──► pos (reopen/cancel order)
               │  inventory.low_stock_alert    │
               └───────────────────────────────┘
```

### Bounded Context Assignments

| Epic | Service | Bounded Context |
|---|---|---|
| 4.1 Identity & Auth | `services/tenant` | Tenant & Identity BC |
| 4.2 Product & Menu | `services/pos` | POS Core & Cart BC |
| 4.3 POS Core | `services/pos` | POS Core & Cart BC |
| 4.4 Payment | `services/payment` | Payment & Checkout BC |
| 4.5 Inventory | `services/inventory` | Inventory & Supply Chain BC |

Product catalog lives in the `pos` service — it is the primary consumer and producer of product data. The `inventory` service tracks stock *levels* for product IDs but does not own product definitions.

### Communication Decisions

| Scenario | Protocol | Why |
|---|---|---|
| Client → Service | gRPC (+ REST via gRPC-Gateway) | Typed contracts, generated clients |
| Service A reads from Service B | gRPC RPC call | Synchronous, typed, traceable |
| After OrderPaid → deduct stock | Kafka event | Decoupled, inventory scales independently |
| After PaymentCompleted → update order | Kafka event | Decoupled, payment service is autonomous |
| Midtrans → Payment service | HTTP webhook | Midtrans requirement |

---

## 2. Pending from Phase 3 — Redis Setup

**Story 3.4.4** (deferred): Redis config for local dev.

### docker-compose.yml additions

```yaml
redis:
  image: redis:8.8-alpine
  container_name: xyn-redis
  ports:
    - "6379:6379"
  command: >
    redis-server
    --maxmemory 256mb
    --maxmemory-policy allkeys-lru
    --save 60 1
    --loglevel warning
  volumes:
    - redis-data:/data
  healthcheck:
    test: ["CMD", "redis-cli", "ping"]
    interval: 5s
    timeout: 3s
    retries: 5
  networks:
    - xyn-net
```

**Rationale for standalone (not Cluster) in dev:** Redis Cluster requires minimum 3 masters and complicates local setup without benefit. The K8s production overlay already has Redis Sentinel/Cluster config. Dev uses standalone with same `go-redis/v9` client — the client abstraction is identical.

### Shared Package: `shared/go/pkg/cache`

New package to wrap `go-redis/v9`:

```
shared/go/pkg/cache/
├── cache.go         ← Client struct, NewClient(cfg Config) (*Client, error)
├── idempotency.go   ← SetNX(key, value, ttl), Get(key), Delete(key)
├── cache_test.go    ← unit tests with mock + integration with testcontainers Redis
└── config.go        ← Config{Addr, Password, DB, MaxRetries}
```

Key naming conventions:
```
idempotency:{key}          TTL 24h   payment service
qris:status:{external_id} TTL 30s   payment service (poll cache)
token:blacklist:{jti}      TTL = token remaining TTL  tenant service
```

### `.env.example` additions

```
REDIS_ADDR=localhost:6379
REDIS_PASSWORD=
REDIS_DB=0
```

---

## 3. Kafka Topic Map

All topics use protobuf encoding. Each message carries OTEL trace context in headers for distributed tracing across service boundaries.

| Topic | Producer | Consumer(s) | Partition Key | Retention |
|---|---|---|---|---|
| `pos.order.created` | pos | analytics (Phase 6) | `tenant_id` | 7d |
| `pos.order.paid` | pos | inventory ✓ | `tenant_id` | 7d |
| `pos.order.cancelled` | pos | analytics (Phase 6) | `tenant_id` | 7d |
| `payment.completed` | payment | pos ✓ | `tenant_id` | 7d |
| `payment.voided` | payment | pos ✓ | `tenant_id` | 7d |
| `inventory.low_stock_alert` | inventory | notification (Phase 6) | `tenant_id` | 3d |

### Event Schemas (in `proto/events/v1/`)

```protobuf
// pos.order.paid
message OrderPaidEvent {
  string order_id    = 1;
  string tenant_id   = 2;
  string branch_id   = 3;
  string cashier_id  = 4;
  repeated OrderPaidItem items = 5;
  string occurred_at = 6; // RFC3339
}
message OrderPaidItem {
  string product_id  = 1;
  string variant_id  = 2; // empty if no variant
  int32  quantity    = 3;
}

// payment.completed
message PaymentCompletedEvent {
  string payment_id  = 1;
  string order_id    = 2;
  string tenant_id   = 3;
  int64  amount      = 4;
  string method      = 5;
  string occurred_at = 6;
}

// payment.voided
message PaymentVoidedEvent {
  string payment_id  = 1;
  string order_id    = 2;
  string tenant_id   = 3;
  string voided_by   = 4; // user_id
  string occurred_at = 5;
}

// inventory.low_stock_alert
message LowStockAlertEvent {
  string product_id   = 1;
  string variant_id   = 2;
  string warehouse_id = 3;
  string tenant_id    = 4;
  int64  current_qty  = 5;
  int64  reorder_point = 6;
  string occurred_at  = 7;
}
```

---

## 4. Epic 4.1 — Identity & Auth (tenant service)

### Overview

Extend the existing `tenant` service with a `user` subdomain. Keycloak handles federated identity (password hashing, OAuth2 flows, MFA in future). The tenant service maintains a local user profile with RBAC roles and PIN for sensitive POS operations.

### Auth Flow

```
1. Client sends email+password to Keycloak OIDC endpoint
2. Keycloak returns keycloak_access_token
3. Client calls tenant service: Login(keycloak_token)
4. Tenant service verifies token with Keycloak introspection endpoint
5. Tenant service looks up local user by keycloak_id
6. Tenant service issues PASETO v4 local token:
   claims = { user_id, tenant_id, branch_scope[], role, jti, exp: now+8h }
7. Client uses PASETO token for all subsequent gRPC calls
8. gRPC auth interceptor validates PASETO on every request
```

### PASETO Token Claims

```go
type Claims struct {
    UserID      uuid.UUID   `json:"user_id"`
    TenantID    uuid.UUID   `json:"tenant_id"`
    BranchScope []uuid.UUID `json:"branch_scope"` // empty = all branches
    Role        string      `json:"role"`
    JTI         string      `json:"jti"`           // for blacklist on logout
    ExpiresAt   time.Time   `json:"exp"`
}
```

### RBAC Permission Matrix

| Operation | owner | manager | cashier | kitchen_staff |
|---|---|---|---|---|
| Register/deactivate user | ✅ | ❌ | ❌ | ❌ |
| View all users | ✅ | ✅ | ❌ | ❌ |
| Create/edit product | ✅ | ✅ | ❌ | ❌ |
| Create order | ✅ | ✅ | ✅ | ❌ |
| Process payment | ✅ | ✅ | ✅ | ❌ |
| Void payment (requires PIN) | ✅ | ✅ | ❌ | ❌ |
| Apply discount | ✅ | ✅ | ✅ (limited %) | ❌ |
| Open/close shift | ✅ | ✅ | ✅ | ❌ |
| View KDS | ✅ | ✅ | ✅ | ✅ |
| View reports | ✅ | ✅ | ❌ | ❌ |
| Adjust stock | ✅ | ✅ | ❌ | ❌ |

### New Files in tenant service

```
services/tenant/internal/
├── domain/user/
│   ├── user.go              ← User aggregate
│   ├── repository.go        ← UserRepository interface
│   ├── errors.go            ← ErrUserNotFound, ErrEmailTaken,
│   │                           ErrInvalidPIN, ErrUserInactive,
│   │                           ErrUnauthorized, ErrBranchAccessDenied
│   └── events.go            ← UserRegistered, UserDeactivated
├── application/
│   ├── command/
│   │   ├── register_user.go     ← Keycloak create + PG sync
│   │   ├── login_user.go        ← exchange Keycloak token → PASETO
│   │   ├── refresh_token.go     ← extend session
│   │   ├── logout_user.go       ← blacklist JTI in Redis
│   │   ├── deactivate_user.go   ← owner only
│   │   ├── set_pin.go           ← bcrypt(PIN, cost=12)
│   │   └── verify_pin.go        ← used before void/refund
│   └── query/
│       ├── get_user.go
│       └── list_users.go
├── infrastructure/
│   ├── postgres/
│   │   ├── user_repo.go
│   │   └── migrations/
│   │       └── 00004_create_users.sql
│   ├── keycloak/
│   │   └── keycloak_client.go   ← Admin REST API: CreateUser, DeleteUser,
│   │                               Introspect, ResetPassword
│   └── redis/
│       └── token_blacklist.go   ← SetNX JTI with token TTL
└── interfaces/grpc/
    └── user_handler.go          ← maps proto ↔ domain
```

### User Domain Model

```go
// domain/user/user.go
type User struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    KeycloakID  string
    Email       string
    FullName    string
    Role        Role
    BranchScope []uuid.UUID // nil = all branches
    IsActive    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
    events      []DomainEvent
}

type Role string
const (
    RoleOwner        Role = "owner"
    RoleManager      Role = "manager"
    RoleCashier      Role = "cashier"
    RoleKitchenStaff Role = "kitchen_staff"
)

func NewUser(tenantID uuid.UUID, keycloakID, email, fullName string, role Role, branchScope []uuid.UUID) (*User, error)
func (u *User) Deactivate() error  // returns ErrUserInactive if already inactive
func (u *User) CanAccessBranch(branchID uuid.UUID) bool
func (u *User) PopEvents() []DomainEvent
```

### DB Migration 00004

```sql
CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id),
    keycloak_id   VARCHAR(36)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL,
    full_name     VARCHAR(255) NOT NULL,
    role          VARCHAR(30)  NOT NULL
                  CHECK (role IN ('owner','manager','cashier','kitchen_staff')),
    branch_scope  UUID[]       NULL,       -- NULL means all branches
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

CREATE TABLE user_pins (
    user_id    UUID  PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pin_hash   TEXT  NOT NULL,             -- bcrypt cost=12
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_tenant_id ON users (tenant_id);
CREATE INDEX idx_users_keycloak_id ON users (keycloak_id);

ALTER TABLE users     ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_pins ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_select ON users FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY users_insert ON users FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY users_update ON users FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY users_delete ON users FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### Proto: `proto/tenant/v1/user.proto`

```protobuf
syntax = "proto3";
package tenant.v1;
import "google/protobuf/empty.proto";

enum Role {
  ROLE_UNSPECIFIED   = 0;
  ROLE_OWNER         = 1;
  ROLE_MANAGER       = 2;
  ROLE_CASHIER       = 3;
  ROLE_KITCHEN_STAFF = 4;
}

message User {
  string id                    = 1;
  string tenant_id             = 2;
  string email                 = 3;
  string full_name             = 4;
  Role   role                  = 5;
  repeated string branch_scope = 6;
  bool   is_active             = 7;
  string created_at            = 8;
  string updated_at            = 9;
}

message RegisterUserRequest {
  string idempotency_key       = 1;
  string email                 = 2;
  string password              = 3;
  string full_name             = 4;
  Role   role                  = 5;
  repeated string branch_scope = 6;
}
message RegisterUserResponse { User user = 1; }

message LoginRequest  { string keycloak_token = 1; }
message LoginResponse {
  string paseto_token = 1;
  string user_id      = 2;
  string expires_at   = 3;
}

message RefreshTokenRequest  { string paseto_token = 1; }
message LogoutRequest        { string paseto_token = 1; }

message GetUserRequest        { string user_id = 1; }
message GetUserResponse       { User user = 1; }

message ListUsersRequest      { string tenant_id = 1; }
message ListUsersResponse     { repeated User users = 1; }

message DeactivateUserRequest { string user_id = 1; }

message SetPINRequest         { string user_id = 1; string pin = 2; }
message VerifyPINRequest      { string user_id = 1; string pin = 2; }
message VerifyPINResponse     { bool valid = 1; }

service UserService {
  rpc RegisterUser(RegisterUserRequest)     returns (RegisterUserResponse);
  rpc Login(LoginRequest)                   returns (LoginResponse);
  rpc RefreshToken(RefreshTokenRequest)     returns (LoginResponse);
  rpc Logout(LogoutRequest)                 returns (google.protobuf.Empty);
  rpc GetUser(GetUserRequest)               returns (GetUserResponse);
  rpc ListUsers(ListUsersRequest)           returns (ListUsersResponse);
  rpc DeactivateUser(DeactivateUserRequest) returns (google.protobuf.Empty);
  rpc SetPIN(SetPINRequest)                 returns (google.protobuf.Empty);
  rpc VerifyPIN(VerifyPINRequest)           returns (VerifyPINResponse);
}
```

### Tests — Epic 4.1

**Unit (domain/user):**
- `TestNewUser_InvalidRole_ReturnsError`
- `TestNewUser_EmptyEmail_ReturnsError`
- `TestUser_Deactivate_AlreadyInactive_ReturnsError`
- `TestUser_CanAccessBranch_NilScope_AllowsAll`
- `TestUser_CanAccessBranch_ScopedBranch_BlocksOther`

**Unit (application/command):**
- `TestRegisterUser_EmailAlreadyTaken_ReturnsError`
- `TestLoginUser_InvalidKeycloakToken_ReturnsError`
- `TestLoginUser_UserInactive_ReturnsError`
- `TestVerifyPIN_WrongPIN_ReturnsFalse`
- `TestSetPIN_HashIsStored_NotPlaintext`

**Integration (testcontainers PostgreSQL + Redis):**
- `TestUserRepo_SaveAndFindByID`
- `TestUserRepo_FindByEmail_NotFound`
- `TestUserRepo_SetPINAndVerify`
- `TestUserRepo_ListByTenant_RLSIsolation`
- `TestTokenBlacklist_JTI_ExpiresCorrectly`

**E2E:**
- `TestE2E_RegisterAndLogin_FullFlow`
- `TestE2E_PINSet_VerifyCorrect_VerifyWrong`
- `TestE2E_Logout_TokenBlacklisted`
- `TestE2E_WrongTenant_Unauthorized`
- `TestE2E_DeactivatedUser_CannotLogin`

---

## 5. Epic 4.2 — Product & Menu (pos service)

### Domain Model

```go
// domain/product/product.go
type Product struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    CategoryID  *uuid.UUID
    SKU         string          // unique per tenant
    Name        string
    Description string
    BasePrice   int64           // minor units (sen). Rp 15.000 = 1500000
    TaxType     TaxType         // ppn | pb1 | none
    Variants    []Variant
    AddonGroups []AddonGroup
    IsActive    bool
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

type TaxType string
const (
    TaxTypePPN  TaxType = "ppn"   // 11%
    TaxTypePB1  TaxType = "pb1"   // 10%
    TaxTypeNone TaxType = "none"
)

type Variant struct {
    ID         uuid.UUID
    ProductID  uuid.UUID
    Name       string
    PriceDelta int64    // can be negative; FinalPrice = BasePrice + PriceDelta
    SKU        string
    IsActive   bool
}

type AddonGroup struct {
    ID            uuid.UUID
    ProductID     uuid.UUID
    Name          string   // "Topping", "Size", "Extra"
    IsRequired    bool
    MaxSelections int
    Addons        []Addon
}

type Addon struct {
    ID           uuid.UUID
    AddonGroupID uuid.UUID
    Name         string
    Price        int64    // always >= 0
    IsActive     bool
}
```

**Business rules:**
- `BasePrice >= 0`; `Addon.Price >= 0`; `Variant.PriceDelta` unbounded but `FinalPrice` must be >= 0
- SKU is unique per tenant (enforced at DB + domain level)
- Soft delete only — `IsActive = false`. Hard delete blocked if product referenced by any order
- `BranchProduct` override: `OverridePrice` replaces `BasePrice` for that branch only. `nil` = use `BasePrice`

### New Files in pos service — Product

```
services/pos/internal/
├── domain/product/
│   ├── product.go          ← Product aggregate + NewProduct, AddVariant, AddAddonGroup
│   ├── category.go         ← Category entity (nested, sort_order)
│   ├── variant.go
│   ├── addon_group.go
│   ├── repository.go       ← ProductRepository, CategoryRepository
│   └── errors.go           ← ErrProductNotFound, ErrSKUAlreadyExists,
│                               ErrCategoryNotFound, ErrProductHasActiveOrders,
│                               ErrVariantPriceInvalid
├── application/
│   ├── command/
│   │   ├── create_product.go
│   │   ├── update_product.go
│   │   ├── archive_product.go      ← sets IsActive=false
│   │   ├── create_category.go
│   │   ├── reorder_categories.go
│   │   └── set_branch_price.go     ← branch-specific price override
│   └── query/
│       ├── get_product.go
│       ├── list_products.go        ← with category filter, branch availability
│       └── lookup_by_sku.go        ← barcode scanner endpoint
└── infrastructure/
    └── postgres/
        ├── product_repo.go
        └── migrations/
            └── 00001_create_products.sql
```

### DB Migration — products

```sql
CREATE TABLE categories (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    name        VARCHAR(255) NOT NULL,
    sort_order  INT          NOT NULL DEFAULT 0,
    parent_id   UUID         REFERENCES categories(id),
    is_active   BOOLEAN      NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE products (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL,
    category_id  UUID         REFERENCES categories(id),
    sku          VARCHAR(100),
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    base_price   BIGINT       NOT NULL CHECK (base_price >= 0),
    tax_type     VARCHAR(20)  NOT NULL DEFAULT 'ppn'
                 CHECK (tax_type IN ('ppn','pb1','none')),
    is_active    BOOLEAN      NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, sku)
);

CREATE TABLE product_variants (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id  UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    price_delta BIGINT       NOT NULL DEFAULT 0,
    sku         VARCHAR(100),
    is_active   BOOLEAN      NOT NULL DEFAULT true
);

CREATE TABLE addon_groups (
    id              UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID    NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    is_required     BOOLEAN NOT NULL DEFAULT false,
    max_selections  INT     NOT NULL DEFAULT 1 CHECK (max_selections >= 1)
);

CREATE TABLE addons (
    id             UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    addon_group_id UUID    NOT NULL REFERENCES addon_groups(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    price          BIGINT  NOT NULL DEFAULT 0 CHECK (price >= 0),
    is_active      BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE branch_products (
    branch_id      UUID   NOT NULL,
    product_id     UUID   NOT NULL REFERENCES products(id),
    override_price BIGINT,               -- NULL = use products.base_price
    is_available   BOOLEAN NOT NULL DEFAULT true,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (branch_id, product_id)
);

CREATE INDEX idx_products_tenant_id   ON products(tenant_id);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_sku         ON products(tenant_id, sku) WHERE sku IS NOT NULL;

-- RLS: all product tables scoped by tenant_id
ALTER TABLE categories  ENABLE ROW LEVEL SECURITY;
ALTER TABLE products    ENABLE ROW LEVEL SECURITY;
-- (variants, addons, addon_groups cascade via product_id — no own tenant_id needed,
--  queries always JOIN through products which is RLS-protected)
CREATE POLICY categories_isolation ON categories
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY products_isolation ON products
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### Tests — Epic 4.2

**Unit (domain/product):**
- `TestNewProduct_NegativeBasePrice_ReturnsError`
- `TestNewProduct_EmptyName_ReturnsError`
- `TestProduct_AddVariant_NegativeFinalPrice_ReturnsError`
- `TestProduct_AddAddonGroup_MaxSelectionsZero_ReturnsError`
- `TestProduct_Archive_AlreadyInactive_ReturnsError`

**Unit (application):**
- `TestCreateProduct_SKUAlreadyExists_ReturnsError`
- `TestArchiveProduct_HasActiveOrders_ReturnsError`
- `TestLookupBySKU_NotFound_ReturnsError`
- `TestSetBranchPrice_NegativePrice_ReturnsError`

**Integration (testcontainers):**
- `TestProductRepo_CreateAndFindByID`
- `TestProductRepo_FindBySKU_UniquePerTenant`
- `TestProductRepo_ListByCategory_RLSIsolation`
- `TestProductRepo_BranchOverridePrice`
- `TestCategoryRepo_NestedCategories`

**E2E:**
- `TestE2E_CreateProductWithVariantsAndAddons`
- `TestE2E_BarcodeScanner_LookupBySKU`
- `TestE2E_ArchiveProduct_RemovedFromListing`
- `TestE2E_BranchSpecificPrice_OverridesBase`

---

## 6. Epic 4.3 — POS Core (pos service)

### Order State Machine

```
                  ┌─────────────────────────────┐
                  │           DRAFT             │
                  │  (add/remove items,         │
                  │   apply discount)           │
                  └─────────────┬───────────────┘
                                │ SubmitOrder
                                ▼
                  ┌─────────────────────────────┐
                  │      PENDING_PAYMENT        │◄─── PaymentVoided event
                  └──────┬──────────────────────┘
                         │                │
          payment.       │                │ CancelOrder
          completed      │                │ (before payment)
          event          ▼                ▼
              ┌──────────────┐   ┌────────────────┐
              │     PAID     │   │   CANCELLED    │
              │   (final)    │   │    (final)     │
              └──────────────┘   └────────────────┘
```

### Domain Model — Order

```go
// domain/order/order.go
type Order struct {
    ID              uuid.UUID
    TenantID        uuid.UUID
    BranchID        uuid.UUID
    ShiftID         *uuid.UUID
    CashierID       uuid.UUID
    OrderNumber     string
    OrderType       OrderType
    TableNumber     string
    Status          OrderStatus
    Items           []OrderItem
    Discount        *Discount
    Subtotal        int64       // sum of item subtotals
    TaxAmount       int64       // calculated from TaxConfig
    DiscountAmount  int64
    Total           int64       // Subtotal + Tax - Discount
    IdempotencyKey  string
    Notes           string
    CreatedAt       time.Time
    UpdatedAt       time.Time
    events          []DomainEvent
}

// Business rules enforced in aggregate methods:
// - AddItem: only in DRAFT status
// - RemoveItem: only in DRAFT status, item must exist
// - ApplyDiscount: only in DRAFT; DiscountPercent max 100; DiscountFixed <= Subtotal
// - Submit: requires at least 1 item
// - Cancel: only in DRAFT or PENDING_PAYMENT
// - MarkPaid: only in PENDING_PAYMENT
```

### Tax Calculation

```go
// domain/order/tax.go
type TaxConfig struct {
    Type TaxType // from branch settings
    Rate int64   // basis points: PPN=1100 (11%), PB1=1000 (10%)
}

// TaxAmount = Subtotal * Rate / 10000
// Total = Subtotal + TaxAmount - DiscountAmount
// For PB1: tax is inclusive (already in price), so just reporting — no add
// For PPN: tax is exclusive (added on top)
```

### Discount Rules

```go
type Discount struct {
    Type   DiscountType  // fixed | percent
    Amount int64         // sen for fixed; basis points for percent (1000 = 10%)
    Scope  DiscountScope // total | item
}

// Validation:
// - DiscountPercent: 0 < amount <= 10000 (0-100%)
// - DiscountFixed: 0 < amount <= subtotal
// - Cashier role: max discount percent configurable per branch (e.g. 10%)
// - Manager/Owner: unlimited discount
```

### Shift Management

```go
// domain/order/shift.go
type Shift struct {
    ID          uuid.UUID
    TenantID    uuid.UUID
    BranchID    uuid.UUID
    CashierID   uuid.UUID
    Status      ShiftStatus  // open | closed
    OpenedAt    time.Time
    ClosedAt    *time.Time
    OpeningCash int64        // cash in drawer at open
    ClosingCash *int64       // counted cash at close
}

// Business rules:
// - One active shift per (branch, cashier) at a time
// - Cannot close an already-closed shift
// - ClosingCash is reconciled with expected cash (OpeningCash + cash_sales)
```

### New Application Layer — Order

```
services/pos/internal/application/
├── command/
│   ├── create_order.go         ← idempotency_key required
│   ├── add_item.go             ← validates product exists + is_available
│   ├── remove_item.go
│   ├── update_item_quantity.go
│   ├── apply_discount.go       ← enforce cashier discount limit
│   ├── submit_order.go         ← DRAFT → PENDING_PAYMENT
│   ├── cancel_order.go
│   ├── mark_order_paid.go      ← called by Kafka consumer (payment.completed)
│   ├── open_shift.go
│   └── close_shift.go
└── query/
    ├── get_order.go
    ├── list_orders.go           ← by shift, by cashier, by date range
    └── get_shift.go
```

### DB Migration — orders

```sql
CREATE TABLE shifts (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID        NOT NULL,
    branch_id     UUID        NOT NULL,
    cashier_id    UUID        NOT NULL,
    status        VARCHAR(20) NOT NULL DEFAULT 'open'
                  CHECK (status IN ('open','closed')),
    opened_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at     TIMESTAMPTZ,
    opening_cash  BIGINT      NOT NULL DEFAULT 0,
    closing_cash  BIGINT,
    UNIQUE (branch_id, cashier_id, status) -- prevent >1 open shift per cashier
);

CREATE TABLE orders (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    branch_id        UUID         NOT NULL,
    shift_id         UUID         REFERENCES shifts(id),
    cashier_id       UUID         NOT NULL,
    order_number     VARCHAR(50)  NOT NULL,
    order_type       VARCHAR(20)  NOT NULL
                     CHECK (order_type IN ('dine_in','takeaway','delivery')),
    table_number     VARCHAR(20),
    status           VARCHAR(30)  NOT NULL DEFAULT 'draft'
                     CHECK (status IN ('draft','pending_payment','paid','cancelled')),
    subtotal         BIGINT       NOT NULL DEFAULT 0,
    tax_amount       BIGINT       NOT NULL DEFAULT 0,
    discount_amount  BIGINT       NOT NULL DEFAULT 0,
    total            BIGINT       NOT NULL DEFAULT 0,
    discount_type    VARCHAR(20),
    discount_value   BIGINT,
    idempotency_key  VARCHAR(255) NOT NULL,
    notes            TEXT,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE order_items (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id      UUID         NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id    UUID         NOT NULL,
    variant_id    UUID,
    product_name  VARCHAR(255) NOT NULL,  -- price snapshot at time of order
    variant_name  VARCHAR(255),
    unit_price    BIGINT       NOT NULL,  -- effective price (with branch override)
    quantity      INT          NOT NULL CHECK (quantity > 0),
    subtotal      BIGINT       NOT NULL,  -- unit_price * quantity
    notes         TEXT
);

CREATE TABLE order_item_addons (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id UUID         NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    addon_id      UUID         NOT NULL,
    addon_name    VARCHAR(255) NOT NULL,  -- snapshot
    price         BIGINT       NOT NULL
);

CREATE INDEX idx_orders_tenant_id ON orders(tenant_id);
CREATE INDEX idx_orders_shift_id  ON orders(shift_id);
CREATE INDEX idx_orders_status    ON orders(tenant_id, status);

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE shifts ENABLE ROW LEVEL SECURITY;
CREATE POLICY orders_isolation ON orders
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY shifts_isolation ON shifts
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### Kafka Events Published by pos service

```go
// domain/order/events.go
type OrderPaidEvent struct {
    OrderID    uuid.UUID
    TenantID   uuid.UUID
    BranchID   uuid.UUID
    CashierID  uuid.UUID
    Items      []OrderPaidItem   // product_id, variant_id, quantity
    OccurredAt time.Time
}
type OrderCancelledEvent struct { OrderID, TenantID uuid.UUID; OccurredAt time.Time }
```

### Kafka Consumer in pos service

Consumes `payment.completed` and `payment.voided`:
```
payment.completed → MarkOrderPaid command → order status = PAID
payment.voided    → (if order PAID) revert to PENDING_PAYMENT
```

### Tests — Epic 4.3

**Unit (domain/order):**
- `TestOrder_AddItem_DraftStatus_Success`
- `TestOrder_AddItem_NotDraft_ReturnsError`
- `TestOrder_Submit_EmptyItems_ReturnsError`
- `TestOrder_StateMachine_AllValidTransitions`
- `TestOrder_StateMachine_InvalidTransition_ReturnsError`
- `TestOrder_TaxCalculation_PPN11Percent`
- `TestOrder_TaxCalculation_PB1Percent`
- `TestOrder_TaxCalculation_TaxTypeNone`
- `TestOrder_DiscountFixed_ExceedsSubtotal_ReturnsError`
- `TestOrder_DiscountPercent_Over100_ReturnsError`
- `TestOrder_Total_SubtotalPlusTaxMinusDiscount`
- `TestOrder_ItemSnapshot_PriceImmutableAfterOrderSubmit`
- `TestShift_CannotCloseAlreadyClosed`
- `TestShift_UniquePerCashierBranch`

**Integration:**
- `TestOrderRepo_CreateAndFindByID`
- `TestOrderRepo_ListByShift`
- `TestOrderRepo_Idempotency_DuplicateKey`
- `TestShiftRepo_OpenAndClose`
- `TestKafkaConsumer_PaymentCompleted_MarksOrderPaid`

**E2E:**
- `TestE2E_Order_FullLifecycle`
- `TestE2E_Shift_OpenCloseZReport`
- `TestE2E_ConcurrentOrders_DifferentCashiers`
- `TestE2E_Discount_CashierLimitEnforced`
- `TestE2E_PendingOrder_ParkAndResume`

---

## 7. Epic 4.4 — Payment Service

### PaymentGateway Interface (Port)

```go
// domain/payment/gateway.go — zero external imports
type PaymentGateway interface {
    CreateQRIS(ctx context.Context, req CreateQRISRequest) (*QRISResult, error)
    GetStatus(ctx context.Context, externalID string) (*GatewayStatus, error)
    Refund(ctx context.Context, req RefundRequest) (*RefundResult, error)
    Cancel(ctx context.Context, externalID string) error
}

type CreateQRISRequest struct {
    ExternalID  string   // our payment_id
    Amount      int64
    Description string
    CustomerID  string
}

type QRISResult struct {
    ExternalID string   // Midtrans order_id
    QRCodeURL  string   // URL to QRIS image
    ExpiresAt  time.Time
}

type GatewayStatus struct {
    ExternalID        string
    Status            GatewayPaymentStatus  // pending|completed|failed|expired
    TransactionTime   time.Time
    GatewayRawResponse json.RawMessage
}
```

`NoopGateway` in `infrastructure/midtrans/noop.go` — always returns success with fake QR URL. Used in unit tests and local dev without Midtrans credentials.

### Payment Aggregate

```go
// domain/payment/payment.go
type Payment struct {
    ID             uuid.UUID
    TenantID       uuid.UUID
    OrderID        uuid.UUID
    IdempotencyKey string
    Method         PaymentMethod
    Status         PaymentStatus
    Amount         int64        // total expected
    AmountPaid     int64        // actual amount collected (cash may be > Amount)
    ChangeAmount   int64        // AmountPaid - Amount for cash
    Splits         []PaymentSplit  // for split payment
    Gateway        string       // "midtrans" or ""
    ExternalID     string
    ExternalQRURL  string
    GatewayResp    json.RawMessage
    VoidedBy       *uuid.UUID
    VoidedAt       *time.Time
    CreatedAt      time.Time
    UpdatedAt      time.Time
    events         []DomainEvent
}
```

### Idempotency Pattern (Redis + PostgreSQL)

```
Request arrives with idempotency_key
  │
  ▼
1. Redis SETNX idempotency:{key} "processing" EX 86400
   ├── acquired (first call):
   │     process payment → gateway → store result in PG
   │     Redis SET idempotency:{key} {serialized result} EX 86400
   │     return result
   └── not acquired (duplicate):
         Redis GET idempotency:{key}
         ├── "processing" → return 409 Conflict (retry after 1s)
         └── {result}     → return cached result (200 OK)
```

### DB Migration — payments

```sql
CREATE TABLE payments (
    id               UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID         NOT NULL,
    order_id         UUID         NOT NULL,
    idempotency_key  VARCHAR(255) NOT NULL,
    method           VARCHAR(30)  NOT NULL
                     CHECK (method IN ('cash','qris','card','split')),
    status           VARCHAR(30)  NOT NULL DEFAULT 'pending'
                     CHECK (status IN ('pending','completed','failed',
                                       'cancelled','voided','refunded')),
    amount           BIGINT       NOT NULL CHECK (amount > 0),
    amount_paid      BIGINT       NOT NULL DEFAULT 0,
    change_amount    BIGINT       NOT NULL DEFAULT 0,
    gateway          VARCHAR(30),               -- 'midtrans' | NULL
    external_id      VARCHAR(255),
    external_qr_url  TEXT,
    gateway_response JSONB,
    voided_by        UUID,
    voided_at        TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, idempotency_key)
);

CREATE TABLE payment_splits (
    id          UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id  UUID    NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    method      VARCHAR(30) NOT NULL,
    amount      BIGINT  NOT NULL CHECK (amount > 0)
);

CREATE TABLE refunds (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id         UUID         NOT NULL REFERENCES payments(id),
    idempotency_key    VARCHAR(255) NOT NULL UNIQUE,
    amount             BIGINT       NOT NULL CHECK (amount > 0),
    reason             TEXT,
    approved_by        UUID         NOT NULL,   -- user_id with manager/owner role
    external_refund_id VARCHAR(255),
    status             VARCHAR(30)  NOT NULL DEFAULT 'pending'
                       CHECK (status IN ('pending','completed','failed')),
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX idx_payments_tenant_id ON payments(tenant_id);
CREATE INDEX idx_payments_order_id  ON payments(order_id);

ALTER TABLE payments ENABLE ROW LEVEL SECURITY;
ALTER TABLE refunds  ENABLE ROW LEVEL SECURITY;
CREATE POLICY payments_isolation ON payments
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### Midtrans Webhook Handler

```
POST /webhooks/midtrans
  1. Parse body → MidtransNotification struct
  2. Validate signature:
       expected = SHA512(order_id + status_code + gross_amount + server_key)
       reject if signature_key != expected → HTTP 400
  3. Map transaction_status:
       "settlement" | "capture" → PaymentCompleted
       "expire" | "deny" | "cancel" → PaymentFailed
       "pending" → no-op (already pending)
  4. Call application: UpdatePaymentStatus(externalID, newStatus, rawBody)
  5. Publish Kafka event: payment.completed or payment.voided
  6. Return HTTP 200 (Midtrans retries on non-2xx up to 5×)

Security: webhook endpoint does NOT require PASETO auth (Midtrans has no token).
          Signature validation IS the auth. Always validate before processing.
```

### Tests — Epic 4.4

**Unit (domain):**
- `TestPayment_IdempotencyKey_Required`
- `TestPayment_StateMachine_CannotVoidWithoutManagerPIN`
- `TestPayment_StateMachine_CannotRefundVoided`
- `TestPayment_SplitPayment_MustSumToTotal`
- `TestPayment_CashChange_Calculated_Correctly`
- `TestPayment_RefundExceedsPaidAmount_ReturnsError`

**Unit (Midtrans webhook):**
- `TestWebhook_SignatureValidation_Valid`
- `TestWebhook_SignatureValidation_InvalidKey_Rejected`
- `TestWebhook_SettlementStatus_MapsToCompleted`
- `TestWebhook_ExpireStatus_MapsToFailed`
- `TestWebhook_PendingStatus_NoStateChange`

**Integration (testcontainers PG + Redis):**
- `TestPaymentRepo_SaveAndFindByID`
- `TestPaymentRepo_Idempotency_DuplicateKeyReturnsExisting`
- `TestIdempotencyStore_SetNX_SecondCallReturnsFalse`
- `TestIdempotencyStore_TTL_ExpiresAfter24h`

**E2E:**
- `TestE2E_CashPayment_OrderMarkedPaid`
- `TestE2E_QRISPayment_WebhookCompletes_InventoryDeducted`
- `TestE2E_SplitPayment_CashAndQRIS`
- `TestE2E_VoidPayment_RequiresManagerPIN`
- `TestE2E_VoidPayment_WrongPIN_Rejected`
- `TestE2E_PartialRefund_CorrectAmountReturned`
- `TestE2E_DuplicatePaymentRequest_IdempotentResponse`

---

## 8. Epic 4.5 — Inventory Service

### Domain Model

```go
// domain/stock/stock_ledger.go
type StockLedger struct {
    ID           uuid.UUID
    TenantID     uuid.UUID
    WarehouseID  uuid.UUID
    ProductID    uuid.UUID
    VariantID    *uuid.UUID
    Quantity     int64      // can be negative (backorder allowed, triggers alert)
    ReorderPoint int64      // when Quantity ≤ ReorderPoint → LowStockAlert event
    UpdatedAt    time.Time
}

func (s *StockLedger) Deduct(qty int64, ref StockReference) (*StockMutation, error)
func (s *StockLedger) Receive(qty int64, ref StockReference) (*StockMutation, error)
func (s *StockLedger) Adjust(qty int64, reason AdjustReason, ref StockReference) (*StockMutation, error)
func (s *StockLedger) IsBelowReorderPoint() bool
```

### BOM/Recipe Flow

```
OrderPaid event arrives with items: [{product_id: "nasi-padang", qty: 2}]
  │
  ▼
inventory service checks BOM:
  BOM exists for "nasi-padang":
    ingredients: [{beras, 200g}, {rendang, 150g}]
  │
  ▼
Deduct per ingredient × order quantity:
  beras   → -400g
  rendang → -300g
  (NOT "nasi-padang" itself — it's a finished product)
  │
  ▼
No BOM for product → deduct product directly
```

### Kafka Consumer Architecture

```go
// infrastructure/kafka/order_paid_consumer.go
// Consumer group: inventory-service
// Topic: pos.order.paid
// At-least-once delivery + idempotency via stock_mutations UNIQUE constraint

func (c *OrderPaidConsumer) Handle(ctx context.Context, event OrderPaidEvent) error {
    for _, item := range event.Items {
        ingredients := c.bomRepo.FindByProduct(item.ProductID)
        if len(ingredients) == 0 {
            // deduct product directly
            ingredients = [{ProductID: item.ProductID, Qty: 1}]
        }
        for _, ing := range ingredients {
            ledger := c.stockRepo.FindOrCreate(ing.ProductID, defaultWarehouse)
            mutation, err := ledger.Deduct(ing.Qty * item.Quantity, StockReference{
                ReferenceID:   event.OrderID,
                ReferenceType: "order",
            })
            // mutation has UNIQUE(reference_id, reference_type, product_id, variant_id)
            // → idempotent: duplicate consume is a no-op
            if ledger.IsBelowReorderPoint() {
                c.eventPublisher.Publish(LowStockAlertEvent{...})
            }
        }
    }
}
```

### DB Migration — inventory

```sql
CREATE TABLE warehouses (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID         NOT NULL,
    branch_id   UUID         NOT NULL,
    name        VARCHAR(255) NOT NULL,
    is_default  BOOLEAN      NOT NULL DEFAULT false,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE TABLE stock_ledger (
    id             UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id      UUID    NOT NULL,
    warehouse_id   UUID    NOT NULL REFERENCES warehouses(id),
    product_id     UUID    NOT NULL,
    variant_id     UUID,
    quantity       BIGINT  NOT NULL DEFAULT 0,
    reorder_point  BIGINT  NOT NULL DEFAULT 0,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (warehouse_id, product_id, COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::UUID))
);

CREATE TABLE stock_mutations (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID        NOT NULL,
    warehouse_id    UUID        NOT NULL,
    product_id      UUID        NOT NULL,
    variant_id      UUID,
    quantity_delta  BIGINT      NOT NULL,   -- negative = deduction
    reason          VARCHAR(50) NOT NULL
                    CHECK (reason IN ('sale','purchase','adjustment',
                                      'transfer_in','transfer_out','opname')),
    reference_id    UUID,
    reference_type  VARCHAR(50),
    notes           TEXT,
    created_by      UUID        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (reference_id, reference_type, product_id,
            COALESCE(variant_id, '00000000-0000-0000-0000-000000000000'::UUID))
);

CREATE TABLE bom_recipes (
    id                     UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id              UUID    NOT NULL,
    product_id             UUID    NOT NULL,
    ingredient_product_id  UUID    NOT NULL,
    ingredient_variant_id  UUID,
    quantity               BIGINT  NOT NULL CHECK (quantity > 0),
    unit                   VARCHAR(20) NOT NULL,
    UNIQUE (product_id, ingredient_product_id,
            COALESCE(ingredient_variant_id, '00000000-0000-0000-0000-000000000000'::UUID))
);

ALTER TABLE stock_ledger   ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_mutations ENABLE ROW LEVEL SECURITY;
CREATE POLICY stock_isolation ON stock_ledger
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY mutation_isolation ON stock_mutations
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
```

### Tests — Epic 4.5

**Unit (domain/stock):**
- `TestStockLedger_Deduct_UpdatesQuantity`
- `TestStockLedger_Deduct_BelowZero_AllowedWithAlert`
- `TestStockLedger_IsBelowReorderPoint_TrueWhenEqual`
- `TestBOM_DeductIngredientsNotProduct`

**Unit (application):**
- `TestDeductStock_NoBOM_DeductsProductDirectly`
- `TestDeductStock_WithBOM_DeductsIngredients`
- `TestDeductStock_IdempotentOnDuplicateOrderID`

**Integration (testcontainers PG + Kafka):**
- `TestStockRepo_AdjustAndRead`
- `TestKafkaConsumer_OrderPaid_DeductsStock`
- `TestKafkaConsumer_OrderPaid_IdempotentDoubleConsume`
- `TestKafkaConsumer_OrderPaid_WithBOM_DeductsIngredients`
- `TestLowStock_AlertPublished_WhenBelowReorderPoint`

**E2E:**
- `TestE2E_OrderPaid_StockDeducted_BOMApplied`
- `TestE2E_ManualAdjustment_AuditTrailComplete`
- `TestE2E_LowStock_AlertPublished`
- `TestE2E_SetReorderPoint_TriggersAlert`

---

## 9. Cross-Cutting Concerns

### Swagger / OpenAPI Generation

Add to `proto/buf.gen.yaml`:
```yaml
- plugin: buf.build/grpc-ecosystem/openapiv2
  out: ../gen/openapi
  opt:
    - logtostderr=true
    - allow_merge=true
    - merge_file_name=api
    - generate_unbound_methods=true
```

Swagger UI via Docker Compose (add to docker-compose.yml):
```yaml
swagger-ui:
  image: swaggerapi/swagger-ui:latest
  container_name: xyn-swagger
  ports:
    - "8090:8080"
  environment:
    SWAGGER_JSON: /api/api.swagger.json
  volumes:
    - ./gen/openapi:/api:ro
  networks:
    - xyn-net
```

Add to Makefile:
```makefile
swagger-gen:
    buf generate proto/ --template proto/buf.gen.yaml
    @echo "OpenAPI spec: gen/openapi/api.swagger.json"
    @echo "Swagger UI:  http://localhost:8090"
```

### Error Mapping Enhancement (shared/go/pkg/errors)

New file `shared/go/pkg/errors/grpc.go`:

```go
// Sentinel errors that all services map their domain errors to
var (
    ErrNotFound      = errors.New("not found")
    ErrAlreadyExists = errors.New("already exists")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrInvalidInput  = errors.New("invalid input")
    ErrConflict      = errors.New("conflict")
    ErrPrecondition  = errors.New("precondition failed")
)

// MapToGRPCStatus converts any error to a gRPC status error.
// All gRPC handlers call this at the boundary.
func MapToGRPCStatus(err error) error {
    switch {
    case err == nil:                        return nil
    case errors.Is(err, ErrNotFound):       return status.Error(codes.NotFound, err.Error())
    case errors.Is(err, ErrAlreadyExists):  return status.Error(codes.AlreadyExists, err.Error())
    case errors.Is(err, ErrUnauthorized):   return status.Error(codes.Unauthenticated, err.Error())
    case errors.Is(err, ErrForbidden):      return status.Error(codes.PermissionDenied, err.Error())
    case errors.Is(err, ErrInvalidInput):   return status.Error(codes.InvalidArgument, err.Error())
    case errors.Is(err, ErrConflict):       return status.Error(codes.AlreadyExists, err.Error())
    case errors.Is(err, ErrPrecondition):   return status.Error(codes.FailedPrecondition, err.Error())
    default:                                return status.Error(codes.Internal, "internal error")
    }
}
```

Each service maps its domain errors to these shared sentinels using `fmt.Errorf("...: %w", ErrNotFound)`.

### Distributed Tracing Enhancement (OTEL)

Additions to `shared/go/pkg/telemetry`:

**1. Kafka carrier for trace propagation:**
```go
// telemetry/kafka.go
type KafkaHeaderCarrier map[string][]byte

func InjectToKafkaHeaders(ctx context.Context, headers map[string][]byte)
func ExtractFromKafkaHeaders(ctx context.Context, headers map[string][]byte) context.Context
```

**2. Claims baggage in gRPC interceptor:**
```go
// middleware/auth.go — after PASETO validation, inject into span
span.SetAttributes(
    attribute.String("tenant.id", claims.TenantID.String()),
    attribute.String("user.id",   claims.UserID.String()),
    attribute.String("user.role", string(claims.Role)),
)
```

**3. Structured log correlation:**
Every log entry automatically includes `trace_id` and `span_id` extracted from context, enabling log → trace correlation in Grafana.

### Makefile Targets

```makefile
# Phase 4 additions
proto-gen:
    cd proto && buf generate

swagger-gen: proto-gen
    @echo "Swagger UI: http://localhost:8090"

test-unit:
    go test -count=1 -race ./services/tenant/... ./services/pos/... \
            ./services/payment/... ./services/inventory/... \
            ./shared/go/...

test-integration:
    go test -count=1 -race -tags=integration \
            -timeout=120s ./services/.../...

test-e2e:
    go test -count=1 -v -tags=e2e -timeout=300s ./tests/e2e/...

test-all: test-unit test-integration test-e2e

lint:
    golangci-lint run ./services/... ./shared/go/...
```

---

## 10. Testing Strategy

### Test Pyramid

```
        ┌───────────────────────────┐
        │   E2E Tests (tests/e2e/)  │  ← Full golden path + edge cases
        │   ~30 tests, 5 scenarios  │    Run against docker-compose
        ├───────────────────────────┤
        │  Integration Tests        │  ← Real PG + Redis + Kafka
        │  per service (~15 each)   │    testcontainers-go
        ├───────────────────────────┤
        │   Unit Tests (majority)   │  ← Domain + Application layer
        │   ~50+ per service        │    No I/O, pure Go, fast
        └───────────────────────────┘
```

### Golden Path E2E Scenario (tests/e2e/full_scenario_test.go)

```
Scenario: Warung Padang — satu hari operasional penuh

Step  1: RegisterUser (owner: Budi) → get PASETO token
Step  2: RegisterUser (cashier: Ani) → scoped to Branch A
Step  3: CreateBranch "Branch A"
Step  4: CreateCategory "Makanan Utama"
Step  5: CreateProduct "Nasi Padang" (Rp 25.000, PPN, SKU: NP-001)
           + Variant "Porsi Kecil" (-Rp 5.000)
           + AddonGroup "Extra Lauk" [Rendang +Rp 8.000, Ayam +Rp 5.000]
Step  6: SetStock warehouse default: Nasi Padang = 100 pcs
Step  7: UpsertBOMRecipe Nasi Padang:
           beras 200g, rendang 150g
Step  8: OpenShift (cashier Ani, Branch A, opening_cash Rp 500.000)
Step  9: CreateOrder (dine_in, table 5, idempotency_key: "ord-001")
Step 10: AddItem Nasi Padang qty=2 + addon Extra Rendang × 2
Step 11: ApplyDiscount 10% total
Step 12: SubmitOrder → status = PENDING_PAYMENT
Step 13: InitiatePayment QRIS (idempotency_key: "pay-001")
           → receive QR URL
Step 14: Simulate Midtrans webhook: settlement for pay-001
Step 15: Assert: payment status = COMPLETED
Step 16: Assert: order status = PAID (via payment.completed Kafka event)
Step 17: Assert: stock beras -400g, rendang -300g (BOM × qty 2)
Step 18: Assert: Kafka pos.order.paid consumed by inventory
Step 19: InitiatePayment again (idempotency_key: "pay-001") → same result (idempotent)
Step 20: VoidPayment with cashier PIN → REJECTED (cashier role forbidden)
Step 21: VoidPayment with manager PIN → SUCCESS
Step 22: Assert: order status = PENDING_PAYMENT (reopened)
Step 23: CreateOrder again (idempotency_key: "ord-001") → returns existing order
Step 24: CloseShift → Z-report: total_sales, cash_sales, qris_sales
Step 25: Assert: all trace IDs connected across pos → payment → inventory spans
```

### Edge Cases Coverage

| Category | Cases |
|---|---|
| Idempotency | Duplicate payment key, duplicate order key, duplicate Kafka consume |
| Race conditions | Concurrent orders same cashier, concurrent stock deduction |
| Auth & RBAC | Wrong tenant token, expired token, insufficient role, branch scope violation |
| Financial | Negative price, zero payment amount, refund > paid amount, discount > subtotal |
| State machines | Invalid order transitions, invalid payment transitions |
| Kafka | Consumer group rebalance, duplicate message, out-of-order message |
| Webhooks | Invalid Midtrans signature, duplicate webhook, unknown transaction ID |
| Stock | BOM missing ingredient, backorder (negative stock), reorder point alert |

---

## 11. Proto Contract Map

### New Proto Files

```
proto/
├── events/v1/
│   └── events.proto          ← NEW: Kafka event message types
├── tenant/v1/
│   ├── tenant.proto          ← existing
│   └── user.proto            ← NEW
├── pos/v1/
│   ├── product.proto         ← NEW
│   └── order.proto           ← NEW
├── payment/v1/
│   └── payment.proto         ← NEW
└── inventory/v1/
    └── inventory.proto       ← NEW
```

### Proto Registration in buf.yaml

The existing `proto/buf.yaml` already covers `proto/` directory. New `.proto` files are automatically picked up — no config change needed.

### gRPC-Gateway REST Endpoints

Each proto service gets HTTP transcoding annotations for REST access via KrakenD:

| gRPC Method | REST Endpoint |
|---|---|
| `UserService.RegisterUser` | `POST /v1/tenants/{tenant_id}/users` |
| `UserService.Login` | `POST /v1/auth/login` |
| `UserService.VerifyPIN` | `POST /v1/auth/pin/verify` |
| `ProductService.CreateProduct` | `POST /v1/products` |
| `ProductService.ListProducts` | `GET /v1/products` |
| `ProductService.LookupBySKU` | `GET /v1/products/sku/{sku}` |
| `OrderService.CreateOrder` | `POST /v1/orders` |
| `OrderService.AddItem` | `POST /v1/orders/{order_id}/items` |
| `OrderService.SubmitOrder` | `POST /v1/orders/{order_id}/submit` |
| `PaymentService.InitiatePayment` | `POST /v1/payments` |
| `PaymentService.VoidPayment` | `POST /v1/payments/{payment_id}/void` |
| `InventoryService.GetStock` | `GET /v1/inventory/{product_id}/stock` |
| `InventoryService.AdjustStock` | `POST /v1/inventory/{product_id}/adjust` |

---

## 12. Implementation Order

Each epic gets its own implementation plan (via `superpowers:writing-plans`) before coding begins.

```
Phase 4a: Redis + Cache package           (prerequisite for payment service)
Phase 4b: Epic 4.1 — Identity & Auth      (prerequisite for all services)
Phase 4c: Epic 4.2 — Product & Menu       (prerequisite for POS orders)
Phase 4d: Epic 4.3 — POS Core             (prerequisite for payment)
Phase 4e: Epic 4.4 — Payment Service      (prerequisite for E2E)
Phase 4f: Epic 4.5 — Inventory Service    (consumes pos.order.paid)
Phase 4g: Cross-cutting + E2E tests       (Swagger, tracing, full scenario)
```

**Branch naming:**
- All Phase 4 work on: `feat/phase4-backend-mvp`
- Merged to `main` at end of each sub-phase via PR with CI green

---

*Design finalized: 2026-06-07 | Phase 4 Master Spec v1.0*
