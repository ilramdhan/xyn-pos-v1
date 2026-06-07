package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	posv1 "github.com/xyn-pos/gen/pos/v1"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

// Server wraps the gRPC server with lifecycle management.
type Server struct {
	grpc *grpc.Server
	port int
}

// NewServer builds the gRPC server with auth + recovery middleware.
// Pass nil for verifyFn to skip auth (test scenarios).
func NewServer(port int, productHandler *ProductHandler, verifyFn sharedauth.VerifyFunc) *Server {
	opts := middleware.Chain(
		middleware.Auth(verifyFn),
		middleware.Recovery(),
	)
	srv := grpc.NewServer(opts...)

	if productHandler != nil {
		posv1.RegisterProductServiceServer(srv, productHandler)
	}

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
