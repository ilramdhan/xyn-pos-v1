package command

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// SetPINInput is the command payload.
type SetPINInput struct {
	UserID uuid.UUID
	PIN    string
}

// SetPINHandler stores a bcrypt-hashed PIN for a user.
type SetPINHandler struct {
	repo user.Repository
}

// NewSetPINHandler creates a handler.
func NewSetPINHandler(repo user.Repository) *SetPINHandler {
	return &SetPINHandler{repo: repo}
}

// Handle validates PIN length (4–6 digits), hashes with bcrypt cost=12, and saves.
func (h *SetPINHandler) Handle(ctx context.Context, in SetPINInput) error {
	if len(in.PIN) < 4 || len(in.PIN) > 6 {
		return user.ErrInvalidPIN
	}

	if _, err := h.repo.FindByID(ctx, in.UserID); err != nil {
		return fmt.Errorf("SetPIN FindByID: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.PIN), 12)
	if err != nil {
		return fmt.Errorf("SetPIN bcrypt: %w", err)
	}

	if err := h.repo.SavePIN(ctx, in.UserID, string(hash)); err != nil {
		return fmt.Errorf("SetPIN SavePIN: %w", err)
	}
	return nil
}
