//go:build integration

package postgres_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/pos/internal/domain/order"
	"github.com/xyn-pos/services/pos/internal/infrastructure/postgres"
)

func TestOrderRepo_CreateAndFindByID(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewOrderRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	o := makeTestOrder(t, tenantID)
	require.NoError(t, repo.Save(ctx, o))

	found, err := repo.FindByID(ctx, o.ID)
	require.NoError(t, err)
	assert.Equal(t, o.OrderNumber, found.OrderNumber)
	assert.Equal(t, order.StatusDraft, found.Status)
}

func TestOrderRepo_Idempotency_DuplicateKey(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewOrderRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	o := makeTestOrder(t, tenantID)
	require.NoError(t, repo.Save(ctx, o))

	found, err := repo.FindByIdempotencyKey(ctx, tenantID, o.IdempotencyKey)
	require.NoError(t, err)
	assert.Equal(t, o.ID, found.ID)
}

func TestShiftRepo_OpenAndClose(t *testing.T) {
	pool := startTestDB(t)
	repo := postgres.NewShiftRepository(pool)
	tenantID := uuid.New()
	setRLS(t, pool, tenantID)
	ctx := context.Background()

	s := order.NewShift(tenantID, uuid.New(), uuid.New(), 500_000_00)
	require.NoError(t, repo.Save(ctx, s))

	found, err := repo.FindByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ShiftStatusOpen, found.Status)

	_ = found.Close(450_000_00)
	require.NoError(t, repo.Update(ctx, found))

	closed, err := repo.FindByID(ctx, s.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ShiftStatusClosed, closed.Status)
	assert.NotNil(t, closed.ClosingCash)
}

func makeTestOrder(t *testing.T, tenantID uuid.UUID) *order.Order {
	t.Helper()
	o, err := order.NewOrder(tenantID, uuid.New(), nil, uuid.New(),
		"ORD-TEST-001", order.OrderTypeDineIn, "5", uuid.NewString(),
		order.TaxConfig{Type: "none", Rate: 0})
	require.NoError(t, err)
	return o
}
