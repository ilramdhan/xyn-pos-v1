// Package story_test — Phase 4 extensions to the Warung Padang Payment story.
//
// Covers:
//   - Story A: Receipt generation from a successful payment
//   - Story B: Full-amount refund of a successful payment
//
// Run with: go test ./services/payment/tests/story/... -v
package story_test

import (
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	paymentdomain "github.com/xyn-pos/services/payment/internal/domain/payment"
)

// TestStoryA_ReceiptGeneration validates the receipt aggregate.
func TestStoryA_ReceiptGeneration(t *testing.T) {
	tenantID := uuid.New()
	orderID := uuid.New()
	amount := int64(5_555_000) // Rp 55,550

	// Build a successful payment.
	p, err := paymentdomain.NewPayment(tenantID, orderID, amount, paymentdomain.MethodQRIS, "idem-receipt-001")
	require.NoError(t, err)
	require.NoError(t, p.MarkSuccess("ext-trx-001"))

	// =========================================================================
	// Step 1: Generate receipt from a successful payment
	// =========================================================================
	t.Log("Step 1: Generate receipt from successful payment")

	r, err := paymentdomain.GenerateReceipt(p)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, r.ID)
	assert.Equal(t, p.ID, r.PaymentID)
	assert.Equal(t, orderID, r.OrderID)
	assert.Equal(t, tenantID, r.TenantID)
	assert.Equal(t, amount, r.Amount)
	assert.Equal(t, paymentdomain.MethodQRIS, r.Method)
	assert.False(t, r.IssuedAt.IsZero())
	assert.True(t, strings.HasPrefix(r.ReceiptNumber, "RCP-"), "receipt number format: %s", r.ReceiptNumber)
	t.Logf("  Receipt: %s  Amount: %d sen", r.ReceiptNumber, r.Amount)

	// =========================================================================
	// Step 2: Receipt emits ReceiptIssuedEvent
	// =========================================================================
	t.Log("Step 2: Verify ReceiptIssuedEvent payload")

	evs := r.PopEvents()
	require.Len(t, evs, 1)
	ev, ok := evs[0].(paymentdomain.ReceiptIssuedEvent)
	require.True(t, ok, "expected ReceiptIssuedEvent")
	assert.Equal(t, r.ID, ev.ReceiptID)
	assert.Equal(t, p.ID, ev.PaymentID)
	assert.Equal(t, orderID, ev.OrderID)
	assert.Equal(t, tenantID, ev.TenantID)
	assert.Equal(t, r.ReceiptNumber, ev.ReceiptNumber)
	assert.Equal(t, amount, ev.Amount)
	assert.False(t, ev.OccurredAt.IsZero())

	// PopEvents clears the slice.
	assert.Empty(t, r.PopEvents())

	// =========================================================================
	// Step 3: Cannot generate receipt from a non-success payment
	// =========================================================================
	t.Log("Step 3: Guard — receipt requires StatusSuccess")

	pending, err := paymentdomain.NewPayment(tenantID, orderID, amount, paymentdomain.MethodQRIS, "idem-receipt-002")
	require.NoError(t, err)
	_, err = paymentdomain.GenerateReceipt(pending)
	assert.ErrorIs(t, err, paymentdomain.ErrReceiptNotAllowed, "pending payment must be rejected")

	failed, err := paymentdomain.NewPayment(tenantID, orderID, amount, paymentdomain.MethodQRIS, "idem-receipt-003")
	require.NoError(t, err)
	require.NoError(t, failed.MarkFailed())
	_, err = paymentdomain.GenerateReceipt(failed)
	assert.ErrorIs(t, err, paymentdomain.ErrReceiptNotAllowed, "failed payment must be rejected")
}

// TestStoryB_RefundPayment validates the full-amount refund scenario.
func TestStoryB_RefundPayment(t *testing.T) {
	tenantID := uuid.New()
	orderID := uuid.New()
	amount := int64(5_555_000)

	// Build a successful payment; clear the PaymentCompletedEvent before testing refund.
	p, err := paymentdomain.NewPayment(tenantID, orderID, amount, paymentdomain.MethodQRIS, "idem-refund-001")
	require.NoError(t, err)
	require.NoError(t, p.MarkSuccess("ext-trx-refund-001"))
	_ = p.PopEvents() // clear PaymentCompletedEvent

	// =========================================================================
	// Step 1: Full refund succeeds
	// =========================================================================
	t.Log("Step 1: Full refund of successful payment")

	err = p.Refund(amount, "customer requested refund")
	require.NoError(t, err)
	assert.Equal(t, paymentdomain.StatusRefunded, p.Status)
	assert.Equal(t, amount, p.RefundedAmount)
	assert.Equal(t, "customer requested refund", p.RefundReason)

	// =========================================================================
	// Step 2: PaymentRefundedEvent emitted
	// =========================================================================
	t.Log("Step 2: Verify PaymentRefundedEvent payload")

	evs := p.PopEvents()
	require.Len(t, evs, 1)
	ev, ok := evs[0].(paymentdomain.PaymentRefundedEvent)
	require.True(t, ok, "expected PaymentRefundedEvent")
	assert.Equal(t, p.ID, ev.PaymentID)
	assert.Equal(t, orderID, ev.OrderID)
	assert.Equal(t, tenantID, ev.TenantID)
	assert.Equal(t, amount, ev.RefundedAmount)
	assert.False(t, ev.OccurredAt.IsZero())

	// =========================================================================
	// Step 3: Invalid refund amounts rejected (zero, negative, exceeds original)
	// =========================================================================
	t.Log("Step 3: Guard — invalid refund amounts rejected")

	p2, err := paymentdomain.NewPayment(tenantID, orderID, amount, paymentdomain.MethodQRIS, "idem-refund-002")
	require.NoError(t, err)
	require.NoError(t, p2.MarkSuccess("ext-trx-refund-002"))

	err = p2.Refund(0, "zero amount")
	assert.ErrorIs(t, err, paymentdomain.ErrPartialRefundInvalid, "zero amount must be rejected")

	err = p2.Refund(amount+1, "exceeds original")
	assert.ErrorIs(t, err, paymentdomain.ErrPartialRefundInvalid, "amount exceeding original must be rejected")

	// =========================================================================
	// Step 4: Cannot refund a non-success payment
	// =========================================================================
	t.Log("Step 4: Guard — refund requires StatusSuccess")

	pending, err := paymentdomain.NewPayment(tenantID, orderID, amount, paymentdomain.MethodQRIS, "idem-refund-003")
	require.NoError(t, err)
	err = pending.Refund(amount, "reason")
	assert.ErrorIs(t, err, paymentdomain.ErrInvalidStatusTransition)

	// =========================================================================
	// Step 5: Cannot refund an already-refunded payment
	// =========================================================================
	t.Log("Step 5: Guard — double refund rejected")

	p3, err := paymentdomain.NewPayment(tenantID, orderID, amount, paymentdomain.MethodQRIS, "idem-refund-004")
	require.NoError(t, err)
	require.NoError(t, p3.MarkSuccess("ext-trx-refund-003"))
	require.NoError(t, p3.Refund(amount, "first refund"))
	err = p3.Refund(amount, "second refund")
	assert.ErrorIs(t, err, paymentdomain.ErrInvalidStatusTransition)
}
