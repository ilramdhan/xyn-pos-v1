package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// PaymentRepository implements payment.PaymentRepository backed by PostgreSQL.
type PaymentRepository struct {
	pool *pgxpool.Pool
}

// NewPaymentRepository creates a PaymentRepository.
func NewPaymentRepository(pool *pgxpool.Pool) *PaymentRepository {
	return &PaymentRepository{pool: pool}
}

// Compile-time check that PaymentRepository satisfies the domain port.
var _ payment.PaymentRepository = (*PaymentRepository)(nil)

// Save inserts a new payment record.
// Returns payment.ErrDuplicateIdempotencyKey on unique constraint violation.
func (r *PaymentRepository) Save(ctx context.Context, p *payment.Payment) error {
	const q = `
		INSERT INTO payments
			(id, tenant_id, order_id, amount, status, method,
			 external_id, snap_token, snap_redirect_url, idempotency_key,
			 created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`

	err := withTenantTx(ctx, r.pool, p.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, q,
			p.ID, p.TenantID, p.OrderID, p.Amount,
			string(p.Status), string(p.Method),
			p.ExternalID, p.SnapToken, p.SnapRedirectURL,
			p.IdempotencyKey, p.CreatedAt, p.UpdatedAt,
		)
		return err
	})
	if err != nil {
		if isUniqueViolation(err) {
			return payment.ErrDuplicateIdempotencyKey
		}
		return fmt.Errorf("paymentRepo.Save: %w", err)
	}
	return nil
}

// Update persists state changes (status, external IDs) to an existing payment.
func (r *PaymentRepository) Update(ctx context.Context, p *payment.Payment) error {
	const q = `
		UPDATE payments SET
			status            = $1,
			external_id       = $2,
			snap_token        = $3,
			snap_redirect_url = $4,
			updated_at        = $5
		WHERE id = $6`

	p.UpdatedAt = time.Now().UTC()

	err := withTenantTx(ctx, r.pool, p.TenantID, func(tx pgx.Tx) error {
		ct, err := tx.Exec(ctx, q,
			string(p.Status), p.ExternalID, p.SnapToken, p.SnapRedirectURL,
			p.UpdatedAt, p.ID,
		)
		if err != nil {
			return err
		}
		if ct.RowsAffected() == 0 {
			return payment.ErrPaymentNotFound
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("paymentRepo.Update: %w", err)
	}
	return nil
}

// FindByID retrieves a payment by primary key.
// RLS is enforced by the session variable set in the gRPC middleware; a
// cross-tenant row is invisible and returns ErrPaymentNotFound.
func (r *PaymentRepository) FindByID(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	const q = `
		SELECT id, tenant_id, order_id, amount, status, method,
		       external_id, snap_token, snap_redirect_url, idempotency_key,
		       created_at, updated_at
		FROM payments WHERE id = $1`

	row := r.pool.QueryRow(ctx, q, id)
	p, err := scanPayment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, payment.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("paymentRepo.FindByID: %w", err)
	}
	return p, nil
}

// FindByOrderID retrieves the payment linked to a given order.
// RLS is enforced by the session variable set in the gRPC middleware.
func (r *PaymentRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*payment.Payment, error) {
	const q = `
		SELECT id, tenant_id, order_id, amount, status, method,
		       external_id, snap_token, snap_redirect_url, idempotency_key,
		       created_at, updated_at
		FROM payments WHERE order_id = $1`

	row := r.pool.QueryRow(ctx, q, orderID)
	p, err := scanPayment(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, payment.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("paymentRepo.FindByOrderID: %w", err)
	}
	return p, nil
}

// FindByIdempotencyKey checks for an existing payment with the same key under a tenant.
func (r *PaymentRepository) FindByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*payment.Payment, error) {
	const q = `
		SELECT id, tenant_id, order_id, amount, status, method,
		       external_id, snap_token, snap_redirect_url, idempotency_key,
		       created_at, updated_at
		FROM payments WHERE tenant_id = $1 AND idempotency_key = $2`

	var p *payment.Payment
	err := withTenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, q, tenantID, key)
		var err error
		p, err = scanPayment(row)
		return err
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, payment.ErrPaymentNotFound
		}
		return nil, fmt.Errorf("paymentRepo.FindByIdempotencyKey: %w", err)
	}
	return p, nil
}

// scanPayment reads a single payment row into a Payment struct.
func scanPayment(row pgx.Row) (*payment.Payment, error) {
	var (
		p         payment.Payment
		statusStr string
		methodStr string
	)
	err := row.Scan(
		&p.ID, &p.TenantID, &p.OrderID, &p.Amount,
		&statusStr, &methodStr,
		&p.ExternalID, &p.SnapToken, &p.SnapRedirectURL, &p.IdempotencyKey,
		&p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Status = payment.PaymentStatus(statusStr)
	p.Method = payment.PaymentMethod(methodStr)
	return &p, nil
}

// withTenantTx runs fn inside a transaction with app.current_tenant_id set for RLS.
func withTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if _, err := tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("set tenant: %w", err)
	}

	if err := fn(tx); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

// isUniqueViolation returns true when err is a PostgreSQL unique-constraint
// violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
