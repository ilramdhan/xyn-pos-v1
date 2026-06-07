package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

// BOMRepository implements stock.BOMRepository.
type BOMRepository struct{ pool *pgxpool.Pool }

// NewBOMRepository creates a BOMRepository.
func NewBOMRepository(pool *pgxpool.Pool) *BOMRepository {
	return &BOMRepository{pool: pool}
}

var _ stock.BOMRepository = (*BOMRepository)(nil)

// FindByProduct returns all BOM recipes for the given product.
func (r *BOMRepository) FindByProduct(ctx context.Context, tenantID, productID uuid.UUID) ([]*stock.BOMRecipe, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("bomRepo.FindByProduct begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("bomRepo.FindByProduct setTenant: %w", err)
	}

	const q = `SELECT id, tenant_id, product_id, ingredient_product_id, quantity_per_unit, unit
		FROM bom_recipes WHERE tenant_id=$1 AND product_id=$2`
	rows, err := tx.Query(ctx, q, tenantID, productID)
	if err != nil {
		return nil, fmt.Errorf("bomRepo.FindByProduct query: %w", err)
	}
	defer rows.Close()

	var results []*stock.BOMRecipe
	for rows.Next() {
		b := &stock.BOMRecipe{}
		if err := rows.Scan(&b.ID, &b.TenantID, &b.ProductID, &b.IngredientProductID, &b.QuantityPerUnit, &b.Unit); err != nil {
			return nil, fmt.Errorf("bomRepo.FindByProduct scan: %w", err)
		}
		results = append(results, b)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("bomRepo.FindByProduct rows: %w", err)
	}
	return results, tx.Commit(ctx)
}

// SaveAll upserts all recipes (idempotent — ON CONFLICT DO UPDATE).
func (r *BOMRepository) SaveAll(ctx context.Context, recipes []*stock.BOMRecipe) error {
	if len(recipes) == 0 {
		return nil
	}
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("bomRepo.SaveAll begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, recipes[0].TenantID); err != nil {
		return fmt.Errorf("bomRepo.SaveAll setTenant: %w", err)
	}

	const q = `INSERT INTO bom_recipes (id, tenant_id, product_id, ingredient_product_id, quantity_per_unit, unit)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (tenant_id, product_id, ingredient_product_id) DO UPDATE
		SET quantity_per_unit=EXCLUDED.quantity_per_unit, unit=EXCLUDED.unit`
	for _, b := range recipes {
		if _, err := tx.Exec(ctx, q,
			b.ID, b.TenantID, b.ProductID, b.IngredientProductID, b.QuantityPerUnit, b.Unit,
		); err != nil {
			return fmt.Errorf("bomRepo.SaveAll exec: %w", err)
		}
	}
	return tx.Commit(ctx)
}

// DeleteByProduct removes all BOM recipes for the given product.
func (r *BOMRepository) DeleteByProduct(ctx context.Context, tenantID, productID uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("bomRepo.DeleteByProduct begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return fmt.Errorf("bomRepo.DeleteByProduct setTenant: %w", err)
	}

	if _, err := tx.Exec(ctx,
		`DELETE FROM bom_recipes WHERE tenant_id=$1 AND product_id=$2`, tenantID, productID,
	); err != nil {
		return fmt.Errorf("bomRepo.DeleteByProduct exec: %w", err)
	}
	return tx.Commit(ctx)
}
