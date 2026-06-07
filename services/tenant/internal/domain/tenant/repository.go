package tenant

import (
	"context"

	"github.com/google/uuid"
)

// Repository is the port for tenant persistence. Infrastructure implements this.
type Repository interface {
	Save(ctx context.Context, t *Tenant) error
	FindByID(ctx context.Context, id uuid.UUID) (*Tenant, error)
	FindBySlug(ctx context.Context, slug string) (*Tenant, error)
	ExistsBySlug(ctx context.Context, slug string) (bool, error)
	ListBranches(ctx context.Context, tenantID uuid.UUID) ([]Branch, error)
}
