-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS receipts (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id     UUID        NOT NULL,
    order_id       UUID        NOT NULL,
    tenant_id      UUID        NOT NULL,
    receipt_number TEXT        NOT NULL,
    amount         BIGINT      NOT NULL CHECK (amount > 0),
    method         TEXT        NOT NULL
                               CHECK (method IN ('qris','bank_transfer','credit_card','cash')),
    issued_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (payment_id),
    UNIQUE (tenant_id, receipt_number)
);

CREATE INDEX IF NOT EXISTS idx_receipts_order_id  ON receipts (order_id);
CREATE INDEX IF NOT EXISTS idx_receipts_tenant_id ON receipts (tenant_id);

ALTER TABLE receipts ENABLE ROW LEVEL SECURITY;

CREATE POLICY receipts_tenant_isolation ON receipts
    USING (tenant_id = (current_setting('app.current_tenant_id', true)::UUID));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP POLICY IF EXISTS receipts_tenant_isolation ON receipts;
DROP TABLE IF EXISTS receipts;
-- +goose StatementEnd
