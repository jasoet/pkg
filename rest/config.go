package rest

import (
	"time"

	"github.com/jasoet/pkg/v2/otel"
)

// Config holds configuration for the REST client.
type Config struct {
	RetryCount       int           `yaml:"retryCount" mapstructure:"retryCount"`
	RetryWaitTime    time.Duration `yaml:"retryWaitTime" mapstructure:"retryWaitTime"`
	RetryMaxWaitTime time.Duration `yaml:"retryMaxWaitTime" mapstructure:"retryMaxWaitTime"`
	Timeout          time.Duration `yaml:"timeout" mapstructure:"timeout"`

	// OpenTelemetry Configuration (optional - nil disables telemetry)
	OTelConfig *otel.Config `yaml:"-" mapstructure:"-"` // Not serializable from config files
}

// DefaultRestConfig returns a default REST configuration with sensible defaults.
func DefaultRestConfig() *Config {
	return &Config{
		RetryCount:       1,
		RetryWaitTime:    2 * time.Second,
		RetryMaxWaitTime: 10 * time.Second,
		Timeout:          30 * time.Second,
	}
}
