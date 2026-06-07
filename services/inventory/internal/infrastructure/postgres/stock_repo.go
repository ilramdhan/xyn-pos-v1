package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

// StockRepository implements stock.StockRepository.
type StockRepository struct{ pool *pgxpool.Pool }

// NewStockRepository creates a StockRepository.
func NewStockRepository(pool *pgxpool.Pool) *StockRepository {
	return &StockRepository{pool: pool}
}

var _ stock.StockRepository = (*StockRepository)(nil)

func setTenant(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID) error {
	_, err := tx.Exec(ctx, `SELECT set_config('app.current_tenant_id', $1, true)`, tenantID.String())
	return err
}

// FindByProductAndBranch retrieves the stock ledger for a specific product at a branch.
func (r *StockRepository) FindByProductAndBranch(
	ctx context.Context,
	tenantID, branchID, productID uuid.UUID,
	variantID *uuid.UUID,
) (*stock.StockLedger, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("stockRepo.FindByProductAndBranch begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("stockRepo.FindByProductAndBranch setTenant: %w", err)
	}

	var row pgx.Row
	if variantID == nil {
		const q = `SELECT id, tenant_id, branch_id, product_id, variant_id, quantity, unit, low_stock_threshold, updated_at
			FROM stock_ledgers WHERE tenant_id=$1 AND branch_id=$2 AND product_id=$3 AND variant_id IS NULL`
		row = tx.QueryRow(ctx, q, tenantID, branchID, productID)
	} else {
		const q = `SELECT id, tenant_id, branch_id, product_id, variant_id, quantity, unit, low_stock_threshold, updated_at
			FROM stock_ledgers WHERE tenant_id=$1 AND branch_id=$2 AND product_id=$3 AND variant_id=$4`
		row = tx.QueryRow(ctx, q, tenantID, branchID, productID, *variantID)
	}

	s, err := scanLedger(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, stock.ErrStockLedgerNotFound
		}
		return nil, fmt.Errorf("stockRepo.FindByProductAndBranch scan: %w", err)
	}
	return s, tx.Commit(ctx)
}

// FindByBranch lists all stock ledgers for a branch.
func (r *StockRepository) FindByBranch(
	ctx context.Context,
	tenantID, branchID uuid.UUID,
	filter stock.StockFilter,
) ([]*stock.StockLedger, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("stockRepo.FindByBranch begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, tenantID); err != nil {
		return nil, fmt.Errorf("stockRepo.FindByBranch setTenant: %w", err)
	}

	q := `SELECT id, tenant_id, branch_id, product_id, variant_id, quantity, unit, low_stock_threshold, updated_at
		FROM stock_ledgers WHERE tenant_id=$1 AND branch_id=$2`
	if filter.LowStockOnly {
		q += ` AND quantity <= low_stock_threshold`
	}
	q += ` ORDER BY product_id`

	rows, err := tx.Query(ctx, q, tenantID, branchID)
	if err != nil {
		return nil, fmt.Errorf("stockRepo.FindByBranch query: %w", err)
	}
	defer rows.Close()

	var results []*stock.StockLedger
	for rows.Next() {
		s, err := scanLedger(rows)
		if err != nil {
			return nil, fmt.Errorf("stockRepo.FindByBranch scan: %w", err)
		}
		results = append(results, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("stockRepo.FindByBranch rows: %w", err)
	}
	return results, tx.Commit(ctx)
}

// Save inserts a new stock ledger.
func (r *StockRepository) Save(ctx context.Context, ledger *stock.StockLedger) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("stockRepo.Save begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, ledger.TenantID); err != nil {
		return fmt.Errorf("stockRepo.Save setTenant: %w", err)
	}

	const q = `INSERT INTO stock_ledgers (id, tenant_id, branch_id, product_id, variant_id, quantity, unit, low_stock_threshold, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`
	_, err = tx.Exec(ctx, q,
		ledger.ID, ledger.TenantID, ledger.BranchID, ledger.ProductID, ledger.VariantID,
		ledger.Quantity, ledger.Unit, ledger.LowStockThreshold, ledger.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("stockRepo.Save exec: %w", err)
	}
	return tx.Commit(ctx)
}

// Update atomically updates the ledger quantity and inserts the movement record.
func (r *StockRepository) Update(ctx context.Context, ledger *stock.StockLedger, movement *stock.StockMovement) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("stockRepo.Update begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := setTenant(ctx, tx, ledger.TenantID); err != nil {
		return fmt.Errorf("stockRepo.Update setTenant: %w", err)
	}

	const updateQ = `UPDATE stock_ledgers SET quantity=$1, updated_at=$2 WHERE id=$3`
	ct, err := tx.Exec(ctx, updateQ, ledger.Quantity, ledger.UpdatedAt, ledger.ID)
	if err != nil {
		return fmt.Errorf("stockRepo.Update ledger: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return stock.ErrStockLedgerNotFound
	}

	const insertQ = `INSERT INTO stock_movements (id, tenant_id, branch_id, product_id, variant_id, delta, type, reference_id, note, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	_, err = tx.Exec(ctx, insertQ,
		movement.ID, movement.TenantID, movement.BranchID, movement.ProductID, movement.VariantID,
		movement.Delta, string(movement.Type), movement.ReferenceID, movement.Note, movement.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("stockRepo.Update movement: %w", err)
	}
	return tx.Commit(ctx)
}

// scannable is implemented by both pgx.Row and pgx.Rows.
type scannable interface {
	Scan(dest ...any) error
}

func scanLedger(row scannable) (*stock.StockLedger, error) {
	s := &stock.StockLedger{}
	err := row.Scan(
		&s.ID, &s.TenantID, &s.BranchID, &s.ProductID, &s.VariantID,
		&s.Quantity, &s.Unit, &s.LowStockThreshold, &s.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return s, nil
}
