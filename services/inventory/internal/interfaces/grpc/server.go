package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	inventoryv1 "github.com/xyn-pos/gen/inventory/v1"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

// Server wraps the gRPC server with lifecycle management.
type Server struct {
	grpc *grpc.Server
	port string
}

// NewServer builds the gRPC server with auth + recovery middleware.
// Pass nil for verifyFn to skip auth (dev/test scenarios).
func NewServer(port string, handler *InventoryHandler, verifyFn sharedauth.VerifyFunc) *Server {
	opts := middleware.Chain(
		middleware.Auth(verifyFn),
		middleware.Recovery(),
	)
	srv := grpc.NewServer(opts...)
	inventoryv1.RegisterInventoryServiceServer(srv, handler)
	return &Server{grpc: srv, port: port}
}

// Serve begins accepting connections. Blocks until the server stops.
func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		return fmt.Errorf("grpc.Server.Listen port=%s: %w", s.port, err)
	}
	return s.grpc.Serve(lis)
}

// GracefulStop drains active RPCs and stops the server.
func (s *Server) GracefulStop() { s.grpc.GracefulStop() }
