package temporal

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"

	"github.com/jasoet/pkg/v3/otel"
)

// NewClient creates a Temporal client. It starts from DefaultConfig and
// applies the given options in order.
func NewClient(opts ...Option) (client.Client, error) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "temporal.NewClient")

	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	logger.Debug("Creating new Temporal client",
		otel.F("hostPort", config.HostPort),
		otel.F("namespace", config.Namespace))

	// Create a zerolog logger for Temporal SDK's logger adapter
	zerologLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("service", "temporal").
		Logger()

	clientOption := client.Options{
		HostPort:  config.HostPort,
		Namespace: config.Namespace,
		Logger:    NewZerologAdapter(zerologLogger),
	}

	// Configure TLS / credentials for TLS-enabled servers and Temporal Cloud.
	if config.TLS != nil {
		clientOption.ConnectionOptions.TLS = config.TLS
	}
	if config.Credentials != nil {
		clientOption.Credentials = config.Credentials
	}

	// Add OTel tracing interceptor if configured. Tracing was explicitly opted
	// into, so a failure to construct the interceptor is fatal — returning it
	// rather than silently proceeding without tracing.
	if config.OTelConfig != nil && config.OTelConfig.IsTracingEnabled() {
		tracerOpts := temporalotel.TracerOptions{
			Tracer: config.OTelConfig.GetTracer("temporal-sdk-go"),
		}
		tracingInterceptor, err := temporalotel.NewTracingInterceptor(tracerOpts)
		if err != nil {
			logger.Error(err, "Failed to create OTel tracing interceptor")
			return nil, fmt.Errorf("create OTel tracing interceptor: %w", err)
		}
		clientOption.Interceptors = append(clientOption.Interceptors, tracingInterceptor)
		logger.Debug("OTel tracing interceptor added to Temporal client")
	}

	// Add OTel metrics handler if configured
	if config.OTelConfig != nil && config.OTelConfig.IsMetricsEnabled() {
		meter := config.OTelConfig.GetMeter("temporal-sdk-go")
		metricsHandler := temporalotel.NewMetricsHandler(temporalotel.MetricsHandlerOptions{
			Meter: meter,
			OnError: func(err error) {
				errLogger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v3/temporal", "temporal.otelMetrics.OnError")
				errLogger.Error(err, "Error in OTel metrics handler")
			},
		})
		clientOption.MetricsHandler = metricsHandler
		logger.Debug("OTel metrics handler added to Temporal client")
	}

	// Apply caller-supplied passthrough hooks last so they can override any
	// option assembled above.
	for _, hook := range config.clientOptionsHooks {
		hook(&clientOption)
	}

	logger.Debug("Connecting to Temporal server")
	c, err := client.Dial(clientOption)
	if err != nil {
		logger.Error(err, "Failed to connect to Temporal server")
		return nil, fmt.Errorf("dial temporal server %q: %w", config.HostPort, err)
	}

	logger.Debug("Successfully connected to Temporal server")
	return c, nil
}
