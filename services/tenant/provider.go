package tenant

import (
	"context"
	"fmt"

	"github.com/xyn-pos/services/tenant/internal/application/command"
	"github.com/xyn-pos/services/tenant/internal/application/query"
	"github.com/xyn-pos/services/tenant/internal/infrastructure/keycloak"
	"github.com/xyn-pos/services/tenant/internal/infrastructure/postgres"
	tenantredis "github.com/xyn-pos/services/tenant/internal/infrastructure/redis"
	grpchandler "github.com/xyn-pos/services/tenant/internal/interfaces/grpc"
	sharedauth "github.com/xyn-pos/shared/pkg/auth"
	sharedcache "github.com/xyn-pos/shared/pkg/cache"
	shareddb "github.com/xyn-pos/shared/pkg/database"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
)

// App is the assembled tenant service: gRPC server + shutdown lifecycle.
type App struct {
	grpc     *grpchandler.Server
	shutdown func()
}

// New builds the full dependency graph bottom-up:
// telemetry → database → redis → repository → application → interface.
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

	// 3. Redis cache (for token blacklist)
	cacheClient, err := sharedcache.NewClient(sharedcache.Config{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})
	if err != nil {
		pool.Close()
		shutdownTelemetry()
		return nil, fmt.Errorf("provider.New redis: %w", err)
	}
	idempotencyStore := sharedcache.NewIdempotencyStore(cacheClient)

	// 4. Tenant repository and application layer
	tenantRepo := postgres.NewTenantRepository(pool)
	createTenantH := command.NewCreateTenantHandler(tenantRepo, nil) // nil publisher = no Kafka in Phase 3
	createBranchH := command.NewCreateBranchHandler(tenantRepo)
	getTenantH := query.NewGetTenantHandler(tenantRepo)
	listBranchesH := query.NewListBranchesHandler(tenantRepo)

	// 5. User infrastructure
	kcClient := keycloak.NewClient(keycloak.Config{
		BaseURL:      cfg.KeycloakBaseURL,
		Realm:        cfg.KeycloakRealm,
		ClientID:     cfg.KeycloakClientID,
		ClientSecret: cfg.KeycloakClientSecret,
	})
	kcAdapter := &keycloakAdapter{client: kcClient}

	// 6. User repository and application layer
	userRepo := postgres.NewUserRepository(pool)

	registerUserH := command.NewRegisterUserHandler(userRepo, kcAdapter, cfg.KeycloakAdminToken)
	loginUserH := command.NewLoginUserHandler(userRepo, kcAdapter, cfg.PASETOKeyHex)

	verifyFn, err := sharedauth.NewLocalVerifier(cfg.PASETOKeyHex)
	if err != nil {
		_ = cacheClient.Close()
		pool.Close()
		shutdownTelemetry()
		return nil, fmt.Errorf("provider.New paseto verifier: %w", err)
	}

	tokenBlacklist := tenantredis.NewTokenBlacklist(idempotencyStore)
	logoutH := command.NewLogoutHandler(verifyFn, tokenBlacklist)
	setPINH := command.NewSetPINHandler(userRepo)
	verifyPINH := command.NewVerifyPINHandler(userRepo)
	deactivateH := command.NewDeactivateUserHandler(userRepo)
	getUserH := query.NewGetUserHandler(userRepo)
	listUsersH := query.NewListUsersHandler(userRepo)

	// 7. Interface layer
	tenantHandler := grpchandler.NewTenantHandler(createTenantH, createBranchH, getTenantH, listBranchesH)
	userHandler := grpchandler.NewUserHandler(
		registerUserH, loginUserH, logoutH,
		setPINH, verifyPINH, deactivateH,
		getUserH, listUsersH,
	)

	grpcSrv := grpchandler.NewServer(cfg.GRPCPort, tenantHandler, userHandler, verifyFn)

	return &App{
		grpc: grpcSrv,
		shutdown: func() {
			_ = cacheClient.Close()
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

// keycloakAdapter bridges keycloak.Client to command.KeycloakClient.
type keycloakAdapter struct {
	client *keycloak.Client
}

func (a *keycloakAdapter) CreateUser(ctx context.Context, adminToken string, req command.KeycloakCreateUserRequest) (string, error) {
	return a.client.CreateUser(ctx, adminToken, keycloak.CreateUserRequest{
		Email:     req.Email,
		Password:  req.Password,
		FirstName: req.FirstName,
	})
}

func (a *keycloakAdapter) Introspect(ctx context.Context, token string) (*command.KeycloakIntrospectResult, error) {
	result, err := a.client.Introspect(ctx, token)
	if err != nil {
		return nil, err
	}
	return &command.KeycloakIntrospectResult{
		Active:  result.Active,
		Subject: result.Subject,
		Email:   result.Email,
	}, nil
}
