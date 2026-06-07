package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// SetBranchPriceInput is the command payload.
type SetBranchPriceInput struct {
	BranchID       uuid.UUID
	ProductID      uuid.UUID
	OverridePrice  *int64 // nil = remove override
	IsAvailable    bool
	IdempotencyKey string
}

// SetBranchPriceHandler sets a branch-specific price override.
type SetBranchPriceHandler struct {
	repo product.ProductRepository
}

// NewSetBranchPriceHandler creates a handler.
func NewSetBranchPriceHandler(repo product.ProductRepository) *SetBranchPriceHandler {
	return &SetBranchPriceHandler{repo: repo}
}

// Handle upserts a branch_products record.
func (h *SetBranchPriceHandler) Handle(ctx context.Context, in SetBranchPriceInput) error {
	if in.OverridePrice != nil && *in.OverridePrice < 0 {
		return fmt.Errorf("SetBranchPrice: override_price must be >= 0")
	}
	bp := product.BranchProduct{
		BranchID:      in.BranchID,
		ProductID:     in.ProductID,
		OverridePrice: in.OverridePrice,
		IsAvailable:   in.IsAvailable,
	}
	if err := h.repo.SetBranchPrice(ctx, bp); err != nil {
		return fmt.Errorf("SetBranchPrice: %w", err)
	}
	return nil
}
