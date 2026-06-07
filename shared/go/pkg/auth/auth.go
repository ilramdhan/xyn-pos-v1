package auth

import (
	"fmt"
	"strings"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
)

// Claims extracted from or issued into a PASETO v4 local token.
type Claims struct {
	TenantID    uuid.UUID
	UserID      uuid.UUID
	Role        string
	Email       string
	JTI         string
	BranchScope []uuid.UUID
	IssuedAt    time.Time
	Expiry      time.Time
}

// VerifyFunc is the signature expected by gRPC middleware.
type VerifyFunc func(token string) (*Claims, error)

// Issuer creates PASETO v4 local tokens.
type Issuer struct {
	key paseto.V4SymmetricKey
}

// NewLocalIssuer returns an Issuer for the given 32-byte hex symmetric key.
func NewLocalIssuer(keyHex string) (*Issuer, error) {
	key, err := paseto.V4SymmetricKeyFromHex(keyHex)
	if err != nil {
		return nil, fmt.Errorf("auth.NewLocalIssuer: invalid key: %w", err)
	}
	return &Issuer{key: key}, nil
}

// Issue signs and encrypts claims into a PASETO v4 local token valid for ttl.
func (i *Issuer) Issue(claims Claims, ttl time.Duration) (string, error) {
	now := time.Now().UTC()
	tok := paseto.NewToken()
	tok.SetIssuedAt(now)
	tok.SetIssuer("xyn-pos")
	tok.SetExpiration(now.Add(ttl))
	tok.SetString("tenant_id", claims.TenantID.String())
	tok.SetString("user_id", claims.UserID.String())
	tok.SetString("role", claims.Role)
	tok.SetString("email", claims.Email)
	tok.SetString("jti", claims.JTI)

	scope := make([]string, len(claims.BranchScope))
	for idx, id := range claims.BranchScope {
		scope[idx] = id.String()
	}
	tok.SetString("branch_scope", strings.Join(scope, ","))

	return tok.V4Encrypt(i.key, nil), nil
}

// NewLocalVerifier returns a VerifyFunc that verifies PASETO v4 local tokens.
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

		tenantIDStr, _ := tok.GetString("tenant_id")
		tenantID, err := uuid.Parse(tenantIDStr)
		if err != nil {
			return nil, fmt.Errorf("auth.Verify: invalid tenant_id: %w", err)
		}

		userIDStr, _ := tok.GetString("user_id")
		userID, err := uuid.Parse(userIDStr)
		if err != nil {
			return nil, fmt.Errorf("auth.Verify: invalid user_id: %w", err)
		}

		role, _ := tok.GetString("role")
		email, _ := tok.GetString("email")
		jti, _ := tok.GetString("jti")
		issuedAt, _ := tok.GetIssuedAt()
		expiry, _ := tok.GetExpiration()

		var branchScope []uuid.UUID
		if scopeStr, _ := tok.GetString("branch_scope"); scopeStr != "" {
			for s := range strings.SplitSeq(scopeStr, ",") {
				if id, err := uuid.Parse(s); err == nil {
					branchScope = append(branchScope, id)
				}
			}
		}

		return &Claims{
			TenantID:    tenantID,
			UserID:      userID,
			Role:        role,
			Email:       email,
			JTI:         jti,
			BranchScope: branchScope,
			IssuedAt:    issuedAt,
			Expiry:      expiry,
		}, nil
	}, nil
}
