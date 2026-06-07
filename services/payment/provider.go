package payment

import "fmt"

// App is the assembled payment service (provider wired in later tasks).
type App struct{}

// NewServer builds the payment service dependency graph.
// Full implementation in Task 7 (PaymentHandler + provider.go).
func NewServer(_ *Config) (*App, error) {
	return nil, fmt.Errorf("payment.NewServer: not yet implemented")
}

// Run starts the payment service.
func (a *App) Run() error { return nil }
