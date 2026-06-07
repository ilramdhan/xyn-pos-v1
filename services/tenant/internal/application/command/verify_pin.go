package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// VerifyPINInput is the command payload.
type VerifyPINInput struct {
	UserID uuid.UUID
	PIN    string
}

// VerifyPINHandler checks a PIN against the stored bcrypt hash.
type VerifyPINHandler struct {
	repo user.Repository
}

// NewVerifyPINHandler creates a handler.
func NewVerifyPINHandler(repo user.Repository) *VerifyPINHandler {
	return &VerifyPINHandler{repo: repo}
}

// Handle returns true if the PIN matches the stored hash.
func (h *VerifyPINHandler) Handle(ctx context.Context, in VerifyPINInput) (bool, error) {
	hash, err := h.repo.FindPINHash(ctx, in.UserID)
	if err != nil {
		return false, fmt.Errorf("VerifyPIN FindPINHash: %w", err)
	}

	err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.PIN))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("VerifyPIN compare: %w", err)
	}
	return true, nil
}
