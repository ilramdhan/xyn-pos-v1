package auth_test

import (
	"testing"
	"time"

	"aidanwoods.dev/go-paseto"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
)

func makeToken(t *testing.T, keyHex string, tenantID, userID uuid.UUID, role string, expiry time.Time) string {
	t.Helper()
	key, err := paseto.V4SymmetricKeyFromHex(keyHex)
	require.NoError(t, err)

	tok := paseto.NewToken()
	tok.SetIssuedAt(time.Now())
	tok.SetIssuer("xyn-pos")
	tok.SetExpiration(expiry)
	tok.SetString("tenant_id", tenantID.String())
	tok.SetString("user_id", userID.String())
	tok.SetString("role", role)

	return tok.V4Encrypt(key, nil)
}

func TestVerify_ValidToken(t *testing.T) {
	keyHex := "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"
	tenantID := uuid.New()
	userID := uuid.New()

	token := makeToken(t, keyHex, tenantID, userID, "admin", time.Now().Add(time.Hour))

	verify, err := sharedauth.NewLocalVerifier(keyHex)
	require.NoError(t, err)

	claims, err := verify(token)
	require.NoError(t, err)
	assert.Equal(t, tenantID, claims.TenantID)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, "admin", claims.Role)
}

func TestVerify_ExpiredToken(t *testing.T) {
	keyHex := "707172737475767778797a7b7c7d7e7f808182838485868788898a8b8c8d8e8f"
	token := makeToken(t, keyHex, uuid.New(), uuid.New(), "admin", time.Now().Add(-time.Hour))

	verify, err := sharedauth.NewLocalVerifier(keyHex)
	require.NoError(t, err)

	_, err = verify(token)
	assert.Error(t, err, "expired token should fail verification")
}

func TestVerify_InvalidKey(t *testing.T) {
	_, err := sharedauth.NewLocalVerifier("not-a-valid-hex-key")
	assert.Error(t, err)
}
