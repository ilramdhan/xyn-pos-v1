package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// VoidPaymentInput is the DTO for voiding a payment.
type VoidPaymentInput struct {
	PaymentID uuid.UUID
	Reason    string
	VoidedBy  uuid.UUID
}

// VoidPaymentHandler voids a successful payment.
type VoidPaymentHandler struct {
	repo      payment.PaymentRepository
	gateway   payment.PaymentGateway
	publisher PaymentEventPublisher
}

// NewVoidPaymentHandler wires the handler.
func NewVoidPaymentHandler(
	repo payment.PaymentRepository,
	gateway payment.PaymentGateway,
	publisher PaymentEventPublisher,
) *VoidPaymentHandler {
	return &VoidPaymentHandler{repo: repo, gateway: gateway, publisher: publisher}
}

// Handle voids a payment and publishes the domain event.
func (h *VoidPaymentHandler) Handle(ctx context.Context, in VoidPaymentInput) error {
	p, err := h.repo.FindByID(ctx, in.PaymentID)
	if err != nil {
		return fmt.Errorf("VoidPayment.FindByID: %w", err)
	}

	if err := h.gateway.VoidTransaction(ctx, p.ExternalID); err != nil {
		return fmt.Errorf("VoidPayment.VoidTransaction: %w", err)
	}

	if err := p.Void(in.Reason, in.VoidedBy); err != nil {
		return fmt.Errorf("VoidPayment.Void: %w", err)
	}

	if err := h.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("VoidPayment.Update: %w", err)
	}

	for _, ev := range p.PopEvents() {
		if voided, ok := ev.(payment.PaymentVoidedEvent); ok {
			if pubErr := h.publisher.PublishVoided(ctx, voided); pubErr != nil {
				slog.WarnContext(ctx, "VoidPayment: failed to publish event", "err", pubErr)
			}
		}
	}
	return nil
}
