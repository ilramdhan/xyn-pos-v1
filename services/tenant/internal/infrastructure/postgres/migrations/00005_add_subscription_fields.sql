-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS subscription_status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ;

ALTER TABLE tenants ADD CONSTRAINT tenants_subscription_status_check
    CHECK (subscription_status IN ('active', 'trial', 'expired', 'cancelled'));

-- Back-fill existing tenants as active (they pre-date trial tracking)
UPDATE tenants SET subscription_status = 'active' WHERE subscription_status = 'active';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_subscription_status_check;
ALTER TABLE tenants
    DROP COLUMN IF EXISTS subscription_status,
    DROP COLUMN IF EXISTS trial_ends_at;
-- +goose StatementEnd
