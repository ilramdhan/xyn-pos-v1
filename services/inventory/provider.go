package inventory

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/inventory/internal/application/command"
	"github.com/xyn-pos/services/inventory/internal/application/query"
	kafkainfra "github.com/xyn-pos/services/inventory/internal/infrastructure/kafka"
	postgresinfra "github.com/xyn-pos/services/inventory/internal/infrastructure/postgres"
	grpchandler "github.com/xyn-pos/services/inventory/internal/interfaces/grpc"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
)

// App is the fully assembled inventory service.
type App struct {
	grpcSrv  *grpchandler.Server
	consumer *kafkainfra.OrderPaidConsumer
	pool     *pgxpool.Pool
}

// NewServer builds the inventory service dependency graph using manual DI.
func NewServer(cfg *Config) (*App, error) {
	ctx := context.Background()

	pool, err := postgresinfra.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("inventory.NewServer postgres: %w", err)
	}

	// Repositories (infrastructure → implements domain ports)
	stockRepo := postgresinfra.NewStockRepository(pool)
	bomRepo := postgresinfra.NewBOMRepository(pool)

	// Application layer — commands
	adjustH := command.NewAdjustStockHandler(stockRepo)
	setBOMH := command.NewSetBOMHandler(bomRepo)
	deductH := command.NewDeductStockForOrderHandler(stockRepo, bomRepo)

	// Application layer — queries
	getStockH := query.NewGetStockHandler(stockRepo)
	listStockH := query.NewListStockHandler(stockRepo)
	getBOMH := query.NewGetBOMHandler(bomRepo)

	// Kafka consumer (depends on deduct command handler)
	consumer, err := kafkainfra.NewOrderPaidConsumer(cfg.KafkaBrokers, deductH)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("inventory.NewServer kafka: %w", err)
	}

	// PASETO verify function — nil in dev mode (no PASETO_KEY_HEX set)
	var verifyFn sharedauth.VerifyFunc
	if cfg.PASETOKeyHex != "" {
		verifyFn, err = sharedauth.NewLocalVerifier(cfg.PASETOKeyHex)
		if err != nil {
			pool.Close()
			consumer.Close()
			return nil, fmt.Errorf("inventory.NewServer paseto: %w", err)
		}
	}

	// Interface layer — gRPC handler + server
	grpcHdlr := grpchandler.NewInventoryHandler(adjustH, setBOMH, getStockH, listStockH, getBOMH)
	grpcSrv := grpchandler.NewServer(cfg.GRPCPort, grpcHdlr, verifyFn)

	return &App{grpcSrv: grpcSrv, consumer: consumer, pool: pool}, nil
}

// Run starts the Kafka consumer goroutine and blocks on the gRPC server.
func (a *App) Run(ctx context.Context) error {
	go func() {
		a.consumer.Run(ctx)
	}()

	if err := a.grpcSrv.Serve(); err != nil {
		return fmt.Errorf("inventory: grpc: %w", err)
	}
	return nil
}

// Stop shuts down all components gracefully.
func (a *App) Stop() {
	a.grpcSrv.GracefulStop()
	a.consumer.Close()
	a.pool.Close()
}
