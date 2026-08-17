package argo

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jasoet/pkg/v3/otel"
)

func TestWithKubeConfig(t *testing.T) {
	config := &Config{}
	path := "/custom/path/to/kubeconfig"

	WithKubeConfig(path)(config)

	assert.Equal(t, path, config.KubeConfigPath)
}

func TestWithContext(t *testing.T) {
	config := &Config{}
	contextName := "production"

	WithContext(contextName)(config)

	assert.Equal(t, contextName, config.Context)
}

func TestWithInCluster(t *testing.T) {
	tests := []struct {
		name      string
		inCluster bool
	}{
		{"Enable in-cluster", true},
		{"Disable in-cluster", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}

			WithInCluster(tt.inCluster)(config)

			assert.Equal(t, tt.inCluster, config.InCluster)
		})
	}
}

func TestWithArgoServer(t *testing.T) {
	config := &Config{}
	url := "https://argo-server:2746"
	token := "Bearer test-token"

	WithArgoServer(url, token)(config)

	assert.Equal(t, url, config.ArgoServerOpts.URL)
	assert.Equal(t, token, config.ArgoServerOpts.AuthToken)
}

func TestWithArgoServerInsecure(t *testing.T) {
	tests := []struct {
		name     string
		insecure bool
	}{
		{"Enable insecure", true},
		{"Disable insecure", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}

			WithArgoServerInsecure(tt.insecure)(config)

			assert.Equal(t, tt.insecure, config.ArgoServerOpts.InsecureSkipVerify)
		})
	}
}

func TestWithArgoServerHTTP1(t *testing.T) {
	tests := []struct {
		name  string
		http1 bool
	}{
		{"Enable HTTP1", true},
		{"Disable HTTP1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}

			WithArgoServerHTTP1(tt.http1)(config)

			assert.Equal(t, tt.http1, config.ArgoServerOpts.HTTP1)
		})
	}
}

func TestWithOTelConfig(t *testing.T) {
	config := &Config{}
	otelConfig := otel.NewConfig("test-service")

	WithOTelConfig(otelConfig)(config)

	assert.NotNil(t, config.OTelConfig)
	assert.Equal(t, otelConfig, config.OTelConfig)
}

func TestWithArgoServerOpts(t *testing.T) {
	config := &Config{}
	serverOpts := ServerOpts{
		URL:                "https://argo-server:2746",
		AuthToken:          "Bearer token",
		InsecureSkipVerify: true,
		HTTP1:              true,
	}

	WithArgoServerOpts(serverOpts)(config)

	assert.Equal(t, serverOpts, config.ArgoServerOpts)
	assert.Equal(t, serverOpts.URL, config.ArgoServerOpts.URL)
	assert.Equal(t, serverOpts.AuthToken, config.ArgoServerOpts.AuthToken)
	assert.Equal(t, serverOpts.InsecureSkipVerify, config.ArgoServerOpts.InsecureSkipVerify)
	assert.Equal(t, serverOpts.HTTP1, config.ArgoServerOpts.HTTP1)
}

func TestWithConfig(t *testing.T) {
	config := &Config{}
	newConfig := &Config{
		KubeConfigPath: "/path/to/kubeconfig",
		Context:        "production",
		InCluster:      true,
		ArgoServerOpts: ServerOpts{
			URL:       "https://argo-server:2746",
			AuthToken: "Bearer token",
		},
	}

	WithConfig(newConfig)(config)

	assert.Equal(t, newConfig.KubeConfigPath, config.KubeConfigPath)
	assert.Equal(t, newConfig.Context, config.Context)
	assert.Equal(t, newConfig.InCluster, config.InCluster)
	assert.Equal(t, newConfig.ArgoServerOpts, config.ArgoServerOpts)
}

func TestMultipleOptions(t *testing.T) {
	config := &Config{}

	WithKubeConfig("/path/to/kubeconfig")(config)
	WithContext("production")(config)
	WithInCluster(false)(config)

	assert.Equal(t, "/path/to/kubeconfig", config.KubeConfigPath)
	assert.Equal(t, "production", config.Context)
	assert.False(t, config.InCluster)
}

func TestChainingOptions(t *testing.T) {
	config := DefaultConfig()

	// Apply multiple options
	opts := []Option{
		WithKubeConfig("/custom/kubeconfig"),
		WithContext("staging"),
		WithArgoServer("https://argo:2746", "Bearer token"),
		WithArgoServerInsecure(true),
	}

	for _, opt := range opts {
		opt(config)
	}

	assert.Equal(t, "/custom/kubeconfig", config.KubeConfigPath)
	assert.Equal(t, "staging", config.Context)
	assert.Equal(t, "https://argo:2746", config.ArgoServerOpts.URL)
	assert.Equal(t, "Bearer token", config.ArgoServerOpts.AuthToken)
	assert.True(t, config.ArgoServerOpts.InsecureSkipVerify)
}
