-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS tenants (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    slug        TEXT        NOT NULL,
    plan        TEXT        NOT NULL DEFAULT 'free',
    status      TEXT        NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT tenants_slug_unique UNIQUE (slug),
    CONSTRAINT tenants_plan_check CHECK (plan IN ('free', 'growth', 'enterprise')),
    CONSTRAINT tenants_status_check CHECK (status IN ('active', 'suspended', 'deleted'))
);

ALTER TABLE tenants ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenants_isolation ON tenants
    USING (id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY tenants_insert ON tenants
    FOR INSERT WITH CHECK (TRUE);

CREATE INDEX IF NOT EXISTS tenants_slug_idx ON tenants (slug);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tenants CASCADE;
-- +goose StatementEnd
