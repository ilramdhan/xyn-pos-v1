package command

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// LoginInput is the command payload.
type LoginInput struct {
	KeycloakToken string
}

// LoginResult contains the issued PASETO token.
type LoginResult struct {
	PASETOToken string
	UserID      uuid.UUID
	JTI         string
	ExpiresAt   time.Time
}

// LoginUserHandler exchanges a Keycloak token for a PASETO token.
type LoginUserHandler struct {
	repo     user.Repository
	keycloak KeycloakClient
	issuer   *sharedauth.Issuer
}

// NewLoginUserHandler creates a handler. keyHex is the 32-byte hex PASETO symmetric key.
func NewLoginUserHandler(repo user.Repository, keycloak KeycloakClient, keyHex string) *LoginUserHandler {
	issuer, _ := sharedauth.NewLocalIssuer(keyHex)
	return &LoginUserHandler{repo: repo, keycloak: keycloak, issuer: issuer}
}

// Handle verifies the Keycloak token, looks up the user, and issues a PASETO token.
func (h *LoginUserHandler) Handle(ctx context.Context, in LoginInput) (*LoginResult, error) {
	kcResult, err := h.keycloak.Introspect(ctx, in.KeycloakToken)
	if err != nil {
		return nil, fmt.Errorf("LoginUser introspect: %w", err)
	}

	u, err := h.repo.FindByKeycloakID(ctx, kcResult.Subject)
	if err != nil {
		return nil, fmt.Errorf("LoginUser FindByKeycloakID: %w", err)
	}

	if !u.IsActive {
		return nil, fmt.Errorf("LoginUser: %w", user.ErrUserInactive)
	}

	jti := uuid.NewString()
	expiresAt := time.Now().Add(8 * time.Hour)
	token, err := h.issuer.Issue(sharedauth.Claims{
		TenantID:    u.TenantID,
		UserID:      u.ID,
		Role:        string(u.Role),
		Email:       u.Email,
		JTI:         jti,
		BranchScope: u.BranchScope,
	}, 8*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("LoginUser issue: %w", err)
	}

	return &LoginResult{
		PASETOToken: token,
		UserID:      u.ID,
		JTI:         jti,
		ExpiresAt:   expiresAt,
	}, nil
}
