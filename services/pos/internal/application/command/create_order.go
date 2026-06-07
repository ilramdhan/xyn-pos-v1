package command

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// CreateOrderInput holds all parameters required to open a new order.
type CreateOrderInput struct {
	TenantID       uuid.UUID
	BranchID       uuid.UUID
	ShiftID        *uuid.UUID
	CashierID      uuid.UUID
	OrderType      order.OrderType
	TableNumber    string
	Notes          string
	IdempotencyKey string
	TaxCfg         order.TaxConfig
}

// OrderEventPublisher publishes domain events after successful state transitions.
type OrderEventPublisher interface {
	PublishOrderPaid(ctx context.Context, event order.OrderPaidEvent) error
	PublishOrderCancelled(ctx context.Context, event order.OrderCancelledEvent) error
}

// CreateOrderHandler creates a new order, returning an existing order on duplicate idempotency key.
type CreateOrderHandler struct {
	repo      order.OrderRepository
	publisher OrderEventPublisher
}

// NewCreateOrderHandler constructs a CreateOrderHandler.
func NewCreateOrderHandler(repo order.OrderRepository, publisher OrderEventPublisher) *CreateOrderHandler {
	return &CreateOrderHandler{repo: repo, publisher: publisher}
}

// Handle executes the create-order use case. Idempotent: returns existing order on duplicate key.
func (h *CreateOrderHandler) Handle(ctx context.Context, in CreateOrderInput) (*order.Order, error) {
	existing, err := h.repo.FindByIdempotencyKey(ctx, in.TenantID, in.IdempotencyKey)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, order.ErrOrderNotFound) {
		return nil, fmt.Errorf("CreateOrder idempotency check: %w", err)
	}

	orderNumber := fmt.Sprintf("ORD-%s", uuid.NewString()[:8])

	o, err := order.NewOrder(
		in.TenantID, in.BranchID, in.ShiftID, in.CashierID,
		orderNumber, in.OrderType, in.TableNumber, in.IdempotencyKey, in.TaxCfg,
	)
	if err != nil {
		return nil, fmt.Errorf("CreateOrder domain: %w", err)
	}
	o.Notes = in.Notes

	if err := h.repo.Save(ctx, o); err != nil {
		return nil, fmt.Errorf("CreateOrder save: %w", err)
	}
	return o, nil
}
