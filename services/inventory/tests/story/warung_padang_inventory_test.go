// Package story_test contains domain-level story tests that validate the stock
// ledger lifecycle without any infrastructure dependencies.
//
// This file covers the Inventory bounded context slice of the "Warung Padang Golden Path":
//   - Stock initialization (Adjust)
//   - Stock deduction on order paid (Deduct)
//   - Low-stock threshold detection (IsLowStock)
//   - StockDeductedEvent emission with correct payload
//   - Insufficient stock rejection
//
// See also:
//   - services/pos/tests/story — POS order lifecycle
//   - services/payment/tests/story — payment state machine
//
// Run with: go test ./services/inventory/tests/story/... -v
package story_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	stockdomain "github.com/xyn-pos/services/inventory/internal/domain/stock"
)

// TestWarungPadangGoldenPath_Inventory covers the Inventory bounded context slice
// of the Warung Padang scenario: stock initialization → deduct on order paid →
// low-stock check → insufficient stock rejection.
//
// Units: grams (g). Nasi Padang uses 300g rice per serving.
func TestWarungPadangGoldenPath_Inventory(t *testing.T) {
	tenantID := uuid.New()
	branchID := uuid.New()
	orderID := uuid.New() // represents the paid order from POS story

	// =========================================================================
	// Step 1: Initialize rice stock ledger
	// =========================================================================
	t.Log("Step 1: Create rice stock ledger (low-stock threshold: 500g)")

	riceID := uuid.New()
	ledger := stockdomain.NewStockLedger(tenantID, branchID, riceID, nil, "g", 500)
	assert.NotEqual(t, uuid.Nil, ledger.ID)
	assert.Equal(t, int64(0), ledger.Quantity)
	assert.Equal(t, "g", ledger.Unit)
	assert.Equal(t, int64(500), ledger.LowStockThreshold)
	assert.True(t, ledger.IsLowStock(), "zero stock should trigger low-stock")

	// =========================================================================
	// Step 2: Add initial stock — 5,000g of rice
	// =========================================================================
	t.Log("Step 2: Adjust stock → add 5000g initial stock")

	movement, err := ledger.Adjust(5000, "initial stock from supplier")
	require.NoError(t, err)
	assert.Equal(t, int64(5000), ledger.Quantity)
	assert.Equal(t, int64(5000), movement.Delta)
	assert.Equal(t, stockdomain.MovementIn, movement.Type)
	assert.False(t, ledger.IsLowStock(), "5000g should not trigger low stock")
	t.Logf("  Rice in stock: %dg", ledger.Quantity)

	// Guard: zero delta rejected
	_, err = ledger.Adjust(0, "zero delta")
	assert.ErrorIs(t, err, stockdomain.ErrInvalidDelta)

	// Guard: cannot go negative
	_, err = ledger.Adjust(-9999, "overcorrection")
	assert.ErrorIs(t, err, stockdomain.ErrInsufficientStock)

	// =========================================================================
	// Step 3: Deduct stock on order paid (2x Nasi Padang = 2 × 300g = 600g)
	// =========================================================================
	t.Log("Step 3: Deduct 600g for 2x Nasi Padang on order paid")

	deductMovement, err := ledger.Deduct(-600, orderID)
	require.NoError(t, err)
	assert.Equal(t, int64(4400), ledger.Quantity)
	assert.Equal(t, int64(-600), deductMovement.Delta)
	assert.Equal(t, stockdomain.MovementOut, deductMovement.Type)
	assert.Equal(t, orderID.String(), deductMovement.ReferenceID)
	assert.False(t, ledger.IsLowStock(), "4400g still well above 500g threshold")
	t.Logf("  Rice remaining: %dg", ledger.Quantity)

	// Guard: positive delta rejected by Deduct
	_, err = ledger.Deduct(100, orderID)
	assert.Error(t, err, "Deduct must reject positive delta")

	// =========================================================================
	// Step 4: Verify domain events
	// =========================================================================
	t.Log("Step 4: Verify domain events from Adjust + Deduct")

	events := ledger.PopEvents()
	// Adjust(5000) → StockAdjustedEvent; Deduct(-600) → StockDeductedEvent
	require.Len(t, events, 2, "expected StockAdjustedEvent + StockDeductedEvent")

	var adjustedEv stockdomain.StockAdjustedEvent
	var deductedEv stockdomain.StockDeductedEvent
	for _, ev := range events {
		switch e := ev.(type) {
		case stockdomain.StockAdjustedEvent:
			adjustedEv = e
		case stockdomain.StockDeductedEvent:
			deductedEv = e
		}
	}

	assert.Equal(t, int64(5000), adjustedEv.Delta, "adjusted event delta must be +5000")
	assert.Equal(t, tenantID, adjustedEv.TenantID)

	assert.Equal(t, int64(-600), deductedEv.Delta, "deducted event delta must be -600")
	assert.Equal(t, orderID, deductedEv.OrderID)
	assert.Equal(t, tenantID, deductedEv.TenantID)
	assert.False(t, deductedEv.OccurredAt.IsZero())

	// Events cleared after pop
	assert.Empty(t, ledger.PopEvents())

	// =========================================================================
	// Step 5: Drain stock to trigger low-stock alert
	// =========================================================================
	t.Log("Step 5: Drain stock near threshold to trigger low-stock")

	_, err = ledger.Deduct(-3950, uuid.New()) // 4400 - 3950 = 450g remaining
	require.NoError(t, err)
	assert.Equal(t, int64(450), ledger.Quantity)
	assert.True(t, ledger.IsLowStock(), "450g ≤ 500g threshold: low-stock alert")
	t.Logf("  Rice remaining: %dg (LOW STOCK — threshold: %dg)", ledger.Quantity, ledger.LowStockThreshold)

	// =========================================================================
	// Step 6: Insufficient stock rejection
	// =========================================================================
	t.Log("Step 6: Insufficient stock — reject order that exceeds available stock")

	_, err = ledger.Deduct(-500, uuid.New()) // would go to -50 — rejected
	assert.ErrorIs(t, err, stockdomain.ErrInsufficientStock)
	assert.Equal(t, int64(450), ledger.Quantity, "quantity must not change on rejected deduction")

	// =========================================================================
	// Step 7: Stock replenishment (new supplier delivery)
	// =========================================================================
	t.Log("Step 7: Replenish stock")

	ledger.PopEvents() // clear events from drain
	_, err = ledger.Adjust(10_000, "supplier delivery")
	require.NoError(t, err)
	assert.Equal(t, int64(10_450), ledger.Quantity)
	assert.False(t, ledger.IsLowStock(), "10450g should not trigger low stock")

	t.Log("=== Inventory Domain Story COMPLETE ===")
	t.Logf("  Rice: %dg | LowStock: %v", ledger.Quantity, ledger.IsLowStock())
}
