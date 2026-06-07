package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// GetPaymentByOrderHandler retrieves the payment linked to an order.
type GetPaymentByOrderHandler struct {
	repo payment.PaymentRepository
}

// NewGetPaymentByOrderHandler wires the handler.
func NewGetPaymentByOrderHandler(repo payment.PaymentRepository) *GetPaymentByOrderHandler {
	return &GetPaymentByOrderHandler{repo: repo}
}

// Handle returns the payment for the given order ID.
func (h *GetPaymentByOrderHandler) Handle(ctx context.Context, orderID uuid.UUID) (*payment.Payment, error) {
	p, err := h.repo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("GetPaymentByOrder.FindByOrderID: %w", err)
	}
	return p, nil
}
