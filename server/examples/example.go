//go:build example

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jasoet/pkg/logging"
	"github.com/jasoet/pkg/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
)

// Example data structures
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services"`
}

type CustomHealthChecker struct {
	services map[string]func() error
}

func (c *CustomHealthChecker) CheckHealth() map[string]string {
	results := make(map[string]string)
	for name, check := range c.services {
		if err := check(); err != nil {
			results[name] = fmt.Sprintf("unhealthy: %v", err)
		} else {
			results[name] = "healthy"
		}
	}
	return results
}

func main() {
	// Initialize logging
	logging.Initialize("server-examples", true)

	fmt.Println("Server Package Examples")
	fmt.Println("=======================")

	// Run different server examples in sequence
	examples := []struct {
		name string
		fn   func()
	}{
		{"Basic Server Setup", basicServerExample},
		{"Server with Custom Configuration", customConfigExample},
		{"Server with Custom Routes and Middleware", customRoutesExample},
		{"Server with Health Checks", healthChecksExample},
		{"Server with Graceful Shutdown", gracefulShutdownExample},
	}

	for i, example := range examples {
		fmt.Printf("\n%d. %s\n", i+1, example.name)
		fmt.Println(strings.Repeat("-", len(example.name)+4))
		example.fn()

		if i < len(examples)-1 {
			fmt.Println("\nPress Enter to continue to next example...")
			fmt.Scanln()
		}
	}
}

func basicServerExample() {
	fmt.Println("Creating a basic HTTP server with default configuration...")

	// Create server with minimal configuration
	config := server.Config{
		Port: 8080,
		// Note: The server package handles graceful shutdown automatically
		// with StartWithConfig(), but for demonstration we'll show the configuration
	}

	fmt.Printf("Server configuration:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Health endpoints: /health, /health/ready, /health/live\n")
	fmt.Printf("- Metrics endpoint: /metrics\n")
	fmt.Printf("- Built-in middleware: logging, CORS, recover\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("server.StartWithConfig(config)")
	fmt.Println("\nNote: StartWithConfig() handles graceful shutdown automatically")
	fmt.Println("For a running example, see other functions in this file.")
	fmt.Println("✓ Basic server example completed")
}

func customConfigExample() {
	fmt.Println("Creating server with custom configuration...")

	// Custom configuration with various options
	config := server.Config{
		Port:            8081,
		ShutdownTimeout: 10 * time.Second,
		EnableMetrics:   true,
		MetricsPath:     "/custom-metrics",
	}

	fmt.Printf("Custom server configuration:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Shutdown Timeout: %v\n", config.ShutdownTimeout)
	fmt.Printf("- Custom Metrics Path: %s\n", config.MetricsPath)

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("server.StartWithConfig(config)")
	fmt.Println("\nNote: The server package automatically provides health endpoints and graceful shutdown")
	fmt.Println("✓ Custom configuration example completed")
}

func customRoutesExample() {
	fmt.Println("Creating server with custom routes and middleware...")

	config := server.Config{
		Port: 8082,
		EchoConfigurer: func(e *echo.Echo) {
			// Add custom middleware
			e.Use(middleware.RequestID())
			e.Use(customLoggingMiddleware())

			// Add custom routes
			api := e.Group("/api/v1")
			api.Use(authMiddleware())

			// User endpoints
			api.GET("/users", getUsersHandler)
			api.GET("/users/:id", getUserHandler)
			api.POST("/users", createUserHandler)
			api.PUT("/users/:id", updateUserHandler)
			api.DELETE("/users/:id", deleteUserHandler)

			// Admin endpoints
			admin := api.Group("/admin")
			admin.Use(adminMiddleware())
			admin.GET("/stats", getStatsHandler)

			// Public endpoints (no auth required)
			e.GET("/public/info", getInfoHandler)
		},
	}

	fmt.Printf("Server with custom routes:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- API Routes: /api/v1/users/*\n")
	fmt.Printf("- Admin Routes: /api/v1/admin/*\n")
	fmt.Printf("- Public Routes: /public/*\n")
	fmt.Printf("- Custom Middleware: RequestID, Auth, Logging\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("server.StartWithConfig(config)")
	fmt.Println("✓ Custom routes example completed")
}

func healthChecksExample() {
	fmt.Println("Creating server with custom health checks...")

	// Create custom health checker
	healthChecker := &CustomHealthChecker{
		services: map[string]func() error{
			"database": func() error {
				// Simulate database health check
				return nil // Healthy
			},
			"redis": func() error {
				// Simulate Redis health check
				return fmt.Errorf("connection failed") // Unhealthy
			},
			"external_api": func() error {
				// Simulate external API health check
				return nil // Healthy
			},
		},
	}

	config := server.Config{
		Port: 8083,
		EchoConfigurer: func(e *echo.Echo) {
			// Add custom health endpoint with the health checker
			e.GET("/custom-health", func(c echo.Context) error {
				results := healthChecker.CheckHealth()
				status := HealthStatus{
					Status:    "ok",
					Timestamp: time.Now(),
					Services:  results,
				}
				return c.JSON(http.StatusOK, status)
			})
		},
	}

	fmt.Printf("Server with custom health checks:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Health services: database, redis, external_api\n")
	fmt.Printf("- Custom health endpoint: /custom-health\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("server.StartWithConfig(config)")
	fmt.Println("✓ Health checks example completed")
}

func gracefulShutdownExample() {
	fmt.Println("Demonstrating graceful shutdown with signal handling...")

	config := server.Config{
		Port:            8084,
		ShutdownTimeout: 10 * time.Second,
		EchoConfigurer: func(e *echo.Echo) {
			// Add a long-running endpoint for testing graceful shutdown
			e.GET("/slow", func(c echo.Context) error {
				fmt.Println("   Processing slow request...")
				time.Sleep(5 * time.Second)
				return c.JSON(200, map[string]string{"status": "completed"})
			})
		},
	}

	fmt.Printf("Server with graceful shutdown:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Shutdown Timeout: %v\n", config.ShutdownTimeout)
	fmt.Printf("- Automatic signal handling (SIGINT, SIGTERM)\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("server.StartWithConfig(config)")
	fmt.Println("\nNote: StartWithConfig() automatically handles graceful shutdown")
	fmt.Println("✓ Graceful shutdown example completed")
}

// Helper functions for testing endpoints

func testEndpoints(port int) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	endpoints := []string{"/health", "/health/ready", "/health/live", "/metrics"}

	for _, endpoint := range endpoints {
		url := baseURL + endpoint
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", endpoint, err)
			continue
		}
		resp.Body.Close()
		fmt.Printf("✓ %s: %d\n", endpoint, resp.StatusCode)
	}
}

func testCustomEndpoints(port int, config *server.Config) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	endpoints := []string{
		config.HealthPath,
		config.ReadyPath,
		config.LivePath,
		config.MetricsPath,
	}

	for _, endpoint := range endpoints {
		url := baseURL + endpoint
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", endpoint, err)
			continue
		}
		resp.Body.Close()
		fmt.Printf("✓ %s: %d\n", endpoint, resp.StatusCode)
	}
}

func testAPIEndpoints(port int) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	tests := []struct {
		method   string
		endpoint string
		expected int
	}{
		{"GET", "/public/info", 200},
		{"GET", "/api/v1/users", 200},
		{"GET", "/api/v1/users/1", 200},
		{"GET", "/api/v1/admin/stats", 403}, // Should fail without admin token
		{"POST", "/api/v1/users", 201},
	}

	for _, test := range tests {
		url := baseURL + test.endpoint

		var resp *http.Response
		var err error

		switch test.method {
		case "GET":
			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer user-token")
			resp, err = http.DefaultClient.Do(req)
		case "POST":
			req, _ := http.NewRequest("POST", url, strings.NewReader(`{"name":"Test User","email":"test@example.com"}`))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer user-token")
			resp, err = http.DefaultClient.Do(req)
		}

		if err != nil {
			fmt.Printf("✗ %s %s: %v\n", test.method, test.endpoint, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == test.expected {
			fmt.Printf("✓ %s %s: %d\n", test.method, test.endpoint, resp.StatusCode)
		} else {
			fmt.Printf("? %s %s: %d (expected %d)\n", test.method, test.endpoint, resp.StatusCode, test.expected)
		}
	}
}

func testHealthEndpoints(port int) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	endpoints := []string{"/health", "/health/ready", "/health/live"}

	for _, endpoint := range endpoints {
		url := baseURL + endpoint
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", endpoint, err)
			continue
		}

		if endpoint == "/health" {
			// Parse and display health details
			var healthStatus map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&healthStatus)
			fmt.Printf("✓ %s: %d\n", endpoint, resp.StatusCode)
			if services, ok := healthStatus["services"].(map[string]interface{}); ok {
				for service, status := range services {
					fmt.Printf("  - %s: %v\n", service, status)
				}
			}
		} else {
			fmt.Printf("✓ %s: %d\n", endpoint, resp.StatusCode)
		}

		resp.Body.Close()
	}
}

// Custom middleware and handlers

func customLoggingMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			err := next(c)

			req := c.Request()
			res := c.Response()

			logger := logging.ContextLogger(req.Context(), "http-request")
			logger.Info().
				Str("method", req.Method).
				Str("path", req.URL.Path).
				Int("status", res.Status).
				Dur("duration", time.Since(start)).
				Str("remote_ip", c.RealIP()).
				Msg("HTTP request")

			return err
		}
	}
}

func authMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			authHeader := c.Request().Header.Get("Authorization")
			if authHeader == "" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Missing authorization header")
			}

			// Simple token validation (in real apps, use proper JWT validation)
			if !strings.HasPrefix(authHeader, "Bearer ") {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid authorization format")
			}

			token := strings.TrimPrefix(authHeader, "Bearer ")
			if token != "user-token" && token != "admin-token" {
				return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
			}

			// Store user info in context
			c.Set("token", token)
			c.Set("is_admin", token == "admin-token")

			return next(c)
		}
	}
}

func adminMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isAdmin := c.Get("is_admin")
			if isAdmin != true {
				return echo.NewHTTPError(http.StatusForbidden, "Admin access required")
			}
			return next(c)
		}
	}
}

// API Handlers

func getUsersHandler(c echo.Context) error {
	users := []User{
		{ID: 1, Name: "Alice", Email: "alice@example.com"},
		{ID: 2, Name: "Bob", Email: "bob@example.com"},
	}
	return c.JSON(http.StatusOK, users)
}

func getUserHandler(c echo.Context) error {
	id := c.Param("id")
	user := User{ID: 1, Name: "Alice", Email: "alice@example.com"}

	// Simulate user lookup by ID
	if id == "1" {
		return c.JSON(http.StatusOK, user)
	}

	return echo.NewHTTPError(http.StatusNotFound, "User not found")
}

func createUserHandler(c echo.Context) error {
	var user User
	if err := c.Bind(&user); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Simulate user creation
	user.ID = 3

	logger := logging.ContextLogger(c.Request().Context(), "user-api")
	logger.Info().
		Int("user_id", user.ID).
		Str("user_name", user.Name).
		Msg("User created")

	return c.JSON(http.StatusCreated, user)
}

func updateUserHandler(c echo.Context) error {
	id := c.Param("id")
	var user User
	if err := c.Bind(&user); err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "Invalid request body")
	}

	// Simulate user update
	user.ID = 1 // Use ID from path

	logger := logging.ContextLogger(c.Request().Context(), "user-api")
	logger.Info().
		Str("user_id", id).
		Str("user_name", user.Name).
		Msg("User updated")

	return c.JSON(http.StatusOK, user)
}

func deleteUserHandler(c echo.Context) error {
	id := c.Param("id")

	logger := logging.ContextLogger(c.Request().Context(), "user-api")
	logger.Info().Str("user_id", id).Msg("User deleted")

	return c.NoContent(http.StatusNoContent)
}

func getStatsHandler(c echo.Context) error {
	stats := map[string]interface{}{
		"total_users":    150,
		"active_users":   120,
		"total_requests": 1000,
		"uptime":         "2h 30m",
	}
	return c.JSON(http.StatusOK, stats)
}

func getInfoHandler(c echo.Context) error {
	info := map[string]interface{}{
		"service":     "server-examples",
		"version":     "1.0.0",
		"environment": "development",
		"timestamp":   time.Now(),
	}
	return c.JSON(http.StatusOK, info)
}
