package grpc

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

// extractTenantIDFromContext retrieves the tenant ID from verified claims in ctx.
func extractTenantIDFromContext(ctx context.Context) (uuid.UUID, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return uuid.Nil, fmt.Errorf("no claims in context")
	}
	return claims.TenantID, nil
}

// extractClaimsFromContext retrieves the full verified claims from ctx.
func extractClaimsFromContext(ctx context.Context) (*sharedauth.Claims, error) {
	claims, ok := middleware.ClaimsFromContext(ctx)
	if !ok || claims == nil {
		return nil, fmt.Errorf("no claims in context")
	}
	return claims, nil
}
