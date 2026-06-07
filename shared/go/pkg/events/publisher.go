package events

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/twmb/franz-go/pkg/kgo"
)

// Envelope wraps every domain event for Kafka.
type Envelope struct {
	EventID    string          `json:"event_id"`
	EventType  string          `json:"event_type"`
	TenantID   string          `json:"tenant_id"`
	OccurredAt time.Time       `json:"occurred_at"`
	Payload    json.RawMessage `json:"payload"`
}

// Publisher defines the interface for publishing domain events.
type Publisher interface {
	Publish(ctx context.Context, topic string, tenantID uuid.UUID, eventType string, payload []byte) error
	Close()
}

// KafkaPublisher publishes events to Kafka using franz-go.
type KafkaPublisher struct {
	client *kgo.Client
}

// NewKafkaPublisher creates a new publisher connected to the given brokers.
func NewKafkaPublisher(brokers []string) (*KafkaPublisher, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordPartitioner(kgo.StickyKeyPartitioner(nil)),
	)
	if err != nil {
		return nil, fmt.Errorf("events.NewKafkaPublisher: %w", err)
	}
	return &KafkaPublisher{client: client}, nil
}

// Publish encodes a domain event into an Envelope and produces it to Kafka.
func (p *KafkaPublisher) Publish(ctx context.Context, topic string, tenantID uuid.UUID, eventType string, payload []byte) error {
	env := Envelope{
		EventID:    uuid.NewString(),
		EventType:  eventType,
		TenantID:   tenantID.String(),
		OccurredAt: time.Now().UTC(),
		Payload:    json.RawMessage(payload),
	}

	data, err := json.Marshal(env)
	if err != nil {
		return fmt.Errorf("events.Publish marshal: %w", err)
	}

	record := &kgo.Record{
		Topic: topic,
		Key:   []byte(tenantID.String()),
		Value: data,
	}

	if err := p.client.ProduceSync(ctx, record).FirstErr(); err != nil {
		return fmt.Errorf("events.Publish produce: %w", err)
	}

	return nil
}

// Close shuts down the Kafka client.
func (p *KafkaPublisher) Close() { p.client.Close() }

// NoopPublisher is a no-op publisher for testing and Phase 3 (Kafka not wired yet).
type NoopPublisher struct{}

// Publish is a no-op implementation; always returns nil.
func (n *NoopPublisher) Publish(_ context.Context, _ string, _ uuid.UUID, _ string, _ []byte) error {
	return nil
}

// Close is a no-op implementation.
func (n *NoopPublisher) Close() {}
