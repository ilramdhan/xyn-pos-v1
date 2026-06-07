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
	"github.com/stretchr/testify/suite"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"

	domainTenant "github.com/xyn-pos/services/tenant/internal/domain/tenant"
	"github.com/xyn-pos/services/tenant/internal/domain/user"
	pgstore "github.com/xyn-pos/services/tenant/internal/infrastructure/postgres"
)

// UserRepositoryIntegrationSuite tests UserRepository against a real PostgreSQL container.
// One container and schema is shared across tests; each test creates its own tenant for isolation.
type UserRepositoryIntegrationSuite struct {
	suite.Suite
	container *tcpostgres.PostgresContainer
	pool      *pgxpool.Pool
	repo      *pgstore.UserRepository
	tenantRepo *pgstore.TenantRepository
}

func (s *UserRepositoryIntegrationSuite) SetupSuite() {
	ctx := context.Background()

	pgContainer, err := tcpostgres.Run(ctx,
		"postgres:18-alpine",
		tcpostgres.WithDatabase("xyn_tenant_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("testpass"),
		tcpostgres.BasicWaitStrategies(),
	)
	s.Require().NoError(err)
	s.container = pgContainer

	dsn, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	s.Require().NoError(err)

	s.pool, err = pgxpool.New(ctx, dsn)
	s.Require().NoError(err)

	sqlDB := stdlib.OpenDBFromPool(s.pool)

	migrationsFS, err := fs.Sub(pgstore.MigrationsFS, "migrations")
	s.Require().NoError(err)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrationsFS)
	s.Require().NoError(err)
	_, err = provider.Up(ctx)
	s.Require().NoError(err)

	s.repo = pgstore.NewUserRepository(s.pool)
	s.tenantRepo = pgstore.NewTenantRepository(s.pool)
}

func (s *UserRepositoryIntegrationSuite) TearDownSuite() {
	ctx := context.Background()
	if s.pool != nil {
		s.pool.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(ctx)
	}
}

// TearDownTest cleans up users and tenants between tests.
func (s *UserRepositoryIntegrationSuite) TearDownTest() {
	_, err := s.pool.Exec(context.Background(), "TRUNCATE tenants CASCADE")
	s.Require().NoError(err)
}

// createTenant creates and persists a new tenant, returning its ID.
func (s *UserRepositoryIntegrationSuite) createTenant() uuid.UUID {
	t, err := domainTenant.NewTenant("Test Tenant", uuid.New().String(), domainTenant.PlanTierFree)
	s.Require().NoError(err)
	s.Require().NoError(s.tenantRepo.Save(context.Background(), t))
	return t.ID
}

// setRLS sets the RLS context for subsequent queries on the pool connection.
// Uses a dedicated connection to set app.current_tenant_id for the session.
func (s *UserRepositoryIntegrationSuite) setRLS(tenantID uuid.UUID) {
	_, err := s.pool.Exec(context.Background(),
		"SELECT set_config('app.current_tenant_id', $1, false)", tenantID.String())
	s.Require().NoError(err)
}

func (s *UserRepositoryIntegrationSuite) TestSaveAndFindByID() {
	ctx := context.Background()
	tenantID := s.createTenant()
	s.setRLS(tenantID)

	u, err := user.NewUser(tenantID, "kc-001", "ani@example.com", "Ani", user.RoleCashier, nil)
	s.Require().NoError(err)

	s.Require().NoError(s.repo.Save(ctx, u))

	found, err := s.repo.FindByID(ctx, u.ID)
	s.Require().NoError(err)
	s.Equal(u.Email, found.Email)
	s.Equal(u.Role, found.Role)
	s.True(found.IsActive)
}

func (s *UserRepositoryIntegrationSuite) TestFindByEmail_NotFound() {
	tenantID := s.createTenant()
	s.setRLS(tenantID)

	_, err := s.repo.FindByEmail(context.Background(), tenantID, "nobody@example.com")
	s.ErrorIs(err, user.ErrUserNotFound)
}

func (s *UserRepositoryIntegrationSuite) TestSave_DuplicateEmail_ReturnsEmailTaken() {
	ctx := context.Background()
	tenantID := s.createTenant()
	s.setRLS(tenantID)

	u1, _ := user.NewUser(tenantID, "kc-a", "same@example.com", "A", user.RoleCashier, nil)
	u2, _ := user.NewUser(tenantID, "kc-b", "same@example.com", "B", user.RoleManager, nil)

	s.Require().NoError(s.repo.Save(ctx, u1))
	err := s.repo.Save(ctx, u2)
	s.ErrorIs(err, user.ErrEmailTaken)
}

func (s *UserRepositoryIntegrationSuite) TestSetPINAndVerify() {
	ctx := context.Background()
	tenantID := s.createTenant()
	s.setRLS(tenantID)

	u, _ := user.NewUser(tenantID, "kc-pin", "pin@example.com", "Pin User", user.RoleCashier, nil)
	s.Require().NoError(s.repo.Save(ctx, u))

	hash := "$2a$12$fakehashfakehashfakehash.fakehashfakehashfakehash123456"
	s.Require().NoError(s.repo.SavePIN(ctx, u.ID, hash))

	stored, err := s.repo.FindPINHash(ctx, u.ID)
	s.Require().NoError(err)
	s.Equal(hash, stored)
}

func (s *UserRepositoryIntegrationSuite) TestListByTenant_RLSIsolation() {
	ctx := context.Background()

	tenant1 := s.createTenant()
	tenant2 := s.createTenant()

	s.setRLS(tenant1)
	u1, _ := user.NewUser(tenant1, "kc-t1", "t1@example.com", "T1", user.RoleOwner, nil)
	s.Require().NoError(s.repo.Save(ctx, u1))

	s.setRLS(tenant2)
	u2, _ := user.NewUser(tenant2, "kc-t2", "t2@example.com", "T2", user.RoleOwner, nil)
	s.Require().NoError(s.repo.Save(ctx, u2))

	s.setRLS(tenant1)
	users, err := s.repo.FindByTenant(ctx, tenant1)
	s.Require().NoError(err)
	s.Len(users, 1)
	s.Equal(u1.Email, users[0].Email)
}

// TestUserRepositoryIntegration is the suite entry point.
func TestUserRepositoryIntegration(t *testing.T) {
	suite.Run(t, new(UserRepositoryIntegrationSuite))
}
