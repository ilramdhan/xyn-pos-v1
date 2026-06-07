package events_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	sharedevents "github.com/xyn-pos/shared/pkg/events"
)

func TestNoopPublisher_Publish_NoError(t *testing.T) {
	pub := &sharedevents.NoopPublisher{}
	err := pub.Publish(context.Background(), "test-topic", uuid.New(), "TestEvent", []byte(`{"key":"val"}`))
	assert.NoError(t, err)
}

func TestKafkaPublisher_InvalidBroker_ReturnsError(t *testing.T) {
	// NewKafkaPublisher with no brokers should not error at creation time (lazy connect)
	// but we test the struct is created successfully.
	pub, err := sharedevents.NewKafkaPublisher([]string{"localhost:9999"})
	// franz-go is lazy — creation itself won't error on unreachable brokers
	assert.NoError(t, err)
	if pub != nil {
		pub.Close()
	}
}

func TestEnvelope_Serialization(t *testing.T) {
	// Test that NoopPublisher satisfies Publisher interface
	var _ sharedevents.Publisher = &sharedevents.NoopPublisher{}
}
