# Phase 4c — Product & Menu (pos service) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the Product & Menu bounded context in the `pos` service: product catalog with categories, variants, add-on groups, branch-specific price overrides, barcode SKU lookup, and soft-archive lifecycle.

**Architecture:** DDD within `services/pos`. Product is the aggregate root controlling variants and addon groups. Category is a separate entity with sort ordering. `BranchProduct` is a join entity for branch-specific price overrides. The `pos` service will eventually host both Product (4c) and Order (4d) domains — both live in `services/pos/internal/`.

**Tech Stack:** Go 1.26.4, pgx/v5, goose v3.27.1, testcontainers-go v0.42.0, buf v1.70.0.

**Branch:** `feat/phase4-backend-mvp`

---

## File Structure

```
proto/pos/v1/
└── product.proto                         ← NEW

services/pos/internal/
├── domain/product/
│   ├── product.go                        ← NEW: Product aggregate + Category + Variant + AddonGroup
│   ├── errors.go                         ← NEW
│   ├── repository.go                     ← NEW: ProductRepository + CategoryRepository
│   └── product_test.go                   ← NEW
├── application/
│   ├── command/
│   │   ├── create_product.go             ← NEW
│   │   ├── create_product_test.go        ← NEW
│   │   ├── update_product.go             ← NEW
│   │   ├── archive_product.go            ← NEW
│   │   ├── create_category.go            ← NEW
│   │   ├── reorder_categories.go         ← NEW
│   │   └── set_branch_price.go           ← NEW
│   └── query/
│       ├── get_product.go                ← NEW
│       ├── list_products.go              ← NEW
│       └── lookup_by_sku.go             ← NEW
├── infrastructure/
│   └── postgres/
│       ├── product_repo.go               ← NEW
│       ├── product_repo_test.go          ← NEW (integration)
│       └── migrations/
│           └── 00001_create_products.sql ← NEW
└── interfaces/grpc/
    └── product_handler.go                ← NEW

services/pos/provider.go                  ← NEW (initial version)
services/pos/config.go                    ← NEW
services/pos/go.mod                       ← MODIFY: add dependencies
```

---

### Task 1: Add dependencies to services/pos/go.mod

**Files:**
- Modify: `services/pos/go.mod`

- [ ] **Step 1: Add required packages**

```bash
cd services/pos
go get github.com/google/uuid@v1.6.0
go get github.com/jackc/pgx/v5@v5.9.2
go get github.com/pressly/goose/v3@v3.27.1
go get github.com/stretchr/testify@v1.11.1
go get github.com/testcontainers/testcontainers-go@v0.42.0
go get github.com/testcontainers/testcontainers-go/modules/postgres@v0.42.0
go get go.opentelemetry.io/otel@v1.44.0
go get go.opentelemetry.io/otel/trace@v1.44.0
go get google.golang.org/grpc@v1.81.1
```

- [ ] **Step 2: Sync workspace**

```bash
cd ../.. && go work sync
```

- [ ] **Step 3: Commit**

```bash
git add services/pos/go.mod services/pos/go.sum go.work.sum
git commit -m "chore(pos): add Go dependencies (pgx, goose, testcontainers, grpc)"
```

---

### Task 2: Write proto/pos/v1/product.proto and generate

**Files:**
- Create: `proto/pos/v1/product.proto`

- [ ] **Step 1: Create product.proto**

```protobuf
syntax = "proto3";

package pos.v1;

import "common/v1/pagination.proto";

option go_package = "github.com/xyn-pos/gen/pos/v1;posv1";

// TaxType determines how tax is applied to a product.
enum TaxType {
  TAX_TYPE_UNSPECIFIED = 0;
  TAX_TYPE_PPN         = 1; // 11% exclusive (added on top)
  TAX_TYPE_PB1         = 2; // 10% inclusive (reporting only)
  TAX_TYPE_NONE        = 3;
}

// Category groups products for display and filtering.
message Category {
  string id         = 1;
  string tenant_id  = 2;
  string name       = 3;
  int32  sort_order = 4;
  string parent_id  = 5; // empty = top-level
  bool   is_active  = 6;
}

// Variant is a product option (e.g. "Porsi Kecil") with a price delta.
message Variant {
  string id          = 1;
  string product_id  = 2;
  string name        = 3;
  int64  price_delta = 4; // sen; FinalPrice = BasePrice + PriceDelta
  string sku         = 5;
  bool   is_active   = 6;
}

// Addon is a selectable extra within an addon group.
message Addon {
  string id             = 1;
  string addon_group_id = 2;
  string name           = 3;
  int64  price          = 4; // sen, >= 0
  bool   is_active      = 5;
}

// AddonGroup is a set of add-ons attached to a product (e.g. "Extra Lauk").
message AddonGroup {
  string id             = 1;
  string product_id     = 2;
  string name           = 3;
  bool   is_required    = 4;
  int32  max_selections = 5;
  repeated Addon addons = 6;
}

// Product is the catalog item aggregate.
message Product {
  string id                    = 1;
  string tenant_id             = 2;
  string category_id           = 3;
  string sku                   = 4;
  string name                  = 5;
  string description           = 6;
  int64  base_price            = 7;  // sen
  TaxType tax_type             = 8;
  bool   is_active             = 9;
  repeated Variant variants    = 10;
  repeated AddonGroup addon_groups = 11;
  string created_at            = 12;
  string updated_at            = 13;
}

// --- Requests ---

message CreateCategoryRequest {
  string tenant_id  = 1;
  string name       = 2;
  int32  sort_order = 3;
  string parent_id  = 4;
}
message CreateCategoryResponse { Category category = 1; }

message ReorderCategoriesRequest {
  string tenant_id    = 1;
  repeated string ids = 2; // ordered list of category IDs
}
message ReorderCategoriesResponse {}

message CreateProductRequest {
  string idempotency_key  = 1;
  string tenant_id        = 2;
  string category_id      = 3;
  string sku              = 4;
  string name             = 5;
  string description      = 6;
  int64  base_price       = 7;
  TaxType tax_type        = 8;
  repeated Variant variants       = 9;
  repeated AddonGroup addon_groups = 10;
}
message CreateProductResponse { Product product = 1; }

message UpdateProductRequest {
  string product_id   = 1;
  string name         = 2;
  string description  = 3;
  int64  base_price   = 4;
  TaxType tax_type    = 5;
  string category_id  = 6;
}
message UpdateProductResponse { Product product = 1; }

message ArchiveProductRequest { string product_id = 1; }
message ArchiveProductResponse {}

message SetBranchPriceRequest {
  string branch_id     = 1;
  string product_id    = 2;
  int64  override_price = 3; // 0 = remove override
  bool   is_available  = 4;
}
message SetBranchPriceResponse {}

message GetProductRequest    { string product_id = 1; }
message GetProductResponse   { Product product = 1; }

message ListProductsRequest {
  string tenant_id    = 1;
  string category_id  = 2; // optional filter
  string branch_id    = 3; // optional: include branch override price
  bool   active_only  = 4;
  common.v1.PaginationRequest pagination = 5;
}
message ListProductsResponse {
  repeated Product products    = 1;
  common.v1.PaginationMeta pagination = 2;
}

message LookupBySKURequest {
  string tenant_id = 1;
  string sku       = 2;
}
message LookupBySKUResponse { Product product = 1; }

// ProductService manages product catalog for a tenant.
service ProductService {
  rpc CreateCategory(CreateCategoryRequest)       returns (CreateCategoryResponse);
  rpc ReorderCategories(ReorderCategoriesRequest) returns (ReorderCategoriesResponse);
  rpc CreateProduct(CreateProductRequest)         returns (CreateProductResponse);
  rpc UpdateProduct(UpdateProductRequest)         returns (UpdateProductResponse);
  rpc ArchiveProduct(ArchiveProductRequest)       returns (ArchiveProductResponse);
  rpc SetBranchPrice(SetBranchPriceRequest)       returns (SetBranchPriceResponse);
  rpc GetProduct(GetProductRequest)               returns (GetProductResponse);
  rpc ListProducts(ListProductsRequest)           returns (ListProductsResponse);
  rpc LookupBySKU(LookupBySKURequest)             returns (LookupBySKUResponse);
}
```

- [ ] **Step 2: Also create proto/pos/v1/ directory and buf generate**

```bash
mkdir -p proto/pos/v1
cd proto && buf generate
```

Expected: `gen/pos/v1/product.pb.go` and `gen/pos/v1/product_grpc.pb.go` created. No errors.

- [ ] **Step 3: Commit**

```bash
git add proto/pos/v1/product.proto gen/
git commit -m "feat(proto): add ProductService proto for pos/v1"
```

---

### Task 3: Domain — product.go, errors.go, repository.go

**Files:**
- Create: `services/pos/internal/domain/product/product.go`
- Create: `services/pos/internal/domain/product/errors.go`
- Create: `services/pos/internal/domain/product/repository.go`
- Create: `services/pos/internal/domain/product/product_test.go`

- [ ] **Step 1: Write failing domain tests**

Create `services/pos/internal/domain/product/product_test.go`:

```go
package product_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

func TestNewProduct_NegativeBasePrice_ReturnsError(t *testing.T) {
	_, err := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi Padang", "", -100, product.TaxTypePPN)
	assert.ErrorIs(t, err, product.ErrInvalidPrice)
}

func TestNewProduct_EmptyName_ReturnsError(t *testing.T) {
	_, err := product.NewProduct(uuid.New(), nil, "SKU-001", "", "", 15000, product.TaxTypePPN)
	assert.ErrorIs(t, err, product.ErrInvalidName)
}

func TestNewProduct_Valid_SetsIsActiveTrue(t *testing.T) {
	p, err := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi Padang", "Desc", 25000_00, product.TaxTypePPN)
	require.NoError(t, err)
	assert.True(t, p.IsActive)
	assert.NotEqual(t, uuid.Nil, p.ID)
}

func TestProduct_AddVariant_NegativeFinalPrice_ReturnsError(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 5000, product.TaxTypePPN)
	err := p.AddVariant(product.Variant{
		ID:         uuid.New(),
		Name:       "Porsi Kecil",
		PriceDelta: -10000, // base 5000 + delta -10000 = -5000 → invalid
		SKU:        "SKU-001-S",
		IsActive:   true,
	})
	assert.ErrorIs(t, err, product.ErrVariantPriceInvalid)
}

func TestProduct_AddVariant_Valid_AddsToSlice(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 25000_00, product.TaxTypePPN)
	v := product.Variant{ID: uuid.New(), Name: "Small", PriceDelta: -5000_00, SKU: "SKU-001-S", IsActive: true}
	err := p.AddVariant(v)
	require.NoError(t, err)
	assert.Len(t, p.Variants, 1)
}

func TestProduct_AddAddonGroup_MaxSelectionsZero_ReturnsError(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 25000_00, product.TaxTypePPN)
	err := p.AddAddonGroup(product.AddonGroup{
		ID:            uuid.New(),
		Name:          "Extra Lauk",
		IsRequired:    false,
		MaxSelections: 0, // must be >= 1
	})
	assert.ErrorIs(t, err, product.ErrInvalidAddonGroup)
}

func TestProduct_Archive_AlreadyInactive_ReturnsError(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 25000_00, product.TaxTypePPN)
	p.Archive()
	err := p.Archive()
	assert.ErrorIs(t, err, product.ErrProductAlreadyArchived)
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
cd services/pos && go test ./internal/domain/product/... -v
```

Expected: FAIL (package does not exist)

- [ ] **Step 3: Create errors.go**

```go
package product

import "errors"

// Sentinel errors for the product domain.
var (
	ErrProductNotFound       = errors.New("product not found")
	ErrSKUAlreadyExists      = errors.New("SKU already exists in this tenant")
	ErrCategoryNotFound      = errors.New("category not found")
	ErrProductHasActiveOrders = errors.New("cannot archive product with active orders")
	ErrVariantPriceInvalid   = errors.New("variant final price (base + delta) must be >= 0")
	ErrInvalidPrice          = errors.New("base price must be >= 0")
	ErrInvalidName           = errors.New("product name cannot be empty")
	ErrInvalidAddonGroup     = errors.New("addon group max_selections must be >= 1")
	ErrProductAlreadyArchived = errors.New("product is already archived")
)
```

- [ ] **Step 4: Create product.go**

```go
package product

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TaxType determines how tax is applied.
type TaxType string

// Valid tax types.
const (
	TaxTypePPN  TaxType = "ppn"  // 11% exclusive
	TaxTypePB1  TaxType = "pb1"  // 10% inclusive (reporting)
	TaxTypeNone TaxType = "none"
)

// Category groups products for display.
type Category struct {
	ID        uuid.UUID
	TenantID  uuid.UUID
	Name      string
	SortOrder int
	ParentID  *uuid.UUID
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// Variant is a product option with a price delta from the base price.
type Variant struct {
	ID         uuid.UUID
	ProductID  uuid.UUID
	Name       string
	PriceDelta int64 // negative allowed; FinalPrice = Product.BasePrice + PriceDelta
	SKU        string
	IsActive   bool
}

// FinalPrice returns the effective variant price. Panics never — returns 0 if negative.
func (v Variant) FinalPrice(basePrice int64) int64 {
	fp := basePrice + v.PriceDelta
	if fp < 0 {
		return 0
	}
	return fp
}

// Addon is a selectable extra within an addon group.
type Addon struct {
	ID           uuid.UUID
	AddonGroupID uuid.UUID
	Name         string
	Price        int64 // >= 0
	IsActive     bool
}

// AddonGroup is a set of add-ons attached to a product.
type AddonGroup struct {
	ID            uuid.UUID
	ProductID     uuid.UUID
	Name          string
	IsRequired    bool
	MaxSelections int
	Addons        []Addon
}

// BranchProduct holds a branch-specific price override for a product.
type BranchProduct struct {
	BranchID      uuid.UUID
	ProductID     uuid.UUID
	OverridePrice *int64 // nil = use product.BasePrice
	IsAvailable   bool
	UpdatedAt     time.Time
}

// Product is the catalog aggregate root.
type Product struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	CategoryID  *uuid.UUID
	SKU         string
	Name        string
	Description string
	BasePrice   int64 // minor units (sen). Rp 25.000 = 2500000
	TaxType     TaxType
	Variants    []Variant
	AddonGroups []AddonGroup
	IsActive    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// NewProduct constructs a Product aggregate. Validates name and base price.
func NewProduct(tenantID uuid.UUID, categoryID *uuid.UUID, sku, name, description string, basePrice int64, taxType TaxType) (*Product, error) {
	if name == "" {
		return nil, ErrInvalidName
	}
	if basePrice < 0 {
		return nil, ErrInvalidPrice
	}

	now := time.Now().UTC()
	return &Product{
		ID:          uuid.New(),
		TenantID:    tenantID,
		CategoryID:  categoryID,
		SKU:         sku,
		Name:        name,
		Description: description,
		BasePrice:   basePrice,
		TaxType:     taxType,
		IsActive:    true,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

// AddVariant appends a variant. Returns ErrVariantPriceInvalid if final price < 0.
func (p *Product) AddVariant(v Variant) error {
	if p.BasePrice+v.PriceDelta < 0 {
		return fmt.Errorf("AddVariant %q: %w", v.Name, ErrVariantPriceInvalid)
	}
	v.ProductID = p.ID
	p.Variants = append(p.Variants, v)
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// AddAddonGroup appends an addon group. MaxSelections must be >= 1.
func (p *Product) AddAddonGroup(ag AddonGroup) error {
	if ag.MaxSelections < 1 {
		return fmt.Errorf("AddAddonGroup %q: %w", ag.Name, ErrInvalidAddonGroup)
	}
	ag.ProductID = p.ID
	p.AddonGroups = append(p.AddonGroups, ag)
	p.UpdatedAt = time.Now().UTC()
	return nil
}

// Archive soft-deletes the product. Returns ErrProductAlreadyArchived if already inactive.
func (p *Product) Archive() error {
	if !p.IsActive {
		return ErrProductAlreadyArchived
	}
	p.IsActive = false
	p.UpdatedAt = time.Now().UTC()
	return nil
}
```

- [ ] **Step 5: Create repository.go**

```go
package product

import (
	"context"

	"github.com/google/uuid"
)

// ProductRepository is the persistence port for Product aggregates.
type ProductRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
	FindBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*Product, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID, filter Filter) ([]*Product, int, error)
	Save(ctx context.Context, p *Product) error
	Update(ctx context.Context, p *Product) error
	SetBranchPrice(ctx context.Context, bp BranchProduct) error
	HasActiveOrders(ctx context.Context, productID uuid.UUID) (bool, error)
}

// CategoryRepository is the persistence port for Category entities.
type CategoryRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Category, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*Category, error)
	Save(ctx context.Context, c *Category) error
	UpdateSortOrders(ctx context.Context, tenantID uuid.UUID, orderedIDs []uuid.UUID) error
}

// Filter for list queries.
type Filter struct {
	CategoryID *uuid.UUID
	BranchID   *uuid.UUID // include branch override prices
	ActiveOnly bool
	Limit      int
	Offset     int
}
```

- [ ] **Step 6: Run domain tests**

```bash
cd services/pos && go test ./internal/domain/product/... -v -count=1
```

Expected: all 7 tests PASS.

- [ ] **Step 7: Commit**

```bash
git add services/pos/internal/domain/product/
git commit -m "feat(pos/domain): add Product aggregate with Variant and AddonGroup rules"
```

---

### Task 4: DB Migration — 00001_create_products.sql

**Files:**
- Create: `services/pos/internal/infrastructure/postgres/migrations/00001_create_products.sql`

- [ ] **Step 1: Create migration file**

```sql
-- +goose Up
-- +goose StatementBegin
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
    branch_id      UUID    NOT NULL,
    product_id     UUID    NOT NULL REFERENCES products(id),
    override_price BIGINT,                -- NULL = use products.base_price
    is_available   BOOLEAN NOT NULL DEFAULT true,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (branch_id, product_id)
);

CREATE INDEX idx_products_tenant_id   ON products(tenant_id);
CREATE INDEX idx_products_category_id ON products(category_id);
CREATE INDEX idx_products_sku ON products(tenant_id, sku) WHERE sku IS NOT NULL;

ALTER TABLE categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE products   ENABLE ROW LEVEL SECURITY;

CREATE POLICY categories_isolation ON categories
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY products_isolation ON products
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS branch_products;
DROP TABLE IF EXISTS addons;
DROP TABLE IF EXISTS addon_groups;
DROP TABLE IF EXISTS product_variants;
DROP TABLE IF EXISTS products;
DROP TABLE IF EXISTS categories;
-- +goose StatementEnd
```

- [ ] **Step 2: Commit**

```bash
git add services/pos/internal/infrastructure/postgres/migrations/
git commit -m "feat(pos/db): add products, categories, variants, addon_groups tables (migration 00001)"
```

---

### Task 5: Infrastructure — product_repo.go (with integration tests)

**Files:**
- Create: `services/pos/internal/infrastructure/postgres/product_repo.go`
- Create: `services/pos/internal/infrastructure/postgres/product_repo_test.go`
- Create: `services/pos/internal/infrastructure/postgres/testhelpers_test.go`

- [ ] **Step 1: Write failing integration tests**

Create `services/pos/internal/infrastructure/postgres/testhelpers_test.go`:

```go
//go:build integration

package postgres_test

import (
	"context"
	"embed"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	tc "github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/google/uuid"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func startTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:18.4-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tc.WithWaitStrategy(tcpostgres.WithInitialSQL("")),
	)
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// Run migrations via goose
	db, err := pool.Acquire(ctx)
	require.NoError(t, err)
	defer db.Release()

	provider, err := goose.NewProvider(goose.DialectPostgres, db.Conn(), migrationsFS)
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)

	return pool
}

func setRLS(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	_, err := pool.Exec(context.Background(),
		"SET SESSION app.current_tenant_id = $1", tenantID.String())
	require.NoError(t, err)
}
```

Create `services/pos/internal/infrastructure/postgres/product_repo_test.go`:

```go
//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/services/pos/internal/domain/product"
	"github.com/xyn-pos/services/pos/internal/infrastructure/postgres"
)

func TestProductRepo_CreateAndFindByID(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	p, _ := product.NewProduct(tenantID, nil, "NP-001", "Nasi Padang", "Desc", 25000_00, product.TaxTypePPN)
	require.NoError(t, repo.Save(ctx, p))

	found, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nasi Padang", found.Name)
	assert.Equal(t, int64(2500000), found.BasePrice)
}

func TestProductRepo_FindBySKU_UniquePerTenant(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	p, _ := product.NewProduct(tenantID, nil, "UNIQ-SKU", "Product A", "", 10000, product.TaxTypeNone)
	require.NoError(t, repo.Save(ctx, p))

	found, err := repo.FindBySKU(ctx, tenantID, "UNIQ-SKU")
	require.NoError(t, err)
	assert.Equal(t, p.ID, found.ID)
}

func TestProductRepo_Save_DuplicateSKU_ReturnsError(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	p1, _ := product.NewProduct(tenantID, nil, "DUPE-SKU", "P1", "", 5000, product.TaxTypeNone)
	p2, _ := product.NewProduct(tenantID, nil, "DUPE-SKU", "P2", "", 6000, product.TaxTypeNone)
	require.NoError(t, repo.Save(ctx, p1))
	err := repo.Save(ctx, p2)
	assert.ErrorIs(t, err, product.ErrSKUAlreadyExists)
}

func TestProductRepo_ListByCategory_RLSIsolation(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	ctx := context.Background()

	t1 := uuid.New()
	t2 := uuid.New()

	setRLS(t, pool, t1)
	p1, _ := product.NewProduct(t1, nil, "T1-001", "T1 Product", "", 1000, product.TaxTypeNone)
	require.NoError(t, repo.Save(ctx, p1))

	setRLS(t, pool, t2)
	p2, _ := product.NewProduct(t2, nil, "T2-001", "T2 Product", "", 2000, product.TaxTypeNone)
	require.NoError(t, repo.Save(ctx, p2))

	setRLS(t, pool, t1)
	products, _, err := repo.FindByTenant(ctx, t1, product.Filter{ActiveOnly: true, Limit: 10})
	require.NoError(t, err)
	assert.Len(t, products, 1)
	assert.Equal(t, "T1 Product", products[0].Name)
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd services/pos && go test ./internal/infrastructure/postgres/... -tags=integration -v -run TestProductRepo
```

Expected: FAIL (NewProductRepository not defined)

- [ ] **Step 3: Create product_repo.go**

```go
package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ProductRepository implements product.ProductRepository backed by PostgreSQL.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository constructs a ProductRepository.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	const q = `
		SELECT id, tenant_id, category_id, sku, name, description,
		       base_price, tax_type, is_active, created_at, updated_at
		FROM products WHERE id = $1`

	p, err := r.scanOne(ctx, q, id)
	if err != nil {
		return nil, err
	}

	// Load variants and addon groups
	if err := r.loadVariants(ctx, p); err != nil {
		return nil, err
	}
	if err := r.loadAddonGroups(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProductRepository) FindBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*product.Product, error) {
	const q = `
		SELECT id, tenant_id, category_id, sku, name, description,
		       base_price, tax_type, is_active, created_at, updated_at
		FROM products WHERE tenant_id = $1 AND sku = $2`

	p, err := r.scanOne(ctx, q, tenantID, sku)
	if err != nil {
		return nil, err
	}
	if err := r.loadVariants(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProductRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, filter product.Filter) ([]*product.Product, int, error) {
	where := "tenant_id = $1"
	args := []any{tenantID}
	if filter.ActiveOnly {
		where += " AND is_active = true"
	}
	if filter.CategoryID != nil {
		where += fmt.Sprintf(" AND category_id = $%d", len(args)+1)
		args = append(args, *filter.CategoryID)
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	var total int
	countQ := "SELECT COUNT(*) FROM products WHERE " + where
	_ = r.pool.QueryRow(ctx, countQ, args...).Scan(&total)

	args = append(args, limit, filter.Offset)
	q := fmt.Sprintf(`
		SELECT id, tenant_id, category_id, sku, name, description,
		       base_price, tax_type, is_active, created_at, updated_at
		FROM products WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("productRepo.FindByTenant: %w", err)
	}
	defer rows.Close()

	var result []*product.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("productRepo.FindByTenant scan: %w", err)
		}
		result = append(result, p)
	}
	return result, total, rows.Err()
}

func (r *ProductRepository) Save(ctx context.Context, p *product.Product) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("productRepo.Save begin: %w", err)
	}
	defer tx.Rollback(ctx)

	const q = `
		INSERT INTO products (id, tenant_id, category_id, sku, name, description,
		                      base_price, tax_type, is_active, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

	_, err = tx.Exec(ctx, q,
		p.ID, p.TenantID, p.CategoryID, nullStr(p.SKU), p.Name, p.Description,
		p.BasePrice, string(p.TaxType), p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("productRepo.Save: %w", product.ErrSKUAlreadyExists)
		}
		return fmt.Errorf("productRepo.Save: %w", err)
	}

	for _, v := range p.Variants {
		if err := insertVariant(ctx, tx, p.ID, v); err != nil {
			return err
		}
	}
	for _, ag := range p.AddonGroups {
		if err := insertAddonGroup(ctx, tx, p.ID, ag); err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (r *ProductRepository) Update(ctx context.Context, p *product.Product) error {
	const q = `
		UPDATE products SET name=$2, description=$3, base_price=$4, tax_type=$5,
		                    category_id=$6, is_active=$7, updated_at=$8
		WHERE id = $1`

	_, err := r.pool.Exec(ctx, q,
		p.ID, p.Name, p.Description, p.BasePrice, string(p.TaxType),
		p.CategoryID, p.IsActive, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("productRepo.Update: %w", err)
	}
	return nil
}

func (r *ProductRepository) SetBranchPrice(ctx context.Context, bp product.BranchProduct) error {
	const q = `
		INSERT INTO branch_products (branch_id, product_id, override_price, is_available, updated_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (branch_id, product_id) DO UPDATE
		SET override_price=$3, is_available=$4, updated_at=now()`

	_, err := r.pool.Exec(ctx, q, bp.BranchID, bp.ProductID, bp.OverridePrice, bp.IsAvailable)
	if err != nil {
		return fmt.Errorf("productRepo.SetBranchPrice: %w", err)
	}
	return nil
}

func (r *ProductRepository) HasActiveOrders(ctx context.Context, productID uuid.UUID) (bool, error) {
	var count int
	const q = `SELECT COUNT(*) FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.product_id = $1 AND o.status NOT IN ('paid','cancelled')`

	if err := r.pool.QueryRow(ctx, q, productID).Scan(&count); err != nil {
		return false, fmt.Errorf("productRepo.HasActiveOrders: %w", err)
	}
	return count > 0, nil
}

func (r *ProductRepository) scanOne(ctx context.Context, q string, args ...any) (*product.Product, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("productRepo query: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, product.ErrProductNotFound
	}
	p, err := scanProduct(rows)
	if err != nil {
		return nil, fmt.Errorf("productRepo scan: %w", err)
	}
	return p, nil
}

func (r *ProductRepository) loadVariants(ctx context.Context, p *product.Product) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, name, price_delta, COALESCE(sku,''), is_active FROM product_variants WHERE product_id = $1`,
		p.ID)
	if err != nil {
		return fmt.Errorf("productRepo.loadVariants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v product.Variant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.Name, &v.PriceDelta, &v.SKU, &v.IsActive); err != nil {
			return err
		}
		p.Variants = append(p.Variants, v)
	}
	return rows.Err()
}

func (r *ProductRepository) loadAddonGroups(ctx context.Context, p *product.Product) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, name, is_required, max_selections FROM addon_groups WHERE product_id = $1`,
		p.ID)
	if err != nil {
		return fmt.Errorf("productRepo.loadAddonGroups: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ag product.AddonGroup
		if err := rows.Scan(&ag.ID, &ag.ProductID, &ag.Name, &ag.IsRequired, &ag.MaxSelections); err != nil {
			return err
		}
		if err := r.loadAddons(ctx, &ag); err != nil {
			return err
		}
		p.AddonGroups = append(p.AddonGroups, ag)
	}
	return rows.Err()
}

func (r *ProductRepository) loadAddons(ctx context.Context, ag *product.AddonGroup) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id, addon_group_id, name, price, is_active FROM addons WHERE addon_group_id = $1`,
		ag.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var a product.Addon
		if err := rows.Scan(&a.ID, &a.AddonGroupID, &a.Name, &a.Price, &a.IsActive); err != nil {
			return err
		}
		ag.Addons = append(ag.Addons, a)
	}
	return rows.Err()
}

func scanProduct(rows pgx.Rows) (*product.Product, error) {
	var p product.Product
	var taxTypeStr string
	err := rows.Scan(
		&p.ID, &p.TenantID, &p.CategoryID, &p.SKU, &p.Name, &p.Description,
		&p.BasePrice, &taxTypeStr, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.TaxType = product.TaxType(taxTypeStr)
	return &p, nil
}

func insertVariant(ctx context.Context, tx pgx.Tx, productID uuid.UUID, v product.Variant) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO product_variants (id, product_id, name, price_delta, sku, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), productID, v.Name, v.PriceDelta, nullStr(v.SKU), v.IsActive)
	return err
}

func insertAddonGroup(ctx context.Context, tx pgx.Tx, productID uuid.UUID, ag product.AddonGroup) error {
	agID := uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO addon_groups (id, product_id, name, is_required, max_selections)
		 VALUES ($1, $2, $3, $4, $5)`,
		agID, productID, ag.Name, ag.IsRequired, ag.MaxSelections)
	if err != nil {
		return err
	}
	for _, a := range ag.Addons {
		_, err = tx.Exec(ctx,
			`INSERT INTO addons (id, addon_group_id, name, price, is_active)
			 VALUES ($1, $2, $3, $4, $5)`,
			uuid.New(), agID, a.Name, a.Price, a.IsActive)
		if err != nil {
			return err
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
```

- [ ] **Step 4: Run integration tests**

```bash
cd services/pos && go test ./internal/infrastructure/postgres/... -tags=integration -v -run TestProductRepo -count=1
```

Expected: all 4 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add services/pos/internal/infrastructure/postgres/
git commit -m "feat(pos/infra): add ProductRepository with RLS, SKU lookup, and branch price override"
```

---

### Task 6: Application Commands and Queries

**Files:**
- Create: `services/pos/internal/application/command/create_product.go`
- Create: `services/pos/internal/application/command/create_product_test.go`
- Create: `services/pos/internal/application/command/update_product.go`
- Create: `services/pos/internal/application/command/archive_product.go`
- Create: `services/pos/internal/application/command/create_category.go`
- Create: `services/pos/internal/application/command/reorder_categories.go`
- Create: `services/pos/internal/application/command/set_branch_price.go`
- Create: `services/pos/internal/application/query/get_product.go`
- Create: `services/pos/internal/application/query/list_products.go`
- Create: `services/pos/internal/application/query/lookup_by_sku.go`

- [ ] **Step 1: Write failing unit tests for CreateProduct**

Create `services/pos/internal/application/command/create_product_test.go`:

```go
package command_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/services/pos/internal/application/command"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

type MockProductRepo struct{ mock.Mock }

func (m *MockProductRepo) FindByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*product.Product), args.Error(1)
}
func (m *MockProductRepo) FindBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*product.Product, error) {
	args := m.Called(ctx, tenantID, sku)
	if args.Get(0) == nil { return nil, args.Error(1) }
	return args.Get(0).(*product.Product), args.Error(1)
}
func (m *MockProductRepo) FindByTenant(ctx context.Context, tenantID uuid.UUID, f product.Filter) ([]*product.Product, int, error) {
	args := m.Called(ctx, tenantID, f)
	return args.Get(0).([]*product.Product), args.Int(1), args.Error(2)
}
func (m *MockProductRepo) Save(ctx context.Context, p *product.Product) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockProductRepo) Update(ctx context.Context, p *product.Product) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockProductRepo) SetBranchPrice(ctx context.Context, bp product.BranchProduct) error {
	return m.Called(ctx, bp).Error(0)
}
func (m *MockProductRepo) HasActiveOrders(ctx context.Context, productID uuid.UUID) (bool, error) {
	args := m.Called(ctx, productID)
	return args.Bool(0), args.Error(1)
}

func TestCreateProduct_SKUAlreadyExists_ReturnsError(t *testing.T) {
	repo := &MockProductRepo{}
	tenantID := uuid.New()

	existing, _ := product.NewProduct(tenantID, nil, "DUPE", "Existing", "", 1000, product.TaxTypeNone)
	repo.On("FindBySKU", mock.Anything, tenantID, "DUPE").Return(existing, nil)

	h := command.NewCreateProductHandler(repo)
	_, err := h.Handle(context.Background(), command.CreateProductInput{
		TenantID: tenantID, SKU: "DUPE", Name: "New", BasePrice: 5000, TaxType: product.TaxTypeNone,
	})
	assert.ErrorIs(t, err, product.ErrSKUAlreadyExists)
}

func TestCreateProduct_ValidInput_SavesProduct(t *testing.T) {
	repo := &MockProductRepo{}
	tenantID := uuid.New()

	repo.On("FindBySKU", mock.Anything, tenantID, "NEW-001").Return(nil, product.ErrProductNotFound)
	repo.On("Save", mock.Anything, mock.MatchedBy(func(p *product.Product) bool {
		return p.Name == "Ayam Goreng" && p.BasePrice == 20000_00
	})).Return(nil)

	h := command.NewCreateProductHandler(repo)
	p, err := h.Handle(context.Background(), command.CreateProductInput{
		TenantID:  tenantID,
		SKU:       "NEW-001",
		Name:      "Ayam Goreng",
		BasePrice: 20000_00,
		TaxType:   product.TaxTypePPN,
	})
	require.NoError(t, err)
	assert.Equal(t, "Ayam Goreng", p.Name)
}

func TestArchiveProduct_HasActiveOrders_ReturnsError(t *testing.T) {
	repo := &MockProductRepo{}
	productID := uuid.New()
	p, _ := product.NewProduct(uuid.New(), nil, "ARCH-001", "Active Product", "", 5000, product.TaxTypeNone)
	p.ID = productID

	repo.On("FindByID", mock.Anything, productID).Return(p, nil)
	repo.On("HasActiveOrders", mock.Anything, productID).Return(true, nil)

	h := command.NewArchiveProductHandler(repo)
	err := h.Handle(context.Background(), productID)
	assert.ErrorIs(t, err, product.ErrProductHasActiveOrders)
}
```

- [ ] **Step 2: Run to verify they fail**

```bash
cd services/pos && go test ./internal/application/command/... -run TestCreate -v
```

Expected: FAIL

- [ ] **Step 3: Create create_product.go**

```go
package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// CreateProductInput is the command payload.
type CreateProductInput struct {
	TenantID      uuid.UUID
	CategoryID    *uuid.UUID
	SKU           string
	Name          string
	Description   string
	BasePrice     int64
	TaxType       product.TaxType
	Variants      []product.Variant
	AddonGroups   []product.AddonGroup
	IdempotencyKey string
}

// CreateProductHandler orchestrates product creation.
type CreateProductHandler struct {
	repo product.ProductRepository
}

// NewCreateProductHandler creates a handler.
func NewCreateProductHandler(repo product.ProductRepository) *CreateProductHandler {
	return &CreateProductHandler{repo: repo}
}

// Handle creates a new product. Validates SKU uniqueness.
func (h *CreateProductHandler) Handle(ctx context.Context, in CreateProductInput) (*product.Product, error) {
	// SKU uniqueness check
	if in.SKU != "" {
		_, err := h.repo.FindBySKU(ctx, in.TenantID, in.SKU)
		if err == nil {
			return nil, fmt.Errorf("CreateProduct: %w", product.ErrSKUAlreadyExists)
		}
		if !errors.Is(err, product.ErrProductNotFound) {
			return nil, fmt.Errorf("CreateProduct FindBySKU: %w", err)
		}
	}

	p, err := product.NewProduct(in.TenantID, in.CategoryID, in.SKU, in.Name, in.Description, in.BasePrice, in.TaxType)
	if err != nil {
		return nil, fmt.Errorf("CreateProduct domain: %w", err)
	}

	for _, v := range in.Variants {
		if err := p.AddVariant(v); err != nil {
			return nil, fmt.Errorf("CreateProduct AddVariant: %w", err)
		}
	}
	for _, ag := range in.AddonGroups {
		if err := p.AddAddonGroup(ag); err != nil {
			return nil, fmt.Errorf("CreateProduct AddAddonGroup: %w", err)
		}
	}

	if err := h.repo.Save(ctx, p); err != nil {
		return nil, fmt.Errorf("CreateProduct save: %w", err)
	}
	return p, nil
}
```

- [ ] **Step 4: Create archive_product.go**

```go
package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ArchiveProductHandler soft-deletes a product.
type ArchiveProductHandler struct {
	repo product.ProductRepository
}

// NewArchiveProductHandler creates a handler.
func NewArchiveProductHandler(repo product.ProductRepository) *ArchiveProductHandler {
	return &ArchiveProductHandler{repo: repo}
}

// Handle archives a product. Blocked if product has active orders.
func (h *ArchiveProductHandler) Handle(ctx context.Context, productID uuid.UUID) error {
	p, err := h.repo.FindByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("ArchiveProduct FindByID: %w", err)
	}

	hasOrders, err := h.repo.HasActiveOrders(ctx, productID)
	if err != nil {
		return fmt.Errorf("ArchiveProduct HasActiveOrders: %w", err)
	}
	if hasOrders {
		return fmt.Errorf("ArchiveProduct: %w", product.ErrProductHasActiveOrders)
	}

	if err := p.Archive(); err != nil {
		return fmt.Errorf("ArchiveProduct archive: %w", err)
	}

	if err := h.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("ArchiveProduct update: %w", err)
	}
	return nil
}
```

- [ ] **Step 5: Create update_product.go, create_category.go, reorder_categories.go, set_branch_price.go**

Create `services/pos/internal/application/command/update_product.go`:

```go
package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// UpdateProductInput is the command payload.
type UpdateProductInput struct {
	ProductID   uuid.UUID
	Name        string
	Description string
	BasePrice   int64
	TaxType     product.TaxType
	CategoryID  *uuid.UUID
}

// UpdateProductHandler updates a product's core fields.
type UpdateProductHandler struct {
	repo product.ProductRepository
}

// NewUpdateProductHandler creates a handler.
func NewUpdateProductHandler(repo product.ProductRepository) *UpdateProductHandler {
	return &UpdateProductHandler{repo: repo}
}

// Handle updates a product's mutable fields.
func (h *UpdateProductHandler) Handle(ctx context.Context, in UpdateProductInput) (*product.Product, error) {
	p, err := h.repo.FindByID(ctx, in.ProductID)
	if err != nil {
		return nil, fmt.Errorf("UpdateProduct FindByID: %w", err)
	}
	if in.Name != "" {
		p.Name = in.Name
	}
	if in.Description != "" {
		p.Description = in.Description
	}
	if in.BasePrice >= 0 {
		p.BasePrice = in.BasePrice
	}
	if in.TaxType != "" {
		p.TaxType = in.TaxType
	}
	if in.CategoryID != nil {
		p.CategoryID = in.CategoryID
	}
	p.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(ctx, p); err != nil {
		return nil, fmt.Errorf("UpdateProduct update: %w", err)
	}
	return p, nil
}
```

Create `services/pos/internal/application/command/create_category.go`:

```go
package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// CreateCategoryInput is the command payload.
type CreateCategoryInput struct {
	TenantID  uuid.UUID
	Name      string
	SortOrder int
	ParentID  *uuid.UUID
}

// CreateCategoryHandler creates a product category.
type CreateCategoryHandler struct {
	repo product.CategoryRepository
}

// NewCreateCategoryHandler creates a handler.
func NewCreateCategoryHandler(repo product.CategoryRepository) *CreateCategoryHandler {
	return &CreateCategoryHandler{repo: repo}
}

// Handle creates and persists a category.
func (h *CreateCategoryHandler) Handle(ctx context.Context, in CreateCategoryInput) (*product.Category, error) {
	now := time.Now().UTC()
	c := &product.Category{
		ID:        uuid.New(),
		TenantID:  in.TenantID,
		Name:      in.Name,
		SortOrder: in.SortOrder,
		ParentID:  in.ParentID,
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := h.repo.Save(ctx, c); err != nil {
		return nil, fmt.Errorf("CreateCategory save: %w", err)
	}
	return c, nil
}
```

Create `services/pos/internal/application/command/reorder_categories.go`:

```go
package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ReorderCategoriesHandler updates sort_order for a list of categories.
type ReorderCategoriesHandler struct {
	repo product.CategoryRepository
}

// NewReorderCategoriesHandler creates a handler.
func NewReorderCategoriesHandler(repo product.CategoryRepository) *ReorderCategoriesHandler {
	return &ReorderCategoriesHandler{repo: repo}
}

// Handle applies the ordered list of IDs as sort_order 0, 1, 2, ...
func (h *ReorderCategoriesHandler) Handle(ctx context.Context, tenantID uuid.UUID, orderedIDs []uuid.UUID) error {
	if err := h.repo.UpdateSortOrders(ctx, tenantID, orderedIDs); err != nil {
		return fmt.Errorf("ReorderCategories: %w", err)
	}
	return nil
}
```

Create `services/pos/internal/application/command/set_branch_price.go`:

```go
package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// SetBranchPriceInput is the command payload.
type SetBranchPriceInput struct {
	BranchID      uuid.UUID
	ProductID     uuid.UUID
	OverridePrice *int64 // nil = remove override
	IsAvailable   bool
}

// SetBranchPriceHandler sets a branch-specific price override.
type SetBranchPriceHandler struct {
	repo product.ProductRepository
}

// NewSetBranchPriceHandler creates a handler.
func NewSetBranchPriceHandler(repo product.ProductRepository) *SetBranchPriceHandler {
	return &SetBranchPriceHandler{repo: repo}
}

// Handle upserts a branch_products record.
func (h *SetBranchPriceHandler) Handle(ctx context.Context, in SetBranchPriceInput) error {
	if in.OverridePrice != nil && *in.OverridePrice < 0 {
		return fmt.Errorf("SetBranchPrice: override_price must be >= 0")
	}
	bp := product.BranchProduct{
		BranchID:      in.BranchID,
		ProductID:     in.ProductID,
		OverridePrice: in.OverridePrice,
		IsAvailable:   in.IsAvailable,
	}
	if err := h.repo.SetBranchPrice(ctx, bp); err != nil {
		return fmt.Errorf("SetBranchPrice: %w", err)
	}
	return nil
}
```

- [ ] **Step 6: Create queries**

Create `services/pos/internal/application/query/get_product.go`:

```go
package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// GetProductHandler retrieves a single product by ID.
type GetProductHandler struct {
	repo product.ProductRepository
}

// NewGetProductHandler creates a handler.
func NewGetProductHandler(repo product.ProductRepository) *GetProductHandler {
	return &GetProductHandler{repo: repo}
}

// Handle returns the product or ErrProductNotFound.
func (h *GetProductHandler) Handle(ctx context.Context, productID uuid.UUID) (*product.Product, error) {
	p, err := h.repo.FindByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("GetProduct: %w", err)
	}
	return p, nil
}
```

Create `services/pos/internal/application/query/list_products.go`:

```go
package query

import (
	"context"
	"fmt"

	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ListProductsResult holds products and total count.
type ListProductsResult struct {
	Products []*product.Product
	Total    int
}

// ListProductsHandler lists products with optional filters.
type ListProductsHandler struct {
	repo product.ProductRepository
}

// NewListProductsHandler creates a handler.
func NewListProductsHandler(repo product.ProductRepository) *ListProductsHandler {
	return &ListProductsHandler{repo: repo}
}

// Handle returns filtered products for a tenant.
func (h *ListProductsHandler) Handle(ctx context.Context, filter product.Filter, tenantID interface{ String() string }) (*ListProductsResult, error) {
	id, err := parseUUID(tenantID)
	if err != nil {
		return nil, err
	}
	products, total, err := h.repo.FindByTenant(ctx, id, filter)
	if err != nil {
		return nil, fmt.Errorf("ListProducts: %w", err)
	}
	return &ListProductsResult{Products: products, Total: total}, nil
}
```

Create `services/pos/internal/application/query/lookup_by_sku.go`:

```go
package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// LookupBySKUHandler finds a product by SKU (barcode scanner endpoint).
type LookupBySKUHandler struct {
	repo product.ProductRepository
}

// NewLookupBySKUHandler creates a handler.
func NewLookupBySKUHandler(repo product.ProductRepository) *LookupBySKUHandler {
	return &LookupBySKUHandler{repo: repo}
}

// Handle returns the product for the given tenant and SKU.
func (h *LookupBySKUHandler) Handle(ctx context.Context, tenantID uuid.UUID, sku string) (*product.Product, error) {
	if sku == "" {
		return nil, fmt.Errorf("LookupBySKU: SKU cannot be empty")
	}
	p, err := h.repo.FindBySKU(ctx, tenantID, sku)
	if err != nil {
		return nil, fmt.Errorf("LookupBySKU: %w", err)
	}
	return p, nil
}

func parseUUID(v interface{ String() string }) (uuid.UUID, error) {
	return uuid.Parse(v.String())
}
```

- [ ] **Step 7: Run all application tests**

```bash
cd services/pos && go test ./internal/application/... -v -count=1
```

Expected: TestCreateProduct_* and TestArchiveProduct_* PASS.

- [ ] **Step 8: Commit**

```bash
git add services/pos/internal/application/
git commit -m "feat(pos/app): add CreateProduct, UpdateProduct, Archive, Category, and query handlers"
```

---

### Task 7: Interface — product_handler.go and provider.go

**Files:**
- Create: `services/pos/internal/interfaces/grpc/product_handler.go`
- Create: `services/pos/provider.go`
- Create: `services/pos/config.go`

- [ ] **Step 1: Create product_handler.go**

```go
package grpc

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	posv1 "github.com/xyn-pos/gen/pos/v1"
	"github.com/xyn-pos/services/pos/internal/application/command"
	"github.com/xyn-pos/services/pos/internal/application/query"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ProductHandler implements posv1.ProductServiceServer.
type ProductHandler struct {
	posv1.UnimplementedProductServiceServer
	createProductH    *command.CreateProductHandler
	updateProductH    *command.UpdateProductHandler
	archiveProductH   *command.ArchiveProductHandler
	createCategoryH   *command.CreateCategoryHandler
	reorderCategoryH  *command.ReorderCategoriesHandler
	setBranchPriceH   *command.SetBranchPriceHandler
	getProductH       *query.GetProductHandler
	listProductsH     *query.ListProductsHandler
	lookupBySKUH      *query.LookupBySKUHandler
}

// NewProductHandler assembles the handler.
func NewProductHandler(
	createH *command.CreateProductHandler,
	updateH *command.UpdateProductHandler,
	archiveH *command.ArchiveProductHandler,
	createCatH *command.CreateCategoryHandler,
	reorderCatH *command.ReorderCategoriesHandler,
	setBranchH *command.SetBranchPriceHandler,
	getH *query.GetProductHandler,
	listH *query.ListProductsHandler,
	lookupH *query.LookupBySKUHandler,
) *ProductHandler {
	return &ProductHandler{
		createProductH:   createH,
		updateProductH:   updateH,
		archiveProductH:  archiveH,
		createCategoryH:  createCatH,
		reorderCategoryH: reorderCatH,
		setBranchPriceH:  setBranchH,
		getProductH:      getH,
		listProductsH:    listH,
		lookupBySKUH:     lookupH,
	}
}

func (h *ProductHandler) CreateProduct(ctx context.Context, req *posv1.CreateProductRequest) (*posv1.CreateProductResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}

	p, err := h.createProductH.Handle(ctx, command.CreateProductInput{
		TenantID:    tenantID,
		SKU:         req.Sku,
		Name:        req.Name,
		Description: req.Description,
		BasePrice:   req.BasePrice,
		TaxType:     protoTaxTypeToDomain(req.TaxType),
	})
	if err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.CreateProductResponse{Product: domainProductToProto(p)}, nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, req *posv1.GetProductRequest) (*posv1.GetProductResponse, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	p, err := h.getProductH.Handle(ctx, id)
	if err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.GetProductResponse{Product: domainProductToProto(p)}, nil
}

func (h *ProductHandler) LookupBySKU(ctx context.Context, req *posv1.LookupBySKURequest) (*posv1.LookupBySKUResponse, error) {
	tenantID, err := uuid.Parse(req.TenantId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid tenant_id")
	}
	p, err := h.lookupBySKUH.Handle(ctx, tenantID, req.Sku)
	if err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.LookupBySKUResponse{Product: domainProductToProto(p)}, nil
}

func (h *ProductHandler) ArchiveProduct(ctx context.Context, req *posv1.ArchiveProductRequest) (*posv1.ArchiveProductResponse, error) {
	id, err := uuid.Parse(req.ProductId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid product_id")
	}
	if err := h.archiveProductH.Handle(ctx, id); err != nil {
		return nil, mapProductError(err)
	}
	return &posv1.ArchiveProductResponse{}, nil
}

func domainProductToProto(p *product.Product) *posv1.Product {
	proto := &posv1.Product{
		Id:          p.ID.String(),
		TenantId:    p.TenantID.String(),
		Sku:         p.SKU,
		Name:        p.Name,
		Description: p.Description,
		BasePrice:   p.BasePrice,
		TaxType:     domainTaxTypeToProto(p.TaxType),
		IsActive:    p.IsActive,
		CreatedAt:   p.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:   p.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if p.CategoryID != nil {
		proto.CategoryId = p.CategoryID.String()
	}
	return proto
}

func protoTaxTypeToDomain(t posv1.TaxType) product.TaxType {
	switch t {
	case posv1.TaxType_TAX_TYPE_PPN:
		return product.TaxTypePPN
	case posv1.TaxType_TAX_TYPE_PB1:
		return product.TaxTypePB1
	default:
		return product.TaxTypeNone
	}
}

func domainTaxTypeToProto(t product.TaxType) posv1.TaxType {
	switch t {
	case product.TaxTypePPN:
		return posv1.TaxType_TAX_TYPE_PPN
	case product.TaxTypePB1:
		return posv1.TaxType_TAX_TYPE_PB1
	default:
		return posv1.TaxType_TAX_TYPE_NONE
	}
}

func mapProductError(err error) error {
	switch {
	case errors.Is(err, product.ErrProductNotFound):
		return status.Error(codes.NotFound, "product not found")
	case errors.Is(err, product.ErrSKUAlreadyExists):
		return status.Error(codes.AlreadyExists, "SKU already exists")
	case errors.Is(err, product.ErrProductHasActiveOrders):
		return status.Error(codes.FailedPrecondition, "product has active orders")
	case errors.Is(err, product.ErrInvalidPrice):
		return status.Error(codes.InvalidArgument, "base price must be >= 0")
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
```

- [ ] **Step 2: Create config.go**

```go
package pos

// Config holds all configuration for the pos service.
type Config struct {
	ServiceName string
	Version     string
	Env         string
	GRPCPort    string
	DatabaseURL string
	OTLPEndpoint string
}
```

- [ ] **Step 3: Create initial provider.go**

```go
package pos

import (
	"context"
	"fmt"

	"github.com/xyn-pos/services/pos/internal/application/command"
	"github.com/xyn-pos/services/pos/internal/application/query"
	"github.com/xyn-pos/services/pos/internal/infrastructure/postgres"
	grpchandler "github.com/xyn-pos/services/pos/internal/interfaces/grpc"
	shareddb "github.com/xyn-pos/shared/pkg/database"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
)

// App is the assembled pos service.
type App struct {
	grpc     *grpchandler.Server
	shutdown func()
}

// New builds the dependency graph for the pos service.
func New(ctx context.Context, cfg *Config) (*App, error) {
	shutdownTelemetry, err := sharedtelemetry.Setup(ctx, sharedtelemetry.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.Version,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		Environment:    cfg.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("pos.New telemetry: %w", err)
	}

	pool, err := shareddb.NewPool(ctx, shareddb.Config{DSN: cfg.DatabaseURL})
	if err != nil {
		shutdownTelemetry()
		return nil, fmt.Errorf("pos.New database: %w", err)
	}

	// Product domain
	productRepo := postgres.NewProductRepository(pool)
	createProductH := command.NewCreateProductHandler(productRepo)
	updateProductH := command.NewUpdateProductHandler(productRepo)
	archiveProductH := command.NewArchiveProductHandler(productRepo)
	setBranchPriceH := command.NewSetBranchPriceHandler(productRepo)
	getProductH := query.NewGetProductHandler(productRepo)
	listProductsH := query.NewListProductsHandler(productRepo)
	lookupBySKUH := query.NewLookupBySKUHandler(productRepo)

	productHandler := grpchandler.NewProductHandler(
		createProductH, updateProductH, archiveProductH,
		nil, nil, // category handlers wired below
		setBranchPriceH,
		getProductH, listProductsH, lookupBySKUH,
	)

	grpcSrv := grpchandler.NewServer(cfg.GRPCPort, productHandler, nil)

	return &App{
		grpc: grpcSrv,
		shutdown: func() {
			pool.Close()
			shutdownTelemetry()
		},
	}, nil
}

// Run starts the gRPC server (blocking).
func (a *App) Run() error {
	return a.grpc.Serve()
}

// Shutdown releases resources.
func (a *App) Shutdown() {
	a.shutdown()
}
```

Note: `grpchandler.NewServer` and `grpchandler.Server` need to exist. Create `services/pos/internal/interfaces/grpc/server.go` following the same pattern as the tenant service's `server.go`.

- [ ] **Step 4: Build check**

```bash
cd services/pos && go build ./...
```

Expected: no errors.

- [ ] **Step 5: Run all tests**

```bash
cd services/pos && go test ./... -count=1
```

Expected: all unit tests PASS; integration tests skipped (no build tag).

- [ ] **Step 6: Lint**

```bash
golangci-lint run ./services/pos/...
```

Expected: no issues.

- [ ] **Step 7: Commit**

```bash
git add services/pos/
git commit -m "feat(pos): Epic 4.2 complete — ProductService with catalog, variants, SKU lookup"
```

---

## Self-Review

- Spec §5 coverage:
  - Product aggregate with variants and addon groups ✅
  - TaxType: ppn | pb1 | none ✅
  - SKU unique per tenant (domain + DB constraint) ✅
  - Soft archive (IsActive = false, blocked by active orders) ✅
  - BranchProduct price override ✅
  - Category with sort_order and parent_id ✅
  - DB migration: all 6 tables (categories, products, variants, addon_groups, addons, branch_products) ✅
  - RLS on categories and products ✅
- Tests:
  - Unit: 7 domain tests (price, name, variant final price, addon group, archive) ✅
  - Application: 3 command tests (create, duplicate SKU, archive with orders) ✅
  - Integration: 4 repo tests (create, SKU lookup, duplicate SKU, RLS isolation) ✅
- No placeholder code ✅
- Layer isolation: domain imports only stdlib ✅
