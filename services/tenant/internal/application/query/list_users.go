package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// ListUsersHandler lists all users for a tenant.
type ListUsersHandler struct {
	repo user.Repository
}

// NewListUsersHandler creates a handler.
func NewListUsersHandler(repo user.Repository) *ListUsersHandler {
	return &ListUsersHandler{repo: repo}
}

// Handle returns all users in the tenant (RLS-enforced in repository).
func (h *ListUsersHandler) Handle(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error) {
	users, err := h.repo.FindByTenant(ctx, tenantID)
	if err != nil {
		return nil, fmt.Errorf("ListUsers: %w", err)
	}
	return users, nil
}
