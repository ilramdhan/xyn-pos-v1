package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// ListOrdersResult wraps the paginated list of orders with the total count.
type ListOrdersResult struct {
	Orders []*order.Order
	Total  int
}

// ListOrdersHandler returns a paginated, filtered list of orders for a tenant.
type ListOrdersHandler struct{ repo order.OrderRepository }

// NewListOrdersHandler constructs a ListOrdersHandler.
func NewListOrdersHandler(repo order.OrderRepository) *ListOrdersHandler {
	return &ListOrdersHandler{repo: repo}
}

// Handle executes the list-orders query.
func (h *ListOrdersHandler) Handle(ctx context.Context, tenantID uuid.UUID, filter order.OrderFilter) (ListOrdersResult, error) {
	orders, total, err := h.repo.FindByFilter(ctx, tenantID, filter)
	if err != nil {
		return ListOrdersResult{}, fmt.Errorf("ListOrders: %w", err)
	}
	return ListOrdersResult{Orders: orders, Total: total}, nil
}
