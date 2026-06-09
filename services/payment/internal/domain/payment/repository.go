package payment

import (
	"context"

	"github.com/google/uuid"
)

// PaymentRepository is the persistence port for Payment aggregates.
type PaymentRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Payment, error)
	FindByOrderID(ctx context.Context, orderID uuid.UUID) (*Payment, error)
	FindByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*Payment, error)
	Save(ctx context.Context, p *Payment) error
	Update(ctx context.Context, p *Payment) error
}

// ReceiptRepository is the persistence port for Receipt aggregates.
type ReceiptRepository interface {
	Save(ctx context.Context, r *Receipt) error
	FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*Receipt, error)
	FindByOrderID(ctx context.Context, orderID uuid.UUID) (*Receipt, error)
}
