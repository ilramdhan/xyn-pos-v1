package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

type BOMLine struct {
	IngredientProductID uuid.UUID
	QuantityPerUnit     int64
	Unit                string
}

type SetBOMInput struct {
	TenantID  uuid.UUID
	ProductID uuid.UUID
	Lines     []BOMLine
}

type SetBOMHandler struct {
	bomRepo stock.BOMRepository
}

func NewSetBOMHandler(repo stock.BOMRepository) *SetBOMHandler {
	return &SetBOMHandler{bomRepo: repo}
}

// Handle replaces the entire BOM for a product atomically.
func (h *SetBOMHandler) Handle(ctx context.Context, in SetBOMInput) error {
	if err := h.bomRepo.DeleteByProduct(ctx, in.TenantID, in.ProductID); err != nil {
		return fmt.Errorf("SetBOM.Delete: %w", err)
	}
	if len(in.Lines) == 0 {
		return nil
	}
	recipes := make([]*stock.BOMRecipe, 0, len(in.Lines))
	for _, line := range in.Lines {
		r, err := stock.NewBOMRecipe(in.TenantID, in.ProductID, line.IngredientProductID, line.QuantityPerUnit, line.Unit)
		if err != nil {
			return fmt.Errorf("SetBOM.NewBOMRecipe: %w", err)
		}
		recipes = append(recipes, r)
	}
	if err := h.bomRepo.SaveAll(ctx, recipes); err != nil {
		return fmt.Errorf("SetBOM.SaveAll: %w", err)
	}
	return nil
}
