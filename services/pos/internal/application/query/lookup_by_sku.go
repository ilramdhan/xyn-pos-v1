package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// LookupBySKUHandler finds a product by SKU (barcode scanner endpoint).
type LookupBySKUHandler struct {
	repo product.ProductRepository
}

// NewLookupBySKUHandler creates a handler.
func NewLookupBySKUHandler(repo product.ProductRepository) *LookupBySKUHandler {
	return &LookupBySKUHandler{repo: repo}
}

// Handle returns the product for the given tenant and SKU.
func (h *LookupBySKUHandler) Handle(ctx context.Context, tenantID uuid.UUID, sku string) (*product.Product, error) {
	if sku == "" {
		return nil, fmt.Errorf("LookupBySKU: SKU cannot be empty")
	}
	p, err := h.repo.FindBySKU(ctx, tenantID, sku)
	if err != nil {
		return nil, fmt.Errorf("LookupBySKU: %w", err)
	}
	return p, nil
}
