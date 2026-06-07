// Package story_test contains domain-level story tests that validate the payment
// state machine without any infrastructure dependencies.
//
// This file covers the Payment bounded context slice of the "Warung Padang Golden Path":
//   - NewPayment → StatusPending
//   - MarkSuccess → StatusSuccess + PaymentCompletedEvent
//   - Double-success rejection (idempotency)
//   - Void path (cancel after success)
//   - MarkFailed path
//
// See also:
//   - services/pos/tests/story — POS order lifecycle
//   - services/inventory/tests/story — stock deduction
//
// Run with: go test ./services/payment/tests/story/... -v
package story_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentdomain "github.com/xyn-pos/services/payment/internal/domain/payment"
)

// TestWarungPadangGoldenPath_Payment covers the Payment bounded context slice of
// the Warung Padang scenario: QRIS payment creation → webhook success → event emission.
//
// Money is in sen (1 Rupiah = 100 sen).
// Order total: 5,555,000 sen (Rp 55,550) after 10% discount + PPN 11%.
func TestWarungPadangGoldenPath_Payment(t *testing.T) {
	tenantID := uuid.New()
	orderID := uuid.New()
	orderTotal := int64(5_555_000) // matches POS story final total

	// =========================================================================
	// Step 1: Create payment in PENDING status
	// =========================================================================
	t.Log("Step 1: Create QRIS payment")

	payment, err := paymentdomain.NewPayment(
		tenantID, orderID, orderTotal,
		paymentdomain.MethodQRIS, "idem-payment-warung-001",
	)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, payment.ID)
	assert.Equal(t, paymentdomain.StatusPending, payment.Status)
	assert.Equal(t, orderTotal, payment.Amount)
	assert.Equal(t, paymentdomain.MethodQRIS, payment.Method)
	assert.Equal(t, orderID, payment.OrderID)
	assert.Equal(t, tenantID, payment.TenantID)
	t.Logf("  Payment ID: %s, Amount: %d sen (Rp %.0f)", payment.ID, payment.Amount, float64(payment.Amount)/100)

	// Guard: invalid amount rejected
	_, err = paymentdomain.NewPayment(tenantID, orderID, 0, paymentdomain.MethodQRIS, "idem-zero")
	assert.ErrorIs(t, err, paymentdomain.ErrInvalidAmount)

	_, err = paymentdomain.NewPayment(tenantID, orderID, -1, paymentdomain.MethodQRIS, "idem-neg")
	assert.ErrorIs(t, err, paymentdomain.ErrInvalidAmount)

	// =========================================================================
	// Step 2: Payment webhook fires — payment succeeds
	// =========================================================================
	t.Log("Step 2: Payment webhook — MarkSuccess")

	midtransExternalID := "midtrans-trx-warung-padang-001"
	err = payment.MarkSuccess(midtransExternalID)
	require.NoError(t, err)
	assert.Equal(t, paymentdomain.StatusSuccess, payment.Status)
	assert.Equal(t, midtransExternalID, payment.ExternalID)

	// =========================================================================
	// Step 3: Verify PaymentCompletedEvent
	// =========================================================================
	t.Log("Step 3: Verify PaymentCompletedEvent payload")

	events := payment.PopEvents()
	require.Len(t, events, 1, "exactly one PaymentCompletedEvent expected")
	completedEv, ok := events[0].(paymentdomain.PaymentCompletedEvent)
	require.True(t, ok, "event must be PaymentCompletedEvent")
	assert.Equal(t, payment.ID, completedEv.PaymentID)
	assert.Equal(t, orderID, completedEv.OrderID, "event must carry the order ID for downstream routing")
	assert.Equal(t, tenantID, completedEv.TenantID)
	assert.Equal(t, orderTotal, completedEv.Amount)
	assert.Equal(t, paymentdomain.MethodQRIS, completedEv.Method)
	assert.False(t, completedEv.OccurredAt.IsZero())

	// Events cleared after pop
	assert.Empty(t, payment.PopEvents())

	// =========================================================================
	// Step 4: Idempotency — cannot mark success twice
	// =========================================================================
	t.Log("Step 4: Idempotency — reject double-success")

	err = payment.MarkSuccess("duplicate-trx")
	assert.Error(t, err, "double-success must be rejected by ErrInvalidStatusTransition")

	// =========================================================================
	// Step 5: Void path — void a successful payment
	// =========================================================================
	t.Log("Step 5: Void payment")

	voidedBy := uuid.New()
	err = payment.Void("customer requested refund", voidedBy)
	require.NoError(t, err)
	assert.Equal(t, paymentdomain.StatusVoided, payment.Status)

	voidEvents := payment.PopEvents()
	require.Len(t, voidEvents, 1, "exactly one PaymentVoidedEvent expected")
	voidedEv, ok := voidEvents[0].(paymentdomain.PaymentVoidedEvent)
	require.True(t, ok, "event must be PaymentVoidedEvent")
	assert.Equal(t, orderID, voidedEv.OrderID)
	assert.Equal(t, voidedBy, voidedEv.VoidedBy)

	// =========================================================================
	// Step 6: Failed payment path (separate payment instance)
	// =========================================================================
	t.Log("Step 6: Failed payment path")

	failedPayment, err := paymentdomain.NewPayment(
		tenantID, orderID, orderTotal,
		paymentdomain.MethodBankTransfer, "idem-payment-warung-002",
	)
	require.NoError(t, err)
	assert.Equal(t, paymentdomain.StatusPending, failedPayment.Status)

	err = failedPayment.MarkFailed()
	require.NoError(t, err)
	assert.Equal(t, paymentdomain.StatusFailed, failedPayment.Status)

	// Cannot void a failed payment — must be SUCCESS first
	err = failedPayment.Void("attempt void on failed", uuid.New())
	assert.Error(t, err, "cannot void a failed payment")

	t.Log("=== Payment Domain Story COMPLETE ===")
	t.Logf("  Payment %s: status=%s (voided)", payment.ID, payment.Status)
}
