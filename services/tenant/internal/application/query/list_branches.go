package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// ListBranchesQuery carries the input for listing branches of a tenant.
type ListBranchesQuery struct {
	TenantID uuid.UUID
}

// ListBranchesHandler handles the ListBranches query.
type ListBranchesHandler struct {
	repo domain.Repository
}

// NewListBranchesHandler constructs a ListBranchesHandler.
func NewListBranchesHandler(repo domain.Repository) *ListBranchesHandler {
	return &ListBranchesHandler{repo: repo}
}

// Handle returns all branches for the given tenant.
func (h *ListBranchesHandler) Handle(ctx context.Context, q ListBranchesQuery) ([]domain.Branch, error) {
	branches, err := h.repo.ListBranches(ctx, q.TenantID)
	if err != nil {
		return nil, fmt.Errorf("ListBranchesHandler.ListBranches: %w", err)
	}
	return branches, nil
}
