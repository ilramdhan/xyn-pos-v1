package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// GetProductHandler retrieves a single product by ID.
type GetProductHandler struct {
	repo product.ProductRepository
}

// NewGetProductHandler creates a handler.
func NewGetProductHandler(repo product.ProductRepository) *GetProductHandler {
	return &GetProductHandler{repo: repo}
}

// Handle returns the product or ErrProductNotFound.
func (h *GetProductHandler) Handle(ctx context.Context, productID uuid.UUID) (*product.Product, error) {
	p, err := h.repo.FindByID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("GetProduct: %w", err)
	}
	return p, nil
}
