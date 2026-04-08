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
			HostPort:   "localhost:7233",
			Namespace:  "default",
			OTelConfig: nil,
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
			HostPort:   "localhost:7233",
			Namespace:  "default",
			OTelConfig: otelCfg,
		}

		require.NotNil(t, config.OTelConfig)
		assert.True(t, config.OTelConfig.IsTracingEnabled())
	})

	t.Run("DefaultConfigHasNilOTelConfig", func(t *testing.T) {
		config := DefaultConfig()
		assert.Nil(t, config.OTelConfig)
	})
}

func TestNewClient_NilOTelConfig(t *testing.T) {
	// When OTelConfig is nil, NewClient should not panic.
	// The client creation will fail due to invalid host, but the OTel
	// interceptor path should be skipped gracefully.
	config := &Config{
		HostPort:   "invalid-host-that-does-not-exist:7233",
		Namespace:  "default",
		OTelConfig: nil,
	}

	// This may or may not error (Temporal client.Dial is lazy), but it must not panic
	c, _ := NewClient(config)
	if c != nil {
		c.Close()
	}
}

func TestNewClient_OTelConfigTracingDisabled(t *testing.T) {
	// OTelConfig with no TracerProvider => tracing disabled => interceptor not added
	config := &Config{
		HostPort:   "invalid-host-that-does-not-exist:7233",
		Namespace:  "default",
		OTelConfig: &otel.Config{},
	}

	c, _ := NewClient(config)
	if c != nil {
		c.Close()
	}
}
