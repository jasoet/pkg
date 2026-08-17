package temporal

import (
	"crypto/tls"

	"go.temporal.io/sdk/client"

	"github.com/jasoet/pkg/v3/otel"
)

type Config struct {
	HostPort   string       `yaml:"hostPort" mapstructure:"hostPort"`
	Namespace  string       `yaml:"namespace" mapstructure:"namespace"`
	OTelConfig *otel.Config `yaml:"-" mapstructure:"-"`

	// TLS configures a TLS connection to the Temporal frontend. Required for
	// Temporal Cloud and any TLS-enabled self-hosted server. Not serializable.
	TLS *tls.Config `yaml:"-" mapstructure:"-"`

	// Credentials authenticates the client — e.g. an API key for Temporal
	// Cloud (client.NewAPIKeyStaticCredentials) or mTLS. Not serializable.
	Credentials client.Credentials `yaml:"-" mapstructure:"-"`

	// clientOptionsHooks are applied, in order, to the fully-assembled
	// client.Options right before dialing. They are the escape hatch for any
	// SDK option this package does not surface directly. Not serializable.
	clientOptionsHooks []func(*client.Options) `yaml:"-" mapstructure:"-"`
}

// DefaultConfig returns a Config with sensible defaults. It is a pure factory
// function and performs no I/O or logging.
func DefaultConfig() *Config {
	return &Config{
		HostPort:  "localhost:7233",
		Namespace: "default",
	}
}

// Option mutates a Config. Options are applied to DefaultConfig by NewClient.
type Option func(*Config)

// WithConfig replaces the entire configuration with c.
func WithConfig(c Config) Option {
	return func(cfg *Config) {
		*cfg = c
	}
}

// WithHostPort sets the Temporal frontend address (host:port).
func WithHostPort(addr string) Option {
	return func(cfg *Config) {
		cfg.HostPort = addr
	}
}

// WithNamespace sets the Temporal namespace.
func WithNamespace(ns string) Option {
	return func(cfg *Config) {
		cfg.Namespace = ns
	}
}

// WithOTelConfig attaches OTel tracing/metrics to the client.
func WithOTelConfig(otelCfg *otel.Config) Option {
	return func(cfg *Config) {
		cfg.OTelConfig = otelCfg
	}
}

// WithTLS enables a TLS connection to the Temporal frontend using tlsCfg.
// Required for Temporal Cloud and TLS-enabled self-hosted servers.
func WithTLS(tlsCfg *tls.Config) Option {
	return func(cfg *Config) {
		cfg.TLS = tlsCfg
	}
}

// WithCredentials sets the client credentials used to authenticate — for
// example an API key for Temporal Cloud via
// client.NewAPIKeyStaticCredentials, or mTLS credentials.
func WithCredentials(creds client.Credentials) Option {
	return func(cfg *Config) {
		cfg.Credentials = creds
	}
}

// WithClientOptions registers a hook that receives the fully-assembled
// client.Options immediately before dialing, letting callers set any SDK
// option this package does not expose directly. Hooks run in registration
// order, last, so they can override everything set beforehand.
func WithClientOptions(fn func(*client.Options)) Option {
	return func(cfg *Config) {
		if fn != nil {
			cfg.clientOptionsHooks = append(cfg.clientOptionsHooks, fn)
		}
	}
}
