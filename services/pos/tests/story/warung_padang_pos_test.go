// Package story_test contains domain-level story tests that validate the POS
// order lifecycle without any infrastructure dependencies.
//
// This file covers the POS bounded context slice of the "Warung Padang Golden Path":
//   - Shift open / close
//   - Order lifecycle: DRAFT → PENDING_PAYMENT → PAID
//   - Adding items, applying discount, PPN 11% tax calculation
//   - OrderPaidEvent emission with correct payload
//
// See also:
//   - services/payment/tests/story — payment state machine
//   - services/inventory/tests/story — stock deduction
//
// Run with: go test ./services/pos/tests/story/... -v
package story_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	orderdomain "github.com/xyn-pos/services/pos/internal/domain/order"
)

// TestWarungPadangGoldenPath_POS covers the POS bounded context slice of the
// Warung Padang scenario: shift open → order create → add items → discount →
// tax → submit → mark paid → shift close.
//
// Money is in sen (1 Rupiah = 100 sen).
// Prices: Nasi Padang = Rp 15,000 = 1,500,000 sen; Rendang = Rp 25,000 = 2,500,000 sen.
func TestWarungPadangGoldenPath_POS(t *testing.T) {
	tenantID := uuid.New()
	branchID := uuid.New()
	cashierID := uuid.New()
	orderNumber := "ORD-WP-" + uuid.New().String()[:8]

	// Tax: PPN 11% (1100 basis points)
	taxCfg := orderdomain.TaxConfig{
		Type: orderdomain.TaxTypePPN,
		Rate: 1100,
	}

	// =========================================================================
	// Step 1: Open cashier shift
	// =========================================================================
	t.Log("Step 1: Open cashier shift")

	shift := orderdomain.NewShift(tenantID, branchID, cashierID, 50_000_000) // Rp 500,000 opening cash
	assert.Equal(t, orderdomain.ShiftStatusOpen, shift.Status)
	assert.NotEqual(t, uuid.Nil, shift.ID)
	assert.Nil(t, shift.ClosedAt)

	// =========================================================================
	// Step 2: Create DRAFT order
	// =========================================================================
	t.Log("Step 2: Create DRAFT order")

	shiftID := shift.ID
	order, err := orderdomain.NewOrder(
		tenantID, branchID, &shiftID, cashierID,
		orderNumber, orderdomain.OrderTypeDineIn,
		"5", "idem-order-warung-001", taxCfg,
	)
	require.NoError(t, err)
	assert.Equal(t, orderdomain.StatusDraft, order.Status)
	assert.Equal(t, orderNumber, order.OrderNumber)
	assert.Equal(t, orderdomain.OrderTypeDineIn, order.OrderType)
	assert.Empty(t, order.Items)

	// =========================================================================
	// Step 3: Add menu items
	// =========================================================================
	t.Log("Step 3: Add menu items — Nasi Padang (x2) and Rendang (x1)")

	productNasiPadang := uuid.New()
	productRendang := uuid.New()

	// Nasi Padang: Rp 15,000 → 1,500,000 sen each
	err = order.AddItem(orderdomain.OrderItem{
		ProductID:   productNasiPadang,
		ProductName: "Nasi Padang",
		Quantity:    2,
		UnitPrice:   1_500_000,
	})
	require.NoError(t, err)
	assert.Len(t, order.Items, 1)
	assert.Equal(t, int64(3_000_000), order.Items[0].Subtotal) // 2 × 1,500,000

	// Rendang: Rp 25,000 → 2,500,000 sen each
	err = order.AddItem(orderdomain.OrderItem{
		ProductID:   productRendang,
		ProductName: "Rendang",
		Quantity:    1,
		UnitPrice:   2_500_000,
	})
	require.NoError(t, err)
	assert.Len(t, order.Items, 2)

	// Subtotal = 3,000,000 + 2,500,000 = 5,500,000 sen (Rp 55,000)
	assert.Equal(t, int64(5_500_000), order.Subtotal)
	t.Logf("  Subtotal: %d sen (Rp %.0f)", order.Subtotal, float64(order.Subtotal)/100)

	// =========================================================================
	// Step 4: Apply 10% discount
	// =========================================================================
	t.Log("Step 4: Apply 10% discount (1000 basis points)")

	err = order.ApplyDiscount(orderdomain.Discount{
		Type:   orderdomain.DiscountTypePercent,
		Amount: 1000, // 1000 bp = 10%
	})
	require.NoError(t, err)

	// Discount = 5,500,000 × 10% = 550,000 sen
	assert.Equal(t, int64(550_000), order.DiscountAmount)
	t.Logf("  Discount: %d sen (Rp %.0f)", order.DiscountAmount, float64(order.DiscountAmount)/100)

	// =========================================================================
	// Step 5: Verify tax and total calculation
	// =========================================================================
	t.Log("Step 5: Verify PPN 11% tax and grand total")

	// recalculate() logic in order.go:
	//   taxAmount = subtotal × 1100 / 10000 = 5,500,000 × 0.11 = 605,000
	//   total (pre-discount) = subtotal + taxAmount = 6,105,000
	//   discountAmount = subtotal × 1000 / 10000 = 550,000
	//   Order.Total = (subtotal + tax) - discount = 6,105,000 - 550,000 = 5,555,000
	expectedTax := int64(5_500_000) * 1100 / 10000 // 605,000
	expectedTotal := (int64(5_500_000) + expectedTax) - int64(550_000)

	assert.Equal(t, expectedTax, order.TaxAmount)
	assert.Equal(t, expectedTotal, order.Total) // 5,555,000
	t.Logf("  Tax: %d sen, Grand total: %d sen (Rp %.0f)",
		order.TaxAmount, order.Total, float64(order.Total)/100)

	// =========================================================================
	// Step 6: Submit order → PENDING_PAYMENT
	// =========================================================================
	t.Log("Step 6: Submit order")

	err = order.Submit()
	require.NoError(t, err)
	assert.Equal(t, orderdomain.StatusPendingPayment, order.Status)

	// Cannot add items once submitted — enforces aggregate invariant
	err = order.AddItem(orderdomain.OrderItem{ProductName: "Es Teh", Quantity: 1, UnitPrice: 300_000})
	assert.ErrorIs(t, err, orderdomain.ErrOrderNotDraft)

	// =========================================================================
	// Step 7: Mark order PAID
	// =========================================================================
	// In production: PaymentCompletedEvent flows via Kafka → pos PaymentConsumer.
	// Here we call MarkPaid() directly to test the aggregate behavior in isolation.
	t.Log("Step 7: Mark order PAID (simulating payment webhook downstream)")

	err = order.MarkPaid()
	require.NoError(t, err)
	assert.Equal(t, orderdomain.StatusPaid, order.Status)

	// Verify OrderPaidEvent is emitted with correct items for inventory deduction
	orderEvents := order.PopEvents()
	require.Len(t, orderEvents, 1, "expected exactly one OrderPaidEvent")
	paidEv, ok := orderEvents[0].(orderdomain.OrderPaidEvent)
	require.True(t, ok, "event must be OrderPaidEvent")
	assert.Equal(t, order.ID, paidEv.OrderID)
	assert.Equal(t, tenantID, paidEv.TenantID)
	assert.Len(t, paidEv.Items, 2, "paid event must carry both items for inventory deduction")

	// Event carries correct product IDs for downstream inventory deduction
	foundNasiPadang := false
	for _, item := range paidEv.Items {
		if item.ProductID == productNasiPadang && item.Quantity == 2 {
			foundNasiPadang = true
		}
	}
	assert.True(t, foundNasiPadang, "OrderPaidEvent must include Nasi Padang x2")

	// Cannot mark paid again — idempotency guard
	err = order.MarkPaid()
	assert.Error(t, err, "double-paid must be rejected")

	// =========================================================================
	// Step 8: Close cashier shift
	// =========================================================================
	t.Log("Step 8: Close cashier shift")

	err = shift.Close(52_000_000) // Rp 520,000 closing cash
	require.NoError(t, err)
	assert.Equal(t, orderdomain.ShiftStatusClosed, shift.Status)
	require.NotNil(t, shift.ClosedAt)
	assert.WithinDuration(t, time.Now().UTC(), *shift.ClosedAt, 2*time.Second)

	// Cannot close again — idempotency guard
	err = shift.Close(0)
	assert.ErrorIs(t, err, orderdomain.ErrShiftAlreadyClosed)

	// =========================================================================
	// Summary
	// =========================================================================
	t.Log("=== POS Domain Golden Path COMPLETE ===")
	t.Logf("  Order %s: status=%s total=%d sen (Rp %.0f)",
		order.OrderNumber, order.Status, order.Total, float64(order.Total)/100)
	t.Logf("  Shift: status=%s", shift.Status)
}
