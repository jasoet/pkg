package otel

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// LoggerProviderOption configures LoggerProvider behavior
type LoggerProviderOption func(*loggerProviderConfig)

// loggerProviderConfig holds configuration for logger provider
type loggerProviderConfig struct {
	serviceName   string
	debug         bool
	consoleOutput bool
	otlpEndpoint  string
	otlpInsecure  bool
}

// WithConsoleOutput enables console logging alongside OTLP
func WithConsoleOutput(enabled bool) LoggerProviderOption {
	return func(cfg *loggerProviderConfig) {
		cfg.consoleOutput = enabled
	}
}

// WithOTLPEndpoint enables OTLP log export
func WithOTLPEndpoint(endpoint string, insecure bool) LoggerProviderOption {
	return func(cfg *loggerProviderConfig) {
		cfg.otlpEndpoint = endpoint
		cfg.otlpInsecure = insecure
	}
}

// NewLoggerProviderWithOptions creates a LoggerProvider with flexible options.
// It supports both console output (zerolog) and OTLP export, or both simultaneously.
//
// Parameters:
//   - serviceName: Name of the service
//   - debug: If true, sets log level to Debug, otherwise Info
//   - opts: Optional configuration options
//
// Returns:
//   - A log.LoggerProvider configured according to the options
//   - An error if OTLP exporter creation fails
//
// Example:
//
//	provider, err := otel.NewLoggerProviderWithOptions("my-service", false,
//	    otel.WithOTLPEndpoint("localhost:4318", true),
//	    otel.WithConsoleOutput(true))
func NewLoggerProviderWithOptions(serviceName string, debug bool, opts ...LoggerProviderOption) (log.LoggerProvider, error) {
	cfg := &loggerProviderConfig{
		serviceName:   serviceName,
		debug:         debug,
		consoleOutput: true, // Default: keep console output
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// If no OTLP endpoint, fall back to console-only (existing behavior)
	if cfg.otlpEndpoint == "" {
		return logging.NewLoggerProvider(serviceName, debug), nil
	}

	// Setup console logging if enabled (for local development)
	if cfg.consoleOutput {
		setupZerologConsole(serviceName, debug)
	}

	// Create OTLP log exporter
	ctx := context.Background()
	exporterOpts := []otlploghttp.Option{
		otlploghttp.WithEndpoint(cfg.otlpEndpoint),
	}
	if cfg.otlpInsecure {
		exporterOpts = append(exporterOpts, otlploghttp.WithInsecure())
	}

	exporter, err := otlploghttp.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
	}

	// Create resource with service name
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTel LoggerProvider with batch processor
	provider := sdklog.NewLoggerProvider(
		sdklog.WithProcessor(sdklog.NewBatchProcessor(exporter)),
		sdklog.WithResource(res),
	)

	return provider, nil
}

// setupZerologConsole configures zerolog for console output
func setupZerologConsole(serviceName string, debug bool) {
	lvl := zerolog.InfoLevel
	if debug {
		lvl = zerolog.DebugLevel
	}

	zlog.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("service", serviceName).
		Int("pid", os.Getpid()).
		Logger().Level(lvl)
}
