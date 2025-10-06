package argo

import (
	"context"
	"fmt"
	"os"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/jasoet/pkg/v2/otel"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// NewClient creates a new Argo Workflows client from the given configuration.
// It returns the updated context and client, or an error if the connection fails.
//
// The client can connect to Argo Workflows in two modes:
// 1. Via Kubernetes API (default) - uses kubeconfig or in-cluster config
// 2. Via Argo Server - uses HTTP/HTTPS connection
//
// Example (default kubeconfig):
//
//	ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())
//	if err != nil {
//	    return err
//	}
//	defer client.Close()
//
// Example (in-cluster):
//
//	ctx, client, err := argo.NewClient(ctx, argo.InClusterConfig())
//
// Example (Argo Server):
//
//	cfg := argo.ServerConfig("https://argo-server:2746", "Bearer token")
//	ctx, client, err := argo.NewClient(ctx, cfg)
func NewClient(ctx context.Context, config *Config) (context.Context, apiclient.Client, error) {
	logger := otel.NewLogHelper(ctx, config.OTelConfig, "github.com/jasoet/pkg/v2/argo", "argo.NewClient")

	logger.Debug("Creating Argo Workflows client",
		otel.F("inCluster", config.InCluster),
		otel.F("kubeConfigPath", config.KubeConfigPath),
		otel.F("argoServerURL", config.ArgoServerOpts.URL),
	)

	// Build Argo client options
	opts := apiclient.Opts{
		Context: ctx,
	}

	// Configure Argo Server mode if URL is provided
	if config.ArgoServerOpts.URL != "" {
		logger.Debug("Using Argo Server connection mode")
		opts.ArgoServerOpts = apiclient.ArgoServerOpts{
			URL:                config.ArgoServerOpts.URL,
			InsecureSkipVerify: config.ArgoServerOpts.InsecureSkipVerify,
			HTTP1:              config.ArgoServerOpts.HTTP1,
		}
		// Set auth supplier if token is provided
		if config.ArgoServerOpts.AuthToken != "" {
			token := config.ArgoServerOpts.AuthToken
			opts.AuthSupplier = func() string {
				return token
			}
		}
	} else {
		// Use Kubernetes API mode
		logger.Debug("Using Kubernetes API connection mode")
		opts.ClientConfigSupplier = func() clientcmd.ClientConfig {
			return buildClientConfig(config)
		}
	}

	// Create the client
	ctx, client, err := apiclient.NewClientFromOpts(opts)
	if err != nil {
		logger.Error(err, "Failed to create Argo Workflows client")
		return nil, nil, fmt.Errorf("failed to create argo client: %w", err)
	}

	logger.Debug("Successfully created Argo Workflows client")
	return ctx, client, nil
}

// NewClientWithOptions creates a new Argo Workflows client using functional options.
// This provides a more flexible way to configure the client.
//
// Example:
//
//	ctx, client, err := argo.NewClientWithOptions(ctx,
//	    argo.WithKubeConfig("/custom/path/kubeconfig"),
//	    argo.WithContext("production"),
//	    argo.WithOTelConfig(otelConfig),
//	)
func NewClientWithOptions(ctx context.Context, opts ...Option) (context.Context, apiclient.Client, error) {
	config := DefaultConfig()
	for _, opt := range opts {
		if err := opt(config); err != nil {
			return nil, nil, fmt.Errorf("failed to apply option: %w", err)
		}
	}
	return NewClient(ctx, config)
}

// buildClientConfig constructs a Kubernetes ClientConfig based on the Argo configuration.
// It handles three scenarios:
// 1. In-cluster configuration (when running inside a Kubernetes pod)
// 2. Explicit kubeconfig path
// 3. Default kubeconfig location (~/.kube/config)
func buildClientConfig(config *Config) clientcmd.ClientConfig {
	// Note: context.Background() used here since we don't have access to the actual context
	// This is acceptable as buildClientConfig is called from within NewClient which has the context
	logger := otel.NewLogHelper(context.Background(), config.OTelConfig, "github.com/jasoet/pkg/v2/argo", "argo.buildClientConfig")

	// For in-cluster mode, use in-cluster config
	if config.InCluster {
		logger.Debug("Building in-cluster client config")
		return &inClusterClientConfig{}
	}

	// Build kubeconfig loading rules
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	// Set explicit path if provided
	if config.KubeConfigPath != "" {
		logger.Debug("Using explicit kubeconfig path", otel.F("path", config.KubeConfigPath))
		loadingRules.ExplicitPath = config.KubeConfigPath
	} else {
		// Use default kubeconfig location
		logger.Debug("Using default kubeconfig location")
	}

	// Build config overrides
	overrides := &clientcmd.ConfigOverrides{}
	if config.Context != "" {
		logger.Debug("Using explicit context", otel.F("context", config.Context))
		overrides.CurrentContext = config.Context
	}

	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loadingRules,
		overrides,
	)
}

// inClusterClientConfig implements clientcmd.ClientConfig for in-cluster usage.
type inClusterClientConfig struct{}

func (c *inClusterClientConfig) RawConfig() (clientcmdapi.Config, error) {
	return clientcmdapi.Config{}, fmt.Errorf("RawConfig not supported for in-cluster config")
}

func (c *inClusterClientConfig) ClientConfig() (*rest.Config, error) {
	logger := otel.NewLogHelper(context.Background(), nil, "github.com/jasoet/pkg/v2/argo", "inClusterClientConfig.ClientConfig")
	logger.Debug("Loading in-cluster config")

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Error(err, "Failed to load in-cluster config")
		return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
	}

	logger.Debug("Successfully loaded in-cluster config")
	return config, nil
}

func (c *inClusterClientConfig) Namespace() (string, bool, error) {
	// Read namespace from the same location that Kubernetes uses
	namespaceBytes, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
	if err != nil {
		return "default", false, err
	}
	return string(namespaceBytes), true, nil
}

func (c *inClusterClientConfig) ConfigAccess() clientcmd.ConfigAccess {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	loadingRules.ExplicitPath = ""
	return loadingRules
}
