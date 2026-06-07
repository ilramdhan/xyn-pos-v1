package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

type GetBOMHandler struct{ repo stock.BOMRepository }

func NewGetBOMHandler(repo stock.BOMRepository) *GetBOMHandler {
	return &GetBOMHandler{repo: repo}
}

func (h *GetBOMHandler) Handle(ctx context.Context, tenantID, productID uuid.UUID) ([]*stock.BOMRecipe, error) {
	recipes, err := h.repo.FindByProduct(ctx, tenantID, productID)
	if err != nil {
		return nil, fmt.Errorf("GetBOM: %w", err)
	}
	return recipes, nil
}
