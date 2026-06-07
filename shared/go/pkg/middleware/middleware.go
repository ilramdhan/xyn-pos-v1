package middleware

import (
	"context"
	"log/slog"
	"runtime/debug"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
)

type contextKey string

const claimsKey contextKey = "claims"

// ClaimsFromContext retrieves verified claims from the context.
func ClaimsFromContext(ctx context.Context) (*sharedauth.Claims, bool) {
	v, ok := ctx.Value(claimsKey).(*sharedauth.Claims)
	return v, ok
}

// Auth returns a gRPC unary interceptor that verifies a PASETO token
// from the "authorization" metadata header. Pass nil verifyFn to skip auth (Phase 3).
func Auth(verifyFn sharedauth.VerifyFunc) grpc.ServerOption {
	return grpc.UnaryInterceptor(func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (any, error) {
		if verifyFn == nil {
			return handler(ctx, req)
		}

		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.Unauthenticated, "missing metadata")
		}

		tokens := md.Get("authorization")
		if len(tokens) == 0 {
			return nil, status.Error(codes.Unauthenticated, "missing authorization token")
		}

		claims, err := verifyFn(tokens[0])
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "invalid token")
		}

		ctx = context.WithValue(ctx, claimsKey, claims)
		return handler(ctx, req)
	})
}

// Recovery returns a gRPC unary interceptor that recovers from panics.
func Recovery() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(func(
		ctx context.Context,
		req any,
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp any, err error) {
		defer func() {
			if r := recover(); r != nil {
				slog.ErrorContext(ctx, "panic recovered", "method", info.FullMethod, "panic", r, "stack", string(debug.Stack()))
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	})
}

// Chain returns a slice of gRPC server options to pass to grpc.NewServer.
// Pass nil for verifyFn during Phase 3 to disable auth.
func Chain(opts ...grpc.ServerOption) []grpc.ServerOption {
	return opts
}
