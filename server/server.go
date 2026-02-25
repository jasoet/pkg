// Package server provides a lifecycle-managed HTTP server built on Echo,
// with health endpoints, graceful shutdown, and optional OTel integration.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	"github.com/labstack/echo/v4"
)

type (
	// Operation is called after the Echo instance is configured but before it starts listening.
	Operation func(e *echo.Echo)
	// Shutdown is called during graceful shutdown before the Echo instance is stopped.
	Shutdown func(e *echo.Echo)
	// EchoConfigurer is called during setup to customise the Echo instance (add routes, middleware, etc.).
	EchoConfigurer func(e *echo.Echo)
)

// Config holds the HTTP server configuration.
type Config struct {
	Port int

	Operation Operation

	Shutdown Shutdown

	Middleware []echo.MiddlewareFunc

	ShutdownTimeout time.Duration

	EchoConfigurer EchoConfigurer

	OTelConfig *otel.Config `yaml:"-" mapstructure:"-"`
}

// Option configures a Config during construction.
type Option func(*Config)

// WithPort sets the server listen port.
func WithPort(port int) Option {
	return func(c *Config) { c.Port = port }
}

// WithOperation sets the Operation callback.
func WithOperation(op Operation) Option {
	return func(c *Config) { c.Operation = op }
}

// WithShutdown sets the Shutdown callback.
func WithShutdown(s Shutdown) Option {
	return func(c *Config) { c.Shutdown = s }
}

// WithMiddleware appends Echo middleware to the chain.
func WithMiddleware(m ...echo.MiddlewareFunc) Option {
	return func(c *Config) { c.Middleware = append(c.Middleware, m...) }
}

// WithShutdownTimeout sets the graceful-shutdown deadline.
func WithShutdownTimeout(d time.Duration) Option {
	return func(c *Config) { c.ShutdownTimeout = d }
}

// WithEchoConfigurer sets a callback that customises the Echo instance.
func WithEchoConfigurer(ec EchoConfigurer) Option {
	return func(c *Config) { c.EchoConfigurer = ec }
}

// WithOTelConfig sets the OpenTelemetry configuration.
func WithOTelConfig(cfg *otel.Config) Option {
	return func(c *Config) { c.OTelConfig = cfg }
}

// DefaultConfig returns a default server configuration.
func DefaultConfig(port int, operation Operation, shutdown Shutdown) Config {
	return Config{
		Port:            port,
		Operation:       operation,
		Shutdown:        shutdown,
		ShutdownTimeout: 10 * time.Second,
	}
}

// NewConfig creates a Config using functional options with sensible defaults.
func NewConfig(opts ...Option) Config {
	cfg := Config{
		ShutdownTimeout: 10 * time.Second,
	}
	for _, opt := range opts {
		opt(&cfg)
	}
	return cfg
}

type httpServer struct {
	echo   *echo.Echo
	config Config
}

// setupEcho configures the Echo instance with middleware and health routes.
func setupEcho(config Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Add custom middleware
	for _, m := range config.Middleware {
		e.Use(m)
	}

	// Register health-check routes (no generic "/" handler — library callers add their own routes)
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

func newHttpServer(config Config) *httpServer {
	e := setupEcho(config)
	return &httpServer{
		echo:   e,
		config: config,
	}
}

func (s *httpServer) start() error {
	if s.config.Operation != nil {
		s.config.Operation(s.echo)
	}

	logger := otel.NewLogHelper(context.Background(), s.config.OTelConfig, "github.com/jasoet/pkg/v2/server", "httpServer.start")

	// Use a real listener to detect bind errors immediately instead of a racy timer.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%v", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.Port, err)
	}
	s.echo.Listener = ln

	logger.Info("Starting server", otel.F("address", ln.Addr().String()))

	go func() {
		if err := s.echo.Start(""); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(err, "Server error")
		}
	}()

	return nil
}

func (s *httpServer) stop() error {
	logger := otel.NewLogHelper(context.Background(), s.config.OTelConfig, "github.com/jasoet/pkg/v2/server", "httpServer.stop")
	logger.Info("Gracefully shutting down server")

	if s.config.Shutdown != nil {
		s.config.Shutdown(s.echo)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	return s.echo.Shutdown(ctx)
}

// StartWithConfig starts the HTTP server with the given configuration and
// blocks until an OS interrupt signal is received, then shuts down gracefully.
func StartWithConfig(config Config) error {
	server := newHttpServer(config)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := server.start(); err != nil {
		return err
	}

	<-ctx.Done()

	return server.stop()
}

// Start starts the HTTP server with simplified configuration.
func Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) error {
	config := DefaultConfig(port, operation, shutdown)
	config.Middleware = middleware
	return StartWithConfig(config)
}
