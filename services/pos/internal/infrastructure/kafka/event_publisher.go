package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/xyn-pos/services/pos/internal/application/command"
	"github.com/xyn-pos/services/pos/internal/domain/order"
)

// EventPublisher implements command.OrderEventPublisher using Kafka.
type EventPublisher struct {
	client *kgo.Client
}

// NewEventPublisher creates a Kafka producer connected to the given brokers.
func NewEventPublisher(brokers []string) (*EventPublisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, fmt.Errorf("kafka.NewEventPublisher: %w", err)
	}
	return &EventPublisher{client: client}, nil
}

// Close releases the Kafka client.
func (p *EventPublisher) Close() { p.client.Close() }

// PublishOrderPaid publishes to the pos.order.paid topic.
func (p *EventPublisher) PublishOrderPaid(ctx context.Context, event order.OrderPaidEvent) error {
	return p.publish(ctx, "pos.order.paid", event.TenantID, event)
}

// PublishOrderCancelled publishes to the pos.order.cancelled topic.
func (p *EventPublisher) PublishOrderCancelled(ctx context.Context, event order.OrderCancelledEvent) error {
	return p.publish(ctx, "pos.order.cancelled", event.TenantID, event)
}

func (p *EventPublisher) publish(ctx context.Context, topic string, tenantID uuid.UUID, payload any) error {
	body, err := json.Marshal(map[string]any{
		"event_id":    uuid.NewString(),
		"occurred_at": time.Now().UTC(),
		"payload":     payload,
	})
	if err != nil {
		return fmt.Errorf("kafka.publish marshal: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(tenantID.String()),
		Value: body,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("kafka.publish %s: %w", topic, err)
	}
	return nil
}

// Compile-time check: EventPublisher must satisfy command.OrderEventPublisher.
var _ command.OrderEventPublisher = (*EventPublisher)(nil)
