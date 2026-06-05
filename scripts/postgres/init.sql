-- PostgreSQL 18.4 — Database initialization script
-- Creates databases, roles, and RLS setup for xyn-pos-v1
-- This file runs on first container start via docker-entrypoint-initdb.d

-- ─────────────────────────────────────────────────────────
-- Application role (RLS enforced — used by all services)
-- ─────────────────────────────────────────────────────────
CREATE ROLE app_user WITH LOGIN PASSWORD 'xyn_app_password' NOINHERIT;

-- Migration role (bypasses RLS — used by Goose only)
CREATE ROLE migration_user WITH LOGIN PASSWORD 'xyn_migration_password' NOINHERIT BYPASSRLS;

-- ─────────────────────────────────────────────────────────
-- Databases (one per bounded context)
-- ─────────────────────────────────────────────────────────
CREATE DATABASE xyn_tenant;
CREATE DATABASE xyn_pos;
CREATE DATABASE xyn_payment;
CREATE DATABASE xyn_inventory;
CREATE DATABASE xyn_kitchen;
CREATE DATABASE xyn_analytics;

-- Keycloak gets its own DB
CREATE DATABASE keycloak;

-- ─────────────────────────────────────────────────────────
-- Grant privileges per database
-- ─────────────────────────────────────────────────────────
\connect xyn_tenant
GRANT ALL PRIVILEGES ON DATABASE xyn_tenant TO app_user;
GRANT ALL PRIVILEGES ON DATABASE xyn_tenant TO migration_user;

\connect xyn_pos
GRANT ALL PRIVILEGES ON DATABASE xyn_pos TO app_user;
GRANT ALL PRIVILEGES ON DATABASE xyn_pos TO migration_user;

\connect xyn_payment
GRANT ALL PRIVILEGES ON DATABASE xyn_payment TO app_user;
GRANT ALL PRIVILEGES ON DATABASE xyn_payment TO migration_user;

\connect xyn_inventory
GRANT ALL PRIVILEGES ON DATABASE xyn_inventory TO app_user;
GRANT ALL PRIVILEGES ON DATABASE xyn_inventory TO migration_user;

\connect xyn_kitchen
GRANT ALL PRIVILEGES ON DATABASE xyn_kitchen TO app_user;
GRANT ALL PRIVILEGES ON DATABASE xyn_kitchen TO migration_user;

\connect xyn_analytics
GRANT ALL PRIVILEGES ON DATABASE xyn_analytics TO app_user;
GRANT ALL PRIVILEGES ON DATABASE xyn_analytics TO migration_user;

\connect keycloak
GRANT ALL PRIVILEGES ON DATABASE keycloak TO xyn_admin;

-- ─────────────────────────────────────────────────────────
-- RLS helper function (call from each service migration)
-- Sets the tenant context variable used by RLS policies
-- ─────────────────────────────────────────────────────────
\connect xyn_pos

CREATE OR REPLACE FUNCTION current_tenant_id() RETURNS uuid AS $$
  SELECT current_setting('app.current_tenant_id', true)::uuid;
$$ LANGUAGE sql STABLE;

-- Template macro for enabling RLS on any table:
-- ALTER TABLE {table} ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE {table} FORCE ROW LEVEL SECURITY;
-- CREATE POLICY tenant_isolation ON {table}
--   USING (tenant_id = current_tenant_id());
-- ALTER TABLE {table} OWNER TO migration_user;
-- GRANT SELECT, INSERT, UPDATE, DELETE ON {table} TO app_user;
