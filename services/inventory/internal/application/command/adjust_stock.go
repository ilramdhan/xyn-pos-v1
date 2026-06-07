package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

type AdjustStockInput struct {
	TenantID  uuid.UUID
	BranchID  uuid.UUID
	ProductID uuid.UUID
	VariantID *uuid.UUID
	Delta     int64
	Unit      string
	Note      string
}

type AdjustStockHandler struct {
	stockRepo stock.StockRepository
}

func NewAdjustStockHandler(repo stock.StockRepository) *AdjustStockHandler {
	return &AdjustStockHandler{stockRepo: repo}
}

// Handle applies a stock delta. Auto-creates the ledger if it does not exist.
func (h *AdjustStockHandler) Handle(ctx context.Context, in AdjustStockInput) (*stock.StockLedger, error) {
	ledger, err := h.stockRepo.FindByProductAndBranch(ctx, in.TenantID, in.BranchID, in.ProductID, in.VariantID)
	if err != nil && !errors.Is(err, stock.ErrStockLedgerNotFound) {
		return nil, fmt.Errorf("AdjustStock.Find: %w", err)
	}
	if errors.Is(err, stock.ErrStockLedgerNotFound) {
		ledger = stock.NewStockLedger(in.TenantID, in.BranchID, in.ProductID, in.VariantID, in.Unit, 0)
		if err := h.stockRepo.Save(ctx, ledger); err != nil {
			return nil, fmt.Errorf("AdjustStock.Save: %w", err)
		}
	}
	movement, err := ledger.Adjust(in.Delta, in.Note)
	if err != nil {
		return nil, fmt.Errorf("AdjustStock.Adjust: %w", err)
	}
	if err := h.stockRepo.Update(ctx, ledger, movement); err != nil {
		return nil, fmt.Errorf("AdjustStock.Update: %w", err)
	}
	return ledger, nil
}
