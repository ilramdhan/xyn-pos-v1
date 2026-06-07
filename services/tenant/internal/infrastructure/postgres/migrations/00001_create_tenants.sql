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

-- Tenants is a global registry: any app role may read all rows
-- (slug uniqueness checks and cross-tenant lookups require unrestricted SELECT).
CREATE POLICY tenants_select ON tenants
    FOR SELECT USING (TRUE);

-- Registration: any app role may insert a new tenant row.
CREATE POLICY tenants_insert ON tenants
    FOR INSERT WITH CHECK (TRUE);

-- A tenant may only update or delete its own row.
CREATE POLICY tenants_self_update ON tenants
    FOR UPDATE USING (id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE POLICY tenants_self_delete ON tenants
    FOR DELETE USING (id = current_setting('app.current_tenant_id', TRUE)::UUID);

CREATE INDEX IF NOT EXISTS tenants_slug_idx ON tenants (slug);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS tenants CASCADE;
-- +goose StatementEnd
