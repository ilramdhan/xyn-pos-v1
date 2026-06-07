package product

import (
	"context"

	"github.com/google/uuid"
)

// ProductRepository is the persistence port for Product aggregates.
type ProductRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Product, error)
	FindBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*Product, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID, filter Filter) ([]*Product, int, error)
	Save(ctx context.Context, p *Product) error
	Update(ctx context.Context, p *Product) error
	SetBranchPrice(ctx context.Context, bp BranchProduct) error
	HasActiveOrders(ctx context.Context, productID uuid.UUID) (bool, error)
}

// CategoryRepository is the persistence port for Category entities.
type CategoryRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*Category, error)
	FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*Category, error)
	Save(ctx context.Context, c *Category) error
	UpdateSortOrders(ctx context.Context, tenantID uuid.UUID, orderedIDs []uuid.UUID) error
}

// Filter for list queries.
type Filter struct {
	CategoryID *uuid.UUID
	BranchID   *uuid.UUID // include branch override prices
	ActiveOnly bool
	Limit      int
	Offset     int
}
