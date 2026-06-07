package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// GetOrderHandler retrieves a single order by its ID.
type GetOrderHandler struct{ repo order.OrderRepository }

// NewGetOrderHandler constructs a GetOrderHandler.
func NewGetOrderHandler(repo order.OrderRepository) *GetOrderHandler {
	return &GetOrderHandler{repo: repo}
}

// Handle executes the get-order query.
func (h *GetOrderHandler) Handle(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	o, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetOrder: %w", err)
	}
	return o, nil
}
