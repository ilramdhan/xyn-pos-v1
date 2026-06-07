-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS stock_ledgers (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID        NOT NULL,
    branch_id           UUID        NOT NULL,
    product_id          UUID        NOT NULL,
    variant_id          UUID,
    quantity            BIGINT      NOT NULL DEFAULT 0,
    unit                TEXT        NOT NULL DEFAULT 'pcs',
    low_stock_threshold BIGINT      NOT NULL DEFAULT 0,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS stock_movements (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID        NOT NULL,
    branch_id    UUID        NOT NULL,
    product_id   UUID        NOT NULL,
    variant_id   UUID,
    delta        BIGINT      NOT NULL,
    type         TEXT        NOT NULL CHECK (type IN ('in','out','adjustment')),
    reference_id TEXT        NOT NULL DEFAULT '',
    note         TEXT        NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS bom_recipes (
    id                      UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID    NOT NULL,
    product_id              UUID    NOT NULL,
    ingredient_product_id   UUID    NOT NULL,
    quantity_per_unit       BIGINT  NOT NULL CHECK (quantity_per_unit > 0),
    unit                    TEXT    NOT NULL DEFAULT 'pcs',

    UNIQUE (tenant_id, product_id, ingredient_product_id)
);

-- Partial unique indexes for stock_ledgers to handle nullable variant_id correctly.
-- PostgreSQL treats NULLs as distinct in standard UNIQUE constraints, so two rows with
-- the same (tenant_id, branch_id, product_id) and variant_id IS NULL would both be allowed.
-- These two partial indexes enforce the correct uniqueness semantics.
CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_ledgers_unique_no_variant
    ON stock_ledgers (tenant_id, branch_id, product_id)
    WHERE variant_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_stock_ledgers_unique_with_variant
    ON stock_ledgers (tenant_id, branch_id, product_id, variant_id)
    WHERE variant_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_stock_ledgers_branch ON stock_ledgers (tenant_id, branch_id);
CREATE INDEX IF NOT EXISTS idx_stock_movements_product ON stock_movements (tenant_id, product_id);
CREATE INDEX IF NOT EXISTS idx_bom_recipes_product ON bom_recipes (tenant_id, product_id);

-- RLS
ALTER TABLE stock_ledgers ENABLE ROW LEVEL SECURITY;
ALTER TABLE stock_movements ENABLE ROW LEVEL SECURITY;
ALTER TABLE bom_recipes ENABLE ROW LEVEL SECURITY;

CREATE POLICY stock_ledgers_tenant_isolation ON stock_ledgers
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY stock_movements_tenant_isolation ON stock_movements
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY bom_recipes_tenant_isolation ON bom_recipes
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP POLICY IF EXISTS bom_recipes_tenant_isolation ON bom_recipes;
DROP POLICY IF EXISTS stock_movements_tenant_isolation ON stock_movements;
DROP POLICY IF EXISTS stock_ledgers_tenant_isolation ON stock_ledgers;
DROP TABLE IF EXISTS bom_recipes;
DROP TABLE IF EXISTS stock_movements;
DROP TABLE IF EXISTS stock_ledgers;
-- +goose StatementEnd
