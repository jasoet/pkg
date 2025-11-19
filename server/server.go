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
