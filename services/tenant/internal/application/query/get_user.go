package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// GetUserHandler retrieves a single user by ID.
type GetUserHandler struct {
	repo user.Repository
}

// NewGetUserHandler creates a handler.
func NewGetUserHandler(repo user.Repository) *GetUserHandler {
	return &GetUserHandler{repo: repo}
}

// Handle returns the user or an error wrapping ErrUserNotFound.
func (h *GetUserHandler) Handle(ctx context.Context, userID uuid.UUID) (*user.User, error) {
	u, err := h.repo.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("GetUser: %w", err)
	}
	return u, nil
}
