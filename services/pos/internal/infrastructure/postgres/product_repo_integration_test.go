//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/pos/internal/domain/product"
	"github.com/xyn-pos/services/pos/internal/infrastructure/postgres"
)

func TestProductRepo_SaveAndFindByID(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	p, err := product.NewProduct(tenantID, nil, "NP-001", "Nasi Padang", "Deskripsi", 25000_00, product.TaxTypePPN)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, p))

	found, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	assert.Equal(t, "Nasi Padang", found.Name)
	assert.Equal(t, int64(2500000), found.BasePrice)
	assert.Equal(t, product.TaxTypePPN, found.TaxType)
	assert.True(t, found.IsActive)
}

func TestProductRepo_FindBySKU(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	p, err := product.NewProduct(tenantID, nil, "UNIQ-SKU", "Product A", "", 10000, product.TaxTypeNone)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, p))

	found, err := repo.FindBySKU(ctx, tenantID, "UNIQ-SKU")
	require.NoError(t, err)
	assert.Equal(t, p.ID, found.ID)
}

func TestProductRepo_DuplicateSKU_ReturnsErrSKUAlreadyExists(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	p1, err := product.NewProduct(tenantID, nil, "DUPE-SKU", "P1", "", 5000, product.TaxTypeNone)
	require.NoError(t, err)
	p2, err := product.NewProduct(tenantID, nil, "DUPE-SKU", "P2", "", 6000, product.TaxTypeNone)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, p1))
	err = repo.Save(ctx, p2)
	assert.ErrorIs(t, err, product.ErrSKUAlreadyExists)
}

func TestProductRepo_FindByTenant_RLSIsolation(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	ctx := context.Background()

	t1 := uuid.New()
	t2 := uuid.New()

	setRLS(t, pool, t1)
	p1, err := product.NewProduct(t1, nil, "T1-001", "T1 Product", "", 1000, product.TaxTypeNone)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, p1))

	setRLS(t, pool, t2)
	p2, err := product.NewProduct(t2, nil, "T2-001", "T2 Product", "", 2000, product.TaxTypeNone)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, p2))

	setRLS(t, pool, t1)
	products, total, err := repo.FindByTenant(ctx, t1, product.Filter{ActiveOnly: true, Limit: 10})
	require.NoError(t, err)
	assert.Equal(t, 1, total)
	assert.Len(t, products, 1)
	assert.Equal(t, "T1 Product", products[0].Name)
}

func TestProductRepo_Archive(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewProductRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	p, err := product.NewProduct(tenantID, nil, "ARCH-001", "To Archive", "", 5000, product.TaxTypeNone)
	require.NoError(t, err)
	require.NoError(t, repo.Save(ctx, p))

	require.NoError(t, p.Archive())
	require.NoError(t, repo.Update(ctx, p))

	found, err := repo.FindByID(ctx, p.ID)
	require.NoError(t, err)
	assert.False(t, found.IsActive)
}
