-- +goose Up
-- +goose StatementBegin
CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id     UUID         NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    keycloak_id   VARCHAR(36)  NOT NULL UNIQUE,
    email         VARCHAR(255) NOT NULL,
    full_name     VARCHAR(255) NOT NULL,
    role          VARCHAR(30)  NOT NULL
                  CHECK (role IN ('owner','manager','cashier','kitchen_staff')),
    branch_scope  UUID[]       NULL,
    is_active     BOOLEAN      NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, email)
);

CREATE TABLE user_pins (
    user_id    UUID  PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    pin_hash   TEXT  NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_users_tenant_id   ON users (tenant_id);
CREATE INDEX idx_users_keycloak_id ON users (keycloak_id);

ALTER TABLE users     ENABLE ROW LEVEL SECURITY;
ALTER TABLE user_pins ENABLE ROW LEVEL SECURITY;

CREATE POLICY users_select ON users FOR SELECT
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY users_insert ON users FOR INSERT
    WITH CHECK (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY users_update ON users FOR UPDATE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);
CREATE POLICY users_delete ON users FOR DELETE
    USING (tenant_id = current_setting('app.current_tenant_id', true)::UUID);

CREATE POLICY user_pins_select ON user_pins FOR SELECT USING (true);
CREATE POLICY user_pins_insert ON user_pins FOR INSERT WITH CHECK (true);
CREATE POLICY user_pins_update ON user_pins FOR UPDATE USING (true);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_pins;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
