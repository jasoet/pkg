//go:build example

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	"github.com/jasoet/pkg/v2/server"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
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
	fmt.Println("Server Package Examples (v2 with OpenTelemetry)")
	fmt.Println("===============================================")

	// Run different server examples in sequence
	examples := []struct {
		name string
		fn   func()
	}{
		{"Basic Server Setup", basicServerExample},
		{"Server with OpenTelemetry Configuration", otelConfigExample},
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

	operation := func(e *echo.Echo) {
		// Register custom routes
		e.GET("/hello", func(c echo.Context) error {
			return c.String(http.StatusOK, "Hello, World!")
		})
	}

	shutdown := func(e *echo.Echo) {
		// Cleanup resources
		fmt.Println("Cleaning up resources...")
	}

	// Create server with minimal configuration
	config := server.DefaultConfig(8080, operation, shutdown)

	fmt.Printf("Server configuration:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Health endpoints: /health, /health/ready, /health/live\n")
	fmt.Printf("- OpenTelemetry: disabled (nil)\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("if err := server.StartWithConfig(config); err != nil { log.Fatal(err) }")
	fmt.Println("\nNote: Without OTelConfig, no request logging or telemetry is enabled")
	fmt.Println("Basic server example completed")
}

func otelConfigExample() {
	fmt.Println("Creating server with OpenTelemetry configuration...")

	operation := func(e *echo.Echo) {
		e.GET("/api/hello", func(c echo.Context) error {
			return c.JSON(http.StatusOK, map[string]string{
				"message": "Hello with telemetry!",
			})
		})
	}

	shutdown := func(e *echo.Echo) {
		fmt.Println("Cleaning up resources...")
	}

	// OTel is configured at the middleware level, not on server.Config.
	// Create an OTel config and use it in Echo middleware:
	otelCfg := otel.NewConfig("server-example").
		WithServiceVersion("1.0.0")

	config := server.DefaultConfig(8081, operation, shutdown)
	config.ShutdownTimeout = 15 * time.Second
	// OTel middleware can be added via config.Middleware or EchoConfigurer
	// Example: e.Use(otelecho.Middleware(otelCfg.ServiceName))

	fmt.Printf("Server configuration:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Shutdown Timeout: %v\n", config.ShutdownTimeout)
	fmt.Printf("- OTel Config Service: %s\n", otelCfg.ServiceName)

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("if err := server.StartWithConfig(config); err != nil { log.Fatal(err) }")
	fmt.Println("\nNote: OTel is configured via Echo middleware, not server.Config")
	fmt.Println("OpenTelemetry configuration example completed")
}

func customRoutesExample() {
	fmt.Println("Creating server with custom routes and middleware...")

	operation := func(e *echo.Echo) {
		// Add custom middleware
		e.Use(middleware.RequestID())

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
	}

	shutdown := func(e *echo.Echo) {
		fmt.Println("Shutting down API server...")
	}

	config := server.DefaultConfig(8082, operation, shutdown)

	fmt.Printf("Server with custom routes:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- API Routes: /api/v1/users/*\n")
	fmt.Printf("- Admin Routes: /api/v1/admin/*\n")
	fmt.Printf("- Public Routes: /public/*\n")
	fmt.Printf("- Custom Middleware: RequestID, Auth, Logging\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("if err := server.StartWithConfig(config); err != nil { log.Fatal(err) }")
	fmt.Println("Custom routes example completed")
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

	operation := func(e *echo.Echo) {
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
	}

	shutdown := func(e *echo.Echo) {
		fmt.Println("Closing health check connections...")
	}

	config := server.DefaultConfig(8083, operation, shutdown)

	fmt.Printf("Server with custom health checks:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Health services: database, redis, external_api\n")
	fmt.Printf("- Custom health endpoint: /custom-health\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("if err := server.StartWithConfig(config); err != nil { log.Fatal(err) }")
	fmt.Println("Health checks example completed")
}

func gracefulShutdownExample() {
	fmt.Println("Demonstrating graceful shutdown with signal handling...")

	operation := func(e *echo.Echo) {
		// Add a long-running endpoint for testing graceful shutdown
		e.GET("/slow", func(c echo.Context) error {
			fmt.Println("   Processing slow request...")
			time.Sleep(5 * time.Second)
			return c.JSON(200, map[string]string{"status": "completed"})
		})
	}

	shutdown := func(e *echo.Echo) {
		fmt.Println("Waiting for in-flight requests to complete...")
		time.Sleep(1 * time.Second)
		fmt.Println("All requests completed, shutting down gracefully")
	}

	config := server.DefaultConfig(8084, operation, shutdown)
	config.ShutdownTimeout = 10 * time.Second

	fmt.Printf("Server with graceful shutdown:\n")
	fmt.Printf("- Port: %d\n", config.Port)
	fmt.Printf("- Shutdown Timeout: %v\n", config.ShutdownTimeout)
	fmt.Printf("- Automatic signal handling (SIGINT, SIGTERM)\n")

	fmt.Println("\nTo start this server, you would call:")
	fmt.Println("if err := server.StartWithConfig(config); err != nil { log.Fatal(err) }")
	fmt.Println("\nNote: StartWithConfig() automatically handles graceful shutdown")
	fmt.Println("Graceful shutdown example completed")
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

func testCustomEndpoints(port int) {
	baseURL := fmt.Sprintf("http://localhost:%d", port)

	endpoints := []string{
		"/health",
		"/health/ready",
		"/health/live",
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
	fmt.Printf("User created: ID=%d, Name=%s\n", user.ID, user.Name)

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
	fmt.Printf("User updated: ID=%s, Name=%s\n", id, user.Name)

	return c.JSON(http.StatusOK, user)
}

func deleteUserHandler(c echo.Context) error {
	id := c.Param("id")
	fmt.Printf("User deleted: ID=%s\n", id)

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
