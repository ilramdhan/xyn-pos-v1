package stock

import (
	"time"

	"github.com/google/uuid"
)

// DomainEvent is the marker interface for stock domain events.
type DomainEvent interface{ stockEvent() }

// StockDeductedEvent is emitted when stock is reduced by an order.
type StockDeductedEvent struct {
	LedgerID   uuid.UUID
	TenantID   uuid.UUID
	BranchID   uuid.UUID
	ProductID  uuid.UUID
	Delta      int64 // negative
	OrderID    uuid.UUID
	OccurredAt time.Time
}

func (StockDeductedEvent) stockEvent() {}

// StockAdjustedEvent is emitted when stock is manually corrected.
type StockAdjustedEvent struct {
	LedgerID   uuid.UUID
	TenantID   uuid.UUID
	BranchID   uuid.UUID
	ProductID  uuid.UUID
	Delta      int64
	OccurredAt time.Time
}

func (StockAdjustedEvent) stockEvent() {}
