package command_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/services/pos/internal/application/command"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

type MockProductRepo struct{ mock.Mock }

func (m *MockProductRepo) FindByID(ctx context.Context, id uuid.UUID) (*product.Product, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.Product), args.Error(1)
}
func (m *MockProductRepo) FindBySKU(ctx context.Context, tenantID uuid.UUID, sku string) (*product.Product, error) {
	args := m.Called(ctx, tenantID, sku)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*product.Product), args.Error(1)
}
func (m *MockProductRepo) FindByTenant(ctx context.Context, tenantID uuid.UUID, f product.Filter) ([]*product.Product, int, error) {
	args := m.Called(ctx, tenantID, f)
	return args.Get(0).([]*product.Product), args.Int(1), args.Error(2)
}
func (m *MockProductRepo) Save(ctx context.Context, p *product.Product) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockProductRepo) Update(ctx context.Context, p *product.Product) error {
	return m.Called(ctx, p).Error(0)
}
func (m *MockProductRepo) SetBranchPrice(ctx context.Context, bp product.BranchProduct) error {
	return m.Called(ctx, bp).Error(0)
}
func (m *MockProductRepo) HasActiveOrders(ctx context.Context, productID uuid.UUID) (bool, error) {
	args := m.Called(ctx, productID)
	return args.Bool(0), args.Error(1)
}

func TestCreateProduct_SKUAlreadyExists_ReturnsError(t *testing.T) {
	repo := &MockProductRepo{}
	tenantID := uuid.New()

	existing, _ := product.NewProduct(tenantID, nil, "DUPE", "Existing", "", 1000, product.TaxTypeNone)
	repo.On("FindBySKU", mock.Anything, tenantID, "DUPE").Return(existing, nil)

	h := command.NewCreateProductHandler(repo)
	_, err := h.Handle(context.Background(), command.CreateProductInput{
		TenantID: tenantID, SKU: "DUPE", Name: "New", BasePrice: 5000, TaxType: product.TaxTypeNone,
	})
	assert.ErrorIs(t, err, product.ErrSKUAlreadyExists)
}

func TestCreateProduct_ValidInput_SavesProduct(t *testing.T) {
	repo := &MockProductRepo{}
	tenantID := uuid.New()

	repo.On("FindBySKU", mock.Anything, tenantID, "NEW-001").Return(nil, product.ErrProductNotFound)
	repo.On("Save", mock.Anything, mock.MatchedBy(func(p *product.Product) bool {
		return p.Name == "Ayam Goreng" && p.BasePrice == 20000_00
	})).Return(nil)

	h := command.NewCreateProductHandler(repo)
	p, err := h.Handle(context.Background(), command.CreateProductInput{
		TenantID:  tenantID,
		SKU:       "NEW-001",
		Name:      "Ayam Goreng",
		BasePrice: 20000_00,
		TaxType:   product.TaxTypePPN,
	})
	require.NoError(t, err)
	assert.Equal(t, "Ayam Goreng", p.Name)
}

func TestArchiveProduct_HasActiveOrders_ReturnsError(t *testing.T) {
	repo := &MockProductRepo{}
	productID := uuid.New()
	p, _ := product.NewProduct(uuid.New(), nil, "ARCH-001", "Active Product", "", 5000, product.TaxTypeNone)
	p.ID = productID

	repo.On("FindByID", mock.Anything, productID).Return(p, nil)
	repo.On("HasActiveOrders", mock.Anything, productID).Return(true, nil)

	h := command.NewArchiveProductHandler(repo)
	err := h.Handle(context.Background(), productID)
	assert.ErrorIs(t, err, product.ErrProductHasActiveOrders)
}
