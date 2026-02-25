package otel

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog"
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
	baseFields []Field // Base fields included in every log call
}

// NewLogHelper creates a logger that uses OTel when available, zerolog otherwise.
// When OTel is enabled, logs are automatically correlated with active spans.
//
// Parameters:
//   - ctx: Context for trace correlation
//   - config: OTel configuration (can be nil for zerolog-only mode)
//   - scopeName: OpenTelemetry scope name (e.g., "github.com/jasoet/pkg/v2/argo")
//   - function: Function name to include in logs (optional, can be empty string)
//
// Example:
//
//	// With OTel configured and function name
//	logger := otel.NewLogHelper(ctx, otelConfig, "github.com/jasoet/pkg/v2/mypackage", "mypackage.DoWork")
//	logger.Debug("Starting work", F("workerId", 123))
//
//	// Without function name (when used with spans)
//	logger := otel.NewLogHelper(ctx, otelConfig, "service.user", "")
//	logger.Info("Work completed")
//
//	// Without OTel (falls back to zerolog)
//	logger := otel.NewLogHelper(ctx, nil, "", "mypackage.DoWork")
//	logger.Info("Work completed")
func NewLogHelper(ctx context.Context, config *Config, scopeName, function string) *LogHelper {
	h := &LogHelper{
		ctx:      ctx,
		function: function,
	}

	if config != nil && config.IsLoggingEnabled() {
		h.otelLogger = config.GetLogger(scopeName)
	} else {
		serviceName := scopeName
		if config != nil && config.ServiceName != "" {
			serviceName = config.ServiceName
		}

		loggerCtx := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
			With().
			Timestamp().
			Str("service", serviceName).
			Int("pid", os.Getpid())

		if function != "" {
			loggerCtx = loggerCtx.Str("function", function)
		}

		h.logger = loggerCtx.Logger()
	}

	return h
}

// WithFields returns a new LogHelper with additional base fields.
// These fields will be automatically included in every log call.
//
// Example:
//
//	logger := otel.NewLogHelper(ctx, cfg, "service.user", "").
//	    WithFields(F("user.id", userID), F("action", "create"))
//	logger.Info("Processing request") // Includes user.id and action
func (h *LogHelper) WithFields(fields ...Field) *LogHelper {
	newHelper := &LogHelper{
		ctx:        h.ctx,
		function:   h.function,
		logger:     h.logger,
		otelLogger: h.otelLogger,
		baseFields: append(append(make([]Field, 0, len(h.baseFields)+len(fields)), h.baseFields...), fields...),
	}
	return newHelper
}

// Debug logs a debug-level message with optional fields.
// If OTel is enabled, automatically adds trace_id and span_id.
//
// Example:
//
//	logger.Debug("Processing request", F("request_id", reqID), F("user", userID))
func (h *LogHelper) Debug(msg string, fields ...Field) {
	h.log(otellog.SeverityDebug, h.logger.Debug, msg, fields...)
}

// Info logs an info-level message with optional fields.
// If OTel is enabled, automatically adds trace_id and span_id.
//
// Example:
//
//	logger.Info("User logged in", F("user_id", 123), F("role", "admin"))
func (h *LogHelper) Info(msg string, fields ...Field) {
	h.log(otellog.SeverityInfo, h.logger.Info, msg, fields...)
}

// Warn logs a warning-level message with optional fields.
// If OTel is enabled, automatically adds trace_id and span_id.
//
// Example:
//
//	logger.Warn("Rate limit approaching", F("current", 95), F("limit", 100))
func (h *LogHelper) Warn(msg string, fields ...Field) {
	h.log(otellog.SeverityWarn, h.logger.Warn, msg, fields...)
}

// log is the internal method that handles both OTel and zerolog logging
func (h *LogHelper) log(severity otellog.Severity, zerologFn func() *zerolog.Event, msg string, fields ...Field) {
	allFields := append(append(make([]Field, 0, len(h.baseFields)+len(fields)), h.baseFields...), fields...)

	if h.otelLogger != nil {
		params := otellog.EnabledParameters{Severity: severity}
		if h.otelLogger.Enabled(h.ctx, params) {
			h.emitOTel(severity, msg, allFields...)
		}
	} else {
		event := zerologFn()
		h.addFields(event, allFields...)
		event.Msg(msg)
	}
}

// Span returns the active span from the logger's context.
// Returns a non-nil span even if no span is active (use span.IsRecording() to check).
//
// Example:
//
//	span := logger.Span()
//	if span.IsRecording() {
//	    span.AddEvent("custom.event")
//	}
func (h *LogHelper) Span() trace.Span {
	return trace.SpanFromContext(h.ctx)
}

// Error logs an error-level message with optional fields.
// Also sets span status to error if a span is active.
//
// Example:
//
//	logger.Error(err, "Failed to process request", F("request_id", reqID), F("attempt", 3))
func (h *LogHelper) Error(err error, msg string, fields ...Field) {
	span := h.Span()
	if span.IsRecording() {
		span.SetStatus(codes.Error, msg)
		if err != nil {
			span.RecordError(err)
		}
	}

	allFields := append(append(make([]Field, 0, len(h.baseFields)+len(fields)), h.baseFields...), fields...)

	if h.otelLogger != nil {
		params := otellog.EnabledParameters{Severity: otellog.SeverityError}
		if h.otelLogger.Enabled(h.ctx, params) {
			if err != nil {
				errorField := F("error", err.Error())
				allFields = append([]Field{errorField}, allFields...)
			}
			h.emitOTel(otellog.SeverityError, msg, allFields...)
		}
	} else {
		event := h.logger.Error().Err(err)
		h.addFields(event, allFields...)
		event.Msg(msg)
	}
}

// emitOTel emits a log via OpenTelemetry with automatic trace correlation.
func (h *LogHelper) emitOTel(severity otellog.Severity, msg string, fields ...Field) {
	var record otellog.Record
	record.SetBody(otellog.StringValue(msg))
	record.SetSeverity(severity)

	if h.function != "" {
		record.AddAttributes(otellog.String("function", h.function))
	}

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
