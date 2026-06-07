package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ArchiveProductHandler soft-deletes a product.
type ArchiveProductHandler struct {
	repo product.ProductRepository
}

// NewArchiveProductHandler creates a handler.
func NewArchiveProductHandler(repo product.ProductRepository) *ArchiveProductHandler {
	return &ArchiveProductHandler{repo: repo}
}

// Handle archives a product. Blocked if product has active orders.
func (h *ArchiveProductHandler) Handle(ctx context.Context, productID uuid.UUID) error {
	p, err := h.repo.FindByID(ctx, productID)
	if err != nil {
		return fmt.Errorf("ArchiveProduct FindByID: %w", err)
	}

	hasOrders, err := h.repo.HasActiveOrders(ctx, productID)
	if err != nil {
		return fmt.Errorf("ArchiveProduct HasActiveOrders: %w", err)
	}
	if hasOrders {
		return fmt.Errorf("ArchiveProduct: %w", product.ErrProductHasActiveOrders)
	}

	if err := p.Archive(); err != nil {
		return fmt.Errorf("ArchiveProduct: %w", err)
	}
	if err := h.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("ArchiveProduct update: %w", err)
	}
	return nil
}
