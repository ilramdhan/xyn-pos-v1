package command_test

import (
	"context"
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
)

type MockTokenBlacklister struct{ mock.Mock }

func (m *MockTokenBlacklister) Blacklist(ctx context.Context, jti string, ttl time.Duration) error {
	return m.Called(ctx, jti, mock.Anything).Error(0)
}

func makeTokenWithJTI(t *testing.T, keyHex, jti string) string {
	t.Helper()
	key, _ := paseto.V4SymmetricKeyFromHex(keyHex)
	tok := paseto.NewToken()
	tok.SetIssuedAt(time.Now())
	tok.SetIssuer("xyn-pos")
	tok.SetExpiration(time.Now().Add(time.Hour))
	tok.SetString("tenant_id", uuid.NewString())
	tok.SetString("user_id", uuid.NewString())
	tok.SetString("jti", jti)
	tok.SetString("role", "cashier")
	tok.SetString("branch_scope", "")
	return tok.V4Encrypt(key, nil)
}

func TestLogoutUser_BlacklistsJTI(t *testing.T) {
	keyHex := "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"
	verify, _ := sharedauth.NewLocalVerifier(keyHex)
	blacklist := &MockTokenBlacklister{}
	blacklist.On("Blacklist", mock.Anything, "test-jti-123", mock.Anything).Return(nil)

	token := makeTokenWithJTI(t, keyHex, "test-jti-123")
	h := command.NewLogoutHandler(verify, blacklist)
	err := h.Handle(context.Background(), command.LogoutInput{PASETOToken: token})
	require.NoError(t, err)
	blacklist.AssertExpectations(t)
}

func TestDeactivateUser_NotOwner_ReturnsUnauthorized(t *testing.T) {
	repo := &MockUserRepository{}
	h := command.NewDeactivateUserHandler(repo)
	err := h.Handle(context.Background(), command.DeactivateUserInput{
		UserID:          uuid.New(),
		RequestedByRole: "cashier",
	})
	assert.Error(t, err)
}

func TestDeactivateUser_Owner_DeactivatesUser(t *testing.T) {
	repo := &MockUserRepository{}
	userID := uuid.New()

	u, _ := user.NewUser(uuid.New(), "kc-id", "owner@example.com", "Owner", user.RoleOwner, nil)
	repo.On("FindByID", mock.Anything, userID).Return(u, nil)
	repo.On("Update", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return !u.IsActive
	})).Return(nil)

	h := command.NewDeactivateUserHandler(repo)
	err := h.Handle(context.Background(), command.DeactivateUserInput{
		UserID:          userID,
		RequestedByRole: user.RoleOwner,
	})
	assert.NoError(t, err)
	repo.AssertExpectations(t)
}
