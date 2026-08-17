package ssh

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"

	"github.com/jasoet/pkg/v3/otel"
)

// TestWithOTelConfig verifies the functional option sets the OTelConfig field.
func TestWithOTelConfig(t *testing.T) {
	t.Run("option sets OTelConfig on the tunnel config", func(t *testing.T) {
		otelCfg := otel.NewConfig("ssh-test")

		tunnel := New(Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			RemoteHost:            "remote.example.com",
			RemotePort:            3306,
			InsecureIgnoreHostKey: true,
		}, WithOTelConfig(otelCfg))

		require.NotNil(t, tunnel)
		require.NotNil(t, tunnel.config.OTelConfig)
		assert.Equal(t, otelCfg, tunnel.config.OTelConfig)
	})

	t.Run("no option leaves OTelConfig nil", func(t *testing.T) {
		tunnel := New(Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			RemoteHost:            "remote.example.com",
			RemotePort:            3306,
			InsecureIgnoreHostKey: true,
		})

		require.NotNil(t, tunnel)
		assert.Nil(t, tunnel.config.OTelConfig)
	})
}

// TestStartEmitsSpan verifies that Start emits an operations-layer span in
// error state on the failure path, using an unreachable host (192.0.2.1 is
// TEST-NET-1, guaranteed unroutable) so no live SSH server is needed.
func TestStartEmitsSpan(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() {
		assert.NoError(t, tp.Shutdown(context.Background()))
	})

	cfg := otel.NewConfig("test-service", otel.WithTracerProvider(tp))
	ctx := otel.ContextWithConfig(context.Background(), cfg)

	tunnel := New(Config{
		Host:                  "192.0.2.1",
		Port:                  22,
		User:                  "testuser",
		Password:              "testpass",
		RemoteHost:            "127.0.0.1",
		RemotePort:            80,
		LocalPort:             0,
		Timeout:               1 * time.Second,
		InsecureIgnoreHostKey: true,
	})

	err := tunnel.Start(ctx)
	require.Error(t, err)

	spans := exporter.GetSpans()
	require.Len(t, spans, 1, "expected exactly one ended span")
	assert.Equal(t, "ssh.Start", spans[0].Name)
	assert.Equal(t, "operations.ssh", spans[0].InstrumentationScope.Name)
	assert.Equal(t, codes.Error, spans[0].Status.Code)
}
