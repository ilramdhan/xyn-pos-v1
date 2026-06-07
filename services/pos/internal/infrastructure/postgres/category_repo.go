package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/pos/internal/domain/product"
)

// CategoryRepository implements product.CategoryRepository backed by PostgreSQL.
type CategoryRepository struct {
	pool *pgxpool.Pool
}

// NewCategoryRepository constructs a CategoryRepository.
func NewCategoryRepository(pool *pgxpool.Pool) *CategoryRepository {
	return &CategoryRepository{pool: pool}
}

func (r *CategoryRepository) FindByID(ctx context.Context, id uuid.UUID) (*product.Category, error) {
	const q = `SELECT id, tenant_id, name, sort_order, parent_id, is_active, created_at, updated_at
	            FROM categories WHERE id = $1`

	rows, err := r.pool.Query(ctx, q, id)
	if err != nil {
		return nil, fmt.Errorf("categoryRepo.FindByID: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, product.ErrCategoryNotFound
	}
	return scanCategory(rows)
}

func (r *CategoryRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*product.Category, error) {
	const q = `SELECT id, tenant_id, name, sort_order, parent_id, is_active, created_at, updated_at
	            FROM categories WHERE tenant_id = $1 ORDER BY sort_order ASC`

	rows, err := r.pool.Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("categoryRepo.FindByTenant: %w", err)
	}
	defer rows.Close()

	var result []*product.Category
	for rows.Next() {
		c, err := scanCategory(rows)
		if err != nil {
			return nil, fmt.Errorf("categoryRepo.FindByTenant scan: %w", err)
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (r *CategoryRepository) Save(ctx context.Context, c *product.Category) error {
	const q = `INSERT INTO categories (id, tenant_id, name, sort_order, parent_id, is_active, created_at, updated_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`

	_, err := r.pool.Exec(ctx, q,
		c.ID, c.TenantID, c.Name, c.SortOrder, c.ParentID, c.IsActive, c.CreatedAt, c.UpdatedAt)
	if err != nil {
		return fmt.Errorf("categoryRepo.Save: %w", err)
	}
	return nil
}

func (r *CategoryRepository) UpdateSortOrders(ctx context.Context, tenantID uuid.UUID, orderedIDs []uuid.UUID) error {
	// Update each category's sort_order based on its position in the slice.
	batch := &pgx.Batch{}
	for i, id := range orderedIDs {
		batch.Queue(`UPDATE categories SET sort_order=$1, updated_at=$2 WHERE id=$3 AND tenant_id=$4`,
			i, time.Now().UTC(), id, tenantID)
	}
	br := r.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range orderedIDs {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("categoryRepo.UpdateSortOrders: %w", err)
		}
	}
	return nil
}

func scanCategory(rows pgx.Rows) (*product.Category, error) {
	var c product.Category
	err := rows.Scan(
		&c.ID, &c.TenantID, &c.Name, &c.SortOrder, &c.ParentID, &c.IsActive, &c.CreatedAt, &c.UpdatedAt,
	)
	return &c, err
}
