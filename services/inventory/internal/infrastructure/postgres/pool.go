package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// NewPool creates a pgxpool and runs Goose migrations.
func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("postgres.NewPool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres.NewPool ping: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	goose.SetBaseFS(MigrationsFS)
	if err := goose.SetDialect("postgres"); err != nil {
		pool.Close()
		return nil, err
	}
	if err := goose.Up(db, "migrations"); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres.NewPool migrate: %w", err)
	}
	return pool, nil
}
