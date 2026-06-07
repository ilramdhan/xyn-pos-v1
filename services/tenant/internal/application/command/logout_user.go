package command

import (
	"context"
	"fmt"
	"time"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
)

// TokenBlacklister is the port for revoking PASETO tokens by JTI.
type TokenBlacklister interface {
	Blacklist(ctx context.Context, jti string, ttl time.Duration) error
}

// LogoutInput is the command payload.
type LogoutInput struct {
	PASETOToken string
}

// LogoutHandler revokes a PASETO token by blacklisting its JTI.
type LogoutHandler struct {
	verify    sharedauth.VerifyFunc
	blacklist TokenBlacklister
}

// NewLogoutHandler creates a handler.
func NewLogoutHandler(verify sharedauth.VerifyFunc, blacklist TokenBlacklister) *LogoutHandler {
	return &LogoutHandler{verify: verify, blacklist: blacklist}
}

// Handle validates the token, extracts JTI, and stores it with remaining TTL.
func (h *LogoutHandler) Handle(ctx context.Context, in LogoutInput) error {
	claims, err := h.verify(in.PASETOToken)
	if err != nil {
		return fmt.Errorf("logout verify: %w", err)
	}

	ttl := time.Until(claims.Expiry)
	if ttl <= 0 {
		return nil
	}

	if err := h.blacklist.Blacklist(ctx, claims.JTI, ttl); err != nil {
		return fmt.Errorf("logout blacklist: %w", err)
	}
	return nil
}
