package payment_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

func TestNewPayment_ValidInputs(t *testing.T) {
	p, err := payment.NewPayment(uuid.New(), uuid.New(), 100000, payment.MethodQRIS, "idem-001")
	require.NoError(t, err)
	assert.Equal(t, payment.StatusPending, p.Status)
	assert.Equal(t, int64(100000), p.Amount)
}

func TestNewPayment_ZeroAmount_ReturnsError(t *testing.T) {
	_, err := payment.NewPayment(uuid.New(), uuid.New(), 0, payment.MethodQRIS, "idem-001")
	assert.ErrorIs(t, err, payment.ErrInvalidAmount)
}

func TestNewPayment_NegativeAmount_ReturnsError(t *testing.T) {
	_, err := payment.NewPayment(uuid.New(), uuid.New(), -100, payment.MethodQRIS, "idem-001")
	assert.ErrorIs(t, err, payment.ErrInvalidAmount)
}

func TestPayment_MarkSuccess_FromPending(t *testing.T) {
	p := newPendingPayment(t)
	err := p.MarkSuccess("mid-trx-123")
	require.NoError(t, err)
	assert.Equal(t, payment.StatusSuccess, p.Status)
	assert.Equal(t, "mid-trx-123", p.ExternalID)
}

func TestPayment_MarkSuccess_AlreadySuccess_ReturnsError(t *testing.T) {
	p := newPendingPayment(t)
	_ = p.MarkSuccess("mid-trx-1")
	err := p.MarkSuccess("mid-trx-2")
	assert.ErrorIs(t, err, payment.ErrInvalidStatusTransition)
}

func TestPayment_Void_FromSuccess(t *testing.T) {
	p := newPendingPayment(t)
	_ = p.MarkSuccess("mid-trx-1")
	err := p.Void("wrong item", uuid.New())
	require.NoError(t, err)
	assert.Equal(t, payment.StatusVoided, p.Status)
}

func TestPayment_Void_FromPending_ReturnsError(t *testing.T) {
	p := newPendingPayment(t)
	err := p.Void("reason", uuid.New())
	assert.ErrorIs(t, err, payment.ErrInvalidStatusTransition)
}

func TestPayment_MarkFailed_FromPending(t *testing.T) {
	p := newPendingPayment(t)
	err := p.MarkFailed()
	require.NoError(t, err)
	assert.Equal(t, payment.StatusFailed, p.Status)
}

func TestPayment_PopEvents_ReturnsAndClears(t *testing.T) {
	p := newPendingPayment(t)
	_ = p.MarkSuccess("mid-trx-1")

	evs := p.PopEvents()
	assert.Len(t, evs, 1)

	evs2 := p.PopEvents()
	assert.Empty(t, evs2)
}

func TestPayment_Void_EmitsVoidedEvent(t *testing.T) {
	p := newPendingPayment(t)
	_ = p.MarkSuccess("mid-trx-1")
	_ = p.PopEvents() // clear success event
	voidedBy := uuid.New()
	_ = p.Void("wrong item", voidedBy)
	evs := p.PopEvents()
	require.Len(t, evs, 1)
	voided, ok := evs[0].(payment.PaymentVoidedEvent)
	require.True(t, ok)
	assert.Equal(t, voidedBy, voided.VoidedBy)
}

func newPendingPayment(t *testing.T) *payment.Payment {
	t.Helper()
	p, err := payment.NewPayment(uuid.New(), uuid.New(), 100_000_00, payment.MethodQRIS, "idem-001")
	require.NoError(t, err)
	return p
}
