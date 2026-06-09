package pos

import (
	"context"
	"fmt"

	"github.com/xyn-pos/services/pos/internal/application/command"
	"github.com/xyn-pos/services/pos/internal/application/query"
	kafkainfra "github.com/xyn-pos/services/pos/internal/infrastructure/kafka"
	"github.com/xyn-pos/services/pos/internal/infrastructure/postgres"
	grpchandler "github.com/xyn-pos/services/pos/internal/interfaces/grpc"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	shareddb "github.com/xyn-pos/shared/pkg/database"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
)

// App is the assembled pos service.
type App struct {
	grpc     *grpchandler.Server
	consumer *kafkainfra.PaymentConsumer
	shutdown func()
}

// New builds the dependency graph bottom-up for the pos service.
func New(ctx context.Context, cfg *Config) (*App, error) {
	// 1. Telemetry
	shutdownTelemetry, err := sharedtelemetry.Setup(ctx, sharedtelemetry.Config{
		ServiceName:    cfg.ServiceName,
		ServiceVersion: cfg.Version,
		OTLPEndpoint:   cfg.OTLPEndpoint,
		Environment:    cfg.Env,
	})
	if err != nil {
		return nil, fmt.Errorf("pos.New telemetry: %w", err)
	}

	// 2. Database
	pool, err := shareddb.NewPool(ctx, shareddb.Config{DSN: cfg.DatabaseURL})
	if err != nil {
		shutdownTelemetry()
		return nil, fmt.Errorf("pos.New database: %w", err)
	}

	// 3. Repositories
	productRepo := postgres.NewProductRepository(pool)
	categoryRepo := postgres.NewCategoryRepository(pool)

	// 4. Application layer — commands
	createProductH := command.NewCreateProductHandler(productRepo)
	updateProductH := command.NewUpdateProductHandler(productRepo)
	archiveProductH := command.NewArchiveProductHandler(productRepo)
	createCategoryH := command.NewCreateCategoryHandler(categoryRepo)
	reorderCategoryH := command.NewReorderCategoriesHandler(categoryRepo)
	setBranchPriceH := command.NewSetBranchPriceHandler(productRepo)

	// 5. Application layer — queries
	getProductH := query.NewGetProductHandler(productRepo)
	listProductsH := query.NewListProductsHandler(productRepo)
	lookupBySKUH := query.NewLookupBySKUHandler(productRepo)

	// 6. PASETO verifier (for gRPC auth middleware)
	var verifyFn sharedauth.VerifyFunc
	if cfg.PASETOKeyHex != "" {
		verifyFn, err = sharedauth.NewLocalVerifier(cfg.PASETOKeyHex)
		if err != nil {
			pool.Close()
			shutdownTelemetry()
			return nil, fmt.Errorf("pos.New paseto verifier: %w", err)
		}
	}

	// 7. Order domain
	orderRepo := postgres.NewOrderRepository(pool)
	shiftRepo := postgres.NewShiftRepository(pool)

	kafkaPublisher, err := kafkainfra.NewEventPublisher(cfg.KafkaBrokers)
	if err != nil {
		pool.Close()
		shutdownTelemetry()
		return nil, fmt.Errorf("pos.New kafka publisher: %w", err)
	}

	createOrderH := command.NewCreateOrderHandler(orderRepo, kafkaPublisher)
	addItemH := command.NewAddItemHandler(orderRepo)
	removeItemH := command.NewRemoveItemHandler(orderRepo)
	updateQtyH := command.NewUpdateItemQuantityHandler(orderRepo)
	applyDiscountH := command.NewApplyDiscountHandler(orderRepo)
	submitOrderH := command.NewSubmitOrderHandler(orderRepo)
	cancelOrderH := command.NewCancelOrderHandler(orderRepo, kafkaPublisher)
	parkOrderH := command.NewParkOrderHandler(orderRepo, kafkaPublisher)
	resumeOrderH := command.NewResumeOrderHandler(orderRepo, kafkaPublisher)
	markOrderPaidH := command.NewMarkOrderPaidHandler(orderRepo, kafkaPublisher)
	openShiftH := command.NewOpenShiftHandler(shiftRepo)
	closeShiftH := command.NewCloseShiftHandler(shiftRepo)
	getOrderH := query.NewGetOrderHandler(orderRepo)
	listOrdersH := query.NewListOrdersHandler(orderRepo)
	getShiftH := query.NewGetShiftHandler(shiftRepo)

	orderHandler := grpchandler.NewOrderHandler(
		createOrderH, addItemH, removeItemH, updateQtyH,
		applyDiscountH, submitOrderH, cancelOrderH,
		parkOrderH, resumeOrderH,
		openShiftH, closeShiftH,
		getOrderH, listOrdersH, getShiftH,
	)

	paymentConsumer, err := kafkainfra.NewPaymentConsumer(cfg.KafkaBrokers, markOrderPaidH)
	if err != nil {
		kafkaPublisher.Close()
		pool.Close()
		shutdownTelemetry()
		return nil, fmt.Errorf("pos.New kafka consumer: %w", err)
	}

	// 8. Interface layer
	productHandler := grpchandler.NewProductHandler(
		createProductH, updateProductH, archiveProductH,
		createCategoryH, reorderCategoryH, setBranchPriceH,
		getProductH, listProductsH, lookupBySKUH,
	)
	grpcSrv := grpchandler.NewServer(cfg.GRPCPort, productHandler, orderHandler, verifyFn)

	return &App{
		grpc:     grpcSrv,
		consumer: paymentConsumer,
		shutdown: func() {
			kafkaPublisher.Close()
			pool.Close()
			shutdownTelemetry()
		},
	}, nil
}

// Start begins serving gRPC traffic. Launches the Kafka consumer in the background
// and blocks until the gRPC server stops.
func (a *App) Start(ctx context.Context) error {
	go a.consumer.Run(ctx)
	return a.grpc.Start()
}

// Stop gracefully drains connections and releases resources.
func (a *App) Stop() {
	a.grpc.GracefulStop()
	a.consumer.Close()
	a.shutdown()
}
