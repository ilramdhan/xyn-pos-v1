package payment

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/payment/internal/application/command"
	"github.com/xyn-pos/services/payment/internal/application/query"
	domainpayment "github.com/xyn-pos/services/payment/internal/domain/payment"
	kafkainfra "github.com/xyn-pos/services/payment/internal/infrastructure/kafka"
	midtransinfra "github.com/xyn-pos/services/payment/internal/infrastructure/midtrans"
	postgresinfra "github.com/xyn-pos/services/payment/internal/infrastructure/postgres"
	redisinfra "github.com/xyn-pos/services/payment/internal/infrastructure/redis"
	grpchandler "github.com/xyn-pos/services/payment/internal/interfaces/grpc"
	httphandler "github.com/xyn-pos/services/payment/internal/interfaces/http"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
)

// Compile-time interface assertions — fail fast if infrastructure diverges from contracts.
var _ command.IdempotencyStore = (*redisinfra.IdempotencyStore)(nil)
var _ command.PaymentEventPublisher = (*kafkainfra.EventPublisher)(nil)
var _ command.ReceiptEventPublisher = (*kafkainfra.EventPublisher)(nil)
var _ domainpayment.PaymentGateway = (*midtransinfra.MidtransGateway)(nil)
var _ domainpayment.PaymentGateway = (*midtransinfra.NoopGateway)(nil)

// App holds the assembled payment service.
type App struct {
	grpcSrv *grpchandler.Server
	httpSrv *httphandler.HTTPServer
	kafka   *kafkainfra.EventPublisher
	redisSt *redisinfra.IdempotencyStore
	pool    *pgxpool.Pool
}

// NewServer builds the full payment service dependency graph.
func NewServer(cfg *Config) (*App, error) {
	ctx := context.Background()

	// Infrastructure: PostgreSQL (runs migrations)
	pool, err := postgresinfra.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("payment.NewServer postgres: %w", err)
	}

	// Infrastructure: Redis idempotency store
	redisSt, err := redisinfra.NewIdempotencyStore(cfg.RedisAddr)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("payment.NewServer redis: %w", err)
	}

	// Infrastructure: Kafka event publisher
	kafkaPub, err := kafkainfra.NewEventPublisher(cfg.KafkaBrokers)
	if err != nil {
		pool.Close()
		_ = redisSt.Close()
		return nil, fmt.Errorf("payment.NewServer kafka: %w", err)
	}

	// Infrastructure: Payment gateway (real or noop)
	var gw domainpayment.PaymentGateway
	if cfg.MidtransServerKey != "" {
		gw = midtransinfra.NewMidtransGateway(cfg.MidtransServerKey, cfg.MidtransIsProduction)
	} else {
		gw = &midtransinfra.NoopGateway{}
	}

	// Repositories
	paymentRepo := postgresinfra.NewPaymentRepository(pool)
	receiptRepo := postgresinfra.NewReceiptRepository(pool)

	// Application: commands
	initiateH := command.NewInitiatePaymentHandler(paymentRepo, gw, redisSt)
	voidH := command.NewVoidPaymentHandler(paymentRepo, gw, kafkaPub)
	webhookH := command.NewHandleWebhookHandler(paymentRepo, kafkaPub, receiptRepo, kafkaPub)

	// Application: queries
	getH := query.NewGetPaymentHandler(paymentRepo)
	getByOrderH := query.NewGetPaymentByOrderHandler(paymentRepo)
	getReceiptH := query.NewGetReceiptHandler(receiptRepo)
	getReceiptByOrderH := query.NewGetReceiptByOrderHandler(receiptRepo)

	// PASETO verifier for gRPC auth middleware
	var verifyFn sharedauth.VerifyFunc
	if cfg.PASETOKeyHex != "" {
		verifyFn, err = sharedauth.NewLocalVerifier(cfg.PASETOKeyHex)
		if err != nil {
			pool.Close()
			_ = redisSt.Close()
			kafkaPub.Close()
			return nil, fmt.Errorf("payment.NewServer paseto verifier: %w", err)
		}
	}

	// Interfaces: gRPC
	grpcHdlr := grpchandler.NewPaymentHandler(initiateH, voidH, getH, getByOrderH, getReceiptH, getReceiptByOrderH)
	grpcSrv := grpchandler.NewServer(cfg.GRPCPort, grpcHdlr, verifyFn)

	// Interfaces: HTTP (Midtrans webhook)
	webhookHTTP := httphandler.NewWebhookHandler(webhookH, cfg.MidtransServerKey)
	httpSrv := httphandler.NewHTTPServer(cfg.HTTPPort, webhookHTTP)

	return &App{
		grpcSrv: grpcSrv,
		httpSrv: httpSrv,
		kafka:   kafkaPub,
		redisSt: redisSt,
		pool:    pool,
	}, nil
}

// Run starts both the gRPC and HTTP servers concurrently.
// It blocks until either server errors or ctx is cancelled.
func (a *App) Run(ctx context.Context) error {
	errCh := make(chan error, 2)

	go func() {
		if err := a.grpcSrv.Serve(); err != nil {
			errCh <- fmt.Errorf("grpc: %w", err)
		}
	}()

	go func() {
		if err := a.httpSrv.Serve(); err != nil {
			errCh <- fmt.Errorf("http: %w", err)
		}
	}()

	select {
	case <-ctx.Done():
		a.Stop(ctx)
		return ctx.Err()
	case err := <-errCh:
		a.Stop(ctx)
		return err
	}
}

// Stop gracefully shuts down all service components.
// The provided context deadline is respected during HTTP server shutdown.
func (a *App) Stop(ctx context.Context) {
	a.grpcSrv.GracefulStop()
	_ = a.httpSrv.Shutdown(ctx)
	a.kafka.Close()
	_ = a.redisSt.Close()
	a.pool.Close()
}
