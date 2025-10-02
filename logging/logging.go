package logging

import (
	"context"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
	"go.opentelemetry.io/otel/trace"
)

var initOnce sync.Once

// Initialize sets up the zerolog global logger with standard fields.
// This function should be called once at the start of your application.
// After calling Initialize, you can use zerolog's log package functions directly
// (log.Debug(), log.Info(), etc.) or create component-specific loggers with ContextLogger.
//
// Parameters:
//   - serviceName: Name of the service, added as a field to all log entries
//   - debug: If true, sets log level to Debug, otherwise Info
func Initialize(serviceName string, debug bool) {
	initOnce.Do(func() {
		level := zerolog.InfoLevel
		if debug {
			level = zerolog.DebugLevel
		}

		zerolog.SetGlobalLevel(level)

		zlog.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Str("service", serviceName).
			Int("pid", os.Getpid()).
			Caller().
			Logger()
	})
}

// ContextLogger creates a logger with context values.
// This function uses the global logger configured by Initialize.
// It adds context values and a component name to the logger.
//
// Parameters:
//   - ctx: Context that may contain values to be added to the logger
//   - component: Name of the component, added as a field to all log entries
//
// Returns:
//   - A zerolog.Logger instance with context and component information
func ContextLogger(ctx context.Context, component string) zerolog.Logger {
	return zlog.With().
		Ctx(ctx).
		Str("component", component).
		Logger()
}

// NewLoggerProvider creates an OpenTelemetry LoggerProvider that writes to zerolog.
// This allows using zerolog as the backend for OpenTelemetry logging.
//
// Parameters:
//   - serviceName: Name of the service, added as a field to all log entries
//   - debug: If true, sets log level to Debug, otherwise Info
//
// Returns:
//   - A log.LoggerProvider that bridges OTel logging to zerolog
func NewLoggerProvider(serviceName string, debug bool) log.LoggerProvider {
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

	return &zerologLoggerProvider{
		logger: zlog.Logger,
	}
}

// zerologLoggerProvider implements log.LoggerProvider
type zerologLoggerProvider struct {
	embedded.LoggerProvider
	logger zerolog.Logger
}

// Logger returns a Logger instance for the given instrumentation scope.
func (p *zerologLoggerProvider) Logger(name string, opts ...log.LoggerOption) log.Logger {
	// Add scope name as a field
	scopedLogger := p.logger.With().Str("scope", name).Logger()

	return &zerologLogger{
		logger: scopedLogger,
		name:   name,
	}
}

// zerologLogger implements log.Logger
type zerologLogger struct {
	embedded.Logger
	logger zerolog.Logger
	name   string
}

// Emit translates an OpenTelemetry log record to zerolog and emits it.
// It automatically extracts trace context (trace_id, span_id) from the context
// to enable log-span correlation in backends like Grafana.
func (l *zerologLogger) Emit(ctx context.Context, record log.Record) {
	// Map OTel severity to zerolog level
	var event *zerolog.Event
	severity := record.Severity()

	switch {
	case severity >= log.SeverityFatal:
		event = l.logger.Fatal()
	case severity >= log.SeverityError:
		event = l.logger.Error()
	case severity >= log.SeverityWarn:
		event = l.logger.Warn()
	case severity >= log.SeverityInfo:
		event = l.logger.Info()
	case severity >= log.SeverityDebug:
		event = l.logger.Debug()
	default:
		event = l.logger.Trace()
	}

	// Add timestamp
	if !record.Timestamp().IsZero() {
		event = event.Time("timestamp", record.Timestamp())
	}

	// Add severity text if present
	severityText := record.SeverityText()
	if severityText != "" {
		event = event.Str("severity", severityText)
	}

	// Extract and add trace context for log-span correlation
	// This is critical for linking logs to traces in Grafana and other backends
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		event = event.
			Str("trace_id", spanCtx.TraceID().String()).
			Str("span_id", spanCtx.SpanID().String())

		// Add trace_flags if the span is sampled
		if spanCtx.IsSampled() {
			event = event.Str("trace_flags", "01")
		} else {
			event = event.Str("trace_flags", "00")
		}
	}

	// Add all attributes from the record
	record.WalkAttributes(func(kv log.KeyValue) bool {
		key := kv.Key
		value := kv.Value

		switch value.Kind() {
		case log.KindBool:
			event = event.Bool(key, value.AsBool())
		case log.KindInt64:
			event = event.Int64(key, value.AsInt64())
		case log.KindFloat64:
			event = event.Float64(key, value.AsFloat64())
		case log.KindString:
			event = event.Str(key, value.AsString())
		case log.KindBytes:
			event = event.Bytes(key, value.AsBytes())
		case log.KindSlice:
			event = event.Interface(key, value.AsSlice())
		case log.KindMap:
			event = event.Interface(key, value.AsMap())
		default:
			event = event.Interface(key, value.AsString())
		}
		return true
	})

	// Get the body/message
	body := record.Body()
	message := body.AsString()
	if message == "" {
		message = "log entry"
	}

	// Emit the log
	event.Msg(message)
}

// Enabled returns whether this logger is enabled for the given severity.
func (l *zerologLogger) Enabled(_ context.Context, param log.EnabledParameters) bool {
	severity := param.Severity
	lvl := l.logger.GetLevel()

	// Map OTel severity to zerolog level and check
	switch {
	case severity >= log.SeverityFatal:
		return lvl <= zerolog.FatalLevel
	case severity >= log.SeverityError:
		return lvl <= zerolog.ErrorLevel
	case severity >= log.SeverityWarn:
		return lvl <= zerolog.WarnLevel
	case severity >= log.SeverityInfo:
		return lvl <= zerolog.InfoLevel
	case severity >= log.SeverityDebug:
		return lvl <= zerolog.DebugLevel
	default:
		return lvl <= zerolog.TraceLevel
	}
}
