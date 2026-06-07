package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	tenantv1 "github.com/xyn-pos/gen/tenant/v1"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

// Server wraps the gRPC server with lifecycle management.
type Server struct {
	grpc *grpc.Server
	port int
}

// NewServer builds the gRPC server with middleware chain.
// Pass nil for verifyFn to skip auth (Phase 3 / test scenarios).
func NewServer(port int, tenantHandler *TenantHandler, userHandler *UserHandler, verifyFn sharedauth.VerifyFunc) *Server {
	opts := middleware.Chain(
		middleware.Auth(verifyFn),
		middleware.Recovery(),
	)
	srv := grpc.NewServer(opts...)

	if userHandler != nil {
		tenantv1.RegisterUserServiceServer(srv, userHandler)
	}

	// TenantServiceServer registration will be added when proto handler is wired.
	_ = tenantHandler

	return &Server{grpc: srv, port: port}
}

// Start begins accepting connections. Blocks until the server stops.
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", s.port))
	if err != nil {
		return fmt.Errorf("grpc.Server.Start listen port=%d: %w", s.port, err)
	}
	return s.grpc.Serve(lis)
}

// GracefulStop drains active RPCs and stops the server.
func (s *Server) GracefulStop() { s.grpc.GracefulStop() }
