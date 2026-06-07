package grpc

import (
	"fmt"
	"net"

	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
	"google.golang.org/grpc"
)

// Server wraps the gRPC server with lifecycle management.
type Server struct {
	grpc *grpc.Server
	port int
}

// NewServer builds the gRPC server with middleware chain.
// Pass nil for verifyFn in Phase 3 (no auth yet).
func NewServer(port int, handler *TenantHandler, verifyFn sharedauth.VerifyFunc) *Server {
	opts := middleware.Chain(
		middleware.Auth(verifyFn),
		middleware.Recovery(),
	)
	srv := grpc.NewServer(opts...)

	// Register proto handler — uncomment after running make proto-gen:
	// tenantpb.RegisterTenantServiceServer(srv, handler)

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
