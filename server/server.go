package server

import (
	"context"
	"errors"
	"fmt"
	"github.com/labstack/echo-contrib/echoprometheus"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests by status code, method, and path",
		},
		[]string{"status", "method", "path"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request latencies in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)

	httpResponseSize = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_response_size_bytes",
			Help:    "HTTP response sizes in bytes",
			Buckets: prometheus.ExponentialBuckets(100, 10, 8),
		},
		[]string{"method", "path"},
	)
)

type Operation func(e *echo.Echo)
type Shutdown func(e *echo.Echo)

// metricsMiddleware creates a middleware that collects HTTP metrics
func metricsMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			req := c.Request()
			res := c.Response()
			start := time.Now()

			err := next(c)

			latency := time.Since(start)
			status := res.Status

			httpRequestsTotal.WithLabelValues(
				fmt.Sprintf("%d", status),
				req.Method,
				req.URL.Path,
			).Inc()

			httpRequestDuration.WithLabelValues(
				req.Method,
				req.URL.Path,
			).Observe(latency.Seconds())

			httpResponseSize.WithLabelValues(
				req.Method,
				req.URL.Path,
			).Observe(float64(res.Size))

			return err
		}
	}
}

func Start(port int, operation Operation, shutdown Shutdown) {
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

	e.GET("/metrics", echoprometheus.NewHandler())

	// Register all metrics
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	prometheus.MustRegister(httpResponseSize)

	// Add metrics middleware
	e.Use(metricsMiddleware())

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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go operation(e)

	go func() {
		log.Info().Msgf("Starting server, on port %d", port)
		if err := e.Start(fmt.Sprintf(":%v", port)); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal().Err(err).Msg("failed to start server")
		}
	}()

	<-ctx.Done()

	log.Info().Msg("gracefully shutting down")
	shutdown(e)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("failed to shutdown server")
	}

}
