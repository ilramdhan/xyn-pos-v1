package payment

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the marker interface for payment domain events.
type DomainEvent interface{ paymentEvent() }

// PaymentCompletedEvent is published when a payment reaches SUCCESS status.
type PaymentCompletedEvent struct {
	PaymentID  uuid.UUID
	OrderID    uuid.UUID
	TenantID   uuid.UUID
	Amount     int64
	Method     PaymentMethod
	OccurredAt time.Time
}

func (PaymentCompletedEvent) paymentEvent() {}

// PaymentVoidedEvent is published when a payment is voided.
type PaymentVoidedEvent struct {
	PaymentID  uuid.UUID
	OrderID    uuid.UUID
	TenantID   uuid.UUID
	VoidedBy   uuid.UUID
	OccurredAt time.Time
}

func (PaymentVoidedEvent) paymentEvent() {}

// ReceiptIssuedEvent is published when a receipt is generated for a successful payment.
type ReceiptIssuedEvent struct {
	ReceiptID     uuid.UUID
	PaymentID     uuid.UUID
	OrderID       uuid.UUID
	TenantID      uuid.UUID
	ReceiptNumber string
	Amount        int64
	Method        PaymentMethod
	OccurredAt    time.Time
}

func (ReceiptIssuedEvent) paymentEvent() {}

// PaymentRefundedEvent is published when a payment is fully or partially refunded.
type PaymentRefundedEvent struct {
	PaymentID      uuid.UUID
	OrderID        uuid.UUID
	TenantID       uuid.UUID
	RefundedAmount int64
	OccurredAt     time.Time
}

func (PaymentRefundedEvent) paymentEvent() {}
