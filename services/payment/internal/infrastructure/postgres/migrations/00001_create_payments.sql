-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS payments (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID        NOT NULL,
    order_id          UUID        NOT NULL,
    amount            BIGINT      NOT NULL CHECK (amount > 0),
    status            TEXT        NOT NULL DEFAULT 'pending'
                                  CHECK (status IN ('pending','success','failed','voided')),
    method            TEXT        NOT NULL
                                  CHECK (method IN ('qris','bank_transfer','credit_card','cash')),
    external_id       TEXT        NOT NULL DEFAULT '',
    snap_token        TEXT        NOT NULL DEFAULT '',
    snap_redirect_url TEXT        NOT NULL DEFAULT '',
    idempotency_key   TEXT        NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (tenant_id, idempotency_key)
);

-- Index for order lookup
CREATE INDEX IF NOT EXISTS idx_payments_order_id  ON payments (order_id);
CREATE INDEX IF NOT EXISTS idx_payments_tenant_id ON payments (tenant_id);

-- Row-Level Security
ALTER TABLE payments ENABLE ROW LEVEL SECURITY;

CREATE POLICY payments_tenant_isolation ON payments
    USING (tenant_id = (current_setting('app.current_tenant_id', true)::UUID));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP POLICY IF EXISTS payments_tenant_isolation ON payments;
DROP TABLE IF EXISTS payments;
-- +goose StatementEnd
