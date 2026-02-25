package grpc

import (
	"encoding/json"
	"fmt"
	"net/http"
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
func (hm *HealthManager) RegisterCheck(name string, checker HealthChecker) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.checks[name] = checker
}

// RemoveCheck removes a health check
func (hm *HealthManager) RemoveCheck(name string) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	delete(hm.checks, name)
}

// SetEnabled enables or disables health checks
func (hm *HealthManager) SetEnabled(enabled bool) {
	hm.mu.Lock()
	defer hm.mu.Unlock()
	hm.enabled = enabled
}

// CheckHealth runs all registered health checks
func (hm *HealthManager) CheckHealth() map[string]HealthCheckResult {
	hm.mu.RLock()
	defer hm.mu.RUnlock()

	if !hm.enabled {
		return map[string]HealthCheckResult{
			"status": {
				Status:    HealthStatusUnknown,
				Timestamp: time.Now(),
				Error:     "health checks disabled",
			},
		}
	}

	results := make(map[string]HealthCheckResult, len(hm.checks))

	for name, checker := range hm.checks {
		start := time.Now()
		result := checker()
		result.Duration = time.Since(start)
		result.Timestamp = time.Now()
		results[name] = result
	}

	return results
}

// overallStatusFromResults derives the aggregate status from a set of results.
func overallStatusFromResults(results map[string]HealthCheckResult) HealthStatus {
	if len(results) == 0 {
		return HealthStatusUp // No checks means healthy
	}
	for _, result := range results {
		if result.Status == HealthStatusDown {
			return HealthStatusDown
		}
	}
	return HealthStatusUp
}

// GetOverallStatus returns the overall health status.
func (hm *HealthManager) GetOverallStatus() HealthStatus {
	return overallStatusFromResults(hm.CheckHealth())
}

// CreateHealthHandlers creates HTTP handlers for health check endpoints
func (hm *HealthManager) CreateHealthHandlers(basePath string) map[string]http.HandlerFunc {
	handlers := make(map[string]http.HandlerFunc)

	// Main health endpoint - returns detailed health information
	handlers[basePath] = func(w http.ResponseWriter, r *http.Request) {
		results := hm.CheckHealth()
		overallStatus := overallStatusFromResults(results)

		response := map[string]interface{}{
			"status":    overallStatus,
			"timestamp": time.Now(),
			"checks":    results,
		}

		w.Header().Set("Content-Type", "application/json")

		if overallStatus == HealthStatusDown {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			w.WriteHeader(http.StatusOK)
		}

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode health check response", http.StatusInternalServerError)
		}
	}

	// Readiness endpoint - simple ready check
	handlers[basePath+"/ready"] = func(w http.ResponseWriter, r *http.Request) {
		status := hm.GetOverallStatus()

		w.Header().Set("Content-Type", "application/json")

		if status == HealthStatusUp {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, `{"status":"ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
		} else {
			w.WriteHeader(http.StatusServiceUnavailable)
			fmt.Fprintf(w, `{"status":"not_ready","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
		}
	}

	// Liveness endpoint - simple alive check
	handlers[basePath+"/live"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"alive","timestamp":"%s"}`, time.Now().Format(time.RFC3339))
	}

	return handlers
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
			// Simple memory check - could be enhanced with actual memory metrics
			return HealthCheckResult{
				Status: HealthStatusUp,
				Details: map[string]interface{}{
					"goroutines": "ok", // placeholder for actual goroutine count
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

	// Liveness endpoint - checks if the service is alive
	e.GET(basePath+"/live", func(c echo.Context) error {
		status := h.GetOverallStatus()
		if status == HealthStatusUp {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status":    "live",
				"timestamp": time.Now(),
			})
		}
		return c.JSON(http.StatusServiceUnavailable, map[string]interface{}{
			"status":    "not live",
			"timestamp": time.Now(),
		})
	})
}

// EchoHealthCheckMiddleware adds health check headers to responses.
// Health status is cached for 5 seconds to avoid running all health checks
// on every HTTP request.
func (h *HealthManager) EchoHealthCheckMiddleware() echo.MiddlewareFunc {
	var (
		cacheMu     sync.RWMutex
		cachedAt    time.Time
		cachedValue HealthStatus
	)
	const cacheTTL = 5 * time.Second

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			cacheMu.RLock()
			status := cachedValue
			valid := time.Since(cachedAt) < cacheTTL
			cacheMu.RUnlock()

			if !valid {
				status = h.GetOverallStatus()
				cacheMu.Lock()
				cachedValue = status
				cachedAt = time.Now()
				cacheMu.Unlock()
			}

			c.Response().Header().Set("X-Health-Status", string(status))
			return next(c)
		}
	}
}

// CreateEchoHealthHandler creates a health check handler for specific checks
func (h *HealthManager) CreateEchoHealthHandler(checkName string) echo.HandlerFunc {
	return func(c echo.Context) error {
		h.mu.RLock()
		checker, exists := h.checks[checkName]
		h.mu.RUnlock()

		if !exists {
			return c.JSON(http.StatusNotFound, map[string]string{
				"error": "health check not found",
				"check": checkName,
			})
		}

		result := checker()
		code := http.StatusOK
		if result.Status != HealthStatusUp {
			code = http.StatusServiceUnavailable
		}

		return c.JSON(code, map[string]interface{}{
			"check":     checkName,
			"status":    result.Status,
			"timestamp": result.Timestamp,
			"duration":  result.Duration,
			"details":   result.Details,
		})
	}
}

// RegisterEchoIndividualHealthChecks registers individual health check endpoints
func (h *HealthManager) RegisterEchoIndividualHealthChecks(e *echo.Echo, basePath string) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for name := range h.checks {
		endpoint := basePath + "/check/" + name
		e.GET(endpoint, h.CreateEchoHealthHandler(name))
	}
}
