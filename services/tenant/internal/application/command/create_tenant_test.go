package command

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	domain "github.com/xyn-pos/services/tenant/internal/domain/tenant"
)

// MockTenantRepository is a hand-written mock.
type MockTenantRepository struct{ mock.Mock }

func (m *MockTenantRepository) Save(ctx context.Context, t *domain.Tenant) error {
	return m.Called(ctx, t).Error(0)
}
func (m *MockTenantRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}
func (m *MockTenantRepository) FindBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	args := m.Called(ctx, slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Tenant), args.Error(1)
}
func (m *MockTenantRepository) ExistsBySlug(ctx context.Context, slug string) (bool, error) {
	args := m.Called(ctx, slug)
	return args.Bool(0), args.Error(1)
}
func (m *MockTenantRepository) ListBranches(ctx context.Context, tenantID uuid.UUID) ([]domain.Branch, error) {
	args := m.Called(ctx, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Branch), args.Error(1)
}

// MockPublisher is a hand-written no-op.
type MockPublisher struct{ mock.Mock }

func (m *MockPublisher) Publish(ctx context.Context, topic string, tenantID uuid.UUID, eventType string, payload []byte) error {
	return m.Called(ctx, topic, tenantID, eventType, payload).Error(0)
}

type CreateTenantSuite struct {
	suite.Suite
	repo    *MockTenantRepository
	handler *CreateTenantHandler
}

func (s *CreateTenantSuite) SetupTest() {
	s.repo = new(MockTenantRepository)
	s.handler = NewCreateTenantHandler(s.repo, nil)
}

func TestCreateTenantSuite(t *testing.T) { suite.Run(t, new(CreateTenantSuite)) }

func (s *CreateTenantSuite) TestHandle_ValidInput_ShouldCreateTenant() {
	s.repo.On("ExistsBySlug", mock.Anything, "my-shop").Return(false, nil)
	s.repo.On("Save", mock.Anything, mock.AnythingOfType("*tenant.Tenant")).Return(nil)

	cmd := CreateTenantCommand{IdempotencyKey: uuid.NewString(), Name: "My Shop", Slug: "my-shop", Plan: domain.PlanTierFree}
	result, err := s.handler.Handle(context.Background(), cmd)

	s.Require().NoError(err)
	s.Equal("my-shop", result.Slug)
	s.Equal(domain.StatusActive, result.Status)
}

func (s *CreateTenantSuite) TestHandle_DuplicateSlug_ShouldReturnErrSlugAlreadyTaken() {
	s.repo.On("ExistsBySlug", mock.Anything, "taken-slug").Return(true, nil)

	cmd := CreateTenantCommand{IdempotencyKey: uuid.NewString(), Name: "Shop", Slug: "taken-slug"}
	_, err := s.handler.Handle(context.Background(), cmd)

	s.ErrorIs(err, domain.ErrSlugAlreadyTaken)
}

func (s *CreateTenantSuite) TestHandle_RepoError_ShouldReturnError() {
	s.repo.On("ExistsBySlug", mock.Anything, "good-slug").Return(false, nil)
	s.repo.On("Save", mock.Anything, mock.Anything).Return(errors.New("db error"))

	cmd := CreateTenantCommand{IdempotencyKey: uuid.NewString(), Name: "Shop", Slug: "good-slug"}
	_, err := s.handler.Handle(context.Background(), cmd)

	s.Error(err)
}
