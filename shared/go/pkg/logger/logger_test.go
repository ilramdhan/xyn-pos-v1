package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/xyn-pos/shared/pkg/logger"
)

func TestNew_DefaultInfo(t *testing.T) {
	l := logger.New(logger.Config{Level: "info", Format: "json"})
	assert.NotNil(t, l)
}

func TestNew_DebugLevel(t *testing.T) {
	l := logger.New(logger.Config{Level: "debug", Format: "text"})
	assert.NotNil(t, l)
}

func TestNew_InvalidLevel_DefaultsToInfo(t *testing.T) {
	l := logger.New(logger.Config{Level: "invalid", Format: "json"})
	assert.NotNil(t, l)
}
