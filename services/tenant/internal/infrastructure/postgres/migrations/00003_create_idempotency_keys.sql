-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS idempotency_keys (
    key         TEXT        NOT NULL,
    tenant_id   UUID        NOT NULL,
    response    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at  TIMESTAMPTZ NOT NULL DEFAULT NOW() + INTERVAL '24 hours',
    PRIMARY KEY (key, tenant_id)
);

ALTER TABLE idempotency_keys ENABLE ROW LEVEL SECURITY;

CREATE POLICY idempotency_keys_isolation ON idempotency_keys
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY idempotency_keys_insert ON idempotency_keys
    FOR INSERT WITH CHECK (TRUE);

CREATE INDEX IF NOT EXISTS idempotency_keys_expires_idx ON idempotency_keys (expires_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS idempotency_keys CASCADE;
-- +goose StatementEnd
