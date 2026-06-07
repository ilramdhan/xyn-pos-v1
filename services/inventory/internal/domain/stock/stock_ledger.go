package stock

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// StockLedger is the aggregate root representing current stock at a location.
type StockLedger struct {
	ID                uuid.UUID
	TenantID          uuid.UUID
	BranchID          uuid.UUID
	ProductID         uuid.UUID
	VariantID         *uuid.UUID // nil if no variant
	Quantity          int64      // current balance (minor units: grams, ml, pcs)
	Unit              string     // "g", "ml", "pcs"
	LowStockThreshold int64      // alert when Quantity <= this
	UpdatedAt         time.Time
	events            []DomainEvent
}

// NewStockLedger creates a StockLedger with zero quantity.
func NewStockLedger(tenantID, branchID, productID uuid.UUID, variantID *uuid.UUID, unit string, lowStockThreshold int64) *StockLedger {
	return &StockLedger{
		ID:                uuid.New(),
		TenantID:          tenantID,
		BranchID:          branchID,
		ProductID:         productID,
		VariantID:         variantID,
		Quantity:          0,
		Unit:              unit,
		LowStockThreshold: lowStockThreshold,
		UpdatedAt:         timeNow(),
	}
}

// Deduct reduces stock by the absolute value of delta (always negative delta).
// Used when an order is paid. Returns ErrInsufficientStock if result < 0.
func (s *StockLedger) Deduct(delta int64, orderID uuid.UUID) (*StockMovement, error) {
	if delta >= 0 {
		return nil, fmt.Errorf("Deduct: delta must be negative, got %d", delta)
	}
	if s.Quantity+delta < 0 {
		return nil, ErrInsufficientStock
	}
	s.Quantity += delta
	s.UpdatedAt = timeNow()
	s.events = append(s.events, StockDeductedEvent{
		LedgerID:   s.ID,
		TenantID:   s.TenantID,
		BranchID:   s.BranchID,
		ProductID:  s.ProductID,
		Delta:      delta,
		OrderID:    orderID,
		OccurredAt: timeNow(),
	})
	mov, _ := NewMovement(s.TenantID, s.BranchID, s.ProductID, s.VariantID, delta, MovementOut, orderID.String(), "order fulfillment")
	return mov, nil
}

// Adjust applies any delta (positive or negative) for manual corrections.
// Returns ErrInvalidDelta if delta is 0.
// Returns ErrInsufficientStock if resulting Quantity < 0.
func (s *StockLedger) Adjust(delta int64, note string) (*StockMovement, error) {
	if delta == 0 {
		return nil, ErrInvalidDelta
	}
	if s.Quantity+delta < 0 {
		return nil, ErrInsufficientStock
	}
	s.Quantity += delta
	s.UpdatedAt = timeNow()
	s.events = append(s.events, StockAdjustedEvent{
		LedgerID:   s.ID,
		TenantID:   s.TenantID,
		BranchID:   s.BranchID,
		ProductID:  s.ProductID,
		Delta:      delta,
		OccurredAt: timeNow(),
	})
	movType := MovementAdjustment
	if delta > 0 {
		movType = MovementIn
	}
	mov, _ := NewMovement(s.TenantID, s.BranchID, s.ProductID, s.VariantID, delta, movType, "", note)
	return mov, nil
}

// IsLowStock returns true when the current quantity is at or below the alert threshold.
func (s *StockLedger) IsLowStock() bool {
	return s.Quantity <= s.LowStockThreshold
}

// PopEvents returns and clears all pending domain events.
func (s *StockLedger) PopEvents() []DomainEvent {
	evs := s.events
	s.events = nil
	return evs
}
