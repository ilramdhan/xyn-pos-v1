package grpc

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	paymentv1 "github.com/xyn-pos/gen/go/payment/v1"
)

// Server wraps the gRPC server.
type Server struct {
	grpc *grpc.Server
	port string
}

// NewServer builds a gRPC server with the PaymentHandler registered.
func NewServer(port string, handler *PaymentHandler) *Server {
	srv := grpc.NewServer()
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
