package otel

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/jasoet/pkg/v2/logging"
	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// LogLevel is an alias for logging.LogLevel for convenience.
// Use logging.LogLevel constants directly (logging.LogLevelDebug, etc.)
type LogLevel = logging.LogLevel

// LoggerProviderOption configures LoggerProvider behavior
type LoggerProviderOption func(*loggerProviderConfig)

// loggerProviderConfig holds configuration for logger provider
type loggerProviderConfig struct {
	serviceName   string
	consoleOutput bool
	otlpEndpoint  string
	otlpInsecure  bool
	logLevel      LogLevel
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

// WithLogLevel sets the log level for console output
// Valid levels: "debug", "info", "warn", "error", "none"
// If not specified, defaults to "info"
func WithLogLevel(level LogLevel) LoggerProviderOption {
	return func(cfg *loggerProviderConfig) {
		cfg.logLevel = level
	}
}

// NewLoggerProviderWithOptions creates a LoggerProvider with flexible options.
// It supports both console output (zerolog) and OTLP export, or both simultaneously.
//
// Parameters:
//   - serviceName: Name of the service
//   - opts: Optional configuration options
//
// Returns:
//   - A log.LoggerProvider configured according to the options
//   - An error if OTLP exporter creation fails
//
// Example:
//
//	provider, err := otel.NewLoggerProviderWithOptions("my-service",
//	    otel.WithLogLevel(logging.LogLevelDebug),
//	    otel.WithOTLPEndpoint("localhost:4318", true),
//	    otel.WithConsoleOutput(true))
func NewLoggerProviderWithOptions(serviceName string, opts ...LoggerProviderOption) (log.LoggerProvider, error) {
	cfg := &loggerProviderConfig{
		serviceName:   serviceName,
		consoleOutput: true, // Default: keep console output
	}

	for _, opt := range opts {
		opt(cfg)
	}

	effectiveLevel := cfg.logLevel
	if effectiveLevel == "" {
		effectiveLevel = logging.LogLevelInfo
	}

	ctx := context.Background()

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	var processors []sdklog.Processor

	if cfg.consoleOutput {
		consoleExporter := newConsoleExporter(serviceName, effectiveLevel)
		processors = append(processors, sdklog.NewSimpleProcessor(consoleExporter))
	}

	if cfg.otlpEndpoint != "" {
		exporterOpts := []otlploghttp.Option{
			otlploghttp.WithEndpoint(cfg.otlpEndpoint),
		}
		if cfg.otlpInsecure {
			exporterOpts = append(exporterOpts, otlploghttp.WithInsecure())
		}

		otlpExporter, err := otlploghttp.New(ctx, exporterOpts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP log exporter: %w", err)
		}

		processors = append(processors, sdklog.NewBatchProcessor(otlpExporter))
	}

	if len(processors) == 0 {
		consoleExporter := newConsoleExporter(serviceName, effectiveLevel)
		processors = append(processors, sdklog.NewSimpleProcessor(consoleExporter))
	}

	providerOpts := []sdklog.LoggerProviderOption{
		sdklog.WithResource(res),
	}
	for _, processor := range processors {
		providerOpts = append(providerOpts, sdklog.WithProcessor(processor))
	}

	provider := sdklog.NewLoggerProvider(providerOpts...)

	return provider, nil
}

// consoleExporter implements sdklog.Exporter for console output via zerolog
type consoleExporter struct {
	logger zerolog.Logger
}

// newConsoleExporter creates a console exporter with zerolog (OTel-aware version)
func newConsoleExporter(serviceName string, logLevel LogLevel) *consoleExporter {
	lvl := logLevelToZerolog(logLevel)

	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("service", serviceName).
		Int("pid", os.Getpid()).
		Logger().
		Level(lvl)

	return &consoleExporter{logger: logger}
}

// Export implements sdklog.Exporter interface
func (e *consoleExporter) Export(ctx context.Context, records []sdklog.Record) error {
	for _, record := range records {
		event := severityToZerologEvent(e.logger, record.Severity())

		if !record.Timestamp().IsZero() {
			event = event.Time("timestamp", record.Timestamp())
		}

		if severityText := record.SeverityText(); severityText != "" {
			event = event.Str("severity", severityText)
		}

		traceID := record.TraceID()
		spanID := record.SpanID()
		if traceID.IsValid() {
			event = event.
				Str("trace_id", traceID.String()).
				Str("span_id", spanID.String())

			if record.TraceFlags().IsSampled() {
				event = event.Str("trace_flags", "01")
			} else {
				event = event.Str("trace_flags", "00")
			}
		}

		record.WalkAttributes(func(kv log.KeyValue) bool {
			event = addAttributeToEvent(event, kv)
			return true
		})

		message := record.Body().AsString()
		if message == "" {
			message = "log entry"
		}
		event.Msg(message)
	}

	return nil
}

// Shutdown implements sdklog.Exporter interface
func (e *consoleExporter) Shutdown(ctx context.Context) error {
	return nil
}

// ForceFlush implements sdklog.Exporter interface
func (e *consoleExporter) ForceFlush(ctx context.Context) error {
	return nil
}

// logLevelToZerolog converts LogLevel to zerolog.Level
func logLevelToZerolog(level LogLevel) zerolog.Level {
	switch level {
	case logging.LogLevelDebug:
		return zerolog.DebugLevel
	case logging.LogLevelInfo:
		return zerolog.InfoLevel
	case logging.LogLevelWarn:
		return zerolog.WarnLevel
	case logging.LogLevelError:
		return zerolog.ErrorLevel
	case logging.LogLevelNone:
		return zerolog.Disabled
	default:
		return zerolog.InfoLevel
	}
}

// severityToZerologEvent maps OTel severity to zerolog event
func severityToZerologEvent(logger zerolog.Logger, severity log.Severity) *zerolog.Event {
	switch {
	case severity >= log.SeverityFatal:
		return logger.WithLevel(zerolog.FatalLevel)
	case severity >= log.SeverityError:
		return logger.Error()
	case severity >= log.SeverityWarn:
		return logger.Warn()
	case severity >= log.SeverityInfo:
		return logger.Info()
	case severity >= log.SeverityDebug:
		return logger.Debug()
	default:
		return logger.Trace()
	}
}

// addAttributeToEvent adds a log attribute to zerolog event
func addAttributeToEvent(event *zerolog.Event, kv log.KeyValue) *zerolog.Event {
	key := kv.Key
	value := kv.Value

	switch value.Kind() {
	case log.KindBool:
		return event.Bool(key, value.AsBool())
	case log.KindInt64:
		return event.Int64(key, value.AsInt64())
	case log.KindFloat64:
		return event.Float64(key, value.AsFloat64())
	case log.KindString:
		return event.Str(key, value.AsString())
	case log.KindBytes:
		return event.Bytes(key, value.AsBytes())
	case log.KindSlice:
		return event.Interface(key, value.AsSlice())
	case log.KindMap:
		return event.Interface(key, value.AsMap())
	default:
		return event.Interface(key, value.AsString())
	}
}
