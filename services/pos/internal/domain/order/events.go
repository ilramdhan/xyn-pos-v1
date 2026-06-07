package order

import (
	"time"

	"github.com/google/uuid"
)

type DomainEvent interface{ orderEvent() }

type OrderPaidItem struct {
	ProductID uuid.UUID
	VariantID *uuid.UUID
	Quantity  int
}

type OrderPaidEvent struct {
	OrderID    uuid.UUID
	TenantID   uuid.UUID
	BranchID   uuid.UUID
	CashierID  uuid.UUID
	Items      []OrderPaidItem
	OccurredAt time.Time
}

func (OrderPaidEvent) orderEvent() {}

type OrderCancelledEvent struct {
	OrderID    uuid.UUID
	TenantID   uuid.UUID
	OccurredAt time.Time
}

func (OrderCancelledEvent) orderEvent() {}
