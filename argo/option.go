package argo

import (
	"github.com/jasoet/pkg/v2/otel"
)

// Option is a functional option for configuring Argo client.
type Option func(*Config) error

// WithKubeConfig sets the path to the kubeconfig file.
// If not set, the default location (~/.kube/config) will be used.
//
// Example:
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithKubeConfig("/custom/path/to/kubeconfig"),
//	)
func WithKubeConfig(path string) Option {
	return func(c *Config) error {
		c.KubeConfigPath = path
		return nil
	}
}

// WithContext sets the kubeconfig context to use.
// If not set, the current context from kubeconfig will be used.
//
// Example:
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithContext("production"),
//	)
func WithContext(context string) Option {
	return func(c *Config) error {
		c.Context = context
		return nil
	}
}

// WithInCluster enables in-cluster configuration mode.
// When true, the client will use the service account token mounted in the pod.
// This is useful when running inside a Kubernetes cluster.
//
// Example:
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithInCluster(true),
//	)
func WithInCluster(inCluster bool) Option {
	return func(c *Config) error {
		c.InCluster = inCluster
		return nil
	}
}

// WithArgoServer configures the client to connect via Argo Server instead of Kubernetes API.
// This is an alternative connection mode that uses HTTP/HTTPS.
//
// Example:
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithArgoServer("https://argo-server:2746", "Bearer token"),
//	)
func WithArgoServer(url, authToken string) Option {
	return func(c *Config) error {
		c.ArgoServerOpts.URL = url
		c.ArgoServerOpts.AuthToken = authToken
		return nil
	}
}

// WithArgoServerInsecure enables insecure mode for Argo Server connection.
// This disables TLS certificate verification.
// WARNING: This should only be used in development/testing environments.
//
// Example:
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithArgoServer("http://argo-server:2746", ""),
//	    argo.WithArgoServerInsecure(true),
//	)
func WithArgoServerInsecure(insecure bool) Option {
	return func(c *Config) error {
		c.ArgoServerOpts.InsecureSkipVerify = insecure
		return nil
	}
}

// WithArgoServerHTTP1 forces HTTP/1.1 instead of HTTP/2 for Argo Server connection.
// This can be useful for debugging or compatibility reasons.
//
// Example:
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithArgoServer("https://argo-server:2746", "Bearer token"),
//	    argo.WithArgoServerHTTP1(true),
//	)
func WithArgoServerHTTP1(http1 bool) Option {
	return func(c *Config) error {
		c.ArgoServerOpts.HTTP1 = http1
		return nil
	}
}

// WithOTelConfig enables OpenTelemetry instrumentation for the Argo client.
// This allows tracing and monitoring of workflow operations.
//
// Example:
//
//	otelConfig := otel.NewConfig("my-service").
//	    WithTracerProvider(tracerProvider).
//	    WithMeterProvider(meterProvider)
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithOTelConfig(otelConfig),
//	)
func WithOTelConfig(otelConfig *otel.Config) Option {
	return func(c *Config) error {
		c.OTelConfig = otelConfig
		return nil
	}
}

// WithArgoServerOpts sets the complete ArgoServerOpts configuration.
// This is useful when you want to configure all Argo Server options at once.
//
// Example:
//
//	serverOpts := argo.ArgoServerOpts{
//	    URL:                "https://argo-server:2746",
//	    AuthToken:          "Bearer token",
//	    InsecureSkipVerify: false,
//	    HTTP1:              false,
//	}
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithArgoServerOpts(serverOpts),
//	)
func WithArgoServerOpts(opts ArgoServerOpts) Option {
	return func(c *Config) error {
		c.ArgoServerOpts = opts
		return nil
	}
}

// WithConfig applies a complete Config to the client.
// This is useful when you have a pre-built configuration from a config file.
//
// Example:
//
//	config := &argo.Config{
//	    KubeConfigPath: "/custom/kubeconfig",
//	    Context:        "production",
//	}
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithConfig(config),
//	)
func WithConfig(config *Config) Option {
	return func(c *Config) error {
		*c = *config
		return nil
	}
}
