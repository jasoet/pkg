package temporal

import (
	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog/log"
	"github.com/uber-go/tally/v4"
	"github.com/uber-go/tally/v4/prometheus"
	"go.temporal.io/sdk/client"
	sdktally "go.temporal.io/sdk/contrib/tally"
	"time"
)

func NewClientWithMetrics(config *Config, metricsEnabled bool) (client.Client, error) {
	logger := log.With().Str("function", "temporal.NewClient").Logger()
	logger.Debug().
		Str("hostPort", config.HostPort).
		Str("namespace", config.Namespace).
		Str("metricsAddress", config.MetricsListenAddress).
		Msg("Creating new Temporal client")

	clientOption := client.Options{
		HostPort:  config.HostPort,
		Namespace: config.Namespace,
		Logger:    NewZerologAdapter(logger),
	}

	if !metricsEnabled {
		scope, err := newPrometheusScope(prometheus.Configuration{
			ListenAddress: config.MetricsListenAddress,
			TimerType:     "histogram",
		})

		if err != nil {
			logger.Error().Err(err).Msg("Failed to create Prometheus scope")
			return nil, err
		}

		clientOption.MetricsHandler = sdktally.NewMetricsHandler(scope)
	}

	logger.Debug().Msg("Connecting to Temporal server")
	c, err := client.Dial(clientOption)

	if err != nil {
		logger.Error().Err(err).Msg("Failed to connect to Temporal server")
		return nil, err
	}

	logger.Debug().Msg("Successfully connected to Temporal server")
	return c, nil
}

func NewClient(config *Config) (client.Client, error) {
	return NewClientWithMetrics(config, true)
}

func newPrometheusScope(c prometheus.Configuration) (tally.Scope, error) {
	logger := log.With().Str("function", "temporal.newPrometheusScope").Logger()
	logger.Debug().Str("listenAddress", c.ListenAddress).Msg("Creating Prometheus reporter")

	reporter, err := c.NewReporter(
		prometheus.ConfigurationOptions{
			Registry: prom.NewRegistry(),
			OnError: func(err error) {
				log.Error().Err(err).Msg("Error in Prometheus reporter")
			},
		},
	)

	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Prometheus reporter")
		return nil, err
	}

	logger.Debug().Msg("Configuring tally scope options")
	scopeOpts := tally.ScopeOptions{
		CachedReporter:  reporter,
		Separator:       prometheus.DefaultSeparator,
		SanitizeOptions: &sdktally.PrometheusSanitizeOptions,
	}

	logger.Debug().Msg("Creating new root scope")
	scope, _ := tally.NewRootScope(scopeOpts, time.Second)
	scope = sdktally.NewPrometheusNamingScope(scope)

	logger.Debug().Msg("Prometheus scope created successfully")
	return scope, nil
}
