package order_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

func TestOrder_AddItem_DraftStatus_Success(t *testing.T) {
	o := newTestOrder(t)
	err := o.AddItem(testItem("Nasi Padang", 25000_00, 1))
	require.NoError(t, err)
	assert.Len(t, o.Items, 1)
}

func TestOrder_AddItem_NotDraft_ReturnsError(t *testing.T) {
	o := newTestOrder(t)
	_ = o.AddItem(testItem("Nasi", 10000, 1))
	_ = o.Submit()
	err := o.AddItem(testItem("Ayam", 15000, 1))
	assert.ErrorIs(t, err, order.ErrOrderNotDraft)
}

func TestOrder_Submit_EmptyItems_ReturnsError(t *testing.T) {
	o := newTestOrder(t)
	err := o.Submit()
	assert.ErrorIs(t, err, order.ErrOrderEmpty)
}

func TestOrder_StateMachine_AllValidTransitions(t *testing.T) {
	o := newTestOrder(t)
	_ = o.AddItem(testItem("Item", 10000, 1))

	assert.Equal(t, order.StatusDraft, o.Status)
	require.NoError(t, o.Submit())
	assert.Equal(t, order.StatusPendingPayment, o.Status)
	require.NoError(t, o.MarkPaid())
	assert.Equal(t, order.StatusPaid, o.Status)
}

func TestOrder_StateMachine_InvalidTransition_ReturnsError(t *testing.T) {
	o := newTestOrder(t)
	_ = o.AddItem(testItem("Item", 10000, 1))
	_ = o.Submit()
	_ = o.MarkPaid()

	err := o.Cancel("")
	assert.ErrorIs(t, err, order.ErrInvalidStatusTransition)
}

func TestOrder_TaxCalculation_PPN11Percent(t *testing.T) {
	o := newTestOrderWithTax(t, order.TaxConfig{Type: "ppn", Rate: 1100})
	_ = o.AddItem(testItem("Item", 100_00, 1))
	assert.Equal(t, int64(100_00), o.Subtotal)
	assert.Equal(t, int64(11_00), o.TaxAmount)
	assert.Equal(t, int64(111_00), o.Total)
}

func TestOrder_TaxCalculation_PB1_NoAddedTax(t *testing.T) {
	o := newTestOrderWithTax(t, order.TaxConfig{Type: "pb1", Rate: 1000})
	_ = o.AddItem(testItem("Item", 100_00, 1))
	assert.Equal(t, int64(100_00), o.Subtotal)
	assert.Equal(t, int64(10_00), o.TaxAmount)
	assert.Equal(t, int64(100_00), o.Total)
}

func TestOrder_TaxCalculation_TaxTypeNone(t *testing.T) {
	o := newTestOrderWithTax(t, order.TaxConfig{Type: "none", Rate: 0})
	_ = o.AddItem(testItem("Item", 100_00, 1))
	assert.Equal(t, int64(0), o.TaxAmount)
	assert.Equal(t, int64(100_00), o.Total)
}

func TestOrder_DiscountFixed_ExceedsSubtotal_ReturnsError(t *testing.T) {
	o := newTestOrder(t)
	_ = o.AddItem(testItem("Item", 100_00, 1))
	err := o.ApplyDiscount(order.Discount{Type: order.DiscountTypeFixed, Amount: 200_00})
	assert.ErrorIs(t, err, order.ErrDiscountExceedsSubtotal)
}

func TestOrder_DiscountPercent_Over100_ReturnsError(t *testing.T) {
	o := newTestOrder(t)
	_ = o.AddItem(testItem("Item", 100_00, 1))
	err := o.ApplyDiscount(order.Discount{Type: order.DiscountTypePercent, Amount: 10001})
	assert.ErrorIs(t, err, order.ErrDiscountPercentInvalid)
}

func TestOrder_Total_SubtotalPlusTaxMinusDiscount(t *testing.T) {
	o := newTestOrderWithTax(t, order.TaxConfig{Type: "ppn", Rate: 1100})
	_ = o.AddItem(testItem("Item", 100_00, 2))
	_ = o.ApplyDiscount(order.Discount{Type: order.DiscountTypeFixed, Amount: 20_00})
	assert.Equal(t, int64(200_00), o.Subtotal)
	assert.Equal(t, int64(22_00), o.TaxAmount)
	assert.Equal(t, int64(20_00), o.DiscountAmount)
	assert.Equal(t, int64(202_00), o.Total)
}

func TestShift_CannotCloseAlreadyClosed(t *testing.T) {
	s := order.NewShift(uuid.New(), uuid.New(), uuid.New(), 500_00_00)
	_ = s.Close(450_00_00)
	err := s.Close(400_00_00)
	assert.ErrorIs(t, err, order.ErrShiftAlreadyClosed)
}

func newTestOrder(t *testing.T) *order.Order {
	t.Helper()
	o, err := order.NewOrder(uuid.New(), uuid.New(), nil,
		uuid.New(), "ORD-001", order.OrderTypeDineIn, "5", "idem-001",
		order.TaxConfig{Type: "none", Rate: 0})
	require.NoError(t, err)
	return o
}

func newTestOrderWithTax(t *testing.T, tax order.TaxConfig) *order.Order {
	t.Helper()
	o, err := order.NewOrder(uuid.New(), uuid.New(), nil,
		uuid.New(), "ORD-001", order.OrderTypeDineIn, "", "idem-001", tax)
	require.NoError(t, err)
	return o
}

func testItem(name string, unitPrice int64, qty int) order.OrderItem {
	return order.OrderItem{
		ID:          uuid.New(),
		ProductID:   uuid.New(),
		ProductName: name,
		UnitPrice:   unitPrice,
		Quantity:    qty,
		Subtotal:    unitPrice * int64(qty),
	}
}
