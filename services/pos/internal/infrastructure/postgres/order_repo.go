package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// OrderRepository implements order.OrderRepository backed by PostgreSQL.
type OrderRepository struct {
	pool *pgxpool.Pool
}

// NewOrderRepository constructs an OrderRepository.
func NewOrderRepository(pool *pgxpool.Pool) *OrderRepository {
	return &OrderRepository{pool: pool}
}

func (r *OrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*order.Order, error) {
	const q = `
		SELECT id, tenant_id, branch_id, shift_id, cashier_id, order_number,
		       order_type, table_number, status, subtotal, tax_amount,
		       discount_amount, total, discount_type, discount_value,
		       idempotency_key, notes, created_at, updated_at
		FROM orders WHERE id = $1`

	o, err := r.scanOne(ctx, q, id)
	if err != nil {
		return nil, err
	}
	if err := r.loadItems(ctx, o); err != nil {
		return nil, err
	}
	return o, nil
}

func (r *OrderRepository) FindByIdempotencyKey(ctx context.Context, tenantID uuid.UUID, key string) (*order.Order, error) {
	const q = `
		SELECT id, tenant_id, branch_id, shift_id, cashier_id, order_number,
		       order_type, table_number, status, subtotal, tax_amount,
		       discount_amount, total, discount_type, discount_value,
		       idempotency_key, notes, created_at, updated_at
		FROM orders WHERE tenant_id = $1 AND idempotency_key = $2`

	return r.scanOne(ctx, q, tenantID, key)
}

func (r *OrderRepository) FindByFilter(ctx context.Context, tenantID uuid.UUID, filter order.OrderFilter) ([]*order.Order, int, error) {
	args := []any{tenantID}
	where := "tenant_id = $1"

	if filter.ShiftID != nil {
		where += fmt.Sprintf(" AND shift_id = $%d", len(args)+1)
		args = append(args, *filter.ShiftID)
	}
	if filter.Status != nil {
		where += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, string(*filter.Status))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	var total int
	_ = r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM orders WHERE "+where, args...).Scan(&total)

	args = append(args, limit, filter.Offset)
	q := fmt.Sprintf(`
		SELECT id, tenant_id, branch_id, shift_id, cashier_id, order_number,
		       order_type, table_number, status, subtotal, tax_amount,
		       discount_amount, total, discount_type, discount_value,
		       idempotency_key, notes, created_at, updated_at
		FROM orders WHERE %s
		ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("orderRepo.FindByFilter: %w", err)
	}
	defer rows.Close()

	var result []*order.Order
	for rows.Next() {
		o, err := scanOrder(rows)
		if err != nil {
			return nil, 0, err
		}
		result = append(result, o)
	}
	return result, total, rows.Err()
}

func (r *OrderRepository) Save(ctx context.Context, o *order.Order) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("orderRepo.Save begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const q = `
		INSERT INTO orders (id, tenant_id, branch_id, shift_id, cashier_id,
		    order_number, order_type, table_number, status,
		    subtotal, tax_amount, discount_amount, total,
		    discount_type, discount_value, idempotency_key, notes, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19)`

	var discountType *string
	var discountValue *int64
	if o.Discount != nil {
		dt := string(o.Discount.Type)
		discountType = &dt
		discountValue = &o.Discount.Amount
	}

	_, err = tx.Exec(ctx, q,
		o.ID, o.TenantID, o.BranchID, o.ShiftID, o.CashierID,
		o.OrderNumber, string(o.OrderType), o.TableNumber, string(o.Status),
		o.Subtotal, o.TaxAmount, o.DiscountAmount, o.Total,
		discountType, discountValue, o.IdempotencyKey, o.Notes, o.CreatedAt, o.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("orderRepo.Save: idempotency key already exists")
		}
		return fmt.Errorf("orderRepo.Save: %w", err)
	}

	for _, item := range o.Items {
		if err := insertOrderItem(ctx, tx, item); err != nil {
			return fmt.Errorf("orderRepo.Save insertItem: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *OrderRepository) Update(ctx context.Context, o *order.Order) error {
	const q = `
		UPDATE orders SET status=$2, subtotal=$3, tax_amount=$4, discount_amount=$5,
		       total=$6, discount_type=$7, discount_value=$8, notes=$9, updated_at=$10
		WHERE id = $1`

	var discountType *string
	var discountValue *int64
	if o.Discount != nil {
		dt := string(o.Discount.Type)
		discountType = &dt
		discountValue = &o.Discount.Amount
	}

	_, err := r.pool.Exec(ctx, q,
		o.ID, string(o.Status), o.Subtotal, o.TaxAmount, o.DiscountAmount,
		o.Total, discountType, discountValue, o.Notes, o.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("orderRepo.Update: %w", err)
	}
	return nil
}

func (r *OrderRepository) AddOrderItem(ctx context.Context, item order.OrderItem) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO order_items (id, order_id, product_id, variant_id, product_name,
		  variant_name, unit_price, quantity, subtotal, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		item.ID, item.OrderID, item.ProductID, item.VariantID,
		item.ProductName, nullableStr(item.VariantName),
		item.UnitPrice, item.Quantity, item.Subtotal, item.Notes,
	)
	if err != nil {
		return fmt.Errorf("orderRepo.AddOrderItem: %w", err)
	}
	return nil
}

func (r *OrderRepository) RemoveOrderItem(ctx context.Context, itemID uuid.UUID) error {
	_, err := r.pool.Exec(ctx, "DELETE FROM order_items WHERE id = $1", itemID)
	if err != nil {
		return fmt.Errorf("orderRepo.RemoveOrderItem: %w", err)
	}
	return nil
}

func (r *OrderRepository) UpdateOrderItemQuantity(ctx context.Context, itemID uuid.UUID, qty int, subtotal int64) error {
	_, err := r.pool.Exec(ctx,
		"UPDATE order_items SET quantity=$2, subtotal=$3 WHERE id=$1",
		itemID, qty, subtotal,
	)
	if err != nil {
		return fmt.Errorf("orderRepo.UpdateOrderItemQuantity: %w", err)
	}
	return nil
}

func (r *OrderRepository) scanOne(ctx context.Context, q string, args ...any) (*order.Order, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("orderRepo query: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if errors.Is(rows.Err(), nil) {
			return nil, order.ErrOrderNotFound
		}
		return nil, rows.Err()
	}
	return scanOrder(rows)
}

func (r *OrderRepository) loadItems(ctx context.Context, o *order.Order) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id, order_id, product_id, variant_id, product_name, COALESCE(variant_name,''),
		        unit_price, quantity, subtotal, COALESCE(notes,'')
		 FROM order_items WHERE order_id = $1`, o.ID)
	if err != nil {
		return fmt.Errorf("orderRepo.loadItems: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var item order.OrderItem
		if err := rows.Scan(
			&item.ID, &item.OrderID, &item.ProductID, &item.VariantID,
			&item.ProductName, &item.VariantName,
			&item.UnitPrice, &item.Quantity, &item.Subtotal, &item.Notes,
		); err != nil {
			return err
		}
		o.Items = append(o.Items, item)
	}
	return rows.Err()
}

func scanOrder(rows pgx.Rows) (*order.Order, error) {
	var o order.Order
	var statusStr, orderTypeStr string
	var discountType *string
	var discountValue *int64

	err := rows.Scan(
		&o.ID, &o.TenantID, &o.BranchID, &o.ShiftID, &o.CashierID,
		&o.OrderNumber, &orderTypeStr, &o.TableNumber, &statusStr,
		&o.Subtotal, &o.TaxAmount, &o.DiscountAmount, &o.Total,
		&discountType, &discountValue,
		&o.IdempotencyKey, &o.Notes, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	o.Status = order.OrderStatus(statusStr)
	o.OrderType = order.OrderType(orderTypeStr)

	if discountType != nil && discountValue != nil {
		o.Discount = &order.Discount{
			Type:   order.DiscountType(*discountType),
			Amount: *discountValue,
		}
	}
	return &o, nil
}

func insertOrderItem(ctx context.Context, tx pgx.Tx, item order.OrderItem) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO order_items (id, order_id, product_id, variant_id, product_name,
		  variant_name, unit_price, quantity, subtotal, notes)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		item.ID, item.OrderID, item.ProductID, item.VariantID,
		item.ProductName, nullableStr(item.VariantName),
		item.UnitPrice, item.Quantity, item.Subtotal, item.Notes,
	)
	return err
}
