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

	"github.com/jasoet/pkg/otel"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	"go.opentelemetry.io/otel/attribute"
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

	// Standard request logging middleware (zerolog)
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogURI:    true,
		LogStatus: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			if v.Error == nil {
				log.Info().
					Str("URI", v.URI).
					Int("status", v.Status).
					Msg("request")
			} else {
				log.Error().Err(v.Error).Msg("request error")
			}

			return nil
		},
	}))

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

			log.Info().Str("service", serviceName).Msg("OpenTelemetry tracing enabled")
		}

		// Add metrics middleware
		if config.OTelConfig.IsMetricsEnabled() {
			e.Use(createMetricsMiddleware(config.OTelConfig))
			log.Info().Msg("OpenTelemetry metrics enabled")
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

// createMetricsMiddleware creates Echo middleware that records HTTP metrics via OpenTelemetry
func createMetricsMiddleware(cfg *otel.Config) echo.MiddlewareFunc {
	meter := cfg.GetMeter("server")

	// Create metrics instruments
	requestCounter, _ := meter.Int64Counter(
		"http.server.request.count",
		metric.WithDescription("Total number of HTTP requests"),
		metric.WithUnit("{request}"),
	)

	requestDuration, _ := meter.Float64Histogram(
		"http.server.request.duration",
		metric.WithDescription("HTTP request duration"),
		metric.WithUnit("ms"),
	)

	activeRequests, _ := meter.Int64UpDownCounter(
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
	go s.config.Operation(s.echo)

	go func() {
		log.Info().Msgf("Starting server, on port %d", s.config.Port)
		if err := s.echo.Start(fmt.Sprintf(":%v", s.config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()
}

func (s *httpServer) stop() error {
	log.Info().Msg("gracefully shutting down")

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
		log.Fatal().Err(err).Msg("failed to shutdown server")
	}
}

// Start starts the HTTP server with simplified configuration
func Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) {
	config := DefaultConfig(port, operation, shutdown)
	config.Middleware = middleware
	StartWithConfig(config)
}
