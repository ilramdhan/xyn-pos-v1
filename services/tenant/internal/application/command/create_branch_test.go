package command

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// CreateBranchSuite tests the CreateBranchHandler use case using the shared
// MockTenantRepository defined in create_tenant_test.go (same package).
type CreateBranchSuite struct {
	suite.Suite
	repo    *MockTenantRepository
	handler *CreateBranchHandler
}

func (s *CreateBranchSuite) SetupTest() {
	s.repo = new(MockTenantRepository)
	s.handler = NewCreateBranchHandler(s.repo)
}

func (s *CreateBranchSuite) TestHandle_Success() {
	ctx := context.Background()

	existing, err := domain.NewTenant("Kopi Kenangan", "kopi-kenangan", domain.PlanTierGrowth)
	s.Require().NoError(err)

	s.repo.On("FindByID", mock.Anything, existing.ID).Return(existing, nil)
	s.repo.On("Save", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).Return(nil)

	cmd := CreateBranchCommand{
		TenantID: existing.ID,
		Name:           "Cabang Sudirman",
		Address: domain.Address{
			Street:  "Jl. Sudirman No. 5",
			City:    "Jakarta",
			Country: "ID",
		},
		Timezone: "Asia/Jakarta",
	}

	branch, err := s.handler.Handle(ctx, cmd)
	s.Require().NoError(err)
	s.Equal("Cabang Sudirman", branch.Name)
	s.Equal(existing.ID, branch.TenantID)
	s.Equal("Asia/Jakarta", branch.Timezone)
	s.repo.AssertExpectations(s.T())
}

func (s *CreateBranchSuite) TestHandle_DefaultTimezone() {
	ctx := context.Background()

	existing, _ := domain.NewTenant("Test Corp", "test-corp", domain.PlanTierGrowth)
	s.repo.On("FindByID", mock.Anything, existing.ID).Return(existing, nil)
	s.repo.On("Save", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).Return(nil)

	// Empty Timezone should default to Asia/Jakarta.
	cmd := CreateBranchCommand{TenantID: existing.ID, Name: "Branch X", Timezone: ""}
	branch, err := s.handler.Handle(ctx, cmd)

	s.Require().NoError(err)
	s.Equal("Asia/Jakarta", branch.Timezone)
}

func (s *CreateBranchSuite) TestHandle_TenantNotFound() {
	ctx := context.Background()
	s.repo.On("FindByID", mock.Anything, mock.Anything).Return(nil, domain.ErrTenantNotFound)

	_, err := s.handler.Handle(ctx, CreateBranchCommand{Name: "Ghost"})
	s.ErrorIs(err, domain.ErrTenantNotFound)
}

func (s *CreateBranchSuite) TestHandle_ExceedsPlanLimit() {
	ctx := context.Background()

	// Free plan: 1 branch max. Pre-fill one branch so the next fails.
	existing, _ := domain.NewTenant("Free Tenant", "free-tenant", domain.PlanTierFree)
	br := domain.NewBranch(existing.ID, "Existing Branch", domain.Address{}, "Asia/Jakarta")
	s.Require().NoError(existing.AddBranch(br))

	s.repo.On("FindByID", mock.Anything, existing.ID).Return(existing, nil)

	cmd := CreateBranchCommand{TenantID: existing.ID, Name: "One Too Many"}
	_, err := s.handler.Handle(ctx, cmd)
	s.ErrorIs(err, domain.ErrBranchLimitReached)
}

func TestCreateBranchSuite(t *testing.T) {
	suite.Run(t, new(CreateBranchSuite))
}
