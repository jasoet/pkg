package otel_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jasoet/pkg/v3/otel"
)

func TestNewConfig_WithOptions(t *testing.T) {
	cfg := otel.NewConfig("svc",
		otel.WithServiceVersion("1.2.3"),
		otel.WithoutTracing(),
		otel.WithoutMetrics(),
	)
	assert.Equal(t, "svc", cfg.ServiceName)
	assert.Equal(t, "1.2.3", cfg.ServiceVersion)
	assert.False(t, cfg.IsTracingEnabled())
	assert.False(t, cfg.IsMetricsEnabled())
	assert.True(t, cfg.IsLoggingEnabled())
}

func TestNewConfig_WithoutLogging(t *testing.T) {
	cfg := otel.NewConfig("svc", otel.WithoutLogging())
	assert.False(t, cfg.IsLoggingEnabled())
	assert.NotNil(t, cfg.GetLogger("scope")) // no-op, never nil
}
