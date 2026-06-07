package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// DeactivateUserInput is the command payload.
type DeactivateUserInput struct {
	UserID          uuid.UUID
	RequestedByRole user.Role
}

// DeactivateUserHandler soft-deletes a user (sets is_active=false).
type DeactivateUserHandler struct {
	repo user.Repository
}

// NewDeactivateUserHandler creates a handler.
func NewDeactivateUserHandler(repo user.Repository) *DeactivateUserHandler {
	return &DeactivateUserHandler{repo: repo}
}

// Handle deactivates the user. Only the owner role is permitted.
func (h *DeactivateUserHandler) Handle(ctx context.Context, in DeactivateUserInput) error {
	if in.RequestedByRole != user.RoleOwner {
		return fmt.Errorf("DeactivateUser: %w", user.ErrUnauthorized)
	}

	u, err := h.repo.FindByID(ctx, in.UserID)
	if err != nil {
		return fmt.Errorf("DeactivateUser FindByID: %w", err)
	}

	if err := u.Deactivate(); err != nil {
		return fmt.Errorf("DeactivateUser deactivate: %w", err)
	}

	if err := h.repo.Update(ctx, u); err != nil {
		return fmt.Errorf("DeactivateUser update: %w", err)
	}
	return nil
}
