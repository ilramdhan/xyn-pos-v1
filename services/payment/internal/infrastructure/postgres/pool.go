package postgres

import (
	"context"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// NewPool creates a pgxpool connection pool and runs all pending migrations.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres.NewPool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres.NewPool ping: %w", err)
	}
	if err := runMigrations(pool); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres.NewPool migrate: %w", err)
	}
	return pool, nil
}

func runMigrations(pool *pgxpool.Pool) error {
	migrationsFS, err := fs.Sub(MigrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("postgres.runMigrations sub: %w", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close() //nolint:errcheck

	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrationsFS)
	if err != nil {
		return fmt.Errorf("postgres.runMigrations provider: %w", err)
	}
	_, err = provider.Up(context.Background())
	return err
}
