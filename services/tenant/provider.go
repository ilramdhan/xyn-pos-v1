package tenant

import (
	"context"
	"fmt"

	shareddb "github.com/xyn-pos/shared/pkg/database"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/application/query"
	"github.com/xyn-pos/services/tenant/internal/infrastructure/postgres"
	grpchandler "github.com/xyn-pos/services/tenant/internal/interfaces/grpc"
)

// App is the assembled tenant service: gRPC server + shutdown lifecycle.
type App struct {
	grpc     *grpchandler.Server
	shutdown func()
}

// New builds the full dependency graph bottom-up:
// telemetry → database → repository → application → interface
func New(ctx context.Context, cfg *Config) (*App, error) {
	// 1. Telemetry (first: spans on everything below)
	shutdownTelemetry, err := sharedtelemetry.Setup(ctx, sharedtelemetry.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.Version,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		Environment:    cfg.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("provider.New telemetry: %w", err)
	}

	// 2. Database pool
	pool, err := shareddb.NewPool(ctx, shareddb.Config{DSN: cfg.DatabaseURL})
	if err != nil {
		shutdownTelemetry()
		return nil, fmt.Errorf("provider.New database: %w", err)
	}

	// 3. Repository (infrastructure)
	repo := postgres.NewTenantRepository(pool)

	// 4. Application layer
	createTenantH := command.NewCreateTenantHandler(repo, nil) // nil publisher = no Kafka in Phase 3
	createBranchH := command.NewCreateBranchHandler(repo)
	getTenantH := query.NewGetTenantHandler(repo)
	listBranchesH := query.NewListBranchesHandler(repo)

	// 5. Interface layer
	handler := grpchandler.NewTenantHandler(createTenantH, createBranchH, getTenantH, listBranchesH)
	grpcSrv := grpchandler.NewServer(cfg.GRPCPort, handler, nil) // nil verifyFn = no auth in Phase 3

	return &App{
		grpc: grpcSrv,
		shutdown: func() {
			pool.Close()
			shutdownTelemetry()
		},
	}, nil
}

// Start begins serving gRPC traffic. Blocks until the server stops.
func (a *App) Start() error { return a.grpc.Start() }

// Stop gracefully drains connections and releases resources.
func (a *App) Stop() {
	a.grpc.GracefulStop()
	a.shutdown()
}
