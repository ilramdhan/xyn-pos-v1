package query_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"

	"github.com/xyn-pos/services/tenant/internal/application/query"
	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// listMockRepo extends mockQueryRepo with a ListBranches implementation.
type listMockRepo struct {
	mockQueryRepo
	listBranchesFn func(ctx context.Context, tenantID uuid.UUID) ([]domain.Branch, error)
}

func (m *listMockRepo) ListBranches(ctx context.Context, tenantID uuid.UUID) ([]domain.Branch, error) {
	return m.listBranchesFn(ctx, tenantID)
}

// ListBranchesSuite tests the ListBranchesHandler.
type ListBranchesSuite struct {
	suite.Suite
	repo    *listMockRepo
	handler *query.ListBranchesHandler
}

func (s *ListBranchesSuite) SetupTest() {
	s.repo = &listMockRepo{}
	s.handler = query.NewListBranchesHandler(s.repo)
}

func (s *ListBranchesSuite) TestHandle_ReturnsBranches() {
	ctx := context.Background()
	tenantID := uuid.New()

	want := []domain.Branch{
		{ID: uuid.New(), TenantID: tenantID, Name: "Branch A"},
		{ID: uuid.New(), TenantID: tenantID, Name: "Branch B"},
	}
	s.repo.listBranchesFn = func(_ context.Context, id uuid.UUID) ([]domain.Branch, error) {
		s.Equal(tenantID, id)
		return want, nil
	}

	got, err := s.handler.Handle(ctx, query.ListBranchesQuery{TenantID: tenantID})
	s.Require().NoError(err)
	s.Len(got, 2)
	s.Equal("Branch A", got[0].Name)
}

func (s *ListBranchesSuite) TestHandle_EmptyList() {
	ctx := context.Background()
	s.repo.listBranchesFn = func(_ context.Context, _ uuid.UUID) ([]domain.Branch, error) {
		return []domain.Branch{}, nil
	}

	got, err := s.handler.Handle(ctx, query.ListBranchesQuery{TenantID: uuid.New()})
	s.Require().NoError(err)
	s.Empty(got)
}

func (s *ListBranchesSuite) TestHandle_RepoError() {
	ctx := context.Background()
	s.repo.listBranchesFn = func(_ context.Context, _ uuid.UUID) ([]domain.Branch, error) {
		return nil, domain.ErrTenantNotFound
	}

	_, err := s.handler.Handle(ctx, query.ListBranchesQuery{TenantID: uuid.New()})
	s.ErrorIs(err, domain.ErrTenantNotFound)
}

func TestListBranchesSuite(t *testing.T) {
	suite.Run(t, new(ListBranchesSuite))
}
