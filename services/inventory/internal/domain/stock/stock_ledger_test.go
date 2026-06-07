package stock_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/inventory/internal/domain/stock"
)

func TestStockLedger_Adjust_Positive(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	mov, err := s.Adjust(10, "initial stock")
	require.NoError(t, err)
	assert.Equal(t, int64(10), s.Quantity)
	assert.NotNil(t, mov)
}

func TestStockLedger_Adjust_ZeroDelta_ReturnsError(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, err := s.Adjust(0, "zero")
	assert.ErrorIs(t, err, stock.ErrInvalidDelta)
}

func TestStockLedger_Adjust_NegativeResultingInNegativeQty(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, _ = s.Adjust(5, "add stock")
	_, err := s.Adjust(-10, "overdraw")
	assert.ErrorIs(t, err, stock.ErrInsufficientStock)
}

func TestStockLedger_Deduct_Success(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, _ = s.Adjust(100, "add stock")
	mov, err := s.Deduct(-10, uuid.New())
	require.NoError(t, err)
	assert.Equal(t, int64(90), s.Quantity)
	assert.Equal(t, int64(-10), mov.Delta)
}

func TestStockLedger_Deduct_PositiveDelta_ReturnsError(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, err := s.Deduct(5, uuid.New())
	assert.Error(t, err)
}

func TestStockLedger_Deduct_InsufficientStock(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, _ = s.Adjust(5, "add stock")
	_, err := s.Deduct(-10, uuid.New())
	assert.ErrorIs(t, err, stock.ErrInsufficientStock)
}

func TestStockLedger_IsLowStock(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, _ = s.Adjust(3, "add stock") // qty=3, threshold=5
	assert.True(t, s.IsLowStock())
}

func TestStockLedger_NotLowStock(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, _ = s.Adjust(10, "add stock") // qty=10, threshold=5
	assert.False(t, s.IsLowStock())
}

func TestStockLedger_PopEvents_ClearsAfterReturn(t *testing.T) {
	s := stock.NewStockLedger(uuid.New(), uuid.New(), uuid.New(), nil, "pcs", 5)
	_, _ = s.Adjust(10, "add")
	evs := s.PopEvents()
	assert.Len(t, evs, 1)
	evs2 := s.PopEvents()
	assert.Empty(t, evs2)
}

func TestBOMRecipe_New_ZeroQty_ReturnsError(t *testing.T) {
	_, err := stock.NewBOMRecipe(uuid.New(), uuid.New(), uuid.New(), 0, "g")
	assert.ErrorIs(t, err, stock.ErrInvalidQuantity)
}

func TestBOMRecipe_New_Valid(t *testing.T) {
	r, err := stock.NewBOMRecipe(uuid.New(), uuid.New(), uuid.New(), 200, "g")
	require.NoError(t, err)
	assert.Equal(t, int64(200), r.QuantityPerUnit)
}
