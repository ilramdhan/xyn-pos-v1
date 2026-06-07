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
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id      UUID         NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    is_required     BOOLEAN      NOT NULL DEFAULT false,
    max_selections  INT          NOT NULL DEFAULT 1 CHECK (max_selections >= 1)
);

CREATE TABLE addons (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    addon_group_id UUID         NOT NULL REFERENCES addon_groups(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    price          BIGINT       NOT NULL DEFAULT 0 CHECK (price >= 0),
    is_active      BOOLEAN      NOT NULL DEFAULT true
);

-- branch_products: branch-level price overrides. No tenant_id needed — tenant isolation
-- is enforced via the products table (which has RLS). branch_id refers to a branch
-- record in the tenant service; FK not enforced here to avoid cross-service coupling.
CREATE TABLE branch_products (
    branch_id      UUID        NOT NULL,
    product_id     UUID        NOT NULL REFERENCES products(id) ON DELETE CASCADE,
    override_price BIGINT,                  -- NULL = use products.base_price
    is_available   BOOLEAN     NOT NULL DEFAULT true,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (branch_id, product_id)
);

CREATE INDEX idx_products_tenant_id   ON products(tenant_id);
CREATE INDEX idx_products_category_id ON products(category_id);
-- Partial index: NULL skus are excluded, so two NULL-sku products don't conflict
CREATE INDEX idx_products_sku         ON products(tenant_id, sku) WHERE sku IS NOT NULL;
CREATE INDEX idx_categories_tenant_id ON categories(tenant_id);

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
