package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

type ListBranchesQuery struct {
	TenantID uuid.UUID
}

type ListBranchesHandler struct {
	repo domain.Repository
}

func NewListBranchesHandler(repo domain.Repository) *ListBranchesHandler {
	return &ListBranchesHandler{repo: repo}
}

func (h *ListBranchesHandler) Handle(ctx context.Context, q ListBranchesQuery) ([]domain.Branch, error) {
	branches, err := h.repo.ListBranches(ctx, q.TenantID)
	if err != nil {
		return nil, fmt.Errorf("ListBranchesHandler.ListBranches: %w", err)
	}
	return branches, nil
}
