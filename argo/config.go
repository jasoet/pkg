package argo

import (
	"context"

	"github.com/jasoet/pkg/v2/otel"
)

// Config represents the configuration for connecting to Argo Workflows.
// It supports multiple connection modes:
// - Kubernetes API (in-cluster or with kubeconfig)
// - Argo Server (HTTP/HTTPS with optional authentication)
type Config struct {
	// KubeConfigPath specifies the path to kubeconfig file.
	// Empty string means:
	// - Use in-cluster config if InCluster is true
	// - Use default kubeconfig location (~/.kube/config) if InCluster is false
	KubeConfigPath string `yaml:"kubeConfigPath" mapstructure:"kubeConfigPath"`

	// Context specifies the kubeconfig context to use.
	// Empty string means use the current context.
	Context string `yaml:"context" mapstructure:"context"`

	// InCluster indicates whether to use in-cluster Kubernetes configuration.
	// When true, the client will use the service account token mounted in the pod.
	InCluster bool `yaml:"inCluster" mapstructure:"inCluster"`

	// ArgoServerOpts configures connection to Argo Server (alternative to direct k8s API).
	// If URL is set, the client will connect via Argo Server instead of k8s API.
	ArgoServerOpts ServerOpts `yaml:"argoServer" mapstructure:"argoServer"`

	// OTelConfig enables OpenTelemetry instrumentation (optional).
	OTelConfig *otel.Config `yaml:"-"`
}

// ServerOpts contains Argo Server connection options.
// This is used when connecting via Argo Server HTTP API instead of direct Kubernetes API.
type ServerOpts struct {
	// URL is the Argo Server base URL (e.g., "https://argo-server:2746")
	URL string `yaml:"url" mapstructure:"url"`

	// AuthToken is the bearer token for authentication (optional).
	// Excluded from serialization to prevent accidental credential exposure.
	AuthToken string `yaml:"-" mapstructure:"-"`

	// InsecureSkipVerify disables TLS certificate verification (not recommended for production)
	InsecureSkipVerify bool `yaml:"insecureSkipVerify" mapstructure:"insecureSkipVerify"`

	// HTTP1 forces HTTP/1.1 instead of HTTP/2
	HTTP1 bool `yaml:"http1" mapstructure:"http1"`
}

// DefaultConfig returns a Config with sensible defaults.
// By default, it uses:
// - Out-of-cluster mode (InCluster = false)
// - Default kubeconfig location (~/.kube/config)
// - Current context from kubeconfig
func DefaultConfig() *Config {
	config := &Config{
		InCluster: false,
		ArgoServerOpts: ServerOpts{
			InsecureSkipVerify: false,
			HTTP1:              false,
		},
	}

	logger := otel.NewLogHelper(context.Background(), config.OTelConfig, "github.com/jasoet/pkg/v2/argo", "argo.DefaultConfig")
	logger.Debug("Created default Argo configuration",
		otel.F("inCluster", config.InCluster),
		otel.F("kubeConfigPath", config.KubeConfigPath),
	)

	return config
}

// InClusterConfig returns a Config for in-cluster usage.
// This is useful when the client runs inside a Kubernetes pod.
func InClusterConfig() *Config {
	config := &Config{
		InCluster: true,
	}

	logger := otel.NewLogHelper(context.Background(), config.OTelConfig, "github.com/jasoet/pkg/v2/argo", "argo.InClusterConfig")
	logger.Debug("Created in-cluster Argo configuration")

	return config
}

// ServerConfig returns a Config for connecting via Argo Server.
// This is an alternative to direct Kubernetes API access.
func ServerConfig(serverURL string, authToken string) *Config {
	config := &Config{
		InCluster: false,
		ArgoServerOpts: ServerOpts{
			URL:                serverURL,
			AuthToken:          authToken,
			InsecureSkipVerify: false,
			HTTP1:              false,
		},
	}

	logger := otel.NewLogHelper(context.Background(), config.OTelConfig, "github.com/jasoet/pkg/v2/argo", "argo.ServerConfig")
	logger.Debug("Created Argo Server configuration", otel.F("serverURL", serverURL))

	return config
}
