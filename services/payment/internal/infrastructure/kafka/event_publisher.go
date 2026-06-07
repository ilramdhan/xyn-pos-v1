package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/xyn-pos/services/payment/internal/domain/payment"
)

const (
	topicPaymentCompleted = "payment.completed"
	topicPaymentVoided    = "payment.voided"
)

// EventPublisher publishes payment domain events to Kafka.
type EventPublisher struct {
	client *kgo.Client
}

// NewEventPublisher creates an EventPublisher connected to the given brokers.
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

// PublishCompleted publishes a PaymentCompletedEvent.
func (p *EventPublisher) PublishCompleted(ctx context.Context, ev payment.PaymentCompletedEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("kafka.PublishCompleted marshal: %w", err)
	}
	rec := &kgo.Record{
		Topic: topicPaymentCompleted,
		Key:   []byte(ev.PaymentID.String()),
		Value: payload,
	}
	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		slog.WarnContext(ctx, "kafka.PublishCompleted failed", "err", err)
		return fmt.Errorf("kafka.PublishCompleted produce: %w", err)
	}
	return nil
}

// PublishVoided publishes a PaymentVoidedEvent.
func (p *EventPublisher) PublishVoided(ctx context.Context, ev payment.PaymentVoidedEvent) error {
	payload, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("kafka.PublishVoided marshal: %w", err)
	}
	rec := &kgo.Record{
		Topic: topicPaymentVoided,
		Key:   []byte(ev.PaymentID.String()),
		Value: payload,
	}
	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		slog.WarnContext(ctx, "kafka.PublishVoided failed", "err", err)
		return fmt.Errorf("kafka.PublishVoided produce: %w", err)
	}
	return nil
}

// Close shuts down the Kafka client.
func (p *EventPublisher) Close() {
	p.client.Close()
}
