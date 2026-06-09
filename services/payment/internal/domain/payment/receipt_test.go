package payment_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

func TestGenerateReceipt_FromSuccessfulPayment(t *testing.T) {
	p := newSuccessfulPayment(t)
	r, err := payment.GenerateReceipt(p)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, r.ID)
	assert.Equal(t, p.ID, r.PaymentID)
	assert.Equal(t, p.OrderID, r.OrderID)
	assert.Equal(t, p.TenantID, r.TenantID)
	assert.Equal(t, p.Amount, r.Amount)
	assert.Equal(t, p.Method, r.Method)
	assert.NotEmpty(t, r.ReceiptNumber)
	assert.False(t, r.IssuedAt.IsZero())
}

func TestGenerateReceipt_NonSuccessPayment_ReturnsError(t *testing.T) {
	tenantID := uuid.New()
	orderID := uuid.New()

	tests := []struct {
		name  string
		setup func() *payment.Payment
	}{
		{
			name: "pending",
			setup: func() *payment.Payment {
				p, _ := payment.NewPayment(tenantID, orderID, 100000, payment.MethodQRIS, "idem-ns-1")
				return p
			},
		},
		{
			name: "failed",
			setup: func() *payment.Payment {
				p, _ := payment.NewPayment(tenantID, orderID, 100000, payment.MethodQRIS, "idem-ns-2")
				_ = p.MarkFailed()
				return p
			},
		},
		{
			name: "voided",
			setup: func() *payment.Payment {
				p, _ := payment.NewPayment(tenantID, orderID, 100000, payment.MethodQRIS, "idem-ns-3")
				_ = p.MarkSuccess("ext-void-001")
				_ = p.Void("test void", uuid.New())
				return p
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := payment.GenerateReceipt(tc.setup())
			assert.ErrorIs(t, err, payment.ErrReceiptNotAllowed)
		})
	}
}

func TestGenerateReceipt_PopEvents_ReturnsReceiptIssuedEvent(t *testing.T) {
	p := newSuccessfulPayment(t)
	r, err := payment.GenerateReceipt(p)
	require.NoError(t, err)
	evs := r.PopEvents()
	require.Len(t, evs, 1)
	ev, ok := evs[0].(payment.ReceiptIssuedEvent)
	require.True(t, ok)
	assert.Equal(t, r.ID, ev.ReceiptID)
	assert.Equal(t, r.PaymentID, ev.PaymentID)
	assert.Equal(t, r.TenantID, ev.TenantID)
	// Second pop must return empty — slice must be cleared.
	assert.Empty(t, r.PopEvents())
}

func newSuccessfulPayment(t *testing.T) *payment.Payment {
	t.Helper()
	p, err := payment.NewPayment(uuid.New(), uuid.New(), 500000, payment.MethodQRIS, "idem-success-001")
	require.NoError(t, err)
	require.NoError(t, p.MarkSuccess("mid-ext-001"))
	return p
}
