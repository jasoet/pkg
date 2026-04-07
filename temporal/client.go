package temporal

import (
	"context"
	"os"
	"time"

	"github.com/rs/zerolog"
	"go.temporal.io/sdk/client"
	temporalotel "go.temporal.io/sdk/contrib/opentelemetry"

	"github.com/jasoet/pkg/v2/otel"
)

func NewClient(config *Config) (client.Client, error) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.NewClient")

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

	// Add OTel tracing interceptor if configured
	if config.OTelConfig != nil && config.OTelConfig.IsTracingEnabled() {
		tracerOpts := temporalotel.TracerOptions{
			Tracer: config.OTelConfig.GetTracer("temporal-sdk-go"),
		}
		tracingInterceptor, err := temporalotel.NewTracingInterceptor(tracerOpts)
		if err != nil {
			logger.Error(err, "Failed to create OTel tracing interceptor, continuing without tracing")
		} else {
			clientOption.Interceptors = append(clientOption.Interceptors, tracingInterceptor)
			logger.Debug("OTel tracing interceptor added to Temporal client")
		}
	}

	// Add OTel metrics handler if configured
	if config.OTelConfig != nil && config.OTelConfig.IsMetricsEnabled() {
		meter := config.OTelConfig.GetMeter("temporal-sdk-go")
		metricsHandler := temporalotel.NewMetricsHandler(temporalotel.MetricsHandlerOptions{
			Meter: meter,
			OnError: func(err error) {
				errLogger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.otelMetrics.OnError")
				errLogger.Error(err, "Error in OTel metrics handler")
			},
		})
		clientOption.MetricsHandler = metricsHandler
		logger.Debug("OTel metrics handler added to Temporal client")
	}

	logger.Debug("Connecting to Temporal server")
	c, err := client.Dial(clientOption)
	if err != nil {
		logger.Error(err, "Failed to connect to Temporal server")
		return nil, err
	}

	logger.Debug("Successfully connected to Temporal server")
	return c, nil
}
