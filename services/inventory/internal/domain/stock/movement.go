package stock

import (
	"time"

	"github.com/google/uuid"
)

// MovementType describes the cause of a stock change.
type MovementType string

const (
	MovementIn         MovementType = "in"
	MovementOut        MovementType = "out"
	MovementAdjustment MovementType = "adjustment"
)

// StockMovement is a single journal entry recording a delta to a stock ledger.
type StockMovement struct {
	ID          uuid.UUID
	TenantID    uuid.UUID
	BranchID    uuid.UUID
	ProductID   uuid.UUID
	VariantID   *uuid.UUID // nil if product has no variants
	Delta       int64      // positive = IN, negative = OUT
	Type        MovementType
	ReferenceID string // order_id, purchase_id, etc.
	Note        string
	CreatedAt   time.Time
}

// NewMovement creates a validated StockMovement.
func NewMovement(
	tenantID, branchID, productID uuid.UUID,
	variantID *uuid.UUID,
	delta int64,
	movType MovementType,
	referenceID, note string,
) (*StockMovement, error) {
	if delta == 0 {
		return nil, ErrInvalidQuantity
	}
	return &StockMovement{
		ID:          uuid.New(),
		TenantID:    tenantID,
		BranchID:    branchID,
		ProductID:   productID,
		VariantID:   variantID,
		Delta:       delta,
		Type:        movType,
		ReferenceID: referenceID,
		Note:        note,
		CreatedAt:   timeNow(),
	}, nil
}
