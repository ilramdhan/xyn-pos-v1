package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
	shareddb "github.com/xyn-pos/shared/pkg/database"
)

// TenantRepository implements domain.Repository using PostgreSQL via pgxpool.
type TenantRepository struct {
	pool   *pgxpool.Pool
	tracer trace.Tracer
}

// NewTenantRepository constructs a TenantRepository with an OTEL tracer.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool, tracer: otel.Tracer("tenant.postgres")}
}

// Save upserts the tenant and all its branches inside an RLS-scoped transaction.
func (r *TenantRepository) Save(ctx context.Context, t *domain.Tenant) error {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.Save")
	defer span.End()

	return shareddb.WithTenantTx(ctx, r.pool, t.ID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			INSERT INTO tenants (id, name, slug, plan, status, subscription_status, trial_ends_at, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			ON CONFLICT (id) DO UPDATE SET
				name                = EXCLUDED.name,
				plan                = EXCLUDED.plan,
				status              = EXCLUDED.status,
				subscription_status = EXCLUDED.subscription_status,
				trial_ends_at       = EXCLUDED.trial_ends_at,
				updated_at          = EXCLUDED.updated_at
		`, t.ID, t.Name, t.Slug, string(t.Plan.Tier), string(t.Status), string(t.SubscriptionStatus), t.TrialEndsAt, t.CreatedAt, t.UpdatedAt)
		if err != nil {
			return fmt.Errorf("TenantRepository.Save upsert: %w", err)
		}

		for _, b := range t.Branches {
			_, err := tx.Exec(ctx, `
				INSERT INTO branches (id, tenant_id, name, street, city, province, postal_code, country, timezone, is_active, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
				ON CONFLICT (id) DO UPDATE SET
					name       = EXCLUDED.name,
					is_active  = EXCLUDED.is_active,
					updated_at = EXCLUDED.updated_at
			`, b.ID, b.TenantID, b.Name,
				b.Address.Street, b.Address.City, b.Address.Province, b.Address.PostalCode, b.Address.Country,
				b.Timezone, b.IsActive, b.CreatedAt, b.UpdatedAt)
			if err != nil {
				return fmt.Errorf("TenantRepository.Save branch id=%s: %w", b.ID, err)
			}
		}
		return nil
	})
}

// FindByID loads a tenant by its UUID, including all branches.
func (r *TenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.FindByID")
	defer span.End()

	var t domain.Tenant
	var planTier, status, subscriptionStatus string
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, slug, plan, status, subscription_status, trial_ends_at, created_at, updated_at
		FROM tenants WHERE id = $1
	`, id).Scan(&t.ID, &t.Name, &t.Slug, &planTier, &status, &subscriptionStatus, &t.TrialEndsAt, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("TenantRepository.FindByID id=%s: %w", id, err)
	}
	t.Plan = domain.PlanFromTier(domain.PlanTier(planTier))
	t.Status = domain.Status(status)
	t.SubscriptionStatus = domain.SubscriptionStatus(subscriptionStatus)

	branches, err := r.ListBranches(ctx, id)
	if err != nil {
		return nil, err
	}
	t.Branches = branches
	return &t, nil
}

// FindBySlug looks up a tenant by its unique slug, then delegates to FindByID.
func (r *TenantRepository) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.FindBySlug")
	defer span.End()

	var id uuid.UUID
	err := r.pool.QueryRow(ctx, `SELECT id FROM tenants WHERE slug = $1`, slug).Scan(&id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, domain.ErrTenantNotFound
		}
		return nil, fmt.Errorf("TenantRepository.FindBySlug slug=%s: %w", slug, err)
	}
	return r.FindByID(ctx, id)
}

// ExistsBySlug reports whether a tenant with the given slug already exists.
func (r *TenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.ExistsBySlug")
	defer span.End()

	var exists bool
	err := r.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM tenants WHERE slug = $1)`, slug).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("TenantRepository.ExistsBySlug: %w", err)
	}
	return exists, nil
}

// ListBranches returns all branches for a tenant, ordered by creation time.
// The query runs inside a transaction so SET LOCAL takes effect for the branches RLS policy.
func (r *TenantRepository) ListBranches(ctx context.Context, tenantID uuid.UUID) ([]domain.Branch, error) {
	ctx, span := r.tracer.Start(ctx, "TenantRepository.ListBranches")
	defer span.End()

	var branches []domain.Branch

	err := shareddb.WithTenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx, `
			SELECT id, tenant_id, name, street, city, province, postal_code, country, timezone, is_active, created_at, updated_at
			FROM branches WHERE tenant_id = $1 ORDER BY created_at ASC
		`, tenantID)
		if err != nil {
			return fmt.Errorf("TenantRepository.ListBranches: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var b domain.Branch
			var updatedAt time.Time
			if err := rows.Scan(
				&b.ID, &b.TenantID, &b.Name,
				&b.Address.Street, &b.Address.City, &b.Address.Province, &b.Address.PostalCode, &b.Address.Country,
				&b.Timezone, &b.IsActive, &b.CreatedAt, &updatedAt,
			); err != nil {
				return fmt.Errorf("TenantRepository.ListBranches scan: %w", err)
			}
			b.UpdatedAt = updatedAt
			branches = append(branches, b)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("TenantRepository.ListBranches: %w", err)
	}
	return branches, nil
}
