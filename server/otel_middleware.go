package server

import (
	"fmt"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// otelScope is the instrumentation scope name for server tracing and metrics.
const otelScope = "http.server"

// otelTracingMiddleware creates Echo middleware that emits one server span per
// request. The span is provisionally named by method and renamed to
// "{method} {route}" with the http.route attribute in a deferred block, so
// unmatched routes (404s) are covered too.
func otelTracingMiddleware(cfg *pkgotel.Config) echo.MiddlewareFunc {
	tracer := cfg.GetTracer(otelScope)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()

			scheme := "http"
			if req.TLS != nil {
				scheme = "https"
			}
			fullURL := fmt.Sprintf("%s://%s%s", scheme, req.Host, req.URL.RequestURI())

			ctx, span := tracer.Start(req.Context(), req.Method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.URLFullKey.String(fullURL),
				),
			)
			defer func() {
				route := c.Path()
				span.SetName(fmt.Sprintf("%s %s", req.Method, route))
				span.SetAttributes(
					semconv.HTTPRouteKey.String(route),
					semconv.HTTPResponseStatusCodeKey.Int(c.Response().Status),
				)
				span.End()
			}()

			c.SetRequest(req.WithContext(ctx))

			err := next(c)
			if err != nil {
				span.RecordError(err)
			}
			return err
		}
	}
}

// otelMetricsMiddleware creates Echo middleware that records a request counter
// and duration histogram per request, attributed by method and status code.
func otelMetricsMiddleware(cfg *pkgotel.Config) echo.MiddlewareFunc {
	meter := cfg.GetMeter(otelScope)

	// Note: errors are intentionally ignored as they only occur with nil meter (checked by GetMeter)
	requestCounter, _ := meter.Int64Counter( //nolint:errcheck
		"http.server.request.count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)

	requestDuration, _ := meter.Float64Histogram( //nolint:errcheck
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			ctx := c.Request().Context()

			err := next(c)

			attrs := metric.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request().Method),
				semconv.HTTPResponseStatusCodeKey.Int(c.Response().Status),
			)
			requestCounter.Add(ctx, 1, attrs)
			requestDuration.Record(ctx, float64(time.Since(start).Milliseconds()), attrs)

			return err
		}
	}
}
