package product_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xyn-pos/services/pos/internal/domain/product"
)

func TestNewProduct_NegativeBasePrice_ReturnsError(t *testing.T) {
	_, err := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi Padang", "", -100, product.TaxTypePPN)
	assert.ErrorIs(t, err, product.ErrInvalidPrice)
}

func TestNewProduct_EmptyName_ReturnsError(t *testing.T) {
	_, err := product.NewProduct(uuid.New(), nil, "SKU-001", "", "", 15000, product.TaxTypePPN)
	assert.ErrorIs(t, err, product.ErrInvalidName)
}

func TestNewProduct_Valid_SetsIsActiveTrue(t *testing.T) {
	p, err := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi Padang", "Desc", 25000_00, product.TaxTypePPN)
	require.NoError(t, err)
	assert.True(t, p.IsActive)
	assert.NotEqual(t, uuid.Nil, p.ID)
}

func TestProduct_AddVariant_NegativeFinalPrice_ReturnsError(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 5000, product.TaxTypePPN)
	err := p.AddVariant(product.Variant{
		ID:         uuid.New(),
		Name:       "Porsi Kecil",
		PriceDelta: -10000, // base 5000 + delta -10000 = -5000 → invalid
		SKU:        "SKU-001-S",
		IsActive:   true,
	})
	assert.ErrorIs(t, err, product.ErrVariantPriceInvalid)
}

func TestProduct_AddVariant_Valid_AddsToSlice(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 25000_00, product.TaxTypePPN)
	v := product.Variant{ID: uuid.New(), Name: "Small", PriceDelta: -5000_00, SKU: "SKU-001-S", IsActive: true}
	err := p.AddVariant(v)
	require.NoError(t, err)
	assert.Len(t, p.Variants, 1)
}

func TestProduct_AddAddonGroup_MaxSelectionsZero_ReturnsError(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 25000_00, product.TaxTypePPN)
	err := p.AddAddonGroup(product.AddonGroup{
		ID:            uuid.New(),
		Name:          "Extra Lauk",
		IsRequired:    false,
		MaxSelections: 0, // must be >= 1
	})
	assert.ErrorIs(t, err, product.ErrInvalidAddonGroup)
}

func TestProduct_Archive_AlreadyInactive_ReturnsError(t *testing.T) {
	p, _ := product.NewProduct(uuid.New(), nil, "SKU-001", "Nasi", "", 25000_00, product.TaxTypePPN)
	p.Archive()
	err := p.Archive()
	assert.ErrorIs(t, err, product.ErrProductAlreadyArchived)
}
