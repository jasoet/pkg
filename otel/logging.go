package otel

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

// LogLevel represents the logging level for the console/OTel log pipeline.
// Trace and Fatal levels are intentionally excluded: Trace is not supported by zerolog natively,
// and Fatal triggers os.Exit which is unsuitable for library use.
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelNone  LogLevel = "none"
)

// LoggerProviderOption configures LoggerProvider behavior
type LoggerProviderOption func(*loggerProviderConfig)

// loggerProviderConfig holds configuration for logger provider
type loggerProviderConfig struct {
	serviceName     string
	consoleOutput   bool
	otlpEndpoint    string
	otlpInsecure    bool
	otlpEndpointSet bool // true once WithOTLPEndpoint has been applied
	logLevel        LogLevel
}

// WithConsoleOutput enables console logging alongside OTLP
func WithConsoleOutput(enabled bool) LoggerProviderOption {
	return func(cfg *loggerProviderConfig) {
		cfg.consoleOutput = enabled
	}
}

// WithOTLPEndpoint enables OTLP log export to the given endpoint.
//
// The endpoint may be provided in either form:
//   - A full URL with scheme, e.g. "https://collector.example.com:4318".
//     The scheme selects http/https and the path (if any) is honored.
//   - A bare host:port without scheme, e.g. "collector.example.com:4318".
//     The default OTLP logs path ("/v1/logs") is used.
//
// The insecure flag forces plaintext HTTP; it is redundant with (and overrides)
// an "http://" scheme.
//
// Supplying an empty endpoint is treated as a configuration error by
// NewLoggerProviderWithOptions rather than silently disabling OTLP export.
func WithOTLPEndpoint(endpoint string, insecure bool) LoggerProviderOption {
	return func(cfg *loggerProviderConfig) {
		cfg.otlpEndpoint = endpoint
		cfg.otlpInsecure = insecure
		cfg.otlpEndpointSet = true
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
// Console output uses a synchronous processor (SimpleProcessor) intentionally,
// so that log records are written immediately and are visible without buffering.
// This is appropriate for development and for ensuring log visibility on process exit.
// OTLP export uses a BatchProcessor for efficiency.
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
//	    otel.WithLogLevel(otel.LogLevelDebug),
//	    otel.WithOTLPEndpoint("https://localhost:4318", true),
//	    otel.WithConsoleOutput(true))
func NewLoggerProviderWithOptions(serviceName string, opts ...LoggerProviderOption) (log.LoggerProvider, error) {
	cfg := &loggerProviderConfig{
		serviceName:   serviceName,
		consoleOutput: true, // Default: keep console output
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.otlpEndpointSet && cfg.otlpEndpoint == "" {
		return nil, fmt.Errorf("otel: WithOTLPEndpoint enabled with an empty endpoint")
	}

	effectiveLevel := cfg.logLevel
	if effectiveLevel == "" {
		effectiveLevel = LogLevelInfo
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
		var exporterOpts []otlploghttp.Option
		if strings.Contains(cfg.otlpEndpoint, "://") {
			// URL-shaped endpoint: honor scheme, host, and path. Passing this
			// to WithEndpoint (host:port only) would embed the scheme in the
			// host and silently break every export.
			exporterOpts = append(exporterOpts, otlploghttp.WithEndpointURL(cfg.otlpEndpoint))
			// WithEndpointURL uses the URL path verbatim; for a bare base URL
			// (no path) it would POST to "/". Fall back to the standard OTLP
			// logs path so "https://collector:4318" targets "/v1/logs".
			if u, perr := url.Parse(cfg.otlpEndpoint); perr != nil || u.Path == "" || u.Path == "/" {
				exporterOpts = append(exporterOpts, otlploghttp.WithURLPath("/v1/logs"))
			}
		} else {
			// Bare host:port endpoint; WithEndpoint applies the default
			// "/v1/logs" path automatically.
			exporterOpts = append(exporterOpts, otlploghttp.WithEndpoint(cfg.otlpEndpoint))
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

	// When console output is explicitly disabled and no OTLP endpoint is set,
	// the provider intentionally has no processors (a silent provider). We do
	// not re-add a console exporter, which would contradict the explicit
	// WithConsoleOutput(false).

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

// Shutdown implements sdklog.Exporter interface.
// It is intentionally a no-op: console output (stderr) requires no teardown.
func (e *consoleExporter) Shutdown(ctx context.Context) error {
	return nil
}

// ForceFlush implements sdklog.Exporter interface.
// It is intentionally a no-op: each record is written synchronously, so there
// is no internal buffer to flush.
func (e *consoleExporter) ForceFlush(ctx context.Context) error {
	return nil
}

// logLevelToZerolog converts LogLevel to zerolog.Level
func logLevelToZerolog(level LogLevel) zerolog.Level {
	switch level {
	case LogLevelDebug:
		return zerolog.DebugLevel
	case LogLevelInfo:
		return zerolog.InfoLevel
	case LogLevelWarn:
		return zerolog.WarnLevel
	case LogLevelError:
		return zerolog.ErrorLevel
	case LogLevelNone:
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
		return event.Interface(key, fmt.Sprint(value))
	}
}
