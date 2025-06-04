package rest

import (
	"testing"
	"time"
)

func TestDefaultRestConfig(t *testing.T) {
	config := DefaultRestConfig()

	// Verify that the config is not nil
	if config == nil {
		t.Fatal("DefaultRestConfig() returned nil")
	}

	// Verify default values
	if config.RetryCount != 1 {
		t.Errorf("Expected RetryCount to be 1, got %d", config.RetryCount)
	}

	if config.RetryWaitTime != 20*time.Second {
		t.Errorf("Expected RetryWaitTime to be 2s, got %s", config.RetryWaitTime)
	}

	if config.RetryMaxWaitTime != 30*time.Second {
		t.Errorf("Expected RetryMaxWaitTime to be 30s, got %s", config.RetryMaxWaitTime)
	}

	if config.Timeout != 50*time.Second {
		t.Errorf("Expected Timeout to be 5s, got %s", config.Timeout)
	}
}

func TestConfigStructFields(t *testing.T) {
	// Create a config with custom values
	config := Config{
		RetryCount:       3,
		RetryWaitTime:    5 * time.Second,
		RetryMaxWaitTime: 60 * time.Second,
		Timeout:          10 * time.Second,
	}

	// Verify custom values
	if config.RetryCount != 3 {
		t.Errorf("Expected RetryCount to be 3, got %d", config.RetryCount)
	}

	if config.RetryWaitTime != 5*time.Second {
		t.Errorf("Expected RetryWaitTime to be 5s, got %s", config.RetryWaitTime)
	}

	if config.RetryMaxWaitTime != 60*time.Second {
		t.Errorf("Expected RetryMaxWaitTime to be 60s, got %s", config.RetryMaxWaitTime)
	}

	if config.Timeout != 10*time.Second {
		t.Errorf("Expected Timeout to be 10s, got %s", config.Timeout)
	}
}
