// Package story_test — Phase 4 Story D: Park / Resume Orders.
//
// Covers:
//   - DRAFT order can be parked → StatusParked + OrderParkedEvent
//   - Parked order can be resumed → StatusDraft + OrderResumedEvent
//   - Items are preserved across park/resume cycle
//   - Non-draft order cannot be parked
//   - Non-parked order cannot be resumed
//
// Run with: go test ./services/pos/tests/story/... -v
package story_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orderdomain "github.com/xyn-pos/services/pos/internal/domain/order"
)

// TestStoryD_ParkResumeOrder validates the park/resume order lifecycle.
func TestStoryD_ParkResumeOrder(t *testing.T) {
	tenantID := uuid.New()
	branchID := uuid.New()
	cashierID := uuid.New()
	taxCfg := orderdomain.TaxConfig{Type: orderdomain.TaxTypePPN, Rate: 1100}

	// =========================================================================
	// Step 1: Create a DRAFT order with items
	// =========================================================================
	t.Log("Step 1: Create DRAFT order with 2 items")

	o, err := orderdomain.NewOrder(
		tenantID, branchID, nil, cashierID,
		"ORD-PARK-001", orderdomain.OrderTypeDineIn, "3",
		"idem-park-001", taxCfg,
	)
	require.NoError(t, err)

	require.NoError(t, o.AddItem(orderdomain.OrderItem{
		ProductID: uuid.New(), ProductName: "Nasi Padang",
		UnitPrice: 1_500_000, Quantity: 2,
	}))
	require.NoError(t, o.AddItem(orderdomain.OrderItem{
		ProductID: uuid.New(), ProductName: "Es Teh", //nolint:misspell // Indonesian: iced tea
		UnitPrice: 500_000, Quantity: 2,
	}))
	require.Len(t, o.Items, 2)
	subtotalBeforePark := o.Subtotal
	assert.Equal(t, orderdomain.StatusDraft, o.Status)

	// =========================================================================
	// Step 2: Park the order
	// =========================================================================
	t.Log("Step 2: Park DRAFT order")

	err = o.Park()
	require.NoError(t, err)
	assert.Equal(t, orderdomain.StatusParked, o.Status)
	assert.Len(t, o.Items, 2, "items must be preserved after park")
	assert.Equal(t, subtotalBeforePark, o.Subtotal, "totals must be preserved after park")
	t.Logf("  Order %s parked. Subtotal: %d sen", o.ID, o.Subtotal)

	// =========================================================================
	// Step 3: Verify OrderParkedEvent
	// =========================================================================
	t.Log("Step 3: Verify OrderParkedEvent payload")

	evs := o.PopEvents()
	require.Len(t, evs, 1)
	parkedEv, ok := evs[0].(orderdomain.OrderParkedEvent)
	require.True(t, ok, "expected OrderParkedEvent")
	assert.Equal(t, o.ID, parkedEv.OrderID)
	assert.Equal(t, tenantID, parkedEv.TenantID)
	assert.Equal(t, branchID, parkedEv.BranchID)
	assert.Equal(t, cashierID, parkedEv.CashierID)
	assert.False(t, parkedEv.OccurredAt.IsZero())

	// =========================================================================
	// Step 4: Resume the order
	// =========================================================================
	t.Log("Step 4: Resume parked order")

	err = o.Resume()
	require.NoError(t, err)
	assert.Equal(t, orderdomain.StatusDraft, o.Status)
	assert.Len(t, o.Items, 2, "items must be preserved after resume")
	assert.Equal(t, subtotalBeforePark, o.Subtotal, "totals must be preserved after resume")

	// =========================================================================
	// Step 5: Verify OrderResumedEvent
	// =========================================================================
	t.Log("Step 5: Verify OrderResumedEvent payload")

	evs = o.PopEvents()
	require.Len(t, evs, 1)
	resumedEv, ok := evs[0].(orderdomain.OrderResumedEvent)
	require.True(t, ok, "expected OrderResumedEvent")
	assert.Equal(t, o.ID, resumedEv.OrderID)
	assert.Equal(t, tenantID, resumedEv.TenantID)
	assert.Equal(t, branchID, resumedEv.BranchID)
	assert.Equal(t, cashierID, resumedEv.CashierID)
	assert.False(t, resumedEv.OccurredAt.IsZero())

	// =========================================================================
	// Step 6: After resume, order can be modified and submitted normally
	// =========================================================================
	t.Log("Step 6: Add item after resume, then submit")

	require.NoError(t, o.AddItem(orderdomain.OrderItem{
		ProductID: uuid.New(), ProductName: "Rendang",
		UnitPrice: 2_500_000, Quantity: 1,
	}))
	assert.Len(t, o.Items, 3)
	require.NoError(t, o.Submit())
	assert.Equal(t, orderdomain.StatusPendingPayment, o.Status)

	// =========================================================================
	// Step 7: Guards — cannot park non-draft orders
	// =========================================================================
	t.Log("Step 7: Guards — park/resume state machine")

	// Cannot park a pending_payment order.
	o2, err := orderdomain.NewOrder(tenantID, branchID, nil, cashierID, "ORD-PARK-002",
		orderdomain.OrderTypeDineIn, "", "idem-park-002", taxCfg)
	require.NoError(t, err)
	require.NoError(t, o2.AddItem(orderdomain.OrderItem{ProductID: uuid.New(), ProductName: "Item", UnitPrice: 100, Quantity: 1}))
	require.NoError(t, o2.Submit())
	err = o2.Park()
	assert.ErrorIs(t, err, orderdomain.ErrInvalidStatusTransition)

	// Cannot resume a draft order (not parked).
	o3, err := orderdomain.NewOrder(tenantID, branchID, nil, cashierID, "ORD-PARK-003",
		orderdomain.OrderTypeDineIn, "", "idem-park-003", taxCfg)
	require.NoError(t, err)
	err = o3.Resume()
	assert.ErrorIs(t, err, orderdomain.ErrOrderNotParked)
}
