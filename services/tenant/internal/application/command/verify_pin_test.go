package command_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/xyn-pos/services/tenant/internal/application/command"
	"golang.org/x/crypto/bcrypt"
)

func TestVerifyPIN_WrongPIN_ReturnsFalse(t *testing.T) {
	repo := &MockUserRepository{}
	userID := uuid.New()

	hash, _ := bcrypt.GenerateFromPassword([]byte("1234"), 12)
	repo.On("FindPINHash", mock.Anything, userID).Return(string(hash), nil)

	h := command.NewVerifyPINHandler(repo)
	valid, err := h.Handle(context.Background(), command.VerifyPINInput{UserID: userID, PIN: "9999"})
	assert.NoError(t, err)
	assert.False(t, valid)
}

func TestVerifyPIN_CorrectPIN_ReturnsTrue(t *testing.T) {
	repo := &MockUserRepository{}
	userID := uuid.New()

	hash, _ := bcrypt.GenerateFromPassword([]byte("5678"), 12)
	repo.On("FindPINHash", mock.Anything, userID).Return(string(hash), nil)

	h := command.NewVerifyPINHandler(repo)
	valid, err := h.Handle(context.Background(), command.VerifyPINInput{UserID: userID, PIN: "5678"})
	assert.NoError(t, err)
	assert.True(t, valid)
}
