package query_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/xyn-pos/services/tenant/internal/application/query"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// mockQueryRepo is a minimal mock of domain.Repository for query tests.
type mockQueryRepo struct {
	findByIDFn func(ctx context.Context, id uuid.UUID) (*domain.Tenant, error)
}

func (m *mockQueryRepo) Save(_ context.Context, _ *domain.Tenant) error { return nil }
func (m *mockQueryRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	return m.findByIDFn(ctx, id)
}
func (m *mockQueryRepo) FindBySlug(_ context.Context, _ string) (*domain.Tenant, error) {
	return nil, nil
}
func (m *mockQueryRepo) ExistsBySlug(_ context.Context, _ string) (bool, error) { return false, nil }
func (m *mockQueryRepo) ListBranches(_ context.Context, _ uuid.UUID) ([]domain.Branch, error) {
	return nil, nil
}

// GetTenantSuite tests the GetTenantHandler.
type GetTenantSuite struct {
	suite.Suite
	repo    *mockQueryRepo
	handler *query.GetTenantHandler
}

func (s *GetTenantSuite) SetupTest() {
	s.repo = &mockQueryRepo{}
	s.handler = query.NewGetTenantHandler(s.repo)
}

func (s *GetTenantSuite) TestHandle_Found() {
	ctx := context.Background()

	want, err := domain.NewTenant("My Cafe", "my-cafe", domain.PlanTierFree)
	s.Require().NoError(err)

	s.repo.findByIDFn = func(_ context.Context, id uuid.UUID) (*domain.Tenant, error) {
		if id == want.ID {
			return want, nil
		}
		return nil, domain.ErrTenantNotFound
	}

	got, err := s.handler.Handle(ctx, query.GetTenantQuery{TenantID: want.ID})
	s.Require().NoError(err)
	s.Equal(want.ID, got.ID)
	s.Equal("My Cafe", got.Name)
}

func (s *GetTenantSuite) TestHandle_NotFound() {
	ctx := context.Background()
	s.repo.findByIDFn = func(_ context.Context, _ uuid.UUID) (*domain.Tenant, error) {
		return nil, domain.ErrTenantNotFound
	}

	_, err := s.handler.Handle(ctx, query.GetTenantQuery{TenantID: uuid.New()})
	s.ErrorIs(err, domain.ErrTenantNotFound)
}

func TestGetTenantSuite(t *testing.T) {
	suite.Run(t, new(GetTenantSuite))
}
