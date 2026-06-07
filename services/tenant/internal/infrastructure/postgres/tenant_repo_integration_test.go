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

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
	pgstore "github.com/xyn-pos/services/tenant/internal/infrastructure/postgres"
)

// RepositoryIntegrationSuite runs all TenantRepository tests against a real
// PostgreSQL 18 container. One container is started per suite; each test
// truncates the tables for isolation.
type RepositoryIntegrationSuite struct {
	suite.Suite
	container *tcpostgres.PostgresContainer
	pool      *pgxpool.Pool
	repo      *pgstore.TenantRepository
}

func (s *RepositoryIntegrationSuite) SetupSuite() {
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

	// Build one pool; reuse it for migrations and the repository.
	s.pool, err = pgxpool.New(ctx, dsn)
	s.Require().NoError(err)

	// Run Goose migrations via the database/sql shim.
	// Do NOT close sqlDB — that would also close s.pool.
	sqlDB := stdlib.OpenDBFromPool(s.pool)

	migrationsFS, err := fs.Sub(pgstore.MigrationsFS, "migrations")
	s.Require().NoError(err)

	provider, err := goose.NewProvider(goose.DialectPostgres, sqlDB, migrationsFS)
	s.Require().NoError(err)
	_, err = provider.Up(ctx)
	s.Require().NoError(err)

	s.repo = pgstore.NewTenantRepository(s.pool)
}

func (s *RepositoryIntegrationSuite) TearDownSuite() {
	ctx := context.Background()
	if s.pool != nil {
		s.pool.Close()
	}
	if s.container != nil {
		_ = s.container.Terminate(ctx)
	}
}

// TearDownTest truncates data between tests for isolation.
func (s *RepositoryIntegrationSuite) TearDownTest() {
	_, err := s.pool.Exec(context.Background(), "TRUNCATE tenants CASCADE")
	s.Require().NoError(err)
}

// newSavedTenant creates a tenant and persists it.
func (s *RepositoryIntegrationSuite) newSavedTenant(name, slug string) *domain.Tenant {
	t, err := domain.NewTenant(name, slug, domain.PlanTierFree)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.Save(context.Background(), t))
	return t
}

func (s *RepositoryIntegrationSuite) TestSave_NewTenant() {
	ctx := context.Background()

	t, err := domain.NewTenant("Warung Padang", "warung-padang", domain.PlanTierFree)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.Save(ctx, t))

	found, err := s.repo.FindByID(ctx, t.ID)
	s.Require().NoError(err)
	s.Equal("Warung Padang", found.Name)
	s.Equal("warung-padang", found.Slug)
	s.Equal(domain.PlanTierFree, found.Plan.Tier)
	s.Equal(domain.StatusActive, found.Status)
}

func (s *RepositoryIntegrationSuite) TestSave_UpdateExistingTenant() {
	ctx := context.Background()
	t := s.newSavedTenant("Original Name", "original-slug")

	t.Name = "Updated Name"
	s.Require().NoError(s.repo.Save(ctx, t))

	found, err := s.repo.FindByID(ctx, t.ID)
	s.Require().NoError(err)
	s.Equal("Updated Name", found.Name)
}

func (s *RepositoryIntegrationSuite) TestFindByID_NotFound() {
	ctx := context.Background()
	_, err := s.repo.FindByID(ctx, uuid.New())
	s.ErrorIs(err, domain.ErrTenantNotFound)
}

func (s *RepositoryIntegrationSuite) TestFindBySlug_Found() {
	ctx := context.Background()
	t := s.newSavedTenant("Coffee Shop", "coffee-shop")

	found, err := s.repo.FindBySlug(ctx, "coffee-shop")
	s.Require().NoError(err)
	s.Equal(t.ID, found.ID)
}

func (s *RepositoryIntegrationSuite) TestFindBySlug_NotFound() {
	ctx := context.Background()
	_, err := s.repo.FindBySlug(ctx, "nonexistent-slug")
	s.ErrorIs(err, domain.ErrTenantNotFound)
}

func (s *RepositoryIntegrationSuite) TestExistsBySlug_True() {
	ctx := context.Background()
	s.newSavedTenant("Bakso Solo", "bakso-solo")

	exists, err := s.repo.ExistsBySlug(ctx, "bakso-solo")
	s.Require().NoError(err)
	s.True(exists)
}

func (s *RepositoryIntegrationSuite) TestExistsBySlug_False() {
	ctx := context.Background()
	exists, err := s.repo.ExistsBySlug(ctx, "does-not-exist")
	s.Require().NoError(err)
	s.False(exists)
}

func (s *RepositoryIntegrationSuite) TestSave_TenantWithBranch() {
	ctx := context.Background()
	t := s.newSavedTenant("Multi Branch Corp", "multi-branch-corp")

	branch := domain.NewBranch(t.ID, "Main Branch", domain.Address{
		Street:     "Jl. Sudirman No. 1",
		City:       "Jakarta",
		Province:   "DKI Jakarta",
		PostalCode: "10110",
		Country:    "ID",
	}, "Asia/Jakarta")
	s.Require().NoError(t.AddBranch(branch))
	s.Require().NoError(s.repo.Save(ctx, t))

	found, err := s.repo.FindByID(ctx, t.ID)
	s.Require().NoError(err)
	s.Len(found.Branches, 1)
	s.Equal("Main Branch", found.Branches[0].Name)
	s.Equal("Jl. Sudirman No. 1", found.Branches[0].Address.Street)
}

func (s *RepositoryIntegrationSuite) TestListBranches_Empty() {
	ctx := context.Background()
	t := s.newSavedTenant("Empty Corp", "empty-corp")

	branches, err := s.repo.ListBranches(ctx, t.ID)
	s.Require().NoError(err)
	s.Empty(branches)
}

func (s *RepositoryIntegrationSuite) TestListBranches_Multiple() {
	ctx := context.Background()

	// Growth plan allows 10 branches.
	t, err := domain.NewTenant("Chain Store", "chain-store", domain.PlanTierGrowth)
	s.Require().NoError(err)
	s.Require().NoError(s.repo.Save(ctx, t))

	addr := domain.Address{City: "Jakarta", Country: "ID"}
	for _, name := range []string{"Branch A", "Branch B", "Branch C"} {
		br := domain.NewBranch(t.ID, name, addr, "Asia/Jakarta")
		s.Require().NoError(t.AddBranch(br))
	}
	s.Require().NoError(s.repo.Save(ctx, t))

	branches, err := s.repo.ListBranches(ctx, t.ID)
	s.Require().NoError(err)
	s.Len(branches, 3)
}

func (s *RepositoryIntegrationSuite) TestAddBranch_EnforcesPlanLimit() {
	ctx := context.Background()
	t := s.newSavedTenant("Free Tenant", "free-tenant")

	addr := domain.Address{City: "Jakarta", Country: "ID"}
	br := domain.NewBranch(t.ID, "Only Branch", addr, "Asia/Jakarta")
	s.Require().NoError(t.AddBranch(br))
	s.Require().NoError(s.repo.Save(ctx, t))

	// Reload and attempt to exceed the free-plan branch limit.
	reloaded, err := s.repo.FindByID(ctx, t.ID)
	s.Require().NoError(err)

	extra := domain.NewBranch(reloaded.ID, "Extra Branch", addr, "Asia/Jakarta")
	err = reloaded.AddBranch(extra)
	s.ErrorIs(err, domain.ErrBranchLimitReached)
}

// TestRepositoryIntegration is the suite entry point.
func TestRepositoryIntegration(t *testing.T) {
	suite.Run(t, new(RepositoryIntegrationSuite))
}
