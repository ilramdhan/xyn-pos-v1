//go:build integration

package postgres_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	"github.com/xyn-pos/services/pos/internal/infrastructure/postgres"
)

func startTestDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()

	container, err := tcpostgres.Run(ctx, "postgres:18-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("test"),
		tcpostgres.WithPassword("test"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = container.Terminate(ctx) })

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	pool, err := pgxpool.New(ctx, dsn)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	// Run migrations via the database/sql shim over pgxpool.
	// Do NOT close sqlDB — that would also close the pool.
	sqlDB := stdlib.OpenDBFromPool(pool)

	migrationsFS, err := fs.Sub(postgres.MigrationsFS, "migrations")
	require.NoError(t, err)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrationsFS)
	require.NoError(t, err)
	_, err = provider.Up(ctx)
	require.NoError(t, err)

	return pool
}

func setRLS(t *testing.T, pool *pgxpool.Pool, tenantID uuid.UUID) {
	t.Helper()
	// SET SESSION does not support parameterised queries; embed the value as a
	// quoted literal. tenantID is a UUID so it is safe to format directly.
	_, err := pool.Exec(context.Background(),
		"SET SESSION app.current_tenant_id = '"+tenantID.String()+"'")
	require.NoError(t, err)
}
