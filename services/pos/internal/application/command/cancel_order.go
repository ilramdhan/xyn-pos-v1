package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// CancelOrderHandler cancels an order and publishes the domain event.
type CancelOrderHandler struct {
	repo      order.OrderRepository
	publisher OrderEventPublisher
}

// NewCancelOrderHandler constructs a CancelOrderHandler.
func NewCancelOrderHandler(repo order.OrderRepository, publisher OrderEventPublisher) *CancelOrderHandler {
	return &CancelOrderHandler{repo: repo, publisher: publisher}
}

// Handle cancels the order with the given reason. Events are best-effort published.
func (h *CancelOrderHandler) Handle(ctx context.Context, orderID uuid.UUID, reason string) error {
	o, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("CancelOrder find: %w", err)
	}
	if err := o.Cancel(reason); err != nil {
		return fmt.Errorf("CancelOrder domain: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return fmt.Errorf("CancelOrder update: %w", err)
	}
	for _, ev := range o.PopEvents() {
		if cancelled, ok := ev.(order.OrderCancelledEvent); ok {
			if err := h.publisher.PublishOrderCancelled(ctx, cancelled); err != nil {
				slog.WarnContext(ctx, "failed to publish OrderCancelledEvent", "err", err)
			}
		}
	}
	return nil
}
