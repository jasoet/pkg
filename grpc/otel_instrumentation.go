package grpc

import (
	"context"
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	pkgotel "github.com/jasoet/pkg/v2/otel"
)

// metadataCarrier adapts gRPC metadata to the OTel TextMapCarrier interface,
// enabling W3C Trace Context (traceparent/tracestate) extraction.
type metadataCarrier metadata.MD

func (mc metadataCarrier) Get(key string) string {
	vals := metadata.MD(mc).Get(key)
	if len(vals) == 0 {
		return ""
	}
	return vals[0]
}

func (mc metadataCarrier) Set(key, val string) {
	metadata.MD(mc).Set(key, val)
}

func (mc metadataCarrier) Keys() []string {
	md := metadata.MD(mc)
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	return keys
}

// ============================================================================
// gRPC Metrics (OpenTelemetry)
// ============================================================================

// createGRPCMetricsInterceptor creates gRPC unary interceptor for metrics
func createGRPCMetricsInterceptor(cfg *pkgotel.Config) grpc.UnaryServerInterceptor {
	if cfg == nil || !cfg.IsMetricsEnabled() {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	meter := cfg.GetMeter("grpc.server")

	// Create metrics instruments
	// Note: errors are intentionally ignored as they only occur with nil meter (checked by GetMeter)
	requestCounter, _ := meter.Int64Counter( //nolint:errcheck
		"rpc.server.request.count",
		metric.WithDescription("Total number of gRPC requests"),
		metric.WithUnit("{request}"),
	)

	requestDuration, _ := meter.Float64Histogram( //nolint:errcheck
		"rpc.server.duration",
		metric.WithDescription("gRPC request duration"),
		metric.WithUnit("ms"),
	)

	activeRequests, _ := meter.Int64UpDownCounter( //nolint:errcheck
		"rpc.server.active_requests",
		metric.WithDescription("Number of active gRPC requests"),
		metric.WithUnit("{request}"),
	)

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Increment active requests
		activeRequests.Add(ctx, 1)
		defer activeRequests.Add(ctx, -1)

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start).Milliseconds()

		// Get status code
		st, _ := status.FromError(err)
		statusCode := int(st.Code())

		// Prepare attributes using semantic conventions
		attrs := []attribute.KeyValue{
			semconv.RPCMethodKey.String(info.FullMethod),
			semconv.RPCSystemKey.String("grpc"),
			attribute.Int("rpc.grpc.status_code", statusCode),
		}

		// Record metrics
		requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		requestDuration.Record(ctx, float64(duration), metric.WithAttributes(attrs...))

		return resp, err
	}
}

// ============================================================================
// gRPC Tracing (OpenTelemetry)
// ============================================================================

// createGRPCTracingInterceptor creates gRPC unary interceptor for distributed tracing
func createGRPCTracingInterceptor(cfg *pkgotel.Config) grpc.UnaryServerInterceptor {
	if cfg == nil || !cfg.IsTracingEnabled() {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	tracer := cfg.GetTracer("grpc.server")

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Extract W3C Trace Context (traceparent/tracestate) from gRPC metadata
		if md, ok := metadata.FromIncomingContext(ctx); ok {
			ctx = otel.GetTextMapPropagator().Extract(ctx, metadataCarrier(md))
		}

		// Start span
		ctx, span := tracer.Start(ctx, info.FullMethod,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				semconv.RPCSystemKey.String("grpc"),
				semconv.RPCMethodKey.String(info.FullMethod),
				semconv.RPCServiceKey.String(extractServiceName(info.FullMethod)),
			),
		)
		defer span.End()

		// Call handler
		resp, err := handler(ctx, req)

		// Record status
		if err != nil {
			st, _ := status.FromError(err)
			span.SetAttributes(
				attribute.Int("rpc.grpc.status_code", int(st.Code())),
				attribute.String("rpc.grpc.status_message", st.Message()),
			)
			span.RecordError(err)
		} else {
			span.SetAttributes(attribute.Int("rpc.grpc.status_code", 0))
		}

		return resp, err
	}
}

// extractServiceName extracts service name from full method name
// e.g., "/package.Service/Method" -> "package.Service"
func extractServiceName(fullMethod string) string {
	if len(fullMethod) == 0 {
		return ""
	}
	// Remove leading slash
	if fullMethod[0] == '/' {
		fullMethod = fullMethod[1:]
	}
	// Find last slash
	for i := len(fullMethod) - 1; i >= 0; i-- {
		if fullMethod[i] == '/' {
			return fullMethod[:i]
		}
	}
	return fullMethod
}

// ============================================================================
// gRPC Logging (OpenTelemetry)
// ============================================================================

// createGRPCLoggingInterceptor creates gRPC unary interceptor for structured logging
func createGRPCLoggingInterceptor(cfg *pkgotel.Config) grpc.UnaryServerInterceptor {
	if cfg == nil || !cfg.IsLoggingEnabled() {
		return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
			return handler(ctx, req)
		}
	}

	logger := cfg.GetLogger("grpc.server")

	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		start := time.Now()

		// Call handler
		resp, err := handler(ctx, req)

		// Calculate duration
		duration := time.Since(start)

		// Determine severity
		severity := otellog.SeverityInfo
		if err != nil {
			severity = otellog.SeverityError
		}

		// Get status
		st, _ := status.FromError(err)

		// Create log attributes
		attrs := []otellog.KeyValue{
			otellog.String("rpc.system", "grpc"),
			otellog.String("rpc.method", info.FullMethod),
			otellog.String("rpc.service", extractServiceName(info.FullMethod)),
			otellog.Int("rpc.grpc.status_code", int(st.Code())),
			otellog.Int64("rpc.duration_ms", duration.Milliseconds()),
		}

		if err != nil {
			attrs = append(attrs, otellog.String("error", err.Error()))
		}

		// Emit log record
		var logRecord otellog.Record
		logRecord.SetTimestamp(start)
		logRecord.SetSeverity(severity)
		logRecord.SetBody(otellog.StringValue(fmt.Sprintf("gRPC %s", info.FullMethod)))
		logRecord.AddAttributes(attrs...)

		logger.Emit(ctx, logRecord)

		return resp, err
	}
}

// ============================================================================
// gRPC Stream Interceptors (OpenTelemetry)
// ============================================================================

// createGRPCStreamLoggingInterceptor creates a gRPC stream interceptor for structured logging
func createGRPCStreamLoggingInterceptor(cfg *pkgotel.Config) grpc.StreamServerInterceptor {
	if cfg == nil || !cfg.IsLoggingEnabled() {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	logger := cfg.GetLogger("grpc.server")

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()

		err := handler(srv, ss)

		duration := time.Since(start)
		severity := otellog.SeverityInfo
		if err != nil {
			severity = otellog.SeverityError
		}

		st, _ := status.FromError(err)

		attrs := []otellog.KeyValue{
			otellog.String("rpc.system", "grpc"),
			otellog.String("rpc.method", info.FullMethod),
			otellog.String("rpc.service", extractServiceName(info.FullMethod)),
			otellog.Int("rpc.grpc.status_code", int(st.Code())),
			otellog.Int64("rpc.duration_ms", duration.Milliseconds()),
			otellog.Bool("rpc.is_client_stream", info.IsClientStream),
			otellog.Bool("rpc.is_server_stream", info.IsServerStream),
		}

		if err != nil {
			attrs = append(attrs, otellog.String("error", err.Error()))
		}

		var logRecord otellog.Record
		logRecord.SetTimestamp(start)
		logRecord.SetSeverity(severity)
		logRecord.SetBody(otellog.StringValue(fmt.Sprintf("gRPC stream %s", info.FullMethod)))
		logRecord.AddAttributes(attrs...)

		logger.Emit(ss.Context(), logRecord)

		return err
	}
}

// createGRPCStreamMetricsInterceptor creates a gRPC stream interceptor for metrics
func createGRPCStreamMetricsInterceptor(cfg *pkgotel.Config) grpc.StreamServerInterceptor {
	if cfg == nil || !cfg.IsMetricsEnabled() {
		return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
			return handler(srv, ss)
		}
	}

	meter := cfg.GetMeter("grpc.server")

	streamCounter, _ := meter.Int64Counter( //nolint:errcheck
		"rpc.server.stream.count",
		metric.WithDescription("Total number of gRPC streams"),
		metric.WithUnit("{stream}"),
	)

	streamDuration, _ := meter.Float64Histogram( //nolint:errcheck
		"rpc.server.stream.duration",
		metric.WithDescription("gRPC stream duration"),
		metric.WithUnit("ms"),
	)

	activeStreams, _ := meter.Int64UpDownCounter( //nolint:errcheck
		"rpc.server.active_streams",
		metric.WithDescription("Number of active gRPC streams"),
		metric.WithUnit("{stream}"),
	)

	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		start := time.Now()
		ctx := ss.Context()

		activeStreams.Add(ctx, 1)
		defer activeStreams.Add(ctx, -1)

		err := handler(srv, ss)

		duration := time.Since(start).Milliseconds()
		st, _ := status.FromError(err)
		statusCode := int(st.Code())

		attrs := []attribute.KeyValue{
			semconv.RPCMethodKey.String(info.FullMethod),
			semconv.RPCSystemKey.String("grpc"),
			attribute.Int("rpc.grpc.status_code", statusCode),
		}

		streamCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		streamDuration.Record(ctx, float64(duration), metric.WithAttributes(attrs...))

		return err
	}
}

// ============================================================================
// HTTP Gateway Metrics (OpenTelemetry)
// ============================================================================

// createHTTPGatewayMetricsMiddleware creates Echo middleware for HTTP gateway metrics
func createHTTPGatewayMetricsMiddleware(cfg *pkgotel.Config) echo.MiddlewareFunc {
	if cfg == nil || !cfg.IsMetricsEnabled() {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	meter := cfg.GetMeter("grpc.gateway")

	// Create metrics instruments
	// Note: errors are intentionally ignored as they only occur with nil meter (checked by GetMeter)
	requestCounter, _ := meter.Int64Counter( //nolint:errcheck
		"http.server.request.count",
		metric.WithDescription("Total number of HTTP gateway requests"),
		metric.WithUnit("{request}"),
	)

	requestDuration, _ := meter.Float64Histogram( //nolint:errcheck
		"http.server.request.duration",
		metric.WithDescription("HTTP gateway request duration"),
		metric.WithUnit("ms"),
	)

	activeRequests, _ := meter.Int64UpDownCounter( //nolint:errcheck
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP gateway requests"),
		metric.WithUnit("{request}"),
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			ctx := c.Request().Context()

			// Increment active requests
			activeRequests.Add(ctx, 1)

			// Process request
			err := next(c)

			// Decrement active requests
			activeRequests.Add(ctx, -1)

			// Calculate duration
			duration := time.Since(start).Milliseconds()

			// Prepare attributes
			attrs := []attribute.KeyValue{
				semconv.HTTPRequestMethodKey.String(c.Request().Method),
				semconv.HTTPRouteKey.String(c.Path()),
				semconv.HTTPResponseStatusCodeKey.Int(c.Response().Status),
			}

			// Record metrics
			requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
			requestDuration.Record(ctx, float64(duration), metric.WithAttributes(attrs...))

			return err
		}
	}
}

// ============================================================================
// HTTP Gateway Tracing (OpenTelemetry)
// ============================================================================

// createHTTPGatewayTracingMiddleware creates Echo middleware for HTTP gateway tracing
func createHTTPGatewayTracingMiddleware(cfg *pkgotel.Config) echo.MiddlewareFunc {
	if cfg == nil || !cfg.IsTracingEnabled() {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	tracer := cfg.GetTracer("grpc.gateway")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			ctx := req.Context()

			// Start span
			ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", req.Method, c.Path()),
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.HTTPRouteKey.String(c.Path()),
					semconv.UserAgentOriginalKey.String(req.UserAgent()),
					attribute.String("http.target", req.RequestURI),
					attribute.String("http.scheme", req.URL.Scheme),
				),
			)
			defer span.End()

			// Update request context with span
			c.SetRequest(req.WithContext(ctx))

			// Process request
			err := next(c)

			// Record response status
			span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(c.Response().Status))

			if err != nil {
				span.RecordError(err)
			}

			return err
		}
	}
}

// ============================================================================
// HTTP Gateway Logging (OpenTelemetry)
// ============================================================================

// createHTTPGatewayLoggingMiddleware creates Echo middleware for HTTP gateway logging
func createHTTPGatewayLoggingMiddleware(cfg *pkgotel.Config) echo.MiddlewareFunc {
	if cfg == nil || !cfg.IsLoggingEnabled() {
		return func(next echo.HandlerFunc) echo.HandlerFunc {
			return next
		}
	}

	logger := cfg.GetLogger("grpc.gateway")

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			req := c.Request()

			// Process request
			err := next(c)

			// Calculate duration
			duration := time.Since(start)

			// Prepare log record
			severity := otellog.SeverityInfo
			if err != nil || c.Response().Status >= 500 {
				severity = otellog.SeverityError
			} else if c.Response().Status >= 400 {
				severity = otellog.SeverityWarn
			}

			// Create log attributes
			attrs := []otellog.KeyValue{
				otellog.String("http.method", req.Method),
				otellog.String("http.route", c.Path()),
				otellog.String("http.url", req.RequestURI),
				otellog.Int("http.status_code", c.Response().Status),
				otellog.Int64("http.request_size", req.ContentLength),
				otellog.Int64("http.response_size", c.Response().Size),
				otellog.Int64("http.duration_ms", duration.Milliseconds()),
			}

			if err != nil {
				attrs = append(attrs, otellog.String("error", err.Error()))
			}

			// Emit log record
			var logRecord otellog.Record
			logRecord.SetTimestamp(start)
			logRecord.SetSeverity(severity)
			logRecord.SetBody(otellog.StringValue(fmt.Sprintf("%s %s", req.Method, req.RequestURI)))
			logRecord.AddAttributes(attrs...)

			logger.Emit(req.Context(), logRecord)

			return err
		}
	}
}

// ============================================================================
// Server Metrics (OpenTelemetry)
// ============================================================================

// registerServerMetrics registers OTel observable gauges for server uptime and
// start time. These replace the legacy Prometheus-based uptime tracking.
func registerServerMetrics(cfg *pkgotel.Config) {
	if cfg == nil || !cfg.IsMetricsEnabled() {
		return
	}
	meter := cfg.GetMeter("grpc.server")
	startTime := time.Now()

	//nolint:errcheck // errors only occur with nil meter (checked by GetMeter)
	meter.Float64ObservableGauge(
		"server.uptime",
		metric.WithDescription("Server uptime in seconds"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(time.Since(startTime).Seconds())
			return nil
		}),
	)

	//nolint:errcheck // errors only occur with nil meter (checked by GetMeter)
	meter.Float64ObservableGauge(
		"server.start_time",
		metric.WithDescription("Server start time as Unix timestamp"),
		metric.WithUnit("s"),
		metric.WithFloat64Callback(func(_ context.Context, o metric.Float64Observer) error {
			o.Observe(float64(startTime.Unix()))
			return nil
		}),
	)
}
