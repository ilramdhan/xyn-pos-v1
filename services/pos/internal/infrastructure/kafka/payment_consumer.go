package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/xyn-pos/services/pos/internal/application/command"
)

// PaymentCompletedMessage is the event payload from the payment service.
type PaymentCompletedMessage struct {
	PaymentID  string `json:"payment_id"`
	OrderID    string `json:"order_id"`
	TenantID   string `json:"tenant_id"`
	Amount     int64  `json:"amount"`
	Method     string `json:"method"`
	OccurredAt string `json:"occurred_at"`
}

// PaymentConsumer consumes payment.completed events and marks orders as paid.
type PaymentConsumer struct {
	client         *kgo.Client
	markOrderPaidH *command.MarkOrderPaidHandler
}

// NewPaymentConsumer creates a Kafka consumer group member for payment topics.
func NewPaymentConsumer(brokers []string, markPaidH *command.MarkOrderPaidHandler) (*PaymentConsumer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ConsumerGroup("pos-service"),
		kgo.ConsumeTopics("payment.completed", "payment.voided"),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka.NewPaymentConsumer: %w", err)
	}
	return &PaymentConsumer{client: client, markOrderPaidH: markPaidH}, nil
}

// Run starts the consumer loop. Blocks until ctx is cancelled.
// Must be called in a dedicated goroutine — exits when ctx.Done() closes.
func (c *PaymentConsumer) Run(ctx context.Context) {
	for {
		fetches := c.client.PollFetches(ctx)
		if fetches.IsClientClosed() || ctx.Err() != nil {
			return
		}

		fetches.EachPartition(func(p kgo.FetchTopicPartition) {
			p.EachRecord(func(record *kgo.Record) {
				if err := c.handle(ctx, record); err != nil {
					slog.ErrorContext(ctx, "PaymentConsumer: failed to handle record",
						"topic", record.Topic,
						"offset", record.Offset,
						"err", err)
				}
			})
		})
	}
}

// Close releases the Kafka client.
func (c *PaymentConsumer) Close() { c.client.Close() }

func (c *PaymentConsumer) handle(ctx context.Context, record *kgo.Record) error {
	switch record.Topic {
	case "payment.completed":
		return c.handleCompleted(ctx, record.Value)
	case "payment.voided":
		slog.InfoContext(ctx, "PaymentConsumer: payment.voided received — revert order to pending_payment (Phase 5)")
		return nil
	}
	return nil
}

func (c *PaymentConsumer) handleCompleted(ctx context.Context, data []byte) error {
	var envelope struct {
		Payload PaymentCompletedMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return fmt.Errorf("PaymentConsumer.handleCompleted unmarshal: %w", err)
	}

	orderID, err := uuid.Parse(envelope.Payload.OrderID)
	if err != nil {
		return fmt.Errorf("PaymentConsumer.handleCompleted invalid order_id %q: %w", envelope.Payload.OrderID, err)
	}

	return c.markOrderPaidH.Handle(ctx, orderID)
}
