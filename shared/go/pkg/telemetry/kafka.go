package telemetry

import (
	"go.opentelemetry.io/otel/propagation"

	"github.com/twmb/franz-go/pkg/kgo"
)

// Compile-time assertion that KafkaHeaderCarrier satisfies propagation.TextMapCarrier.
var _ propagation.TextMapCarrier = (*KafkaHeaderCarrier)(nil)

// KafkaHeaderCarrier adapts kgo.Record headers to satisfy propagation.TextMapCarrier.
// This allows OpenTelemetry trace context to be injected into and extracted from
// Kafka message headers for distributed tracing across service boundaries.
type KafkaHeaderCarrier struct {
	record *kgo.Record
}

// NewKafkaHeaderCarrier wraps a kgo.Record for OTel context propagation.
func NewKafkaHeaderCarrier(rec *kgo.Record) *KafkaHeaderCarrier {
	return &KafkaHeaderCarrier{record: rec}
}

// Get returns the value for the given header key, or empty string if not found.
func (c *KafkaHeaderCarrier) Get(key string) string {
	for _, h := range c.record.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

// Set sets or replaces the header with the given key.
func (c *KafkaHeaderCarrier) Set(key, value string) {
	for i, h := range c.record.Headers {
		if h.Key == key {
			c.record.Headers[i].Value = []byte(value)
			return
		}
	}
	c.record.Headers = append(c.record.Headers, kgo.RecordHeader{
		Key:   key,
		Value: []byte(value),
	})
}

// Keys returns all header keys present in the record.
func (c *KafkaHeaderCarrier) Keys() []string {
	keys := make([]string, len(c.record.Headers))
	for i, h := range c.record.Headers {
		keys[i] = h.Key
	}
	return keys
}
