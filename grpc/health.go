package grpc

import (
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// HealthStatus represents the status of a health check
type HealthStatus string

const (
	// HealthStatusUp indicates the service is healthy
	HealthStatusUp HealthStatus = "UP"
	// HealthStatusDown indicates the service is unhealthy
	HealthStatusDown HealthStatus = "DOWN"
	// HealthStatusUnknown indicates the health status is unknown
	HealthStatusUnknown HealthStatus = "UNKNOWN"
)

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Duration  time.Duration          `json:"duration"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Error     string                 `json:"error,omitempty"`
}

// HealthChecker defines the interface for health check functions
type HealthChecker func() HealthCheckResult

// HealthManager manages health checks for the server
type HealthManager struct {
	mu      sync.RWMutex
	checks  map[string]HealthChecker
	enabled bool
}

// NewHealthManager creates a new health manager
func NewHealthManager() *HealthManager {
	return &HealthManager{
		checks:  make(map[string]HealthChecker),
		enabled: true,
	}
}

// RegisterCheck registers a health check with the given name
func (h *HealthManager) RegisterCheck(name string, checker HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checks[name] = checker
}

// RemoveCheck removes a health check
func (h *HealthManager) RemoveCheck(name string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.checks, name)
}

// SetEnabled enables or disables health checks
func (h *HealthManager) SetEnabled(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.enabled = enabled
}

// CheckHealth runs all registered health checks
func (h *HealthManager) CheckHealth() map[string]HealthCheckResult {
	h.mu.RLock()
	enabled := h.enabled
	checkers := make(map[string]HealthChecker, len(h.checks))
	for k, v := range h.checks {
		checkers[k] = v
	}
	h.mu.RUnlock()

	if !enabled {
		return map[string]HealthCheckResult{
			"status": {
				Status:    HealthStatusUnknown,
				Timestamp: time.Now(),
				Error:     "health checks disabled",
			},
		}
	}

	results := make(map[string]HealthCheckResult, len(checkers))

	for name, checker := range checkers {
		start := time.Now()
		result := checker()
		result.Duration = time.Since(start)
		result.Timestamp = time.Now()
		results[name] = result
	}

	return results
}

// overallStatusFromResults derives the aggregate status from a set of results.
// Any checker returning HealthStatusDown or HealthStatusUnknown causes the
// overall status to be HealthStatusDown (fail-safe / conservative policy).
func overallStatusFromResults(results map[string]HealthCheckResult) HealthStatus {
	if len(results) == 0 {
		return HealthStatusUp // No checks means healthy
	}
	for _, result := range results {
		if result.Status == HealthStatusDown || result.Status == HealthStatusUnknown {
			return HealthStatusDown
		}
	}
	return HealthStatusUp
}

// GetOverallStatus returns the overall health status.
func (h *HealthManager) GetOverallStatus() HealthStatus {
	return overallStatusFromResults(h.CheckHealth())
}

// DefaultHealthCheckers returns a set of default health checkers
func DefaultHealthCheckers() map[string]HealthChecker {
	return map[string]HealthChecker{
		"server": func() HealthCheckResult {
			return HealthCheckResult{
				Status: HealthStatusUp,
				Details: map[string]interface{}{
					"uptime": time.Since(startTime).String(),
				},
			}
		},
		"memory": func() HealthCheckResult {
			return HealthCheckResult{
				Status: HealthStatusUp,
				Details: map[string]interface{}{
					"goroutines": runtime.NumGoroutine(),
				},
			}
		},
	}
}

var startTime = time.Now() // Package-level variable to track server start time

// RegisterEchoHealthChecks registers health check endpoints with Echo
func (h *HealthManager) RegisterEchoHealthChecks(e *echo.Echo, basePath string) {
	// Overall health endpoint
	e.GET(basePath, func(c echo.Context) error {
		results := h.CheckHealth()
		overallStatus := overallStatusFromResults(results)

		response := map[string]interface{}{
			"status":    overallStatus,
			"timestamp": time.Now(),
			"checks":    results,
		}

		code := http.StatusOK
		if overallStatus == HealthStatusDown {
			code = http.StatusServiceUnavailable
		}
		return c.JSON(code, response)
	})

	// Readiness endpoint - checks if the service is ready to serve requests
	e.GET(basePath+"/ready", func(c echo.Context) error {
		status := h.GetOverallStatus()
		if status == HealthStatusUp {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status":    "ready",
				"timestamp": time.Now(),
			})
		}
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "not ready",
			"timestamp": time.Now(),
		})
	})

	// Liveness endpoint - always returns 200 UP; only fails if the process is dead.
	// Kubernetes liveness probes must not run application-level health checks.
	e.GET(basePath+"/live", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "UP",
			"timestamp": time.Now(),
		})
	})
}
