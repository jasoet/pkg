// Package server provides a lifecycle-managed HTTP server built on Echo,
// with health endpoints, graceful shutdown, and optional OTel integration.
package server

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/jasoet/pkg/v3/otel"
)

type (
	// Operation is called after the Echo instance is configured but before it starts listening.
	Operation func(e *echo.Echo)
	// Shutdown is called during graceful shutdown before the Echo instance is stopped.
	Shutdown func(e *echo.Echo)
	// EchoConfigurer is called during setup to customize the Echo instance (add routes, middleware, etc.).
	EchoConfigurer func(e *echo.Echo)
)

// Config holds the HTTP server configuration.
type Config struct {
	// Port specifies the listen port. Use 0 for OS-assigned ephemeral port.
	Port int `yaml:"port" mapstructure:"port"`

	// Operation is called synchronously before the server starts listening. Panics in Operation will propagate to the caller of Start.
	Operation Operation

	Shutdown Shutdown

	Middleware []echo.MiddlewareFunc

	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" mapstructure:"shutdownTimeout"`

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

// WithEchoConfigurer sets a callback that customizes the Echo instance.
func WithEchoConfigurer(ec EchoConfigurer) Option {
	return func(c *Config) { c.EchoConfigurer = ec }
}

// WithOTelConfig sets the OpenTelemetry configuration.
func WithOTelConfig(cfg *otel.Config) Option {
	return func(c *Config) { c.OTelConfig = cfg }
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

// Server is a lifecycle-managed HTTP server with programmatic Start/Shutdown.
// Create one with New, then call Start (blocking) and Shutdown from another
// goroutine to stop it gracefully.
type Server struct {
	config       Config
	echo         *echo.Echo
	mu           sync.Mutex
	listener     net.Listener
	running      bool
	stopped      bool
	shutdownOnce sync.Once
	shutdownErr  error
}

// New creates a Server from functional options. It validates the configuration
// (port must be 0-65535) and prepares the Echo instance, but does not bind or
// serve — call Start for that.
func New(opts ...Option) (*Server, error) {
	cfg := NewConfig(opts...)
	if cfg.Port < 0 || cfg.Port > 65535 {
		return nil, fmt.Errorf("invalid port: %d (must be 0-65535)", cfg.Port)
	}
	return &Server{
		config: cfg,
		echo:   setupEcho(cfg),
	}, nil
}

// Echo returns the underlying Echo instance so callers can register routes
// or adjust settings before Start.
func (s *Server) Echo() *echo.Echo {
	return s.echo
}

// Addr returns the bound listener address (e.g. "[::]:8080"), or an empty
// string if the server is not listening yet. With Port 0 this is how callers
// discover the OS-assigned port once Start has bound the listener.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Start binds the listener, runs the Operation callback, and serves HTTP,
// blocking until Shutdown is called or serving fails. It returns nil on a
// clean Shutdown (http.ErrServerClosed is filtered out). Calling Start while
// the server is already running returns an error immediately. A stopped
// Server cannot be restarted — create a new one with New.
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("server is already running")
	}
	if s.stopped {
		s.mu.Unlock()
		return errors.New("server cannot be restarted; create a new one with New")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.listener = nil
		s.mu.Unlock()
	}()

	if s.config.Operation != nil {
		s.config.Operation(s.echo)
	}

	// Logger uses context.Background() intentionally: server lifecycle logs are not tied to any request context.
	logger := otel.NewLogHelper(context.Background(), s.config.OTelConfig, "github.com/jasoet/pkg/v3/server", "Server.Start")

	// Use a real listener to detect bind errors immediately instead of a racy timer.
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", fmt.Sprintf(":%v", s.config.Port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", s.config.Port, err)
	}
	s.mu.Lock()
	s.listener = ln
	s.mu.Unlock()
	s.echo.Listener = ln

	logger.Info("Starting server", otel.F("address", ln.Addr().String()))

	if err := s.echo.Start(""); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// Shutdown gracefully stops the server. It invokes the Shutdown callback and
// then drains the Echo server, honoring ShutdownTimeout (applied on top of the
// caller's context, whichever deadline is earlier). Start returns nil once the
// shutdown completes. Shutdown is idempotent: the callback runs exactly once.
func (s *Server) Shutdown(ctx context.Context) error {
	s.shutdownOnce.Do(func() {
		s.mu.Lock()
		s.stopped = true
		s.mu.Unlock()

		// Logger uses context.Background() intentionally: server lifecycle logs are not tied to any request context.
		logger := otel.NewLogHelper(context.Background(), s.config.OTelConfig, "github.com/jasoet/pkg/v3/server", "Server.Shutdown")
		logger.Info("Gracefully shutting down server")

		ctx, cancel := context.WithTimeout(ctx, s.config.ShutdownTimeout)
		defer cancel()

		if s.config.Shutdown != nil {
			s.config.Shutdown(s.echo)
		}

		s.shutdownErr = s.echo.Shutdown(ctx)
	})
	return s.shutdownErr
}

// setupEcho configures the Echo instance with middleware and health routes.
func setupEcho(config Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Set HTTP timeouts to prevent slow-client and resource exhaustion attacks
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 30 * time.Second
	e.Server.WriteTimeout = 30 * time.Second
	e.Server.IdleTimeout = 120 * time.Second

	// Enforce a default body size limit to prevent request body attacks
	e.Use(middleware.BodyLimit("4M"))

	// Auto-install OTel request instrumentation when configured, before user middleware
	if config.OTelConfig != nil {
		if config.OTelConfig.IsTracingEnabled() {
			e.Use(otelTracingMiddleware(config.OTelConfig))
		}
		if config.OTelConfig.IsMetricsEnabled() {
			e.Use(otelMetricsMiddleware(config.OTelConfig))
		}
	}

	// Add custom middleware
	for _, m := range config.Middleware {
		e.Use(m)
	}

	// Register health-check routes (no generic "/" handler — library callers add their own routes).
	// These routes are registered AFTER the user middleware above, so user middleware (including
	// auth) applies to them. Callers that need unauthenticated Kubernetes probes must not register
	// global auth middleware, or must exempt these paths themselves.
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
