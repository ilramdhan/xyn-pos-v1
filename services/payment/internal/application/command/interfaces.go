package command

import (
	"context"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// PaymentEventPublisher publishes domain events after successful persistence.
type PaymentEventPublisher interface {
	PublishCompleted(ctx context.Context, ev payment.PaymentCompletedEvent) error
	PublishVoided(ctx context.Context, ev payment.PaymentVoidedEvent) error
	PublishRefunded(ctx context.Context, ev payment.PaymentRefundedEvent) error
}

// IdempotencyStore provides distributed idempotency locking.
type IdempotencyStore interface {
	Acquire(ctx context.Context, key string) (bool, error)
	Release(ctx context.Context, key string) error
}

// ReceiptEventPublisher publishes receipt domain events.
type ReceiptEventPublisher interface {
	PublishReceiptIssued(ctx context.Context, ev payment.ReceiptIssuedEvent) error
}
