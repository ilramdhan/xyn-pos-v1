-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS branches (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID        NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name        TEXT        NOT NULL,
    street      TEXT        NOT NULL DEFAULT '',
    city        TEXT        NOT NULL DEFAULT '',
    province    TEXT        NOT NULL DEFAULT '',
    postal_code TEXT        NOT NULL DEFAULT '',
    country     TEXT        NOT NULL DEFAULT 'ID',
    timezone    TEXT        NOT NULL DEFAULT 'Asia/Jakarta',
    is_active   BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE branches ENABLE ROW LEVEL SECURITY;

CREATE POLICY branches_isolation ON branches
    USING (tenant_id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY branches_insert ON branches
    FOR INSERT WITH CHECK (TRUE);

CREATE INDEX IF NOT EXISTS branches_tenant_idx ON branches (tenant_id);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS branches CASCADE;
-- +goose StatementEnd
