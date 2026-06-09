package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// ResumeOrderHandler restores a parked order to draft status.
type ResumeOrderHandler struct {
	repo      order.OrderRepository
	publisher OrderEventPublisher
}

// NewResumeOrderHandler constructs a ResumeOrderHandler.
func NewResumeOrderHandler(repo order.OrderRepository, publisher OrderEventPublisher) *ResumeOrderHandler {
	return &ResumeOrderHandler{repo: repo, publisher: publisher}
}

// Handle resumes the parked order with the given ID.
func (h *ResumeOrderHandler) Handle(ctx context.Context, orderID uuid.UUID) error {
	o, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("ResumeOrder find: %w", err)
	}
	if err := o.Resume(); err != nil {
		return fmt.Errorf("ResumeOrder domain: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return fmt.Errorf("ResumeOrder update: %w", err)
	}
	for _, ev := range o.PopEvents() {
		if resumed, ok := ev.(order.OrderResumedEvent); ok {
			if err := h.publisher.PublishOrderResumed(ctx, resumed); err != nil {
				slog.WarnContext(ctx, "failed to publish OrderResumedEvent", "err", err)
			}
		}
	}
	return nil
}
