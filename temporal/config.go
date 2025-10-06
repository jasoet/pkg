package temporal

import (
	"context"

	"github.com/jasoet/pkg/v2/otel"
)

type Config struct {
	HostPort             string `yaml:"hostPort" mapstructure:"hostPort"`
	Namespace            string `yaml:"namespace" mapstructure:"namespace"`
	MetricsListenAddress string `yaml:"metricsListenAddress" mapstructure:"metricsListenAddress"`
}

func DefaultConfig() *Config {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/temporal", "temporal.DefaultConfig")

	config := &Config{
		HostPort:             "localhost:7233",
		Namespace:            "default",
		MetricsListenAddress: "0.0.0.0:9090",
	}

	logger.Debug("Created default Temporal configuration",
		otel.F("hostPort", config.HostPort),
		otel.F("namespace", config.Namespace),
		otel.F("metricsAddress", config.MetricsListenAddress))

	return config
}
