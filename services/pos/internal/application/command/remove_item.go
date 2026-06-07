package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// RemoveItemInput identifies the item to remove from an order.
type RemoveItemInput struct {
	OrderID uuid.UUID
	ItemID  uuid.UUID
}

// RemoveItemHandler removes a line item from a draft order.
type RemoveItemHandler struct{ repo order.OrderRepository }

// NewRemoveItemHandler constructs a RemoveItemHandler.
func NewRemoveItemHandler(repo order.OrderRepository) *RemoveItemHandler {
	return &RemoveItemHandler{repo: repo}
}

// Handle executes the remove-item use case.
func (h *RemoveItemHandler) Handle(ctx context.Context, in RemoveItemInput) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, fmt.Errorf("RemoveItem find: %w", err)
	}
	if err := o.RemoveItem(in.ItemID); err != nil {
		return nil, fmt.Errorf("RemoveItem domain: %w", err)
	}
	if err := h.repo.RemoveOrderItem(ctx, in.ItemID); err != nil {
		return nil, fmt.Errorf("RemoveItem persist: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("RemoveItem update totals: %w", err)
	}
	return o, nil
}
