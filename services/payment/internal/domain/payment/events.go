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
