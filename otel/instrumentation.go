package otel

import (
	"context"
	"fmt"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// SpanHelper provides a convenient way to create and manage spans with automatic
// error handling and status management. It's designed for use in service and
// repository layers where consistent span instrumentation is needed.
//
// Usage pattern:
//
//	func (s *Service) DoWork(ctx context.Context, id string) error {
//	    span := otel.StartSpan(ctx, "service.example", "Service.DoWork",
//	        otel.WithAttribute("entity.id", id))
//	    defer span.End()
//
//	    // IMPORTANT: Use span.Context() for child operations to maintain trace correlation
//	    if err := s.repository.Save(span.Context(), data); err != nil {
//	        return span.Error(err, "failed to save data")
//	    }
//
//	    return span.Success()
//	}
type SpanHelper struct {
	ctx  context.Context
	span trace.Span
}

// SpanOption allows customizing span creation
type SpanOption func(*spanConfig)

type spanConfig struct {
	attributes []attribute.KeyValue
	spanKind   trace.SpanKind
}

// WithAttribute adds an attribute to the span
func WithAttribute(key string, value any) SpanOption {
	return func(cfg *spanConfig) {
		cfg.attributes = append(cfg.attributes, toAttribute(key, value))
	}
}

// WithAttributes adds multiple attributes to the span
func WithAttributes(fields ...Field) SpanOption {
	return func(cfg *spanConfig) {
		for _, field := range fields {
			cfg.attributes = append(cfg.attributes, toAttribute(field.Key, field.Value))
		}
	}
}

// WithSpanKind sets the span kind (Internal, Server, Client, Producer, Consumer)
func WithSpanKind(kind trace.SpanKind) SpanOption {
	return func(cfg *spanConfig) {
		cfg.spanKind = kind
	}
}

// StartSpan creates a new span with the given tracer name and operation name.
// The tracer name should follow the pattern "layer.component" (e.g., "service.event", "repository.ticket").
// The operation name should be descriptive (e.g., "EventService.CancelEvent", "TicketRepository.FindByID").
//
// This is the recommended way to create spans in service and repository layers.
//
// Example:
//
//	// In service layer
//	span := otel.StartSpan(ctx, "service.event", "EventService.CancelEvent",
//	    otel.WithAttribute("event.id", eventID))
//	defer span.End()
//
//	// In repository layer
//	span := otel.StartSpan(ctx, "repository.event", "EventRepository.FindByID",
//	    otel.WithAttribute("event.id", eventID),
//	    otel.WithAttribute("db.operation", "select"))
//	defer span.End()
func StartSpan(ctx context.Context, tracerName, operationName string, opts ...SpanOption) *SpanHelper {
	cfg := &spanConfig{
		attributes: make([]attribute.KeyValue, 0),
		spanKind:   trace.SpanKindInternal,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	// Use TracerProvider from config if available, otherwise use global provider
	var tracer trace.Tracer
	if config := ConfigFromContext(ctx); config != nil && config.TracerProvider != nil {
		tracer = config.TracerProvider.Tracer(tracerName)
	} else {
		tracer = otel.Tracer(tracerName)
	}

	ctx, span := tracer.Start(ctx, operationName,
		trace.WithSpanKind(cfg.spanKind),
		trace.WithAttributes(cfg.attributes...),
	)

	return &SpanHelper{
		ctx:  ctx,
		span: span,
	}
}

// Context returns the context with the span attached.
// Use this when calling child functions that need the span context.
func (h *SpanHelper) Context() context.Context {
	return h.ctx
}

// Span returns the underlying trace.Span for advanced usage.
func (h *SpanHelper) Span() trace.Span {
	return h.span
}

// Logger creates a LogHelper that is automatically correlated with this span.
// Returns a LogHelper with the default zerolog logger if no config is stored in the context.
// Use ContextWithConfig() to store config in context before creating spans.
//
// Example:
//
//	ctx = otel.ContextWithConfig(ctx, cfg)
//	span := otel.StartSpan(ctx, "service.user", "CreateUser",
//	    otel.WithAttribute("user.id", userID))
//	defer span.End()
//
//	logger := span.Logger("service.user")
//
//	logger.Info("Creating user", F("email", email))
func (h *SpanHelper) Logger(scopeName string) *LogHelper {
	config := ConfigFromContext(h.ctx)
	return NewLogHelper(h.ctx, config, scopeName, "")
}

// FunctionLogger creates a LogHelper that is automatically correlated with this span.
// Returns a LogHelper with the default zerolog logger if no config is stored in the context.
// Use ContextWithConfig() to store config in context before creating spans.
//
// Example:
//
//	ctx = otel.ContextWithConfig(ctx, cfg)
//	span := otel.StartSpan(ctx, "service.user", "CreateUser",
//	    otel.WithAttribute("user.id", userID))
//	defer span.End()
//
//	logger := span.FunctionLogger("service.user","function.name")
//	logger.Info("Creating user", F("email", email))
func (h *SpanHelper) FunctionLogger(scopeName string, function string) *LogHelper {
	config := ConfigFromContext(h.ctx)
	return NewLogHelper(h.ctx, config, scopeName, function)
}

// AddAttribute adds a single attribute to the span.
func (h *SpanHelper) AddAttribute(key string, value any) *SpanHelper {
	h.span.SetAttributes(toAttribute(key, value))
	return h
}

// AddAttributes adds multiple attributes to the span.
func (h *SpanHelper) AddAttributes(fields ...Field) *SpanHelper {
	attributes := make([]attribute.KeyValue, 0, len(fields))
	for _, field := range fields {
		attributes = append(attributes, toAttribute(field.Key, field.Value))
	}
	h.span.SetAttributes(attributes...)
	return h
}

// AddEvent adds an event to the span with optional attributes.
//
// Example:
//
//	span.AddEvent("cache.hit", F("key", cacheKey), F("ttl", ttl))
func (h *SpanHelper) AddEvent(name string, fields ...Field) *SpanHelper {
	attributes := make([]attribute.KeyValue, 0, len(fields))
	for _, field := range fields {
		attributes = append(attributes, toAttribute(field.Key, field.Value))
	}
	h.span.AddEvent(name, trace.WithAttributes(attributes...))
	return h
}

// LogEvent creates both a span event and a log entry for better correlation.
// This is useful for significant events that should appear in both traces and logs.
//
// Example:
//
//	logger := span.Logger("service.cache")
//	span.LogEvent(logger, "cache.miss",
//	    F("key", cacheKey),
//	    F("reason", "expired"))
func (h *SpanHelper) LogEvent(logger *LogHelper, eventName string, fields ...Field) *SpanHelper {
	// Add span event
	h.AddEvent(eventName, fields...)

	// Add log entry if logger provided
	if logger != nil {
		logger.Info(eventName, fields...)
	}

	return h
}

// Error records an error and sets the span status to error.
// Returns the error unchanged for easy error propagation.
//
// Example:
//
//	if err := doWork(); err != nil {
//	    return span.Error(err, "work failed")
//	}
func (h *SpanHelper) Error(err error, message string) error {
	if err != nil {
		h.span.RecordError(err)
		h.span.SetStatus(codes.Error, message)
	}
	return err
}

// Success marks the span as successful.
// This is optional but provides explicit success signaling.
//
// Example:
//
//	span.Success()
func (h *SpanHelper) Success() {
	h.span.SetStatus(codes.Ok, "")
}

// End finishes the span. Always defer this after creating a span.
//
// Example:
//
//	span := otel.StartSpan(ctx, "service", "Operation")
//	defer span.End()
func (h *SpanHelper) End() {
	h.span.End()
}

// toAttribute converts a value to an OpenTelemetry attribute
func toAttribute(key string, value any) attribute.KeyValue {
	switch v := value.(type) {
	case string:
		return attribute.String(key, v)
	case bool:
		return attribute.Bool(key, v)
	case int:
		return attribute.Int(key, v)
	case int64:
		return attribute.Int64(key, v)
	case float64:
		return attribute.Float64(key, v)
	default:
		return attribute.String(key, toString(v))
	}
}

// toString converts any value to a string
func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	if stringer, ok := v.(interface{ String() string }); ok {
		return stringer.String()
	}
	return fmt.Sprint(v)
}

// LayerContext provides unified access to both span and logger for a layer operation.
// This combines span tracing and logging with automatic correlation.
// Base fields are automatically included in Error() and Success() log calls.
type LayerContext struct {
	Span   *SpanHelper
	Logger *LogHelper
	fields []Field // Base fields for all logs
}

// Context returns the span's context for passing to child operations.
func (lc *LayerContext) Context() context.Context {
	return lc.Span.Context()
}

// End finishes the span. Always defer this after creating a LayerContext.
func (lc *LayerContext) End() {
	lc.Span.End()
}

// Error records an error to both span and logs, then returns the error.
// Base fields from StartX are automatically included in the log via the Logger.
// Additional fields are also added as span attributes for correlation.
//
// Example:
//
//	if err := repo.Save(lc.Context(), data); err != nil {
//	    return lc.Error(err, "failed to save", F("id", id))
//	}
func (lc *LayerContext) Error(err error, msg string, fields ...Field) error {
	// Add fields as span attributes
	if len(fields) > 0 {
		lc.Span.AddAttributes(fields...)
	}
	if lc.Logger != nil {
		lc.Logger.Error(err, msg, fields...)
	}
	return lc.Span.Error(err, msg)
}

// Success marks the operation as successful in both span and logs.
// Base fields from StartX are automatically included in the log via the Logger.
// Additional fields are also added as span attributes for correlation.
// The message is added as a span event named "success" for observability.
//
// Example:
//
//	lc.Success("User created successfully", F("user_id", userID))
func (lc *LayerContext) Success(msg string, fields ...Field) {
	// Add fields as span attributes
	if len(fields) > 0 {
		lc.Span.AddAttributes(fields...)
	}
	// Add success message as span event
	lc.Span.AddEvent("success", F("message", msg))
	if lc.Logger != nil {
		lc.Logger.Info(msg, fields...)
	}
	lc.Span.Success()
}

// LayeredSpanHelper provides convenience methods for common span patterns across
// handler, service, and repository layers with consistent naming and attributes.
// All methods return LayerContext which provides both span and logger for unified
// tracing and logging with automatic correlation.
type LayeredSpanHelper struct{}

// StartHandler creates a LayerContext for HTTP handler layer operations.
// Combines span and logger with automatic correlation.
//
// Example:
//
//	func (h *EventHandler) Create(c echo.Context) error {
//	    lc := otel.Layers.StartHandler(c.Request().Context(), "event", "Create",
//	        F("event.type", eventType))
//	    defer lc.End()
//
//	    lc.Logger.Info("Creating event", F("user_id", userID))
//	    if err := h.service.Create(lc.Context(), req); err != nil {
//	        return lc.Error(err, "failed to create event")
//	    }
//	    return lc.Success("Event created")
//	}
func (l *LayeredSpanHelper) StartHandler(ctx context.Context, component, operation string, fields ...Field) *LayerContext {
	tracerName := "handler." + component
	operationName := component + "." + operation

	allFields := append([]Field{F("layer", "handler")}, fields...)

	span := StartSpan(ctx, tracerName, operationName,
		WithAttributes(allFields...),
		WithSpanKind(trace.SpanKindServer))

	return &LayerContext{
		Span:   span,
		Logger: span.Logger(tracerName).WithFields(allFields...),
		fields: allFields,
	}
}

// StartService creates a LayerContext for service layer business logic.
// Combines span and logger with automatic correlation.
//
// Example:
//
//	func (s *EventService) CancelEvent(ctx context.Context, eventID string) error {
//	    lc := otel.Layers.StartService(ctx, "event", "CancelEvent",
//	        F("event.id", eventID))
//	    defer lc.End()
//
//	    lc.Logger.Info("Canceling event")
//	    if err := s.repo.Update(lc.Context(), data); err != nil {
//	        return lc.Error(err, "failed to update event")
//	    }
//	    return lc.Success("Event cancelled")
//	}
func (l *LayeredSpanHelper) StartService(ctx context.Context, component, operation string, fields ...Field) *LayerContext {
	tracerName := "service." + component
	operationName := component + "." + operation

	allFields := append([]Field{F("layer", "service")}, fields...)

	span := StartSpan(ctx, tracerName, operationName,
		WithAttributes(allFields...),
		WithSpanKind(trace.SpanKindInternal))

	return &LayerContext{
		Span:   span,
		Logger: span.Logger(tracerName).WithFields(allFields...),
		fields: allFields,
	}
}

// StartOperations creates a LayerContext for operations layer orchestration.
// Combines span and logger with automatic correlation.
//
// Example:
//
//	func (o *EventOps) ProcessQueue(ctx context.Context, queueName string) error {
//	    lc := otel.Layers.StartOperations(ctx, "event", "ProcessQueue",
//	        F("queue.name", queueName))
//	    defer lc.End()
//
//	    lc.Logger.Info("Processing queue")
//	    if err := o.service.Process(lc.Context()); err != nil {
//	        return lc.Error(err, "failed to process queue")
//	    }
//	    return lc.Success("Queue processed")
//	}
func (l *LayeredSpanHelper) StartOperations(ctx context.Context, component, operation string, fields ...Field) *LayerContext {
	tracerName := "operations." + component
	operationName := component + "." + operation

	allFields := append([]Field{F("layer", "operations")}, fields...)

	span := StartSpan(ctx, tracerName, operationName,
		WithAttributes(allFields...),
		WithSpanKind(trace.SpanKindInternal))

	return &LayerContext{
		Span:   span,
		Logger: span.Logger(tracerName).WithFields(allFields...),
		fields: allFields,
	}
}

// StartMiddleware creates a LayerContext for middleware layer operations.
// Combines span and logger with automatic correlation.
//
// Example:
//
//	func AuthMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
//	    return func(c echo.Context) error {
//	        lc := otel.Layers.StartMiddleware(c.Request().Context(), "auth", "ValidateToken",
//	            F("http.path", c.Path()),
//	            F("http.method", c.Request().Method))
//	        defer lc.End()
//
//	        lc.Logger.Info("Validating authentication token")
//	        token := c.Request().Header.Get("Authorization")
//	        if token == "" {
//	            return lc.Error(errors.New("missing token"), "authentication failed")
//	        }
//
//	        // Pass updated context to next handler
//	        c.SetRequest(c.Request().WithContext(lc.Context()))
//	        if err := next(c); err != nil {
//	            return lc.Error(err, "request failed")
//	        }
//	        return lc.Success("Request processed successfully")
//	    }
//	}
func (l *LayeredSpanHelper) StartMiddleware(ctx context.Context, component, operation string, fields ...Field) *LayerContext {
	tracerName := "middleware." + component
	operationName := component + "." + operation

	allFields := append([]Field{F("layer", "middleware")}, fields...)

	span := StartSpan(ctx, tracerName, operationName,
		WithAttributes(allFields...),
		WithSpanKind(trace.SpanKindServer))

	return &LayerContext{
		Span:   span,
		Logger: span.Logger(tracerName).WithFields(allFields...),
		fields: allFields,
	}
}

// StartRepository creates a LayerContext for repository layer data access.
// Combines span and logger with automatic correlation.
//
// Example:
//
//	func (r *EventRepository) FindByID(ctx context.Context, eventID string) (*Event, error) {
//	    lc := otel.Layers.StartRepository(ctx, "event", "FindByID",
//	        F("event.id", eventID),
//	        F("db.operation", "select"))
//	    defer lc.End()
//
//	    lc.Logger.Debug("Querying database")
//	    event, err := r.db.QueryRow(lc.Context(), query, eventID)
//	    if err != nil {
//	        return nil, lc.Error(err, "query failed")
//	    }
//	    lc.Success("Event found")
//	    return event, nil
//	}
func (l *LayeredSpanHelper) StartRepository(ctx context.Context, component, operation string, fields ...Field) *LayerContext {
	tracerName := "repository." + component
	operationName := component + "." + operation

	allFields := append([]Field{F("layer", "repository")}, fields...)

	span := StartSpan(ctx, tracerName, operationName,
		WithAttributes(allFields...),
		WithSpanKind(trace.SpanKindClient))

	return &LayerContext{
		Span:   span,
		Logger: span.Logger(tracerName).WithFields(allFields...),
		fields: allFields,
	}
}

// Layers provides convenience methods for creating layer-specific spans
var Layers = &LayeredSpanHelper{}
