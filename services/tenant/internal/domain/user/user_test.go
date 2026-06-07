package user_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

func TestNewUser_InvalidRole_ReturnsError(t *testing.T) {
	_, err := user.NewUser(uuid.New(), "kc-id", "a@b.com", "Test", "invalid_role", nil)
	assert.ErrorIs(t, err, user.ErrInvalidRole)
}

func TestNewUser_EmptyEmail_ReturnsError(t *testing.T) {
	_, err := user.NewUser(uuid.New(), "kc-id", "", "Test", user.RoleCashier, nil)
	assert.ErrorIs(t, err, user.ErrInvalidEmail)
}

func TestNewUser_Valid_SetsIsActiveTrue(t *testing.T) {
	u, err := user.NewUser(uuid.New(), "kc-001", "cashier@example.com", "Ani", user.RoleCashier, nil)
	require.NoError(t, err)
	assert.True(t, u.IsActive)
	assert.NotEqual(t, uuid.Nil, u.ID)
}

func TestUser_Deactivate_AlreadyInactive_ReturnsError(t *testing.T) {
	u, _ := user.NewUser(uuid.New(), "kc-id", "a@b.com", "Test", user.RoleCashier, nil)
	_ = u.Deactivate()
	err := u.Deactivate()
	assert.ErrorIs(t, err, user.ErrUserInactive)
}

func TestUser_CanAccessBranch_NilScope_AllowsAll(t *testing.T) {
	u, _ := user.NewUser(uuid.New(), "kc-id", "a@b.com", "Test", user.RoleOwner, nil)
	assert.True(t, u.CanAccessBranch(uuid.New()))
	assert.True(t, u.CanAccessBranch(uuid.New()))
}

func TestUser_CanAccessBranch_ScopedBranch_BlocksOther(t *testing.T) {
	branchA := uuid.New()
	u, _ := user.NewUser(uuid.New(), "kc-id", "a@b.com", "Test", user.RoleCashier, []uuid.UUID{branchA})
	assert.True(t, u.CanAccessBranch(branchA))
	assert.False(t, u.CanAccessBranch(uuid.New()))
}

func TestNewUser_PopEvents_ContainsUserRegisteredEvent(t *testing.T) {
	u, _ := user.NewUser(uuid.New(), "kc-id", "a@b.com", "Test", user.RoleCashier, nil)
	evs := u.PopEvents()
	require.Len(t, evs, 1)
	_, ok := evs[0].(user.UserRegisteredEvent)
	assert.True(t, ok)
}

func TestUser_PopEvents_ClearsAfterCall(t *testing.T) {
	u, _ := user.NewUser(uuid.New(), "kc-id", "a@b.com", "Test", user.RoleCashier, nil)
	_ = u.PopEvents()
	assert.Empty(t, u.PopEvents())
}
