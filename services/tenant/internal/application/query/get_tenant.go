package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// GetTenantQuery carries the input for retrieving a tenant by ID.
type GetTenantQuery struct {
	TenantID uuid.UUID
}

// GetTenantHandler handles the GetTenant query.
type GetTenantHandler struct {
	repo domain.Repository
}

// NewGetTenantHandler constructs a GetTenantHandler.
func NewGetTenantHandler(repo domain.Repository) *GetTenantHandler {
	return &GetTenantHandler{repo: repo}
}

// Handle retrieves a tenant by ID.
func (h *GetTenantHandler) Handle(ctx context.Context, q GetTenantQuery) (*domain.Tenant, error) {
	t, err := h.repo.FindByID(ctx, q.TenantID)
	if err != nil {
		return nil, fmt.Errorf("GetTenantHandler.FindByID: %w", err)
	}
	return t, nil
}
