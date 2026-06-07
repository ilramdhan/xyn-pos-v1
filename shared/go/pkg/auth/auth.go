package auth

import (
	"fmt"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

// Claims extracted from a verified PASETO v4 token.
type Claims struct {
	TenantID uuid.UUID
	UserID   uuid.UUID
	Role     string
	Email    string
	IssuedAt time.Time
	Expiry   time.Time
}

// VerifyFunc is the signature expected by gRPC middleware.
type VerifyFunc func(token string) (*Claims, error)

// NewLocalVerifier returns a VerifyFunc that verifies PASETO v4 local tokens
// using the provided symmetric key (32 bytes hex or raw).
func NewLocalVerifier(keyHex string) (VerifyFunc, error) {
	key, err := paseto.V4SymmetricKeyFromHex(keyHex)
	if err != nil {
		return nil, fmt.Errorf("auth.NewLocalVerifier: invalid key: %w", err)
	}

	parser := paseto.NewParser()
	parser.AddRule(paseto.NotExpired())
	parser.AddRule(paseto.IssuedBy("xyn-pos"))

	return func(token string) (*Claims, error) {
		tok, err := parser.ParseV4Local(key, token, nil)
		if err != nil {
			return nil, fmt.Errorf("auth.Verify: %w", err)
		}

		tenantIDStr, err := tok.GetString("tenant_id")
		if err != nil {
			return nil, fmt.Errorf("auth.Verify: missing tenant_id: %w", err)
		}
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return nil, fmt.Errorf("auth.Verify: invalid tenant_id: %w", err)
		}

		userIDStr, err := tok.GetString("user_id")
		if err != nil {
			return nil, fmt.Errorf("auth.Verify: missing user_id: %w", err)
		}
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("auth.Verify: invalid user_id: %w", err)
		}

		role, _ := tok.GetString("role")
		email, _ := tok.GetString("email")
		issuedAt, _ := tok.GetIssuedAt()
		expiry, _ := tok.GetExpiration()

		return &Claims{
			TenantID: tenantID,
			UserID:   userID,
			Role:     role,
			Email:    email,
			IssuedAt: issuedAt,
			Expiry:   expiry,
		}, nil
	}, nil
}
