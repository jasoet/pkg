package temporal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"

	"github.com/jasoet/pkg/v2/otel"
)

func TestConfigWithOTelConfig(t *testing.T) {
	t.Run("NilOTelConfig", func(t *testing.T) {
		config := &Config{
			HostPort:             "localhost:7233",
			Namespace:            "default",
			MetricsListenAddress: "0.0.0.0:9090",
			OTelConfig:           nil,
		}

		assert.Nil(t, config.OTelConfig)
	})

	t.Run("WithOTelConfig", func(t *testing.T) {
		tp := sdktrace.NewTracerProvider()
		defer func() { _ = tp.Shutdown(nil) }()

		otelCfg := &otel.Config{
			TracerProvider: tp,
		}

		config := &Config{
			HostPort:             "localhost:7233",
			Namespace:            "default",
			MetricsListenAddress: "0.0.0.0:9090",
			OTelConfig:           otelCfg,
		}

		require.NotNil(t, config.OTelConfig)
		assert.True(t, config.OTelConfig.IsTracingEnabled())
	})

	t.Run("DefaultConfigHasNilOTelConfig", func(t *testing.T) {
		config := DefaultConfig()
		assert.Nil(t, config.OTelConfig)
	})
}

func TestNewClientWithMetrics_NilOTelConfig(t *testing.T) {
	// When OTelConfig is nil, NewClientWithMetrics should not panic.
	// The client creation will fail due to invalid host, but the OTel
	// interceptor path should be skipped gracefully.
	config := &Config{
		HostPort:             "invalid-host-that-does-not-exist:7233",
		Namespace:            "default",
		MetricsListenAddress: "0.0.0.0:0",
		OTelConfig:           nil,
	}

	// This may or may not error (Temporal client.Dial is lazy), but it must not panic
	c, closer, _ := NewClientWithMetrics(config, false)
	if c != nil {
		c.Close()
	}
	if closer != nil {
		closer.Close()
	}
}

func TestNewClientWithMetrics_OTelConfigTracingDisabled(t *testing.T) {
	// OTelConfig with no TracerProvider => tracing disabled => interceptor not added
	config := &Config{
		HostPort:             "invalid-host-that-does-not-exist:7233",
		Namespace:            "default",
		MetricsListenAddress: "0.0.0.0:0",
		OTelConfig:           &otel.Config{},
	}

	c, closer, _ := NewClientWithMetrics(config, false)
	if c != nil {
		c.Close()
	}
	if closer != nil {
		closer.Close()
	}
}
