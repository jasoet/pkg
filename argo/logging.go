package argo

import (
	"context"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/codes"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/trace"
)

// logHelper provides OTel-aware logging that automatically correlates logs with traces.
// It uses OTel logging when available (with automatic trace_id/span_id injection),
// otherwise falls back to plain zerolog.
type logHelper struct {
	ctx        context.Context
	config     *Config
	function   string
	logger     zerolog.Logger
	otelLogger otellog.Logger
}

// newLogHelper creates a logger that uses OTel when available, zerolog otherwise.
// When OTel is enabled, logs are automatically correlated with active spans.
func newLogHelper(ctx context.Context, config *Config, function string) *logHelper {
	h := &logHelper{
		ctx:      ctx,
		config:   config,
		function: function,
		logger:   log.With().Str("function", function).Logger(),
	}

	// Use OTel logger if available
	if config != nil && config.OTelConfig != nil && config.OTelConfig.IsLoggingEnabled() {
		h.otelLogger = config.OTelConfig.GetLogger("github.com/jasoet/pkg/v2/argo")
	}

	return h
}

// Debug logs a debug-level message with optional key-value pairs.
// If OTel is enabled, automatically adds trace_id and span_id.
func (h *logHelper) Debug(msg string, keysAndValues ...interface{}) {
	if h.otelLogger != nil {
		h.emitOTel(otellog.SeverityDebug, msg, keysAndValues...)
	} else {
		event := h.logger.Debug()
		h.addFields(event, keysAndValues...)
		event.Msg(msg)
	}
}

// Info logs an info-level message with optional key-value pairs.
func (h *logHelper) Info(msg string, keysAndValues ...interface{}) {
	if h.otelLogger != nil {
		h.emitOTel(otellog.SeverityInfo, msg, keysAndValues...)
	} else {
		event := h.logger.Info()
		h.addFields(event, keysAndValues...)
		event.Msg(msg)
	}
}

// Warn logs a warning-level message with optional key-value pairs.
func (h *logHelper) Warn(msg string, keysAndValues ...interface{}) {
	if h.otelLogger != nil {
		h.emitOTel(otellog.SeverityWarn, msg, keysAndValues...)
	} else {
		event := h.logger.Warn()
		h.addFields(event, keysAndValues...)
		event.Msg(msg)
	}
}

// Error logs an error-level message with optional key-value pairs.
// Also sets span status to error if a span is active.
func (h *logHelper) Error(err error, msg string, keysAndValues ...interface{}) {
	// Set span status to error if we have an active span
	span := trace.SpanFromContext(h.ctx)
	if span.IsRecording() {
		span.SetStatus(codes.Error, msg)
		span.RecordError(err)
	}

	if h.otelLogger != nil {
		kvs := append([]interface{}{"error", err.Error()}, keysAndValues...)
		h.emitOTel(otellog.SeverityError, msg, kvs...)
	} else {
		event := h.logger.Error().Err(err)
		h.addFields(event, keysAndValues...)
		event.Msg(msg)
	}
}

// emitOTel emits a log via OpenTelemetry with automatic trace correlation.
func (h *logHelper) emitOTel(severity otellog.Severity, msg string, keysAndValues ...interface{}) {
	var record otellog.Record
	record.SetBody(otellog.StringValue(msg))
	record.SetSeverity(severity)

	// Add function name
	record.AddAttributes(otellog.String("function", h.function))

	// Add key-value pairs
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				continue
			}

			switch v := keysAndValues[i+1].(type) {
			case string:
				record.AddAttributes(otellog.String(key, v))
			case bool:
				record.AddAttributes(otellog.Bool(key, v))
			case int:
				record.AddAttributes(otellog.Int64(key, int64(v)))
			case int64:
				record.AddAttributes(otellog.Int64(key, v))
			case float64:
				record.AddAttributes(otellog.Float64(key, v))
			default:
				record.AddAttributes(otellog.String(key, ""))
			}
		}
	}

	h.otelLogger.Emit(h.ctx, record)
}

// addFields adds key-value pairs to a zerolog event.
func (h *logHelper) addFields(event *zerolog.Event, keysAndValues ...interface{}) *zerolog.Event {
	for i := 0; i < len(keysAndValues); i += 2 {
		if i+1 < len(keysAndValues) {
			key, ok := keysAndValues[i].(string)
			if !ok {
				continue
			}

			switch v := keysAndValues[i+1].(type) {
			case string:
				event = event.Str(key, v)
			case bool:
				event = event.Bool(key, v)
			case int:
				event = event.Int(key, v)
			case int64:
				event = event.Int64(key, v)
			case float64:
				event = event.Float64(key, v)
			}
		}
	}
	return event
}
