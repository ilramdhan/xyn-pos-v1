package telemetry_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/twmb/franz-go/pkg/kgo"

	"github.com/xyn-pos/shared/pkg/telemetry"
)

func TestKafkaHeaderCarrier_GetSetKeys(t *testing.T) {
	rec := &kgo.Record{}
	c := telemetry.NewKafkaHeaderCarrier(rec)

	// Set a value and get it back
	c.Set("traceparent", "00-trace-span-01")
	assert.Equal(t, "00-trace-span-01", c.Get("traceparent"))

	// Keys includes the set key
	assert.Contains(t, c.Keys(), "traceparent")

	// Get missing key returns empty string
	assert.Equal(t, "", c.Get("nonexistent"))
}

func TestKafkaHeaderCarrier_SetOverwrites(t *testing.T) {
	rec := &kgo.Record{}
	c := telemetry.NewKafkaHeaderCarrier(rec)

	c.Set("traceparent", "original")
	c.Set("traceparent", "updated")
	assert.Equal(t, "updated", c.Get("traceparent"))

	// Only one header with this key
	count := 0
	for _, k := range c.Keys() {
		if k == "traceparent" {
			count++
		}
	}
	assert.Equal(t, 1, count)
}

func TestKafkaHeaderCarrier_MultipleKeys(t *testing.T) {
	rec := &kgo.Record{}
	c := telemetry.NewKafkaHeaderCarrier(rec)

	c.Set("traceparent", "tp-value")
	c.Set("tracestate", "ts-value")
	c.Set("baggage", "bg-value")

	assert.Equal(t, "tp-value", c.Get("traceparent"))
	assert.Equal(t, "ts-value", c.Get("tracestate"))
	assert.Equal(t, "bg-value", c.Get("baggage"))
	assert.Len(t, c.Keys(), 3)
}

func TestKafkaHeaderCarrier_ExistingHeaders(t *testing.T) {
	rec := &kgo.Record{
		Headers: []kgo.RecordHeader{
			{Key: "existing", Value: []byte("value")},
		},
	}
	c := telemetry.NewKafkaHeaderCarrier(rec)
	assert.Equal(t, "value", c.Get("existing"))
	assert.Contains(t, c.Keys(), "existing")
}
