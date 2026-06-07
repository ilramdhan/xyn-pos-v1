package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// GetPaymentHandler retrieves a payment by primary key.
type GetPaymentHandler struct {
	repo payment.PaymentRepository
}

// NewGetPaymentHandler wires the handler.
func NewGetPaymentHandler(repo payment.PaymentRepository) *GetPaymentHandler {
	return &GetPaymentHandler{repo: repo}
}

// Handle returns the payment or ErrPaymentNotFound.
func (h *GetPaymentHandler) Handle(ctx context.Context, id uuid.UUID) (*payment.Payment, error) {
	p, err := h.repo.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("GetPayment.FindByID: %w", err)
	}
	return p, nil
}
