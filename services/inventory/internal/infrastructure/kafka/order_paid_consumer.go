package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/xyn-pos/services/inventory/internal/application/command"
)

const topicPaymentCompleted = "payment.completed"

type paymentCompletedEvent struct {
	PaymentID  uuid.UUID `json:"PaymentID"`
	OrderID    uuid.UUID `json:"OrderID"`
	TenantID   uuid.UUID `json:"TenantID"`
	Amount     int64     `json:"Amount"`
	OccurredAt time.Time `json:"OccurredAt"`
}

// OrderPaidConsumer listens for payment.completed events and deducts stock.
type OrderPaidConsumer struct {
	client  *kgo.Client
	deductH *command.DeductStockForOrderHandler
}

func NewOrderPaidConsumer(brokers []string, deductH *command.DeductStockForOrderHandler) (*OrderPaidConsumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("inventory-service"),
		kgo.ConsumeTopics(topicPaymentCompleted),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka.NewOrderPaidConsumer: %w", err)
	}
	return &OrderPaidConsumer{client: client, deductH: deductH}, nil
}

// Run processes messages until ctx is cancelled.
func (c *OrderPaidConsumer) Run(ctx context.Context) {
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return
		}
		if errs := fetches.Errors(); len(errs) > 0 {
			for _, e := range errs {
				slog.ErrorContext(ctx, "kafka fetch error", "err", e.Err)
			}
			continue
		}
		fetches.EachRecord(func(rec *kgo.Record) {
			c.processRecord(ctx, rec)
		})
	}
}

func (c *OrderPaidConsumer) processRecord(ctx context.Context, rec *kgo.Record) {
	var ev paymentCompletedEvent
	if err := json.Unmarshal(rec.Value, &ev); err != nil {
		slog.ErrorContext(ctx, "OrderPaidConsumer: unmarshal failed", "err", err)
		return
	}
	// Phase 4 MVP: log the event. Full deduction requires order items from POS service (Phase 5).
	// The DeductStockForOrderHandler is wired and ready; call it here when items are available.
	slog.InfoContext(ctx, "OrderPaidConsumer: payment.completed received",
		"order_id", ev.OrderID, "tenant_id", ev.TenantID)
}

// Close shuts down the Kafka client.
func (c *OrderPaidConsumer) Close() {
	c.client.Close()
}
