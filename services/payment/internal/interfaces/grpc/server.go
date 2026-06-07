package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	paymentv1 "github.com/xyn-pos/gen/payment/v1"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	"github.com/xyn-pos/shared/pkg/middleware"
)

// Server wraps the gRPC server.
type Server struct {
	grpc *grpc.Server
	port string
}

// NewServer builds a gRPC server with auth + recovery middleware and the PaymentHandler registered.
// Pass nil for verifyFn to skip auth (dev/test scenarios).
func NewServer(port string, handler *PaymentHandler, verifyFn sharedauth.VerifyFunc) *Server {
	opts := middleware.Chain(
		middleware.Auth(verifyFn),
		middleware.Recovery(),
	)
	srv := grpc.NewServer(opts...)
	paymentv1.RegisterPaymentServiceServer(srv, handler)
	return &Server{grpc: srv, port: port}
}

// Serve starts listening on the configured port.
func (s *Server) Serve() error {
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", s.port))
	if err != nil {
		return fmt.Errorf("grpc.Server.Listen: %w", err)
	}
	return s.grpc.Serve(lis)
}

// GracefulStop stops accepting new connections and waits for in-flight RPCs.
func (s *Server) GracefulStop() {
	s.grpc.GracefulStop()
}
