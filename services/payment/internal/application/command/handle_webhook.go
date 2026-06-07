package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

// WebhookInput is the parsed Midtrans webhook payload.
type WebhookInput struct {
	PaymentID   uuid.UUID // our internal payment ID (order_id sent to Midtrans)
	ExternalID  string    // Midtrans transaction_id
	StatusCode  string    // "200" = success, "201" = pending, "202" = denied
	FraudStatus string    // "accept", "deny", "challenge"
}

// HandleWebhookHandler processes Midtrans payment notification webhooks.
type HandleWebhookHandler struct {
	repo      payment.PaymentRepository
	publisher PaymentEventPublisher
}

// NewHandleWebhookHandler wires the handler.
func NewHandleWebhookHandler(
	repo payment.PaymentRepository,
	publisher PaymentEventPublisher,
) *HandleWebhookHandler {
	return &HandleWebhookHandler{repo: repo, publisher: publisher}
}

// Handle processes a Midtrans webhook and transitions payment state accordingly.
func (h *HandleWebhookHandler) Handle(ctx context.Context, in WebhookInput) error {
	p, err := h.repo.FindByID(ctx, in.PaymentID)
	if err != nil {
		return fmt.Errorf("HandleWebhook.FindByID: %w", err)
	}

	switch {
	case in.StatusCode == "200" && in.FraudStatus != "deny":
		if err := p.MarkSuccess(in.ExternalID); err != nil {
			return fmt.Errorf("HandleWebhook.MarkSuccess: %w", err)
		}
	case in.StatusCode == "202" || in.FraudStatus == "deny":
		if err := p.MarkFailed(); err != nil {
			return fmt.Errorf("HandleWebhook.MarkFailed: %w", err)
		}
	default:
		// "201" pending — no state change
		return nil
	}

	if err := h.repo.Update(ctx, p); err != nil {
		return fmt.Errorf("HandleWebhook.Update: %w", err)
	}

	for _, ev := range p.PopEvents() {
		if completed, ok := ev.(payment.PaymentCompletedEvent); ok {
			if pubErr := h.publisher.PublishCompleted(ctx, completed); pubErr != nil {
				slog.WarnContext(ctx, "HandleWebhook: failed to publish event", "err", pubErr)
			}
		}
	}
	return nil
}
