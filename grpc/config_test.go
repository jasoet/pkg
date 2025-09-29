package grpc

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	// Test default values
	assert.NotEmpty(t, config.GRPCPort)
	assert.Equal(t, H2CMode, config.Mode)
	assert.Equal(t, 30*time.Second, config.ShutdownTimeout)
	assert.Equal(t, 5*time.Second, config.ReadTimeout)
	assert.Equal(t, 10*time.Second, config.WriteTimeout)
	assert.Equal(t, 60*time.Second, config.IdleTimeout)
	assert.Equal(t, 15*time.Minute, config.MaxConnectionIdle)
	assert.Equal(t, 30*time.Minute, config.MaxConnectionAge)
	assert.Equal(t, 5*time.Second, config.MaxConnectionAgeGrace)

	// Test feature flags
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableHealthCheck)
	assert.True(t, config.EnableLogging)
	assert.True(t, config.EnableReflection)

	// Test paths
	assert.Equal(t, "/metrics", config.MetricsPath)
	assert.Equal(t, "/health", config.HealthPath)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config
		expectError bool
	}{
		{
			name: "valid H2C config",
			config: Config{
				GRPCPort: "8080",
				Mode:     H2CMode,
			},
			expectError: false,
		},
		{
			name: "valid separate config",
			config: Config{
				GRPCPort: "9090",
				HTTPPort: "9091",
				Mode:     SeparateMode,
			},
			expectError: false,
		},
		{
			name: "missing gRPC port",
			config: Config{
				Mode: H2CMode,
			},
			expectError: true,
		},
		{
			name: "separate mode missing HTTP port",
			config: Config{
				GRPCPort: "9090",
				Mode:     SeparateMode,
			},
			expectError: true,
		},
		{
			name: "invalid mode",
			config: Config{
				GRPCPort: "8080",
				Mode:     "invalid",
			},
			expectError: true,
		},
		{
			name: "negative shutdown timeout",
			config: Config{
				GRPCPort:        "8080",
				Mode:            H2CMode,
				ShutdownTimeout: -1 * time.Second,
			},
			expectError: true,
		},
		{
			name: "negative read timeout",
			config: Config{
				GRPCPort:    "8080",
				Mode:        H2CMode,
				ReadTimeout: -1 * time.Second,
			},
			expectError: true,
		},
		{
			name: "negative write timeout",
			config: Config{
				GRPCPort:     "8080",
				Mode:         H2CMode,
				WriteTimeout: -1 * time.Second,
			},
			expectError: true,
		},
		{
			name: "negative idle timeout",
			config: Config{
				GRPCPort:    "8080",
				Mode:        H2CMode,
				IdleTimeout: -1 * time.Second,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigAddresses(t *testing.T) {
	tests := []struct {
		name         string
		config       Config
		expectedGRPC string
		expectedHTTP string
	}{
		{
			name: "default ports",
			config: Config{
				GRPCPort: "8080",
				HTTPPort: "8081",
			},
			expectedGRPC: ":8080",
			expectedHTTP: ":8081",
		},
		{
			name: "custom ports",
			config: Config{
				GRPCPort: "9090",
				HTTPPort: "9091",
			},
			expectedGRPC: ":9090",
			expectedHTTP: ":9091",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			grpcAddr := tt.config.GetGRPCAddress()
			assert.Equal(t, tt.expectedGRPC, grpcAddr)

			httpAddr := tt.config.GetHTTPAddress()
			assert.Equal(t, tt.expectedHTTP, httpAddr)
		})
	}
}

func TestConfigSetDefaults(t *testing.T) {
	config := Config{
		GRPCPort: "8080",
		Mode:     H2CMode,
	}

	config.SetDefaults()

	// Verify defaults were set
	assert.Equal(t, 30*time.Second, config.ShutdownTimeout)
	assert.Equal(t, "/metrics", config.MetricsPath)
	assert.Equal(t, "/health", config.HealthPath)
	assert.True(t, config.EnableMetrics)
	assert.True(t, config.EnableHealthCheck)
}
