package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type GetTenantQuery struct {
	TenantID uuid.UUID
}

type GetTenantHandler struct {
	repo domain.Repository
}

func NewGetTenantHandler(repo domain.Repository) *GetTenantHandler {
	return &GetTenantHandler{repo: repo}
}

func (h *GetTenantHandler) Handle(ctx context.Context, q GetTenantQuery) (*domain.Tenant, error) {
	t, err := h.repo.FindByID(ctx, q.TenantID)
	if err != nil {
		return nil, fmt.Errorf("GetTenantHandler.FindByID: %w", err)
	}
	return t, nil
}
