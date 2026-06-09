-- +goose Up
-- +goose StatementBegin
ALTER TABLE payments
    ADD COLUMN IF NOT EXISTS refunded_amount BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS refund_reason   TEXT   NOT NULL DEFAULT '';

-- Drop and recreate the status CHECK constraint to include 'refunded'
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_status_check;
ALTER TABLE payments ADD CONSTRAINT payments_status_check
    CHECK (status IN ('pending','success','failed','voided','refunded'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE payments DROP CONSTRAINT IF EXISTS payments_status_check;
ALTER TABLE payments ADD CONSTRAINT payments_status_check
    CHECK (status IN ('pending','success','failed','voided'));

ALTER TABLE payments
    DROP COLUMN IF EXISTS refunded_amount,
    DROP COLUMN IF EXISTS refund_reason;
-- +goose StatementEnd
