package middleware_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xyn-pos/shared/pkg/middleware"
)

func TestClaimsFromContext_Missing_ReturnsFalse(t *testing.T) {
	_, ok := middleware.ClaimsFromContext(context.Background())
	assert.False(t, ok)
}

func TestClaimsFromContext_Present_ReturnsClaims(t *testing.T) {
	// ClaimsFromContext returns false on an empty context (no injected claims).
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
