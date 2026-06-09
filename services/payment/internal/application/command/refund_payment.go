package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// RefundPaymentInput is the DTO for refunding a payment.
type RefundPaymentInput struct {
	PaymentID    uuid.UUID
	TenantID     uuid.UUID
	RefundAmount int64
	Reason       string
}

// RefundPaymentHandler refunds a successful payment.
type RefundPaymentHandler struct {
	repo      payment.PaymentRepository
	gateway   payment.PaymentGateway
	publisher PaymentEventPublisher
}

// NewRefundPaymentHandler wires the handler.
func NewRefundPaymentHandler(
	repo payment.PaymentRepository,
	gateway payment.PaymentGateway,
	publisher PaymentEventPublisher,
) *RefundPaymentHandler {
	return &RefundPaymentHandler{repo: repo, gateway: gateway, publisher: publisher}
}

// Handle refunds a payment via the gateway and publishes the domain event.
func (h *RefundPaymentHandler) Handle(ctx context.Context, in RefundPaymentInput) error {
	p, err := h.repo.FindByID(ctx, in.PaymentID)
	if err != nil {
		return fmt.Errorf("RefundPayment.FindByID: %w", err)
	}

	if p.TenantID != in.TenantID {
		return fmt.Errorf("RefundPayment.TenantCheck: %w", payment.ErrPaymentNotFound)
	}

	if err := p.Refund(in.RefundAmount, in.Reason); err != nil {
		return fmt.Errorf("RefundPayment.Refund: %w", err)
	}

	if err := h.gateway.RefundTransaction(ctx, p.ExternalID, in.RefundAmount); err != nil {
		return fmt.Errorf("RefundPayment.RefundTransaction: %w", err)
	}

	if err := h.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("RefundPayment.Update: %w", err)
	}

	for _, ev := range p.PopEvents() {
		if refunded, ok := ev.(payment.PaymentRefundedEvent); ok {
			if pubErr := h.publisher.PublishRefunded(ctx, refunded); pubErr != nil {
				slog.WarnContext(ctx, "RefundPayment: failed to publish event", "err", pubErr)
			}
		}
	}
	return nil
}
