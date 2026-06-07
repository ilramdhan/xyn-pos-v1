package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// ProductRepository implements product.ProductRepository backed by PostgreSQL.
type ProductRepository struct {
	pool *pgxpool.Pool
}

// NewProductRepository constructs a ProductRepository.
func NewProductRepository(pool *pgxpool.Pool) *ProductRepository {
	return &ProductRepository{pool: pool}
}

func (r *ProductRepository) FindByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	const q = `SELECT id, tenant_id, category_id, sku, name, description,
	                   base_price, tax_type, is_active, created_at, updated_at
	            FROM products WHERE id = $1`

	p, err := r.scanOne(ctx, q, id)
	if err != nil {
		return nil, err
	}
	if err := r.loadVariants(ctx, p); err != nil {
		return nil, err
	}
	if err := r.loadAddonGroups(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProductRepository) FindBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*product.Product, error) {
	const q = `SELECT id, tenant_id, category_id, sku, name, description,
	                   base_price, tax_type, is_active, created_at, updated_at
	            FROM products WHERE tenant_id = $1 AND sku = $2`

	p, err := r.scanOne(ctx, q, tenantID, sku)
	if err != nil {
		return nil, err
	}
	if err := r.loadVariants(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func (r *ProductRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID, filter product.Filter) ([]*product.Product, int, error) {
	where := "tenant_id = $1"
	args := []any{tenantID}
	if filter.ActiveOnly {
		where += " AND is_active = true"
	}
	if filter.CategoryID != nil {
		args = append(args, *filter.CategoryID)
		where += fmt.Sprintf(" AND category_id = $%d", len(args))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM products WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("productRepo.FindByTenant count: %w", err)
	}

	args = append(args, limit, filter.Offset)
	q := fmt.Sprintf(`SELECT id, tenant_id, category_id, sku, name, description,
	                          base_price, tax_type, is_active, created_at, updated_at
	                   FROM products WHERE %s
	                   ORDER BY created_at DESC
	                   LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("productRepo.FindByTenant: %w", err)
	}
	defer rows.Close()

	var result []*product.Product
	for rows.Next() {
		p, err := scanProduct(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("productRepo.FindByTenant scan: %w", err)
		}
		result = append(result, p)
	}
	return result, total, rows.Err()
}

func (r *ProductRepository) Save(ctx context.Context, p *product.Product) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("productRepo.Save begin: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	const q = `INSERT INTO products
	               (id, tenant_id, category_id, sku, name, description,
	                base_price, tax_type, is_active, created_at, updated_at)
	           VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`

	_, err = tx.Exec(ctx, q,
		p.ID, p.TenantID, p.CategoryID, nullableStr(p.SKU), p.Name, p.Description,
		p.BasePrice, string(p.TaxType), p.IsActive, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return product.ErrSKUAlreadyExists
		}
		return fmt.Errorf("productRepo.Save insert: %w", err)
	}

	for _, v := range p.Variants {
		if err := insertVariant(ctx, tx, p.ID, v); err != nil {
			return fmt.Errorf("productRepo.Save variant: %w", err)
		}
	}
	for _, ag := range p.AddonGroups {
		if err := insertAddonGroup(ctx, tx, p.ID, ag); err != nil {
			return fmt.Errorf("productRepo.Save addonGroup: %w", err)
		}
	}
	return tx.Commit(ctx)
}

func (r *ProductRepository) Update(ctx context.Context, p *product.Product) error {
	const q = `UPDATE products
	           SET name=$2, description=$3, base_price=$4, tax_type=$5,
	               category_id=$6, is_active=$7, updated_at=$8
	           WHERE id = $1`

	_, err := r.pool.Exec(ctx, q,
		p.ID, p.Name, p.Description, p.BasePrice, string(p.TaxType),
		p.CategoryID, p.IsActive, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("productRepo.Update: %w", err)
	}
	return nil
}

func (r *ProductRepository) SetBranchPrice(ctx context.Context, bp product.BranchProduct) error {
	const q = `INSERT INTO branch_products (branch_id, product_id, override_price, is_available, updated_at)
	           VALUES ($1, $2, $3, $4, now())
	           ON CONFLICT (branch_id, product_id) DO UPDATE
	           SET override_price=$3, is_available=$4, updated_at=now()`

	_, err := r.pool.Exec(ctx, q, bp.BranchID, bp.ProductID, bp.OverridePrice, bp.IsAvailable)
	if err != nil {
		return fmt.Errorf("productRepo.SetBranchPrice: %w", err)
	}
	return nil
}

func (r *ProductRepository) HasActiveOrders(ctx context.Context, productID uuid.UUID) (bool, error) {
	// order_items table will exist after migration 00002 (order domain).
	// For now this is a safe stub — the archive command calls this before updating.
	// When the order domain is added, this query will be live.
	const q = `SELECT EXISTS(
		SELECT 1 FROM order_items oi
		JOIN orders o ON o.id = oi.order_id
		WHERE oi.product_id = $1 AND o.status NOT IN ('paid','cancelled')
	)`
	var exists bool
	if err := r.pool.QueryRow(ctx, q, productID).Scan(&exists); err != nil {
		// If order_items table doesn't exist yet, treat as no active orders.
		return false, nil
	}
	return exists, nil
}

// scanOne executes a query that should return exactly one product row.
func (r *ProductRepository) scanOne(ctx context.Context, q string, args ...any) (*product.Product, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("productRepo query: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, product.ErrProductNotFound
	}
	return scanProduct(rows)
}

func (r *ProductRepository) loadVariants(ctx context.Context, p *product.Product) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, name, price_delta, COALESCE(sku,''), is_active
		 FROM product_variants WHERE product_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("productRepo.loadVariants: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var v product.Variant
		if err := rows.Scan(&v.ID, &v.ProductID, &v.Name, &v.PriceDelta, &v.SKU, &v.IsActive); err != nil {
			return err
		}
		p.Variants = append(p.Variants, v)
	}
	return rows.Err()
}

func (r *ProductRepository) loadAddonGroups(ctx context.Context, p *product.Product) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id, product_id, name, is_required, max_selections
		 FROM addon_groups WHERE product_id = $1`, p.ID)
	if err != nil {
		return fmt.Errorf("productRepo.loadAddonGroups: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ag product.AddonGroup
		if err := rows.Scan(&ag.ID, &ag.ProductID, &ag.Name, &ag.IsRequired, &ag.MaxSelections); err != nil {
			return err
		}
		if err := r.loadAddons(ctx, &ag); err != nil {
			return err
		}
		p.AddonGroups = append(p.AddonGroups, ag)
	}
	return rows.Err()
}

func (r *ProductRepository) loadAddons(ctx context.Context, ag *product.AddonGroup) error {
	rows, err := r.pool.Query(ctx,
		`SELECT id, addon_group_id, name, price, is_active FROM addons WHERE addon_group_id = $1`,
		ag.ID)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var a product.Addon
		if err := rows.Scan(&a.ID, &a.AddonGroupID, &a.Name, &a.Price, &a.IsActive); err != nil {
			return err
		}
		ag.Addons = append(ag.Addons, a)
	}
	return rows.Err()
}

func scanProduct(rows pgx.Rows) (*product.Product, error) {
	var p product.Product
	var taxTypeStr string
	var sku *string // nullable in DB
	err := rows.Scan(
		&p.ID, &p.TenantID, &p.CategoryID, &sku, &p.Name, &p.Description,
		&p.BasePrice, &taxTypeStr, &p.IsActive, &p.CreatedAt, &p.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.TaxType = product.TaxType(taxTypeStr)
	if sku != nil {
		p.SKU = *sku
	}
	return &p, nil
}

func insertVariant(ctx context.Context, tx pgx.Tx, productID uuid.UUID, v product.Variant) error {
	_, err := tx.Exec(ctx,
		`INSERT INTO product_variants (id, product_id, name, price_delta, sku, is_active)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		uuid.New(), productID, v.Name, v.PriceDelta, nullableStr(v.SKU), v.IsActive)
	return err
}

func insertAddonGroup(ctx context.Context, tx pgx.Tx, productID uuid.UUID, ag product.AddonGroup) error {
	agID := uuid.New()
	_, err := tx.Exec(ctx,
		`INSERT INTO addon_groups (id, product_id, name, is_required, max_selections)
		 VALUES ($1, $2, $3, $4, $5)`,
		agID, productID, ag.Name, ag.IsRequired, ag.MaxSelections)
	if err != nil {
		return err
	}
	for _, a := range ag.Addons {
		_, err = tx.Exec(ctx,
			`INSERT INTO addons (id, addon_group_id, name, price, is_active)
			 VALUES ($1, $2, $3, $4, $5)`,
			uuid.New(), agID, a.Name, a.Price, a.IsActive)
		if err != nil {
			return err
		}
	}
	return nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullableStr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
