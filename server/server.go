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

type Config struct {
	Port int

	Operation Operation

	Shutdown Shutdown

	Middleware []echo.MiddlewareFunc

	EnableMetrics bool

	MetricsPath string

	MetricsSubsystem string

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

type httpServer struct {
	echo   *echo.Echo
	config Config
}

func setupEcho(config Config) *echo.Echo {
	e := echo.New()
	e.HideBanner = true

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

	if config.EnableMetrics {
		e.GET(config.MetricsPath, echoprometheus.NewHandler())
		e.Use(echoprometheus.NewMiddleware(config.MetricsSubsystem))
	}

	for _, m := range config.Middleware {
		e.Use(m)
	}

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

func Start(port int, operation Operation, shutdown Shutdown, middleware ...echo.MiddlewareFunc) {
	config := DefaultConfig(port, operation, shutdown)
	config.Middleware = middleware
	StartWithConfig(config)
}
