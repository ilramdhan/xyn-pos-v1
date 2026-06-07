package postgres

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// ShiftRepository implements order.ShiftRepository backed by PostgreSQL.
type ShiftRepository struct {
	pool *pgxpool.Pool
}

// NewShiftRepository constructs a ShiftRepository.
func NewShiftRepository(pool *pgxpool.Pool) *ShiftRepository {
	return &ShiftRepository{pool: pool}
}

func (r *ShiftRepository) FindByID(ctx context.Context, id uuid.UUID) (*order.Shift, error) {
	const q = `SELECT id, tenant_id, branch_id, cashier_id, status,
		          opened_at, closed_at, opening_cash, closing_cash
		   FROM shifts WHERE id = $1`

	rows, err := r.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("shiftRepo.FindByID: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, order.ErrShiftNotFound
	}
	return scanShift(rows)
}

func (r *ShiftRepository) FindOpenByBranchAndCashier(ctx context.Context, branchID, cashierID uuid.UUID) (*order.Shift, error) {
	const q = `SELECT id, tenant_id, branch_id, cashier_id, status,
		          opened_at, closed_at, opening_cash, closing_cash
		   FROM shifts WHERE branch_id = $1 AND cashier_id = $2 AND status = 'open'`

	rows, err := r.pool.Query(ctx, q, branchID, cashierID)
	if err != nil {
		return nil, fmt.Errorf("shiftRepo.FindOpenByBranchAndCashier: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, order.ErrShiftNotFound
	}
	return scanShift(rows)
}

func (r *ShiftRepository) Save(ctx context.Context, s *order.Shift) error {
	const q = `INSERT INTO shifts (id, tenant_id, branch_id, cashier_id, status,
		          opened_at, opening_cash)
		   VALUES ($1,$2,$3,$4,$5,$6,$7)`

	_, err := r.pool.Exec(ctx, q,
		s.ID, s.TenantID, s.BranchID, s.CashierID, string(s.Status), s.OpenedAt, s.OpeningCash)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("shiftRepo.Save: %w", order.ErrShiftAlreadyOpen)
		}
		return fmt.Errorf("shiftRepo.Save: %w", err)
	}
	return nil
}

func (r *ShiftRepository) Update(ctx context.Context, s *order.Shift) error {
	const q = `UPDATE shifts SET status=$2, closed_at=$3, closing_cash=$4 WHERE id=$1`
	_, err := r.pool.Exec(ctx, q, s.ID, string(s.Status), s.ClosedAt, s.ClosingCash)
	if err != nil {
		return fmt.Errorf("shiftRepo.Update: %w", err)
	}
	return nil
}

func scanShift(rows pgx.Rows) (*order.Shift, error) {
	var s order.Shift
	var statusStr string
	err := rows.Scan(
		&s.ID, &s.TenantID, &s.BranchID, &s.CashierID, &statusStr,
		&s.OpenedAt, &s.ClosedAt, &s.OpeningCash, &s.ClosingCash,
	)
	if err != nil {
		return nil, err
	}
	s.Status = order.ShiftStatus(statusStr)
	return &s, nil
}
