package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// SubmitOrderHandler transitions a draft order to pending_payment, ready for checkout.
type SubmitOrderHandler struct{ repo order.OrderRepository }

// NewSubmitOrderHandler constructs a SubmitOrderHandler.
func NewSubmitOrderHandler(repo order.OrderRepository) *SubmitOrderHandler {
	return &SubmitOrderHandler{repo: repo}
}

// Handle executes the submit-order use case.
func (h *SubmitOrderHandler) Handle(ctx context.Context, orderID uuid.UUID) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("SubmitOrder find: %w", err)
	}
	if err := o.Submit(); err != nil {
		return nil, fmt.Errorf("SubmitOrder domain: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("SubmitOrder update: %w", err)
	}
	return o, nil
}
