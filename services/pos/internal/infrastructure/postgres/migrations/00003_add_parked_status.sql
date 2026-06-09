-- +goose Up
-- +goose StatementBegin
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('draft', 'pending_payment', 'paid', 'cancelled', 'parked'));
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_status_check
    CHECK (status IN ('draft', 'pending_payment', 'paid', 'cancelled'));
-- +goose StatementEnd
