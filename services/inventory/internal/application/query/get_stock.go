package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

type GetStockHandler struct{ repo stock.StockRepository }

func NewGetStockHandler(repo stock.StockRepository) *GetStockHandler {
	return &GetStockHandler{repo: repo}
}

func (h *GetStockHandler) Handle(ctx context.Context, tenantID, branchID, productID uuid.UUID, variantID *uuid.UUID) (*stock.StockLedger, error) {
	s, err := h.repo.FindByProductAndBranch(ctx, tenantID, branchID, productID, variantID)
	if err != nil {
		return nil, fmt.Errorf("GetStock: %w", err)
	}
	return s, nil
}
