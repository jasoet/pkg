package temporal

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.temporal.io/sdk/mocks"

	"github.com/jasoet/pkg/v3/otel"
)

func TestNewClientOptions(t *testing.T) {
	t.Run("OptionsAssembleConfig", func(t *testing.T) {
		// NewClient starts from DefaultConfig and applies each Option in
		// order; this asserts the assembled Config carries every option.
		otelCfg := &otel.Config{}

		cfg := DefaultConfig()
		for _, opt := range []Option{
			WithHostPort("x:1"),
			WithNamespace("ns"),
			WithOTelConfig(otelCfg),
		} {
			opt(cfg)
		}

		assert.Equal(t, "x:1", cfg.HostPort)
		assert.Equal(t, "ns", cfg.Namespace)
		assert.Same(t, otelCfg, cfg.OTelConfig)
	})

	t.Run("WithConfigReplacesDefaults", func(t *testing.T) {
		custom := Config{HostPort: "custom-host:9999", Namespace: "custom-ns"}

		cfg := DefaultConfig()
		WithConfig(custom)(cfg)

		assert.Equal(t, custom, *cfg)
	})

	t.Run("NewClientAppliesHostPortOption", func(t *testing.T) {
		// client.Dial performs a health check by default, so dialing an
		// unreachable address fails. If WithHostPort were dropped the dial
		// would target the default "localhost:7233" instead — which could
		// succeed on machines running a local server and hide the bug.
		c, err := NewClient(WithHostPort("127.0.0.1:1"), WithNamespace("ns"))
		if c != nil {
			c.Close()
		}
		require.Error(t, err)
	})
}

func TestNewScheduleManagerTyped(t *testing.T) {
	mockClient := mocks.NewClient(t)

	sm, err := NewScheduleManager(mockClient)
	require.NoError(t, err)
	require.NotNil(t, sm)
	assert.Same(t, mockClient, sm.GetClient())

	// Close must not close the caller-owned client.
	sm.Close(context.Background())
}
