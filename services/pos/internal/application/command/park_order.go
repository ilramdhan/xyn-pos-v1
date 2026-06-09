package command

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// ParkOrderHandler parks a draft order so it can be resumed later.
type ParkOrderHandler struct {
	repo      order.OrderRepository
	publisher OrderEventPublisher
}

// NewParkOrderHandler constructs a ParkOrderHandler.
func NewParkOrderHandler(repo order.OrderRepository, publisher OrderEventPublisher) *ParkOrderHandler {
	return &ParkOrderHandler{repo: repo, publisher: publisher}
}

// Handle parks the order with the given ID.
func (h *ParkOrderHandler) Handle(ctx context.Context, orderID uuid.UUID) error {
	o, err := h.repo.FindByID(ctx, orderID)
	if err != nil {
		return fmt.Errorf("ParkOrder find: %w", err)
	}
	if err := o.Park(); err != nil {
		return fmt.Errorf("ParkOrder domain: %w", err)
	}
	if err := h.repo.Update(ctx, o); err != nil {
		return fmt.Errorf("ParkOrder update: %w", err)
	}
	for _, ev := range o.PopEvents() {
		if parked, ok := ev.(order.OrderParkedEvent); ok {
			if err := h.publisher.PublishOrderParked(ctx, parked); err != nil {
				slog.WarnContext(ctx, "failed to publish OrderParkedEvent", "err", err)
			}
		}
	}
	return nil
}
