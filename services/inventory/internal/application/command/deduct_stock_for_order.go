package command

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

type OrderItem struct {
	ProductID uuid.UUID
	VariantID *uuid.UUID
	Quantity  int64
}

type DeductStockInput struct {
	TenantID uuid.UUID
	BranchID uuid.UUID
	OrderID  uuid.UUID
	Items    []OrderItem
}

type DeductStockForOrderHandler struct {
	stockRepo stock.StockRepository
	bomRepo   stock.BOMRepository
}

func NewDeductStockForOrderHandler(stockRepo stock.StockRepository, bomRepo stock.BOMRepository) *DeductStockForOrderHandler {
	return &DeductStockForOrderHandler{stockRepo: stockRepo, bomRepo: bomRepo}
}

// Handle deducts stock for each paid order item, applying BOM recipes where defined.
func (h *DeductStockForOrderHandler) Handle(ctx context.Context, in DeductStockInput) error {
	for _, item := range in.Items {
		boms, err := h.bomRepo.FindByProduct(ctx, in.TenantID, item.ProductID)
		if err != nil && !errors.Is(err, stock.ErrBOMRecipeNotFound) {
			return fmt.Errorf("DeductStock.FindBOM product=%s: %w", item.ProductID, err)
		}

		if len(boms) > 0 {
			for _, bom := range boms {
				delta := -(bom.QuantityPerUnit * item.Quantity)
				h.deductOne(ctx, in.TenantID, in.BranchID, bom.IngredientProductID, nil, delta, in.OrderID)
			}
		} else {
			h.deductOne(ctx, in.TenantID, in.BranchID, item.ProductID, item.VariantID, -item.Quantity, in.OrderID)
		}
	}
	return nil
}

func (h *DeductStockForOrderHandler) deductOne(
	ctx context.Context,
	tenantID, branchID, productID uuid.UUID,
	variantID *uuid.UUID,
	delta int64,
	orderID uuid.UUID,
) {
	ledger, err := h.stockRepo.FindByProductAndBranch(ctx, tenantID, branchID, productID, variantID)
	if errors.Is(err, stock.ErrStockLedgerNotFound) {
		return // no stock tracking for this product — skip silently
	}
	if err != nil {
		slog.Error("deductOne: FindByProductAndBranch failed", "product", productID, "err", err)
		return
	}

	movement, err := ledger.Deduct(delta, orderID)
	if err != nil {
		slog.Warn("deductOne: stock issue", "product", productID, "err", err)
		return
	}

	if err := h.stockRepo.Update(ctx, ledger, movement); err != nil {
		slog.Error("deductOne: Update failed", "product", productID, "err", err)
	}
}
