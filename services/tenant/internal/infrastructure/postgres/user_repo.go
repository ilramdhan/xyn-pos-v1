package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/xyn-pos/services/tenant/internal/domain/user"
)

// UserRepository implements user.Repository backed by PostgreSQL.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository constructs a UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// isUniqueViolation reports whether err is a PostgreSQL unique-constraint violation (SQLSTATE 23505).
func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// FindByID returns a user by primary key.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	const q = `SELECT id, tenant_id, keycloak_id, email, full_name, role,
	                   branch_scope, is_active, created_at, updated_at
	           FROM users WHERE id = $1`
	return r.scanOne(ctx, q, id)
}

// FindByEmail returns a user by tenant and email address.
func (r *UserRepository) FindByEmail(ctx context.Context, tenantID uuid.UUID, email string) (*user.User, error) {
	const q = `SELECT id, tenant_id, keycloak_id, email, full_name, role,
	                   branch_scope, is_active, created_at, updated_at
	           FROM users WHERE tenant_id = $1 AND email = $2`
	return r.scanOne(ctx, q, tenantID, email)
}

// FindByKeycloakID returns a user by their Keycloak subject ID.
func (r *UserRepository) FindByKeycloakID(ctx context.Context, keycloakID string) (*user.User, error) {
	const q = `SELECT id, tenant_id, keycloak_id, email, full_name, role,
	                   branch_scope, is_active, created_at, updated_at
	           FROM users WHERE keycloak_id = $1`
	return r.scanOne(ctx, q, keycloakID)
}

// FindByTenant returns all users belonging to a tenant.
func (r *UserRepository) FindByTenant(ctx context.Context, tenantID uuid.UUID) ([]*user.User, error) {
	const q = `SELECT id, tenant_id, keycloak_id, email, full_name, role,
	                   branch_scope, is_active, created_at, updated_at
	           FROM users WHERE tenant_id = $1 ORDER BY created_at ASC`

	rows, err := r.pool.Query(ctx, q, tenantID)
	if err != nil {
		return nil, fmt.Errorf("userRepo.FindByTenant: %w", err)
	}
	defer rows.Close()

	var result []*user.User
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, fmt.Errorf("userRepo.FindByTenant scan: %w", err)
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

// Save inserts a new user record.
func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	const q = `INSERT INTO users (id, tenant_id, keycloak_id, email, full_name, role,
	                               branch_scope, is_active, created_at, updated_at)
	           VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)`

	branchScope := uuidSliceToStrings(u.BranchScope)
	_, err := r.pool.Exec(ctx, q,
		u.ID, u.TenantID, u.KeycloakID, u.Email, u.FullName, string(u.Role),
		branchScope, u.IsActive, u.CreatedAt, u.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("userRepo.Save: %w", user.ErrEmailTaken)
		}
		return fmt.Errorf("userRepo.Save: %w", err)
	}
	return nil
}

// Update persists changes to an existing user record.
func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	const q = `UPDATE users SET role = $2, branch_scope = $3, is_active = $4, updated_at = $5
	           WHERE id = $1`

	branchScope := uuidSliceToStrings(u.BranchScope)
	_, err := r.pool.Exec(ctx, q, u.ID, string(u.Role), branchScope, u.IsActive, u.UpdatedAt)
	if err != nil {
		return fmt.Errorf("userRepo.Update: %w", err)
	}
	return nil
}

// SavePIN upserts the bcrypt PIN hash for a user.
func (r *UserRepository) SavePIN(ctx context.Context, userID uuid.UUID, pinHash string) error {
	const q = `INSERT INTO user_pins (user_id, pin_hash, updated_at) VALUES ($1, $2, now())
	           ON CONFLICT (user_id) DO UPDATE SET pin_hash = $2, updated_at = now()`
	_, err := r.pool.Exec(ctx, q, userID, pinHash)
	if err != nil {
		return fmt.Errorf("userRepo.SavePIN: %w", err)
	}
	return nil
}

// FindPINHash returns the bcrypt PIN hash for a user.
func (r *UserRepository) FindPINHash(ctx context.Context, userID uuid.UUID) (string, error) {
	var hash string
	err := r.pool.QueryRow(ctx, `SELECT pin_hash FROM user_pins WHERE user_id = $1`, userID).Scan(&hash)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("userRepo.FindPINHash: %w", user.ErrUserNotFound)
	}
	if err != nil {
		return "", fmt.Errorf("userRepo.FindPINHash: %w", err)
	}
	return hash, nil
}

func (r *UserRepository) scanOne(ctx context.Context, q string, args ...any) (*user.User, error) {
	rows, err := r.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("userRepo query: %w", err)
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, user.ErrUserNotFound
	}
	u, err := scanUser(rows)
	if err != nil {
		return nil, fmt.Errorf("userRepo scan: %w", err)
	}
	return u, nil
}

func scanUser(rows pgx.Rows) (*user.User, error) {
	var u user.User
	var roleStr string
	var branchScopeIDs []uuid.UUID

	err := rows.Scan(
		&u.ID, &u.TenantID, &u.KeycloakID, &u.Email, &u.FullName,
		&roleStr, &branchScopeIDs, &u.IsActive, &u.CreatedAt, &u.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	u.Role = user.Role(roleStr)
	u.BranchScope = branchScopeIDs
	return &u, nil
}

func uuidSliceToStrings(ids []uuid.UUID) []string {
	if len(ids) == 0 {
		return nil
	}
	result := make([]string, len(ids))
	for i, id := range ids {
		result[i] = id.String()
	}
	return result
}
