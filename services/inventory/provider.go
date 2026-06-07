package inventory

import (
	"context"
	"fmt"
)

// App is the assembled inventory service.
type App struct{}

// NewServer builds the inventory service dependency graph.
// Full implementation in Task 71.
func NewServer(_ *Config) (*App, error) {
	return nil, fmt.Errorf("inventory.NewServer: not yet implemented")
}

// Run starts the inventory service.
func (a *App) Run(_ context.Context) error { return nil }

// Stop shuts down the inventory service.
func (a *App) Stop() {}
