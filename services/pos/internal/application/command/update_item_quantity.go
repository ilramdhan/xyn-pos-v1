package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// UpdateItemQuantityInput identifies which item to change and its new quantity.
type UpdateItemQuantityInput struct {
	OrderID  uuid.UUID
	ItemID   uuid.UUID
	Quantity int
}

// UpdateItemQuantityHandler updates the quantity of a line item on a draft order.
type UpdateItemQuantityHandler struct{ repo order.OrderRepository }

// NewUpdateItemQuantityHandler constructs an UpdateItemQuantityHandler.
func NewUpdateItemQuantityHandler(repo order.OrderRepository) *UpdateItemQuantityHandler {
	return &UpdateItemQuantityHandler{repo: repo}
}

// Handle executes the update-item-quantity use case.
func (h *UpdateItemQuantityHandler) Handle(ctx context.Context, in UpdateItemQuantityInput) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, fmt.Errorf("UpdateItemQty find: %w", err)
	}
	if err := o.UpdateItemQuantity(in.ItemID, in.Quantity); err != nil {
		return nil, fmt.Errorf("UpdateItemQty domain: %w", err)
	}
	// Find the updated item to persist its new quantity and subtotal.
	for _, item := range o.Items {
		if item.ID == in.ItemID {
			if err := h.repo.UpdateOrderItemQuantity(ctx, item.ID, item.Quantity, item.Subtotal); err != nil {
				return nil, fmt.Errorf("UpdateItemQty persist: %w", err)
			}
			break
		}
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("UpdateItemQty update totals: %w", err)
	}
	return o, nil
}
