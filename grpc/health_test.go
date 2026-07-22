package grpc

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthManager(t *testing.T) {
	hm := NewHealthManager()

	assert.NotNil(t, hm)
	assert.NotNil(t, hm.checks)
	assert.True(t, hm.enabled)
}

func TestHealthManagerRegisterCheck(t *testing.T) {
	hm := NewHealthManager()

	testChecker := func() HealthCheckResult {
		return HealthCheckResult{
			Status: HealthStatusUp,
			Details: map[string]interface{}{
				"test": "value",
			},
		}
	}

	hm.RegisterCheck("test_service", testChecker)

	assert.Len(t, hm.checks, 1)
	assert.Contains(t, hm.checks, "test_service")
}

func TestHealthManagerCheckHealth(t *testing.T) {
	hm := NewHealthManager()

	// Register a healthy service
	hm.RegisterCheck("healthy_service", func() HealthCheckResult {
		return HealthCheckResult{
			Status: HealthStatusUp,
			Details: map[string]interface{}{
				"connection": "active",
			},
		}
	})

	// Register an unhealthy service
	hm.RegisterCheck("unhealthy_service", func() HealthCheckResult {
		return HealthCheckResult{
			Status: HealthStatusDown,
			Details: map[string]interface{}{
				"error": "connection failed",
			},
		}
	})

	// Check individual health checks
	checks := hm.CheckHealth()

	// Get overall status
	overallStatus := hm.GetOverallStatus()
	assert.Equal(t, HealthStatusDown, overallStatus)
	assert.Len(t, checks, 2)

	// Verify individual check results
	assert.Equal(t, HealthStatusUp, checks["healthy_service"].Status)
	assert.Equal(t, HealthStatusDown, checks["unhealthy_service"].Status)
}

func TestHealthManagerAllHealthy(t *testing.T) {
	hm := NewHealthManager()

	// Register only healthy services
	hm.RegisterCheck("service1", func() HealthCheckResult {
		return HealthCheckResult{Status: HealthStatusUp}
	})

	hm.RegisterCheck("service2", func() HealthCheckResult {
		return HealthCheckResult{Status: HealthStatusUp}
	})

	overallStatus := hm.GetOverallStatus()

	assert.Equal(t, HealthStatusUp, overallStatus)
}

func TestHealthManagerNoChecks(t *testing.T) {
	hm := NewHealthManager()

	checks := hm.CheckHealth()
	overallStatus := hm.GetOverallStatus()

	assert.Equal(t, HealthStatusUp, overallStatus)
	assert.Empty(t, checks)
}

func TestDefaultHealthCheckers(t *testing.T) {
	defaultCheckers := DefaultHealthCheckers()

	assert.NotEmpty(t, defaultCheckers, "Expected default health checkers to be provided")

	// Test that default checkers return valid results
	for name, checker := range defaultCheckers {
		result := checker()

		assert.NotEmpty(t, result.Status, "Default checker %s returned empty status", name)

		// Status should be one of the valid statuses
		validStatuses := []HealthStatus{HealthStatusUp, HealthStatusDown, HealthStatusUnknown}
		assert.Contains(t, validStatuses, result.Status,
			"Default checker %s returned invalid status: %s", name, result.Status)
	}
}

func TestHealthStatusConstants(t *testing.T) {
	// Test that health status constants are defined correctly
	assert.Equal(t, HealthStatus("UP"), HealthStatusUp)
	assert.Equal(t, HealthStatus("DOWN"), HealthStatusDown)
	assert.Equal(t, HealthStatus("UNKNOWN"), HealthStatusUnknown)
}

func TestHealthCheckResultSerialization(t *testing.T) {
	result := HealthCheckResult{
		Status: HealthStatusUp,
		Details: map[string]interface{}{
			"database": "connected",
			"latency":  "5ms",
			"count":    42,
		},
	}

	// Test JSON serialization
	data, err := json.Marshal(result)
	assert.NoError(t, err)

	// Test JSON deserialization
	var unmarshaled HealthCheckResult
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, result.Status, unmarshaled.Status)
	assert.Len(t, unmarshaled.Details, len(result.Details))
}

func TestHealthManagerRemoveCheck(t *testing.T) {
	hm := NewHealthManager()

	// Register a check
	testChecker := func() HealthCheckResult {
		return HealthCheckResult{Status: HealthStatusUp}
	}
	hm.RegisterCheck("test_service", testChecker)
	assert.Len(t, hm.checks, 1)

	// Remove the check
	hm.RemoveCheck("test_service")
	assert.Len(t, hm.checks, 0)
	assert.NotContains(t, hm.checks, "test_service")

	// Removing non-existent check should not cause error
	hm.RemoveCheck("non_existent")
	assert.Len(t, hm.checks, 0)
}

func TestHealthManagerSetEnabled(t *testing.T) {
	hm := NewHealthManager()

	// Initially enabled
	assert.True(t, hm.enabled)

	// Register a healthy service
	hm.RegisterCheck("service", func() HealthCheckResult {
		return HealthCheckResult{Status: HealthStatusUp}
	})

	// Disable health checks
	hm.SetEnabled(false)
	assert.False(t, hm.enabled)

	// Check health should return unknown status when disabled
	checks := hm.CheckHealth()
	assert.Len(t, checks, 1)
	assert.Equal(t, HealthStatusUnknown, checks["status"].Status)
	assert.Equal(t, "health checks disabled", checks["status"].Error)

	// Re-enable health checks
	hm.SetEnabled(true)
	assert.True(t, hm.enabled)

	// Check health should work normally
	checks = hm.CheckHealth()
	assert.Len(t, checks, 1)
	assert.Contains(t, checks, "service")
	assert.Equal(t, HealthStatusUp, checks["service"].Status)
}
