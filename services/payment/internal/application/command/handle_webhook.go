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
	repo        payment.PaymentRepository
	publisher   PaymentEventPublisher
	receiptRepo payment.ReceiptRepository
	receiptPub  ReceiptEventPublisher
}

// NewHandleWebhookHandler wires the handler.
func NewHandleWebhookHandler(
	repo payment.PaymentRepository,
	publisher PaymentEventPublisher,
	receiptRepo payment.ReceiptRepository,
	receiptPub ReceiptEventPublisher,
) *HandleWebhookHandler {
	return &HandleWebhookHandler{
		repo:        repo,
		publisher:   publisher,
		receiptRepo: receiptRepo,
		receiptPub:  receiptPub,
	}
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
		switch e := ev.(type) {
		case payment.PaymentCompletedEvent:
			if pubErr := h.publisher.PublishCompleted(ctx, e); pubErr != nil {
				slog.WarnContext(ctx, "HandleWebhook: failed to publish completed event", "err", pubErr)
			}
		case payment.PaymentVoidedEvent:
			if pubErr := h.publisher.PublishVoided(ctx, e); pubErr != nil {
				slog.WarnContext(ctx, "HandleWebhook: failed to publish voided event", "err", pubErr)
			}
		}
	}

	if p.Status == payment.StatusSuccess {
		receipt, err := payment.GenerateReceipt(p)
		if err != nil {
			slog.WarnContext(ctx, "HandleWebhook: failed to generate receipt", "err", err)
		} else {
			if saveErr := h.receiptRepo.Save(ctx, receipt); saveErr != nil {
				slog.WarnContext(ctx, "HandleWebhook: failed to save receipt", "err", saveErr)
			} else {
				for _, ev := range receipt.PopEvents() {
					if issued, ok := ev.(payment.ReceiptIssuedEvent); ok {
						if pubErr := h.receiptPub.PublishReceiptIssued(ctx, issued); pubErr != nil {
							slog.WarnContext(ctx, "HandleWebhook: failed to publish receipt event", "err", pubErr)
						}
					}
				}
			}
		}
	}
	return nil
}
