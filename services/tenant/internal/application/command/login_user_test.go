package command_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

func TestLoginUser_InvalidKeycloakToken_ReturnsError(t *testing.T) {
	repo := &MockUserRepository{}
	kc := &MockKeycloakClient{}
	kc.On("Introspect", mock.Anything, "bad-token").Return(nil, fmt.Errorf("token inactive"))

	h := command.NewLoginUserHandler(repo, kc, "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f")
	_, err := h.Handle(context.Background(), command.LoginInput{KeycloakToken: "bad-token"})
	assert.Error(t, err)
}

func TestLoginUser_UserInactive_ReturnsError(t *testing.T) {
	repo := &MockUserRepository{}
	kc := &MockKeycloakClient{}

	kc.On("Introspect", mock.Anything, "valid-token").Return(&command.KeycloakIntrospectResult{
		Active: true, Subject: "kc-inactive",
	}, nil)

	u, _ := user.NewUser(uuid.New(), "kc-inactive", "inactive@example.com", "Inactive", user.RoleCashier, nil)
	_ = u.Deactivate()
	repo.On("FindByKeycloakID", mock.Anything, "kc-inactive").Return(u, nil)

	h := command.NewLoginUserHandler(repo, kc, "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f")
	_, err := h.Handle(context.Background(), command.LoginInput{KeycloakToken: "valid-token"})
	assert.ErrorIs(t, err, user.ErrUserInactive)
}

func TestLoginUser_ValidToken_ReturnsPASETOToken(t *testing.T) {
	repo := &MockUserRepository{}
	kc := &MockKeycloakClient{}

	tenantID := uuid.New()
	kc.On("Introspect", mock.Anything, "valid-token").Return(&command.KeycloakIntrospectResult{
		Active: true, Subject: "kc-active",
	}, nil)

	u, _ := user.NewUser(tenantID, "kc-active", "active@example.com", "Active", user.RoleCashier, nil)
	repo.On("FindByKeycloakID", mock.Anything, "kc-active").Return(u, nil)

	keyHex := "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"
	h := command.NewLoginUserHandler(repo, kc, keyHex)
	result, err := h.Handle(context.Background(), command.LoginInput{KeycloakToken: "valid-token"})
	require.NoError(t, err)
	assert.NotEmpty(t, result.PASETOToken)
	assert.NotEmpty(t, result.JTI)
}
