package query

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// GetReceiptHandler retrieves a receipt by payment ID.
type GetReceiptHandler struct {
	repo payment.ReceiptRepository
}

// NewGetReceiptHandler wires the handler.
func NewGetReceiptHandler(repo payment.ReceiptRepository) *GetReceiptHandler {
	return &GetReceiptHandler{repo: repo}
}

// Handle returns the receipt for the given payment, or ErrReceiptNotFound.
func (h *GetReceiptHandler) Handle(ctx context.Context, paymentID uuid.UUID) (*payment.Receipt, error) {
	r, err := h.repo.FindByPaymentID(ctx, paymentID)
	if err != nil {
		return nil, fmt.Errorf("GetReceipt.FindByPaymentID: %w", err)
	}
	return r, nil
}

// GetReceiptByOrderHandler retrieves a receipt by order ID.
type GetReceiptByOrderHandler struct {
	repo payment.ReceiptRepository
}

// NewGetReceiptByOrderHandler wires the handler.
func NewGetReceiptByOrderHandler(repo payment.ReceiptRepository) *GetReceiptByOrderHandler {
	return &GetReceiptByOrderHandler{repo: repo}
}

// Handle returns the receipt for the given order, or ErrReceiptNotFound.
func (h *GetReceiptByOrderHandler) Handle(ctx context.Context, orderID uuid.UUID) (*payment.Receipt, error) {
	r, err := h.repo.FindByOrderID(ctx, orderID)
	if err != nil {
		return nil, fmt.Errorf("GetReceiptByOrder.FindByOrderID: %w", err)
	}
	return r, nil
}
