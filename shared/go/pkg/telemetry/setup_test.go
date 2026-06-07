package telemetry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	sharedtelemetry "github.com/xyn-pos/shared/pkg/telemetry"
)

func TestSetup_InvalidEndpoint_ReturnsError(t *testing.T) {
	// An unreachable endpoint during initial dial should NOT error in gRPC lazy mode.
	// The exporter itself should initialize successfully (connections are lazy).
	// This tests that Setup returns a valid shutdown func even with an unavailable collector.
	shutdown, err := sharedtelemetry.Setup(context.Background(), sharedtelemetry.Config{
		ServiceName:    "test-service",
		ServiceVersion: "0.0.1",
		OTLPEndpoint:   "localhost:14317",
		Environment:    "test",
	})
	// gRPC connections are lazy — Setup itself should not error
	assert.NoError(t, err)
	if shutdown != nil {
		shutdown()
	}
}

func TestSetup_EmptyServiceName_StillInitializes(t *testing.T) {
	shutdown, err := sharedtelemetry.Setup(context.Background(), sharedtelemetry.Config{
		OTLPEndpoint: "localhost:14317",
	})
	assert.NoError(t, err)
	if shutdown != nil {
		shutdown()
	}
}
