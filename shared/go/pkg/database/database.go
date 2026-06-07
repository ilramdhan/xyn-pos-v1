package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config holds database connection configuration.
type Config struct {
	DSN      string
	MaxConns int32
	MinConns int32
}

// NewPool creates a pgxpool with the provided config.
// Returns an error if the pool cannot connect.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database.NewPool parse: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database.NewPool connect: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database.NewPool ping: %w", err)
	}

	return pool, nil
}

// WithTenantTx runs fn inside a transaction with app.current_tenant_id set via SET LOCAL.
// This is required by PostgreSQL RLS policies on every table.
func WithTenantTx(ctx context.Context, pool *pgxpool.Pool, tenantID uuid.UUID, fn func(pgx.Tx) error) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("database.WithTenantTx begin: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	// set_config with is_local=true is equivalent to SET LOCAL and supports $1 parameters.
	if _, err = tx.Exec(ctx, "SELECT set_config('app.current_tenant_id', $1, true)", tenantID.String()); err != nil {
		return fmt.Errorf("database.WithTenantTx set tenant: %w", err)
	}

	if err = fn(tx); err != nil {
		return err
	}

	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("database.WithTenantTx commit: %w", err)
	}

	return nil
}
