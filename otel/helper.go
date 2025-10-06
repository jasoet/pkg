package otel

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

// Field represents a key-value pair for structured logging.
// Use the F() function to create fields for type-safe logging.
type Field struct {
	Key   string
	Value any
}

// F creates a Field for structured logging.
// This provides a type-safe, readable way to add context to log messages.
//
// Example:
//
//	logger.Info("User logged in", F("user_id", 123), F("email", "user@example.com"))
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// LogHelper provides OTel-aware logging that automatically correlates logs with traces.

// It uses OTel logging when available (with automatic trace_id/span_id injection),
// otherwise falls back to plain zerolog.
//
// This is the standard logging pattern for all packages in github.com/jasoet/pkg/v2:
//   - When OTel is configured: uses OTel LoggerProvider for automatic log-span correlation
//   - When OTel is not configured: falls back to zerolog
//
// Usage:
//
//	logger := otel.NewLogHelper(ctx, cfg, "scope-name", "function-name")
//	logger.Debug("message", "key", "value")
//	logger.Info("message", "key", "value")
//	logger.Error(err, "message", "key", "value")
type LogHelper struct {
	ctx        context.Context
	function   string
	logger     zerolog.Logger
	otelLogger otellog.Logger
}

// NewLogHelper creates a logger that uses OTel when available, zerolog otherwise.
// When OTel is enabled, logs are automatically correlated with active spans.
//
// Parameters:
//   - ctx: Context for trace correlation
//   - config: OTel configuration (can be nil for zerolog-only mode)
//   - scopeName: OpenTelemetry scope name (e.g., "github.com/jasoet/pkg/v2/argo")
//   - function: Function name to include in logs (e.g., "argo.NewClient")
//
// Example:
//
//	// With OTel configured
//	logger := otel.NewLogHelper(ctx, otelConfig, "github.com/jasoet/pkg/v2/mypackage", "mypackage.DoWork")
//	logger.Debug("Starting work", "workerId", 123)
//
//	// Without OTel (falls back to zerolog)
//	logger := otel.NewLogHelper(ctx, nil, "", "mypackage.DoWork")
//	logger.Info("Work completed")
func NewLogHelper(ctx context.Context, config *Config, scopeName, function string) *LogHelper {
	h := &LogHelper{
		ctx:      ctx,
		function: function,
		logger:   log.With().Str("function", function).Logger(),
	}

	// Use OTel logger if available
	if config != nil && config.IsLoggingEnabled() {
		h.otelLogger = config.GetLogger(scopeName)
	}

	return h
}

// Debug logs a debug-level message with optional fields.
// If OTel is enabled, automatically adds trace_id and span_id.
//
// Example:
//
//	logger.Debug("Processing request", F("request_id", reqID), F("user", userID))
func (h *LogHelper) Debug(msg string, fields ...Field) {
	if h.otelLogger != nil {
		h.emitOTel(otellog.SeverityDebug, msg, fields...)
	} else {
		event := h.logger.Debug()
		h.addFields(event, fields...)
		event.Msg(msg)
	}
}

// Info logs an info-level message with optional fields.
// If OTel is enabled, automatically adds trace_id and span_id.
//
// Example:
//
//	logger.Info("User logged in", F("user_id", 123), F("role", "admin"))
func (h *LogHelper) Info(msg string, fields ...Field) {
	if h.otelLogger != nil {
		h.emitOTel(otellog.SeverityInfo, msg, fields...)
	} else {
		event := h.logger.Info()
		h.addFields(event, fields...)
		event.Msg(msg)
	}
}

// Warn logs a warning-level message with optional fields.
// If OTel is enabled, automatically adds trace_id and span_id.
//
// Example:
//
//	logger.Warn("Rate limit approaching", F("current", 95), F("limit", 100))
func (h *LogHelper) Warn(msg string, fields ...Field) {
	if h.otelLogger != nil {
		h.emitOTel(otellog.SeverityWarn, msg, fields...)
	} else {
		event := h.logger.Warn()
		h.addFields(event, fields...)
		event.Msg(msg)
	}
}

// Error logs an error-level message with optional fields.
// Also sets span status to error if a span is active.
//
// Example:
//
//	logger.Error(err, "Failed to process request", F("request_id", reqID), F("attempt", 3))
func (h *LogHelper) Error(err error, msg string, fields ...Field) {
	// Set span status to error if we have an active span
	span := trace.SpanFromContext(h.ctx)
	if span.IsRecording() {
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)
	}

	if h.otelLogger != nil {
		// Add error field at the beginning
		errorField := F("error", err.Error())
		allFields := append([]Field{errorField}, fields...)
		h.emitOTel(otellog.SeverityError, msg, allFields...)
	} else {
		event := h.logger.Error().Err(err)
		h.addFields(event, fields...)
		event.Msg(msg)
	}
}

// emitOTel emits a log via OpenTelemetry with automatic trace correlation.
func (h *LogHelper) emitOTel(severity otellog.Severity, msg string, fields ...Field) {
	var record otellog.Record
	record.SetBody(otellog.StringValue(msg))
	record.SetSeverity(severity)

	// Add function name
	record.AddAttributes(otellog.String("function", h.function))

	// Add fields
	for _, field := range fields {
		switch v := field.Value.(type) {
		case string:
			record.AddAttributes(otellog.String(field.Key, v))
		case bool:
			record.AddAttributes(otellog.Bool(field.Key, v))
		case int:
			record.AddAttributes(otellog.Int64(field.Key, int64(v)))
		case int64:
			record.AddAttributes(otellog.Int64(field.Key, v))
		case float64:
			record.AddAttributes(otellog.Float64(field.Key, v))
		case time.Duration:
			record.AddAttributes(otellog.String(field.Key, v.String()))
		default:
			// For other types, convert to string using fmt.Sprint
			record.AddAttributes(otellog.String(field.Key, fmt.Sprint(v)))
		}
	}

	h.otelLogger.Emit(h.ctx, record)
}

// addFields adds Field key-value pairs to a zerolog event.
func (h *LogHelper) addFields(event *zerolog.Event, fields ...Field) *zerolog.Event {
	for _, field := range fields {
		switch v := field.Value.(type) {
		case string:
			event = event.Str(field.Key, v)
		case bool:
			event = event.Bool(field.Key, v)
		case int:
			event = event.Int(field.Key, v)
		case int64:
			event = event.Int64(field.Key, v)
		case float64:
			event = event.Float64(field.Key, v)
		case time.Duration:
			event = event.Str(field.Key, v.String())
		default:
			event = event.Str(field.Key, fmt.Sprint(v))
		}
	}
	return event
}
