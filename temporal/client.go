package temporal

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
)

func NewClientWithMetrics(config *Config, metricsEnabled bool) (client.Client, io.Closer, error) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.NewClientWithMetrics")

	logger.Debug("Creating new Temporal client",
		otel.F("hostPort", config.HostPort),
		otel.F("namespace", config.Namespace),
		otel.F("metricsAddress", config.MetricsListenAddress))

	// Create a zerolog logger for Temporal SDK's logger adapter
	zerologLogger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339}).
		With().
		Timestamp().
		Str("service", "temporal").
		Logger()

	clientOption := client.Options{
		HostPort:  config.HostPort,
		Namespace: config.Namespace,
		Logger:    NewZerologAdapter(zerologLogger),
	}

	var metricsCloser io.Closer
	if metricsEnabled {
		scope, closer, err := newPrometheusScope(prometheus.Configuration{
			ListenAddress: config.MetricsListenAddress,
			TimerType:     "histogram",
		})
		if err != nil {
			logger.Error(err, "Failed to create Prometheus scope")
			return nil, nil, err
		}
		metricsCloser = closer

		clientOption.MetricsHandler = sdktally.NewMetricsHandler(scope)
	}

	logger.Debug("Connecting to Temporal server")
	c, err := client.Dial(clientOption)
	if err != nil {
		logger.Error(err, "Failed to connect to Temporal server")
		if metricsCloser != nil {
			metricsCloser.Close()
		}
		return nil, nil, err
	}

	logger.Debug("Successfully connected to Temporal server")
	return c, metricsCloser, nil
}

func NewClient(config *Config) (client.Client, io.Closer, error) {
	return NewClientWithMetrics(config, true)
}

func newPrometheusScope(c prometheus.Configuration) (tally.Scope, io.Closer, error) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.newPrometheusScope")

	logger.Debug("Creating Prometheus reporter", otel.F("listenAddress", c.ListenAddress))

	reporter, err := c.NewReporter(
		prometheus.ConfigurationOptions{
			Registry: prom.NewRegistry(),
			OnError: func(err error) {
				errLogger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.prometheusReporter.OnError")
				errLogger.Error(err, "Error in Prometheus reporter")
			},
		},
	)
	if err != nil {
		logger.Error(err, "Failed to create Prometheus reporter")
		return nil, nil, err
	}

	logger.Debug("Configuring tally scope options")
	scopeOpts := tally.ScopeOptions{
		CachedReporter:  reporter,
		Separator:       prometheus.DefaultSeparator,
		SanitizeOptions: &sdktally.PrometheusSanitizeOptions,
	}

	logger.Debug("Creating new root scope")
	scope, closer := tally.NewRootScope(scopeOpts, time.Second)
	scope = sdktally.NewPrometheusNamingScope(scope)

	logger.Debug("Prometheus scope created successfully")
	return scope, closer, nil
}
