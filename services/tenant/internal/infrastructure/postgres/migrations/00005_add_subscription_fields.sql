-- +goose Up
-- +goose StatementBegin
ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS subscription_status TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN IF NOT EXISTS trial_ends_at TIMESTAMPTZ;

ALTER TABLE tenants ADD CONSTRAINT IF NOT EXISTS tenants_subscription_status_check
    CHECK (subscription_status IN ('active', 'trial', 'expired', 'cancelled'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE tenants DROP CONSTRAINT IF EXISTS tenants_subscription_status_check;
ALTER TABLE tenants
    DROP COLUMN IF EXISTS subscription_status,
    DROP COLUMN IF EXISTS trial_ends_at;
-- +goose StatementEnd
