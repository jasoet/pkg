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

	"github.com/labstack/echo/v4"
	"github.com/rs/zerolog/log"
)

type (
	Operation      func(e *echo.Echo)
	Shutdown       func(e *echo.Echo)
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
}

// DefaultConfig returns a default server configuration
func DefaultConfig(port int, operation Operation, shutdown Shutdown) Config {
	return Config{
		Port:            port,
		Operation:       operation,
		Shutdown:        shutdown,
		ShutdownTimeout: 10 * time.Second,
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

func newHttpServer(config Config) *httpServer {
	e := setupEcho(config)
	return &httpServer{
		echo:   e,
		config: config,
	}
}

func (s *httpServer) start() error {
	s.config.Operation(s.echo)

	errCh := make(chan error, 1)
	go func() {
		log.Info().Int("port", s.config.Port).Msg("Starting server")
		if err := s.echo.Start(fmt.Sprintf(":%v", s.config.Port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	// Give the server a moment to fail on bind errors
	select {
	case err := <-errCh:
		return fmt.Errorf("failed to start server: %w", err)
	case <-time.After(100 * time.Millisecond):
		return nil
	}
}

func (s *httpServer) stop() error {
	log.Info().Msg("Gracefully shutting down server")

	s.config.Shutdown(s.echo)

	ctx, cancel := context.WithTimeout(context.Background(), s.config.ShutdownTimeout)
	defer cancel()

	return s.echo.Shutdown(ctx)
}

// StartWithConfig starts the HTTP server with the given configuration
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

// Start starts the HTTP server with simplified configuration
func Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) error {
	config := DefaultConfig(port, operation, shutdown)
	config.Middleware = middleware
	return StartWithConfig(config)
}
