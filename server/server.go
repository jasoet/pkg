package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Operation func(e *echo.Echo)
type Shutdown func(e *echo.Echo)

// Config ServerConfig contains configuration options for the server
type Config struct {
	// Port to run the server on
	Port int

	// Operation function to execute during server execution
	Operation Operation

	// Shutdown function to execute before server shutdown
	Shutdown Shutdown

	// Middleware functions to add to the server
	Middleware []echo.MiddlewareFunc

	// EnableMetrics determines whether to enable Prometheus metrics
	EnableMetrics bool

	// MetricsPath is the path to expose Prometheus metrics
	MetricsPath string

	// MetricsSubsystem is the subsystem name for Prometheus metrics
	MetricsSubsystem string

	// ShutdownTimeout is the timeout for graceful shutdown
	ShutdownTimeout time.Duration
}

// DefaultConfig returns a default server configuration
func DefaultConfig(port int, operation Operation, shutdown Shutdown) Config {
	return Config{
		Port:             port,
		Operation:        operation,
		Shutdown:         shutdown,
		EnableMetrics:    true,
		MetricsPath:      "/metrics",
		MetricsSubsystem: "echo",
		ShutdownTimeout:  10 * time.Second,
	}
}

// Server represents an HTTP server instance
type Server struct {
	echo   *echo.Echo
	config Config
}

// setupEcho configures an Echo instance based on the provided configuration
func setupEcho(config Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

	// Add request logger middleware
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

	// Setup Prometheus metrics if enabled
	if config.EnableMetrics {
		e.GET(config.MetricsPath, echoprometheus.NewHandler())
		e.Use(echoprometheus.NewMiddleware(config.MetricsSubsystem))
	}

	// Add custom middleware
	for _, m := range config.Middleware {
		e.Use(m)
	}

	e.GET("/", func(c echo.Context) error {
		return c.String(http.StatusOK, "Home")
	})

	// Add health check endpoints
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "UP"})
	})

	e.GET("/health/ready", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "READY"})
	})

	e.GET("/health/live", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "ALIVE"})
	})

	return e
}

// NewServer creates a new server instance with the given configuration
func NewServer(config Config) *Server {
	e := setupEcho(config)
	return &Server{
		echo:   e,
		config: config,
	}
}

// Start starts the server and returns immediately
func (s *Server) Start() {
	// Run the operation function
	go s.config.Operation(s.echo)

	// Start the server
	go func() {
		log.Info().Msgf("Starting server, on port %d", s.config.Port)
		if err := s.echo.Start(fmt.Sprintf(":%v", s.config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()
}

// Stop gracefully stops the server
func (s *Server) Stop() error {
	log.Info().Msg("gracefully shutting down")

	// Run the shutdown function
	s.config.Shutdown(s.echo)

	// Shutdown the server with timeout
	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	return s.echo.Shutdown(ctx)
}

// StartWithConfig starts the server with the given configuration and blocks until interrupted
func StartWithConfig(config Config) {
	server := NewServer(config)

	// Setup signal handling
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	server.Start()

	// Wait for termination signal
	<-ctx.Done()

	// Stop the server
	if err := server.Stop(); err != nil {
		log.Fatal().Err(err).Msg("failed to shutdown server")
	}
}

// Start starts the server with the given port, operation, and shutdown functions
// Optional middleware can be passed using variadic parameters
func Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) {
	config := DefaultConfig(port, operation, shutdown)
	config.Middleware = middleware
	StartWithConfig(config)
}
