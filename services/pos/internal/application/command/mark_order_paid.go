package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// MarkOrderPaidHandler is called by the Kafka PaymentConsumer when payment.completed arrives.
type MarkOrderPaidHandler struct {
	repo      order.OrderRepository
	publisher OrderEventPublisher
}

// NewMarkOrderPaidHandler constructs a MarkOrderPaidHandler.
func NewMarkOrderPaidHandler(repo order.OrderRepository, publisher OrderEventPublisher) *MarkOrderPaidHandler {
	return &MarkOrderPaidHandler{repo: repo, publisher: publisher}
}

// Handle marks the order as paid and publishes the domain event.
func (h *MarkOrderPaidHandler) Handle(ctx context.Context, orderID uuid.UUID) error {
	o, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("MarkOrderPaid find: %w", err)
	}
	if err := o.MarkPaid(); err != nil {
		return fmt.Errorf("MarkOrderPaid domain: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return fmt.Errorf("MarkOrderPaid update: %w", err)
	}
	for _, ev := range o.PopEvents() {
		if paid, ok := ev.(order.OrderPaidEvent); ok {
			if err := h.publisher.PublishOrderPaid(ctx, paid); err != nil {
				slog.WarnContext(ctx, "failed to publish OrderPaidEvent", "err", err)
			}
		}
	}
	return nil
}
