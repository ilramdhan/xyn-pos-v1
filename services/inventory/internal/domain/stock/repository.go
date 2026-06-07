package stock

import (
	"context"

	"github.com/google/uuid"
)

// StockFilter is used for list queries.
type StockFilter struct {
	LowStockOnly bool
}

// StockRepository persists StockLedger aggregates and movements.
type StockRepository interface {
	FindByProductAndBranch(ctx context.Context, tenantID, branchID, productID uuid.UUID, variantID *uuid.UUID) (*StockLedger, error)
	FindByBranch(ctx context.Context, tenantID, branchID uuid.UUID, filter StockFilter) ([]*StockLedger, error)
	Save(ctx context.Context, ledger *StockLedger) error
	Update(ctx context.Context, ledger *StockLedger, movement *StockMovement) error
}

// BOMRepository persists Bill-of-Materials recipes.
type BOMRepository interface {
	FindByProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]*BOMRecipe, error)
	SaveAll(ctx context.Context, recipes []*BOMRecipe) error
	DeleteByProduct(ctx context.Context, tenantID, productID uuid.UUID) error
}
