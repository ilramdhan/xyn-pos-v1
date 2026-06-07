-- +goose Up
-- +goose StatementBegin
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
    UNIQUE (branch_id, cashier_id, status)
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
    product_name  VARCHAR(255) NOT NULL,
    variant_name  VARCHAR(255),
    unit_price    BIGINT       NOT NULL,
    quantity      INT          NOT NULL CHECK (quantity > 0),
    subtotal      BIGINT       NOT NULL,
    notes         TEXT
);

CREATE TABLE order_item_addons (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    order_item_id UUID         NOT NULL REFERENCES order_items(id) ON DELETE CASCADE,
    addon_id      UUID         NOT NULL,
    addon_name    VARCHAR(255) NOT NULL,
    price         BIGINT       NOT NULL
);

CREATE INDEX idx_orders_tenant_id ON orders(tenant_id);
CREATE INDEX idx_orders_shift_id  ON orders(shift_id);
CREATE INDEX idx_orders_status    ON orders(tenant_id, status);
CREATE INDEX idx_shifts_tenant_id ON shifts(tenant_id);

ALTER TABLE orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE shifts ENABLE ROW LEVEL SECURITY;

CREATE POLICY orders_isolation ON orders
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY shifts_isolation ON shifts
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS order_item_addons;
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS shifts;
-- +goose StatementEnd
