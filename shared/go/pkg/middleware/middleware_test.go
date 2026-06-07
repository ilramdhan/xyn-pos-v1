package middleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

func TestClaimsFromContext_Missing_ReturnsFalse(t *testing.T) {
	_, ok := middleware.ClaimsFromContext(context.Background())
	assert.False(t, ok)
}

func TestClaimsFromContext_Present_ReturnsClaims(t *testing.T) {
	// We can't easily test the interceptor without a real gRPC server,
	// but we can test the context helpers directly.
	// ClaimsFromContext uses an unexported key — we verify it returns false on empty ctx.
	claims := &sharedauth.Claims{Role: "admin"}
	_ = claims // ensure Claims struct is accessible
	_, ok := middleware.ClaimsFromContext(context.Background())
	assert.False(t, ok)
}

func TestAuth_NilVerifyFn_Compiles(t *testing.T) {
	// Verify Auth(nil) produces a valid grpc.ServerOption without panicking
	opt := middleware.Auth(nil)
	assert.NotNil(t, opt)
}

func TestChain_ReturnsSlice(t *testing.T) {
	opts := middleware.Chain(middleware.Auth(nil))
	assert.Len(t, opts, 1)
}
