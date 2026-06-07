package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// AddItemInput holds all parameters to add a line item to a draft order.
type AddItemInput struct {
	OrderID     uuid.UUID
	ProductID   uuid.UUID
	VariantID   *uuid.UUID
	ProductName string
	VariantName string
	UnitPrice   int64
	Quantity    int
	Notes       string
	Addons      []order.OrderItemAddon
}

// AddItemHandler adds a product line item to a draft order.
type AddItemHandler struct{ repo order.OrderRepository }

// NewAddItemHandler constructs an AddItemHandler.
func NewAddItemHandler(repo order.OrderRepository) *AddItemHandler {
	return &AddItemHandler{repo: repo}
}

// Handle executes the add-item use case.
func (h *AddItemHandler) Handle(ctx context.Context, in AddItemInput) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, in.OrderID)
	if err != nil {
		return nil, fmt.Errorf("AddItem find: %w", err)
	}
	item := order.OrderItem{
		ProductID:   in.ProductID,
		VariantID:   in.VariantID,
		ProductName: in.ProductName,
		VariantName: in.VariantName,
		UnitPrice:   in.UnitPrice,
		Quantity:    in.Quantity,
		Notes:       in.Notes,
		Addons:      in.Addons,
	}
	if err := o.AddItem(item); err != nil {
		return nil, fmt.Errorf("AddItem domain: %w", err)
	}
	// The aggregate sets ID, OrderID, and Subtotal on the item — get the newly appended item.
	newItem := o.Items[len(o.Items)-1]
	if err := h.repo.AddOrderItem(ctx, newItem); err != nil {
		return nil, fmt.Errorf("AddItem persist item: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return nil, fmt.Errorf("AddItem update totals: %w", err)
	}
	return o, nil
}
