package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

type ListStockHandler struct{ repo stock.StockRepository }

func NewListStockHandler(repo stock.StockRepository) *ListStockHandler {
	return &ListStockHandler{repo: repo}
}

func (h *ListStockHandler) Handle(ctx context.Context, tenantID, branchID uuid.UUID, filter stock.StockFilter) ([]*stock.StockLedger, error) {
	result, err := h.repo.FindByBranch(ctx, tenantID, branchID, filter)
	if err != nil {
		return nil, fmt.Errorf("ListStock: %w", err)
	}
	return result, nil
}
