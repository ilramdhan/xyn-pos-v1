package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// HTTPServer wraps the standard http.Server for the webhook endpoint.
type HTTPServer struct {
	server *http.Server
}

// NewHTTPServer creates an HTTPServer with the webhook route registered.
func NewHTTPServer(port string, webhookH *WebhookHandler) *HTTPServer {
	mux := http.NewServeMux()
	mux.Handle("/webhook/payment", webhookH)
	return &HTTPServer{
		server: &http.Server{
			Addr:              fmt.Sprintf(":%s", port),
			Handler:           mux,
			ReadHeaderTimeout: 10 * time.Second,
		},
	}
}

// Serve starts listening. Blocks until error occurs.
func (s *HTTPServer) Serve() error {
	return s.server.ListenAndServe()
}

// Shutdown gracefully stops the HTTP server.
func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}
