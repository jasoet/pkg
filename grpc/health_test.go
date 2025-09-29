package grpc

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

func TestHealthCheckHandlers(t *testing.T) {
	hm := NewHealthManager()

	// Register a test service
	hm.RegisterCheck("test_service", func() HealthCheckResult {
		return HealthCheckResult{
			Status: HealthStatusUp,
			Details: map[string]interface{}{
				"version": "1.0.0",
			},
		}
	})

	handlers := hm.CreateHealthHandlers("/health")

	// Test main health endpoint
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler, exists := handlers["/health"]
	assert.True(t, exists, "Expected /health handler to exist")

	handler(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, string(HealthStatusUp), response["status"])

	checks, ok := response["checks"].(map[string]interface{})
	assert.True(t, ok, "Expected checks to be a map")
	assert.Len(t, checks, 1)
}

func TestHealthCheckReadinessHandler(t *testing.T) {
	hm := NewHealthManager()

	// Register an unhealthy service
	hm.RegisterCheck("database", func() HealthCheckResult {
		return HealthCheckResult{
			Status: HealthStatusDown,
			Details: map[string]interface{}{
				"error": "connection timeout",
			},
		}
	})

	handlers := hm.CreateHealthHandlers("/health")

	// Test readiness endpoint
	req := httptest.NewRequest("GET", "/health/ready", nil)
	w := httptest.NewRecorder()

	handler, exists := handlers["/health/ready"]
	assert.True(t, exists, "Expected /health/ready handler to exist")

	handler(w, req)

	// Should return 503 Service Unavailable when not ready
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "not_ready", response["status"])
}

func TestHealthCheckLivenessHandler(t *testing.T) {
	hm := NewHealthManager()

	handlers := hm.CreateHealthHandlers("/health")

	// Test liveness endpoint
	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()

	handler, exists := handlers["/health/live"]
	assert.True(t, exists, "Expected /health/live handler to exist")

	handler(w, req)

	// Liveness should always return OK unless the service is completely dead
	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	assert.Equal(t, "alive", response["status"])
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
