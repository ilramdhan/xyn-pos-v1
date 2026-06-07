package command_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

func TestSetPIN_TooShort_ReturnsError(t *testing.T) {
	h := command.NewSetPINHandler(&MockUserRepository{})
	err := h.Handle(context.Background(), command.SetPINInput{UserID: uuid.New(), PIN: "12"})
	assert.ErrorIs(t, err, user.ErrInvalidPIN)
}

func TestSetPIN_HashIsStored_NotPlaintext(t *testing.T) {
	repo := &MockUserRepository{}
	userID := uuid.New()

	u, _ := user.NewUser(uuid.New(), "kc-id", "user@example.com", "User", user.RoleCashier, nil)
	repo.On("FindByID", mock.Anything, userID).Return(u, nil)
	repo.On("SavePIN", mock.Anything, userID, mock.MatchedBy(func(hash string) bool {
		return len(hash) > 20 && hash[:4] == "$2a$"
	})).Return(nil)

	h := command.NewSetPINHandler(repo)
	err := h.Handle(context.Background(), command.SetPINInput{UserID: userID, PIN: "1234"})
	require.NoError(t, err)
	repo.AssertExpectations(t)
}
