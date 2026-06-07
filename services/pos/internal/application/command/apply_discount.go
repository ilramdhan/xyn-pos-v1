package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// ApplyDiscountInput specifies the discount to apply to a draft order.
type ApplyDiscountInput struct {
	OrderID      uuid.UUID
	DiscountType order.DiscountType
	Amount       int64
}

// ApplyDiscountHandler applies or replaces the discount on a draft order.
type ApplyDiscountHandler struct{ repo order.OrderRepository }

// NewApplyDiscountHandler constructs an ApplyDiscountHandler.
func NewApplyDiscountHandler(repo order.OrderRepository) *ApplyDiscountHandler {
	return &ApplyDiscountHandler{repo: repo}
}

// Handle executes the apply-discount use case.
func (h *ApplyDiscountHandler) Handle(ctx context.Context, in ApplyDiscountInput) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, fmt.Errorf("ApplyDiscount find: %w", err)
	}
	if err := o.ApplyDiscount(order.Discount{Type: in.DiscountType, Amount: in.Amount}); err != nil {
		return nil, fmt.Errorf("ApplyDiscount domain: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("ApplyDiscount update: %w", err)
	}
	return o, nil
}
