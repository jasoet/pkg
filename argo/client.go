package argo

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v3/pkg/apiclient"
	"github.com/rs/zerolog/log"
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
//	cfg := argo.ArgoServerConfig("https://argo-server:2746", "Bearer token")
//	ctx, client, err := argo.NewClient(ctx, cfg)
func NewClient(ctx context.Context, config *Config) (context.Context, apiclient.Client, error) {
	logger := log.With().
		Str("function", "argo.NewClient").
		Bool("inCluster", config.InCluster).
		Str("kubeConfigPath", config.KubeConfigPath).
		Str("argoServerURL", config.ArgoServerOpts.URL).
		Logger()

	logger.Debug().Msg("Creating Argo Workflows client")

	// Build Argo client options
	opts := apiclient.Opts{
		Context: ctx,
	}

	// Configure Argo Server mode if URL is provided
	if config.ArgoServerOpts.URL != "" {
		logger.Debug().Msg("Using Argo Server connection mode")
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
		logger.Debug().Msg("Using Kubernetes API connection mode")
		opts.ClientConfigSupplier = func() clientcmd.ClientConfig {
			return buildClientConfig(config)
		}
	}

	// Create the client
	ctx, client, err := apiclient.NewClientFromOpts(opts)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create Argo Workflows client")
		return nil, nil, fmt.Errorf("failed to create argo client: %w", err)
	}

	logger.Debug().Msg("Successfully created Argo Workflows client")
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
	logger := log.With().Str("function", "argo.buildClientConfig").Logger()

	// For in-cluster mode, use in-cluster config
	if config.InCluster {
		logger.Debug().Msg("Building in-cluster client config")
		return &inClusterClientConfig{}
	}

	// Build kubeconfig loading rules
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()

	// Set explicit path if provided
	if config.KubeConfigPath != "" {
		logger.Debug().Str("path", config.KubeConfigPath).Msg("Using explicit kubeconfig path")
		loadingRules.ExplicitPath = config.KubeConfigPath
	} else {
		// Use default kubeconfig location
		logger.Debug().Msg("Using default kubeconfig location")
	}

	// Build config overrides
	overrides := &clientcmd.ConfigOverrides{}
	if config.Context != "" {
		logger.Debug().Str("context", config.Context).Msg("Using explicit context")
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
	logger := log.With().Str("function", "inClusterClientConfig.ClientConfig").Logger()
	logger.Debug().Msg("Loading in-cluster config")

	config, err := rest.InClusterConfig()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to load in-cluster config")
		return nil, fmt.Errorf("failed to load in-cluster config: %w", err)
	}

	logger.Debug().Msg("Successfully loaded in-cluster config")
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

// GetCmdConfig returns an interactive ClientConfig for use with command-line tools.
// This is useful for tools that need to prompt users for input.
//
// Deprecated: Use NewClient or NewClientWithOptions instead for better control.
func GetCmdConfig() clientcmd.ClientConfig {
	logger := log.With().Str("function", "argo.GetCmdConfig").Logger()
	logger.Debug().Msg("Creating interactive client config")

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	return clientcmd.NewInteractiveDeferredLoadingClientConfig(
		loadingRules,
		&clientcmd.ConfigOverrides{},
		os.Stdin,
	)
}

// GetRestConfig returns a Kubernetes REST config for direct API access.
// This is useful when you need to interact with Kubernetes API outside of Argo.
//
// Deprecated: Use NewClient with appropriate Config instead.
func GetRestConfig(inCluster bool) (*rest.Config, error) {
	logger := log.With().
		Str("function", "argo.GetRestConfig").
		Bool("inCluster", inCluster).
		Logger()

	var kubeConfig string
	if !inCluster {
		kubeConfig = filepath.Join(clientcmd.RecommendedConfigDir, clientcmd.RecommendedFileName)
		logger.Debug().Str("kubeConfigPath", kubeConfig).Msg("Using kubeconfig file")
	} else {
		logger.Debug().Msg("Using in-cluster config")
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeConfig)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to create REST config")
		return nil, fmt.Errorf("failed to create config from kubeconfig: %w", err)
	}

	logger.Debug().Msg("Successfully created REST config")
	return config, nil
}
