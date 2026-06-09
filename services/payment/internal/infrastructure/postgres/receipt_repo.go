package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// ReceiptRepository implements payment.ReceiptRepository backed by PostgreSQL.
type ReceiptRepository struct {
	pool *pgxpool.Pool
}

// NewReceiptRepository creates a ReceiptRepository.
func NewReceiptRepository(pool *pgxpool.Pool) *ReceiptRepository {
	return &ReceiptRepository{pool: pool}
}

// Compile-time check that ReceiptRepository satisfies the domain port.
var _ payment.ReceiptRepository = (*ReceiptRepository)(nil)

// Save inserts a new receipt record.
// ON CONFLICT (payment_id) DO NOTHING provides idempotency — a duplicate Save is a silent no-op.
func (r *ReceiptRepository) Save(ctx context.Context, rec *payment.Receipt) error {
	const q = `
		INSERT INTO receipts
			(id, payment_id, order_id, tenant_id, receipt_number, amount, method, issued_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (payment_id) DO NOTHING`

	err := withTenantTx(ctx, r.pool, rec.TenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, q,
			rec.ID, rec.PaymentID, rec.OrderID, rec.TenantID,
			rec.ReceiptNumber, rec.Amount, string(rec.Method), rec.IssuedAt,
		)
		return err
	})
	if err != nil {
		return fmt.Errorf("receiptRepo.Save: %w", err)
	}
	return nil
}

// FindByPaymentID retrieves the receipt linked to a given payment.
// RLS is enforced by the session variable set in the gRPC middleware.
func (r *ReceiptRepository) FindByPaymentID(ctx context.Context, paymentID uuid.UUID) (*payment.Receipt, error) {
	const q = `
		SELECT id, payment_id, order_id, tenant_id, receipt_number, amount, method, issued_at
		FROM receipts WHERE payment_id = $1`

	row := r.pool.QueryRow(ctx, q, paymentID)
	rec, err := scanReceipt(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, payment.ErrReceiptNotFound
		}
		return nil, fmt.Errorf("receiptRepo.FindByPaymentID: %w", err)
	}
	return rec, nil
}

// FindByOrderID retrieves the receipt linked to a given order.
// RLS is enforced by the session variable set in the gRPC middleware.
func (r *ReceiptRepository) FindByOrderID(ctx context.Context, orderID uuid.UUID) (*payment.Receipt, error) {
	const q = `
		SELECT id, payment_id, order_id, tenant_id, receipt_number, amount, method, issued_at
		FROM receipts WHERE order_id = $1`

	row := r.pool.QueryRow(ctx, q, orderID)
	rec, err := scanReceipt(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, payment.ErrReceiptNotFound
		}
		return nil, fmt.Errorf("receiptRepo.FindByOrderID: %w", err)
	}
	return rec, nil
}

// scanReceipt reads a single receipt row into a Receipt struct.
func scanReceipt(row pgx.Row) (*payment.Receipt, error) {
	var (
		rec       payment.Receipt
		methodStr string
	)
	err := row.Scan(
		&rec.ID, &rec.PaymentID, &rec.OrderID, &rec.TenantID,
		&rec.ReceiptNumber, &rec.Amount, &methodStr, &rec.IssuedAt,
	)
	if err != nil {
		return nil, err
	}
	rec.Method = payment.PaymentMethod(methodStr)
	return &rec, nil
}
