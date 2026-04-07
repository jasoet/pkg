package temporal

import (
	"github.com/jasoet/pkg/v2/otel"
)

type Config struct {
	HostPort   string       `yaml:"hostPort" mapstructure:"hostPort"`
	Namespace  string       `yaml:"namespace" mapstructure:"namespace"`
	OTelConfig *otel.Config `yaml:"-" mapstructure:"-"`
}

// DefaultConfig returns a Config with sensible defaults. It is a pure factory
// function and performs no I/O or logging.
func DefaultConfig() *Config {
	return &Config{
		HostPort:  "localhost:7233",
		Namespace: "default",
	}
}
