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
//
// Function-typed and other non-serializable fields carry `yaml:"-"
// mapstructure:"-"` so that decoding a YAML document (e.g. via the config
// package) that happens to contain keys like `operation:` does not fail; only
// Port, BindAddress and ShutdownTimeout are populated from configuration.
type Config struct {
	// Port specifies the listen port. Use 0 for OS-assigned ephemeral port.
	Port int `yaml:"port" mapstructure:"port"`

	// BindAddress specifies the interface address to bind to (e.g. "127.0.0.1"
	// for loopback-only). Empty binds all interfaces.
	BindAddress string `yaml:"bindAddress" mapstructure:"bindAddress"`

	// Operation is called synchronously before the server starts listening. Panics in Operation will propagate to the caller of Start.
	Operation Operation `yaml:"-" mapstructure:"-"`

	Shutdown Shutdown `yaml:"-" mapstructure:"-"`

	Middleware []echo.MiddlewareFunc `yaml:"-" mapstructure:"-"`

	ShutdownTimeout time.Duration `yaml:"shutdownTimeout" mapstructure:"shutdownTimeout"`

	EchoConfigurer EchoConfigurer `yaml:"-" mapstructure:"-"`

	OTelConfig *otel.Config `yaml:"-" mapstructure:"-"`
}

// Option configures a Config during construction.
type Option func(*Config)

// WithPort sets the server listen port.
func WithPort(port int) Option {
	return func(c *Config) { c.Port = port }
}

// WithBindAddress sets the interface address to bind to (e.g. "127.0.0.1" to
// listen on loopback only). The empty string (the default) binds all interfaces.
func WithBindAddress(addr string) Option {
	return func(c *Config) { c.BindAddress = addr }
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

// WithShutdownTimeout sets the graceful-shutdown deadline. A value of 0 (or
// negative) disables the additional deadline, so Shutdown honors only the
// caller-supplied context instead of expiring immediately.
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
// string if the server is not listening yet or has already shut down. With
// Port 0 this is how callers discover the OS-assigned port once Start has
// bound the listener.
func (s *Server) Addr() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return ""
	}
	return s.listener.Addr().String()
}

// Start runs the Operation callback first, then binds the listener and serves
// HTTP, blocking until Shutdown is called or serving fails. Because Operation
// runs before binding, Addr() returns "" inside Operation (notably with Port 0
// — the OS-assigned port is only known after binding). It returns nil on a
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
	address := fmt.Sprintf("%s:%d", s.config.BindAddress, s.config.Port)
	ln, err := (&net.ListenConfig{}).Listen(context.Background(), "tcp", address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", address, err)
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
// Calling Shutdown on a server that was never started is a no-op and returns
// nil, leaving the server free to Start later.
func (s *Server) Shutdown(ctx context.Context) error {
	s.mu.Lock()
	neverStarted := !s.running && !s.stopped
	s.mu.Unlock()
	if neverStarted {
		return nil
	}

	s.shutdownOnce.Do(func() {
		s.mu.Lock()
		s.stopped = true
		s.mu.Unlock()

		// Logger uses context.Background() intentionally: server lifecycle logs are not tied to any request context.
		logger := otel.NewLogHelper(context.Background(), s.config.OTelConfig, "github.com/jasoet/pkg/v3/server", "Server.Shutdown")
		logger.Info("Gracefully shutting down server")

		// A non-positive ShutdownTimeout would produce an already-expired
		// context and force an instant hard shutdown; treat it as "no extra
		// deadline" and honor only the caller's context instead.
		if s.config.ShutdownTimeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, s.config.ShutdownTimeout)
			defer cancel()
		}

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
	// Suppress Echo's "⇨ http server started on ..." stdout line; the bound
	// address is already emitted through the structured lifecycle log in Start.
	e.HidePort = true

	// Set HTTP timeouts to prevent slow-client and resource exhaustion attacks
	e.Server.ReadHeaderTimeout = 5 * time.Second
	e.Server.ReadTimeout = 30 * time.Second
	e.Server.WriteTimeout = 30 * time.Second
	e.Server.IdleTimeout = 120 * time.Second

	// Auto-install OTel request instrumentation FIRST (outermost) when
	// configured, so it observes everything installed below it: the 413s emitted
	// by BodyLimit and the 500s produced by Recover on a panicking handler.
	if config.OTelConfig != nil {
		if config.OTelConfig.IsTracingEnabled() {
			e.Use(otelTracingMiddleware(config.OTelConfig))
		}
		if config.OTelConfig.IsMetricsEnabled() {
			e.Use(otelMetricsMiddleware(config.OTelConfig))
		}
	}

	// Recover from panics in handlers, converting them into 500 responses so a
	// panicking handler does not drop the connection (and is observed as a 500
	// by the OTel middleware above rather than silently reported as a success).
	e.Use(middleware.Recover())

	// Enforce a default body size limit to prevent request body attacks. A body
	// larger than the limit is rejected with 413, which the OTel middleware
	// above records.
	e.Use(middleware.BodyLimit("4M"))

	// Add custom middleware
	for _, m := range config.Middleware {
		e.Use(m)
	}

	// Register health-check routes (no generic "/" handler — library callers add their own routes).
	// Echo applies global middleware to ALL routes regardless of registration order, so user
	// middleware (including auth) applies to these health routes too. Callers that need
	// unauthenticated Kubernetes probes must not register global auth middleware, or must
	// exempt these paths themselves.
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
