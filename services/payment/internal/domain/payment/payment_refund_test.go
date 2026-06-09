package payment_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

func TestRefund_FromSuccessPayment(t *testing.T) {
	p := newSuccessPayment(t)

	err := p.Refund(p.Amount, "customer request")
	require.NoError(t, err)
	assert.Equal(t, payment.StatusRefunded, p.Status)
	assert.Equal(t, p.Amount, p.RefundedAmount)
	assert.Equal(t, "customer request", p.RefundReason)
}

func TestRefund_PartialRefund(t *testing.T) {
	p := newSuccessPayment(t)

	err := p.Refund(5000, "partial return")
	require.NoError(t, err)
	assert.Equal(t, payment.StatusRefunded, p.Status)
	assert.Equal(t, int64(5000), p.RefundedAmount)
}

func TestRefund_EmitsPaymentRefundedEvent(t *testing.T) {
	p := newSuccessPayment(t)

	err := p.Refund(p.Amount, "reason")
	require.NoError(t, err)

	evs := p.PopEvents()
	require.Len(t, evs, 1)
	ev, ok := evs[0].(payment.PaymentRefundedEvent)
	require.True(t, ok)
	assert.Equal(t, p.ID, ev.PaymentID)
	assert.Equal(t, p.OrderID, ev.OrderID)
	assert.Equal(t, p.TenantID, ev.TenantID)
	assert.Equal(t, p.Amount, ev.RefundedAmount)
	assert.False(t, ev.OccurredAt.IsZero())

	assert.Empty(t, p.PopEvents()) // idempotent pop
}

func TestRefund_NonSuccessStatus_ReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		status payment.PaymentStatus
	}{
		{"pending", payment.StatusPending},
		{"failed", payment.StatusFailed},
		{"voided", payment.StatusVoided},
		{"refunded", payment.StatusRefunded},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := &payment.Payment{
				ID:     uuid.New(),
				Status: tc.status,
				Amount: 10000,
			}
			err := p.Refund(5000, "reason")
			assert.ErrorIs(t, err, payment.ErrInvalidStatusTransition)
		})
	}
}

func TestRefund_InvalidAmount_ReturnsError(t *testing.T) {
	tests := []struct {
		name   string
		amount int64
	}{
		{"zero", 0},
		{"negative", -100},
		{"exceeds payment", 10001}, // newSuccessPayment has Amount=10000
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			p := newSuccessPayment(t)
			err := p.Refund(tc.amount, "reason")
			assert.ErrorIs(t, err, payment.ErrPartialRefundInvalid)
		})
	}
}

// newSuccessPayment creates a Payment in StatusSuccess for refund tests.
func newSuccessPayment(t *testing.T) *payment.Payment {
	t.Helper()
	p := &payment.Payment{
		ID:       uuid.New(),
		TenantID: uuid.New(),
		OrderID:  uuid.New(),
		Amount:   10000,
		Status:   payment.StatusSuccess,
		Method:   payment.MethodQRIS,
	}
	return p
}
