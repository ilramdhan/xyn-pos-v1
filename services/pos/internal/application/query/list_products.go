package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ListProductsResult holds products and total count.
type ListProductsResult struct {
	Products []*product.Product
	Total    int
}

// ListProductsHandler lists products with optional filters.
type ListProductsHandler struct {
	repo product.ProductRepository
}

// NewListProductsHandler creates a handler.
func NewListProductsHandler(repo product.ProductRepository) *ListProductsHandler {
	return &ListProductsHandler{repo: repo}
}

// Handle returns filtered products for a tenant.
func (h *ListProductsHandler) Handle(ctx context.Context, tenantID uuid.UUID, filter product.Filter) (*ListProductsResult, error) {
	products, total, err := h.repo.FindByTenant(ctx, tenantID, filter)
	if err != nil {
		return nil, fmt.Errorf("ListProducts: %w", err)
	}
	return &ListProductsResult{Products: products, Total: total}, nil
}
