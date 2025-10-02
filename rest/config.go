package rest

import (
	"time"

	"github.com/jasoet/pkg/otel"
)

// Config RestConfig contains configuration for REST client
type Config struct {
	RetryCount       int           `yaml:"retryCount" mapstructure:"retryCount"`
	RetryWaitTime    time.Duration `yaml:"retryWaitTime" mapstructure:"retryWaitTime"`
	RetryMaxWaitTime time.Duration `yaml:"retryMaxWaitTime" mapstructure:"retryMaxWaitTime"`
	Timeout          time.Duration `yaml:"timeout" mapstructure:"timeout"`

	// OpenTelemetry Configuration (optional - nil disables telemetry)
	OTelConfig *otel.Config `yaml:"-" mapstructure:"-"` // Not serializable from config files
}

// DefaultRestConfig returns a default REST configuration
func DefaultRestConfig() *Config {
	return &Config{
		RetryCount:       1,
		RetryWaitTime:    20 * time.Second,
		RetryMaxWaitTime: 30 * time.Second,
		Timeout:          50 * time.Second,
	}
}
