package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	"github.com/labstack/echo/v4"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel/attribute"
	otellog "go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/metric"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

type (
	Operation      func(e *echo.Echo)
	Shutdown       func(e *echo.Echo)
	EchoConfigurer func(e *echo.Echo)
)

// Config holds the HTTP server configuration.
// BREAKING CHANGE from v1: Prometheus metrics replaced with OpenTelemetry.
// Removed: EnableMetrics, MetricsPath, MetricsSubsystem
// Added: OTelConfig
type Config struct {
	Port int

	Operation Operation

	Shutdown Shutdown

	Middleware []echo.MiddlewareFunc

	ShutdownTimeout time.Duration

	EchoConfigurer EchoConfigurer

	// OpenTelemetry configuration (optional - nil disables telemetry)
	// Use otel.NewConfig("service-name") to get default logging
	// Replaces EnableMetrics, MetricsPath, MetricsSubsystem from v1
	OTelConfig *otel.Config
}

// DefaultConfig returns a default server configuration
func DefaultConfig(port int, operation Operation, shutdown Shutdown) Config {
	return Config{
		Port:            port,
		Operation:       operation,
		Shutdown:        shutdown,
		ShutdownTimeout: 10 * time.Second,
		OTelConfig:      nil, // Telemetry disabled by default
	}
}

type httpServer struct {
	echo   *echo.Echo
	config Config
}

// setupEcho configures the Echo instance with middleware and routes
func setupEcho(config Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// OpenTelemetry logging middleware
	// Note: Uses default stdout LoggerProvider from otel.NewConfig() if OTelConfig provided
	// If OTelConfig is nil, logging is disabled
	if config.OTelConfig != nil && config.OTelConfig.IsLoggingEnabled() {
		e.Use(createLoggingMiddleware(config.OTelConfig))
	}

	// OpenTelemetry instrumentation (if configured)
	if config.OTelConfig != nil {
		// Add tracing middleware
		if config.OTelConfig.IsTracingEnabled() {
			serviceName := config.OTelConfig.ServiceName
			if serviceName == "" {
				serviceName = "http-server"
			}

			e.Use(otelecho.Middleware(serviceName,
				otelecho.WithTracerProvider(config.OTelConfig.TracerProvider),
			))
		}

		// Add metrics middleware
		if config.OTelConfig.IsMetricsEnabled() {
			e.Use(createMetricsMiddleware(config.OTelConfig))
		}
	}

	// Add custom middleware
	for _, m := range config.Middleware {
		e.Use(m)
	}

	// Register standard routes
	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Home")
	})

	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "UP"})
	})

	e.GET("/health/ready", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "READY"})
	})

	e.GET("/health/live", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ALIVE"})
	})

	// Apply custom Echo configuration if provided
	if config.EchoConfigurer != nil {
		config.EchoConfigurer(e)
	}

	return e
}

// createLoggingMiddleware creates Echo middleware that logs HTTP requests via OpenTelemetry
func createLoggingMiddleware(cfg *otel.Config) echo.MiddlewareFunc {
	logger := cfg.GetLogger("server.http")

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

// createMetricsMiddleware creates Echo middleware that records HTTP metrics via OpenTelemetry
func createMetricsMiddleware(cfg *otel.Config) echo.MiddlewareFunc {
	meter := cfg.GetMeter("server")

	// Create metrics instruments
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

	activeRequests, _ := meter.Int64UpDownCounter( //nolint:errcheck
		"http.server.active_requests",
		metric.WithDescription("Number of active HTTP requests"),
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

func newHttpServer(config Config) *httpServer {
	e := setupEcho(config)
	return &httpServer{
		echo:   e,
		config: config,
	}
}

func (s *httpServer) start() {
	s.config.Operation(s.echo)

	go func() {
		fmt.Printf("Starting server on port %d\n", s.config.Port)
		if err := s.echo.Start(fmt.Sprintf(":%v", s.config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintf(os.Stderr, "Failed to start server: %v\n", err)
			os.Exit(1)
		}
	}()
}

func (s *httpServer) stop() error {
	fmt.Println("Gracefully shutting down server...")

	s.config.Shutdown(s.echo)

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	return s.echo.Shutdown(ctx)
}

// StartWithConfig starts the HTTP server with the given configuration
func StartWithConfig(config Config) {
	server := newHttpServer(config)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server.start()

	<-ctx.Done()

	if err := server.stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to shutdown server: %v\n", err)
		os.Exit(1)
	}
}

// Start starts the HTTP server with simplified configuration
func Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) {
	config := DefaultConfig(port, operation, shutdown)
	config.Middleware = middleware
	StartWithConfig(config)
}
