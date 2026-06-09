package order_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

func TestPark_Success(t *testing.T) {
	o := newTestOrder(t)
	err := o.Park()
	require.NoError(t, err)
	assert.Equal(t, order.StatusParked, o.Status)
	evs := o.PopEvents()
	require.Len(t, evs, 1)
	_, ok := evs[0].(order.OrderParkedEvent)
	assert.True(t, ok, "expected OrderParkedEvent")
}

func TestPark_NonDraft_Fails(t *testing.T) {
	tests := []struct {
		name   string
		setup  func(*order.Order)
	}{
		{
			name: "pending_payment",
			setup: func(o *order.Order) {
				_ = o.AddItem(testItem("Item", 10000, 1))
				_ = o.Submit()
			},
		},
		{
			name: "paid",
			setup: func(o *order.Order) {
				_ = o.AddItem(testItem("Item", 10000, 1))
				_ = o.Submit()
				_ = o.MarkPaid()
			},
		},
		{
			name: "cancelled",
			setup: func(o *order.Order) {
				_ = o.Cancel("reason")
			},
		},
		{
			name: "already_parked",
			setup: func(o *order.Order) {
				_ = o.Park()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := newTestOrder(t)
			tc.setup(o)
			_ = o.PopEvents()
			err := o.Park()
			assert.ErrorIs(t, err, order.ErrInvalidStatusTransition)
		})
	}
}

func TestResume_Success(t *testing.T) {
	o := newTestOrder(t)
	require.NoError(t, o.Park())
	_ = o.PopEvents()

	err := o.Resume()
	require.NoError(t, err)
	assert.Equal(t, order.StatusDraft, o.Status)
	evs := o.PopEvents()
	require.Len(t, evs, 1)
	_, ok := evs[0].(order.OrderResumedEvent)
	assert.True(t, ok, "expected OrderResumedEvent")
}

func TestResume_NonParked_Fails(t *testing.T) {
	tests := []struct {
		name  string
		setup func(*order.Order)
	}{
		{name: "draft", setup: func(o *order.Order) {}},
		{
			name: "pending_payment",
			setup: func(o *order.Order) {
				_ = o.AddItem(testItem("Item", 10000, 1))
				_ = o.Submit()
			},
		},
		{
			name: "paid",
			setup: func(o *order.Order) {
				_ = o.AddItem(testItem("Item", 10000, 1))
				_ = o.Submit()
				_ = o.MarkPaid()
			},
		},
		{
			name: "cancelled",
			setup: func(o *order.Order) {
				_ = o.Cancel("reason")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			o := newTestOrder(t)
			tc.setup(o)
			_ = o.PopEvents()
			err := o.Resume()
			assert.ErrorIs(t, err, order.ErrOrderNotParked)
		})
	}
}

func TestParkResume_EventFields(t *testing.T) {
	o := newTestOrder(t)
	require.NoError(t, o.Park())
	evs := o.PopEvents()
	require.Len(t, evs, 1)

	ev, ok := evs[0].(order.OrderParkedEvent)
	require.True(t, ok)
	assert.Equal(t, o.ID, ev.OrderID)
	assert.Equal(t, o.TenantID, ev.TenantID)
	assert.Equal(t, o.BranchID, ev.BranchID)
	assert.Equal(t, o.CashierID, ev.CashierID)
	assert.False(t, ev.OccurredAt.IsZero())

	require.NoError(t, o.Resume())
	evs = o.PopEvents()
	require.Len(t, evs, 1)

	rev, ok := evs[0].(order.OrderResumedEvent)
	require.True(t, ok)
	assert.Equal(t, o.ID, rev.OrderID)
	assert.Equal(t, o.TenantID, rev.TenantID)
	assert.Equal(t, o.BranchID, rev.BranchID)
	assert.Equal(t, o.CashierID, rev.CashierID)
	assert.False(t, rev.OccurredAt.IsZero())
}
