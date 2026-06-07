package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ReorderCategoriesHandler updates sort_order for a list of categories.
type ReorderCategoriesHandler struct {
	repo product.CategoryRepository
}

// NewReorderCategoriesHandler creates a handler.
func NewReorderCategoriesHandler(repo product.CategoryRepository) *ReorderCategoriesHandler {
	return &ReorderCategoriesHandler{repo: repo}
}

// Handle applies the ordered list of IDs as sort_order 0, 1, 2, ...
func (h *ReorderCategoriesHandler) Handle(ctx context.Context, tenantID uuid.UUID, orderedIDs []uuid.UUID) error {
	if err := h.repo.UpdateSortOrders(ctx, tenantID, orderedIDs); err != nil {
		return fmt.Errorf("ReorderCategories: %w", err)
	}
	return nil
}
