package command_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// MockUserRepository stubs user.Repository for unit tests.
type MockUserRepository struct{ mock.Mock }

func (m *MockUserRepository) Save(ctx context.Context, u *user.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *MockUserRepository) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*user.User, error) {
	args := m.Called(ctx, tenantID, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}
func (m *MockUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}
func (m *MockUserRepository) FindByKeycloakID(ctx context.Context, kID string) (*user.User, error) {
	args := m.Called(ctx, kID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*user.User), args.Error(1)
}
func (m *MockUserRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error) {
	args := m.Called(ctx, tenantID)
	return args.Get(0).([]*user.User), args.Error(1)
}
func (m *MockUserRepository) Update(ctx context.Context, u *user.User) error {
	return m.Called(ctx, u).Error(0)
}
func (m *MockUserRepository) SavePIN(ctx context.Context, userID uuid.UUID, pinHash string) error {
	return m.Called(ctx, userID, pinHash).Error(0)
}
func (m *MockUserRepository) FindPINHash(ctx context.Context, userID uuid.UUID) (string, error) {
	args := m.Called(ctx, userID)
	return args.String(0), args.Error(1)
}

// MockKeycloakClient stubs Keycloak calls.
type MockKeycloakClient struct{ mock.Mock }

func (m *MockKeycloakClient) CreateUser(ctx context.Context, adminToken string, req command.KeycloakCreateUserRequest) (string, error) {
	args := m.Called(ctx, adminToken, req)
	return args.String(0), args.Error(1)
}
func (m *MockKeycloakClient) Introspect(ctx context.Context, token string) (*command.KeycloakIntrospectResult, error) {
	args := m.Called(ctx, token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*command.KeycloakIntrospectResult), args.Error(1)
}

func TestRegisterUser_EmailAlreadyTaken_ReturnsError(t *testing.T) {
	repo := &MockUserRepository{}
	kc := &MockKeycloakClient{}

	existing, _ := user.NewUser(uuid.New(), "kc-existing", "ani@example.com", "Ani", user.RoleCashier, nil)
	repo.On("FindByEmail", mock.Anything, mock.Anything, "ani@example.com").Return(existing, nil)

	h := command.NewRegisterUserHandler(repo, kc, "admin-token")
	_, err := h.Handle(context.Background(), command.RegisterUserInput{
		TenantID: uuid.New(), Email: "ani@example.com", Password: "pass123", FullName: "Ani2", Role: user.RoleCashier,
	})
	assert.ErrorIs(t, err, user.ErrEmailTaken)
}

func TestRegisterUser_Success_SavesUser(t *testing.T) {
	repo := &MockUserRepository{}
	kc := &MockKeycloakClient{}
	tenantID := uuid.New()

	repo.On("FindByEmail", mock.Anything, tenantID, "new@example.com").Return(nil, user.ErrUserNotFound)
	kc.On("CreateUser", mock.Anything, "admin-token", mock.Anything).Return("kc-new-123", nil)
	repo.On("Save", mock.Anything, mock.MatchedBy(func(u *user.User) bool {
		return u.Email == "new@example.com" && u.KeycloakID == "kc-new-123"
	})).Return(nil)

	h := command.NewRegisterUserHandler(repo, kc, "admin-token")
	result, err := h.Handle(context.Background(), command.RegisterUserInput{
		TenantID: tenantID, Email: "new@example.com", Password: "pass123", FullName: "New", Role: user.RoleManager,
	})
	require.NoError(t, err)
	assert.Equal(t, "new@example.com", result.Email)
}

// keep errors imported to avoid unused import if tests don't use it directly.
var _ = errors.New
