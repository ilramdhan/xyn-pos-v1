package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// KeycloakCreateUserRequest is defined here (not in keycloak package) to avoid coupling.
type KeycloakCreateUserRequest struct {
	Email     string
	Password  string
	FirstName string
}

// KeycloakIntrospectResult is the parsed Keycloak introspection response.
type KeycloakIntrospectResult struct {
	Active  bool
	Subject string
	Email   string
}

// KeycloakClient is the port the application layer requires from Keycloak.
type KeycloakClient interface {
	CreateUser(ctx context.Context, adminToken string, req KeycloakCreateUserRequest) (string, error)
	Introspect(ctx context.Context, token string) (*KeycloakIntrospectResult, error)
}

// RegisterUserInput is the command payload.
type RegisterUserInput struct {
	TenantID       uuid.UUID
	Email          string
	Password       string
	FullName       string
	Role           user.Role
	BranchScope    []uuid.UUID
	IdempotencyKey string
}

// RegisterUserHandler orchestrates Keycloak user creation and local DB sync.
type RegisterUserHandler struct {
	repo       user.Repository
	keycloak   KeycloakClient
	adminToken string
}

// NewRegisterUserHandler creates a handler.
func NewRegisterUserHandler(repo user.Repository, keycloak KeycloakClient, adminToken string) *RegisterUserHandler {
	return &RegisterUserHandler{repo: repo, keycloak: keycloak, adminToken: adminToken}
}

// Handle registers a new user: checks uniqueness → Keycloak create → local save.
func (h *RegisterUserHandler) Handle(ctx context.Context, in RegisterUserInput) (*user.User, error) {
	_, err := h.repo.FindByEmail(ctx, in.TenantID, in.Email)
	if err == nil {
		return nil, fmt.Errorf("RegisterUser: %w", user.ErrEmailTaken)
	}
	if !errors.Is(err, user.ErrUserNotFound) {
		return nil, fmt.Errorf("RegisterUser FindByEmail: %w", err)
	}

	kcID, err := h.keycloak.CreateUser(ctx, h.adminToken, KeycloakCreateUserRequest{
		Email:     in.Email,
		Password:  in.Password,
		FirstName: in.FullName,
	})
	if err != nil {
		return nil, fmt.Errorf("RegisterUser keycloak.CreateUser: %w", err)
	}

	u, err := user.NewUser(in.TenantID, kcID, in.Email, in.FullName, in.Role, in.BranchScope)
	if err != nil {
		return nil, fmt.Errorf("RegisterUser domain: %w", err)
	}

	if err := h.repo.Save(ctx, u); err != nil {
		return nil, fmt.Errorf("RegisterUser save: %w", err)
	}
	return u, nil
}
