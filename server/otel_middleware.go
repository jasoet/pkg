package server

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
	"go.opentelemetry.io/otel/trace"

	pkgotel "github.com/jasoet/pkg/v3/otel"
)

// otelScope is the instrumentation scope name for server tracing and metrics.
const otelScope = "http.server"

// serverPropagator extracts inbound W3C trace context and baggage from request
// headers so server spans continue the caller's trace instead of starting a new
// root. It mirrors the propagator used by the grpc package's interceptors and
// the rest client's outbound injection, so a rest -> server hop stays a single
// trace end to end.
var serverPropagator = propagation.NewCompositeTextMapPropagator(
	propagation.TraceContext{},
	propagation.Baggage{},
)

// unmatchedRouteName is used as the span name for requests that do not match any
// registered route (e.g. 404s), where echo.Context.Path() is empty.
const unmatchedRouteName = "unmatched"

// sensitiveQueryParams lists query-parameter names whose values are redacted
// from the url.full span attribute so secrets (tokens, keys, passwords) do not
// leak into traces. Matching is case-insensitive.
var sensitiveQueryParams = map[string]struct{}{
	"access_token":  {},
	"refresh_token": {},
	"id_token":      {},
	"token":         {},
	"api_key":       {},
	"apikey":        {},
	"key":           {},
	"secret":        {},
	"client_secret": {},
	"password":      {},
	"passwd":        {},
	"pwd":           {},
	"authorization": {},
	"auth":          {},
	"sig":           {},
	"signature":     {},
}

// resolveStatus returns the HTTP status code that will actually be sent for the
// request. After next(c) returns on the error path, Echo's HTTPErrorHandler has
// not run yet (it runs later, in Echo.ServeHTTP), so c.Response().Status is
// still the default 200. The true status is therefore derived from the returned
// error: an *echo.HTTPError carries the intended code, any other error maps to
// 500. When the response has already been committed (a handler that wrote a
// status and also returned an error), the committed status is authoritative.
func resolveStatus(c echo.Context, err error) int {
	if err == nil || c.Response().Committed {
		return c.Response().Status
	}
	var he *echo.HTTPError
	if errors.As(err, &he) {
		return he.Code
	}
	return http.StatusInternalServerError
}

// redactedURLFull builds the url.full attribute value, replacing the values of
// sensitive query parameters with "REDACTED" so secrets are not persisted in
// traces. Non-sensitive parameters and their ordering are preserved unchanged.
func redactedURLFull(scheme string, req *http.Request) string {
	u := req.URL
	rawQuery := u.RawQuery
	if rawQuery != "" {
		if q := u.Query(); len(q) > 0 {
			redacted := false
			for key, values := range q {
				if _, ok := sensitiveQueryParams[strings.ToLower(key)]; !ok {
					continue
				}
				for i := range values {
					values[i] = "REDACTED"
				}
				redacted = true
			}
			if redacted {
				rawQuery = q.Encode()
			}
		}
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	target := path
	if rawQuery != "" {
		target += "?" + rawQuery
	}
	return scheme + "://" + req.Host + target
}

// otelTracingMiddleware creates Echo middleware that emits one server span per
// request. The span is provisionally named by method and renamed to
// "{method} {route}" with the http.route attribute in a deferred block, so
// unmatched routes (404s) are covered too. The response status code is derived
// after the handler chain returns (see resolveStatus) so that error responses
// and 404s record their real status and set an Error span status for 5xx.
func otelTracingMiddleware(cfg *pkgotel.Config) echo.MiddlewareFunc {
	tracer := cfg.GetTracer(otelScope)

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) (err error) {
			req := c.Request()

			scheme := "http"
			if req.TLS != nil {
				scheme = "https"
			}
			fullURL := redactedURLFull(scheme, req)

			// Join the caller's trace when the request carries W3C trace
			// context; with no inbound headers this is a no-op and the span
			// below becomes a root as before.
			parentCtx := serverPropagator.Extract(req.Context(), propagation.HeaderCarrier(req.Header))

			ctx, span := tracer.Start(parentCtx, req.Method,
				trace.WithSpanKind(trace.SpanKindServer),
				trace.WithAttributes(
					semconv.HTTPRequestMethodKey.String(req.Method),
					semconv.URLFullKey.String(fullURL),
				),
			)
			defer func() {
				status := resolveStatus(c, err)

				route := c.Path()
				if route != "" {
					span.SetName(req.Method + " " + route)
					span.SetAttributes(semconv.HTTPRouteKey.String(route))
				} else {
					// Unmatched route (e.g. 404): Path() is empty. Avoid a
					// dangling "GET " span name and an empty http.route.
					span.SetName(req.Method + " " + unmatchedRouteName)
				}
				span.SetAttributes(semconv.HTTPResponseStatusCodeKey.Int(status))

				if err != nil {
					span.RecordError(err)
				}
				// Per HTTP semantic conventions, only 5xx marks a server span as
				// an error; 4xx is a client fault, not a server error.
				if status >= http.StatusInternalServerError {
					span.SetStatus(codes.Error, http.StatusText(status))
				}
				span.End()
			}()

			c.SetRequest(req.WithContext(ctx))

			err = next(c)
			return err
		}
	}
}

// otelMetricsMiddleware creates Echo middleware that records a request counter
// and duration histogram per request, attributed by method and status code. The
// status code is derived after the handler chain returns (see resolveStatus) so
// error responses and 404s are attributed with their real status rather than a
// premature 200.
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

			status := resolveStatus(c, err)
			attrs := metric.WithAttributes(
				semconv.HTTPRequestMethodKey.String(c.Request().Method),
				semconv.HTTPResponseStatusCodeKey.Int(status),
			)
			requestCounter.Add(ctx, 1, attrs)
			// Record fractional milliseconds; truncating to whole Milliseconds()
			// would floor every sub-millisecond handler to 0.
			requestDuration.Record(ctx, float64(time.Since(start))/float64(time.Millisecond), attrs)

			return err
		}
	}
}
