package temporal

import (
	"github.com/rs/zerolog/log"
)

type Config struct {
	HostPort             string `yaml:"hostPort" mapstructure:"hostPort"`
	Namespace            string `yaml:"namespace" mapstructure:"namespace"`
	MetricsListenAddress string `yaml:"metricsListenAddress" mapstructure:"metricsListenAddress"`
}

func DefaultConfig() *Config {
	logger := log.With().Str("function", "temporal.DefaultConfig").Logger()

	config := &Config{
		HostPort:             "localhost:7233",
		Namespace:            "default",
		MetricsListenAddress: "0.0.0.0:9090",
	}

	logger.Debug().
		Str("hostPort", config.HostPort).
		Str("namespace", config.Namespace).
		Str("metricsAddress", config.MetricsListenAddress).
		Msg("Created default Temporal configuration")

	return config
}
