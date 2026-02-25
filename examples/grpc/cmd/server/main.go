//go:build example

package main

import (
	"log"
	"os"

	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"

	grpcserver "github.com/jasoet/pkg/v2/grpc"
	calculatorv1 "github.com/jasoet/pkg/v2/examples/grpc/gen/calculator/v1"
	"github.com/jasoet/pkg/v2/examples/grpc/internal/service"
)

func main() {
	port := "50051"
	if p := os.Getenv("PORT"); p != "" {
		port = p
	}

	log.Printf("Starting Calculator gRPC server example on port %s", port)
	log.Printf("This example demonstrates:")
	log.Printf("  - Unary RPC (Add, Subtract, Multiply, Divide)")
	log.Printf("  - Server streaming RPC (Factorial)")
	log.Printf("  - Client streaming RPC (Sum)")
	log.Printf("  - Bidirectional streaming RPC (RunningAverage)")

	// Service registrar for calculator service
	serviceRegistrar := func(srv *grpc.Server) {
		calculatorService := service.NewCalculatorService()
		calculatorv1.RegisterCalculatorServiceServer(srv, calculatorService)
		log.Printf("Calculator service registered")
	}

	// Start the server using our reusable component with options pattern
	if err := grpcserver.Start(port, serviceRegistrar,
		// Server mode
		grpcserver.WithH2CMode(),

		// Enable features
		grpcserver.WithReflection(),
		grpcserver.WithHealthCheck(),
		grpcserver.WithMetrics(),

		// Configure Echo with custom routes
		grpcserver.WithEchoConfigurer(func(e *echo.Echo) {
			// Add a status endpoint
			e.GET("/status", func(c echo.Context) error {
				return c.JSON(200, map[string]interface{}{
					"service":     "calculator",
					"status":      "running",
					"description": "gRPC Calculator Service with Echo integration",
					"endpoints": map[string]string{
						"grpc_gateway": "/api/v1/",
						"health":       "/health",
						"metrics":      "/metrics",
						"status":       "/status",
						"calculator":   "/calculator",
					},
				})
			})

			// Add a simple calculator endpoint via REST (in addition to gRPC)
			e.GET("/calculator", func(c echo.Context) error {
				return c.JSON(200, map[string]interface{}{
					"calculator": "REST Calculator API",
					"operations": []string{"add", "subtract", "multiply", "divide"},
					"usage":      "Use /calculator/{operation}?a=1&b=2",
					"grpc":       "Full gRPC API available via /api/v1/",
				})
			})

			// Add simple REST calculator operations
			e.GET("/calculator/add", func(c echo.Context) error {
				a := c.QueryParam("a")
				b := c.QueryParam("b")
				if a == "" || b == "" {
					return c.JSON(400, map[string]string{"error": "Missing parameters a and b"})
				}
				return c.JSON(200, map[string]string{
					"operation": "add",
					"a":         a,
					"b":         b,
					"note":      "Use gRPC API for actual calculation with type safety",
				})
			})

			log.Printf("Custom Echo routes configured:")
			log.Printf("  - Status: http://localhost:%s/status", port)
			log.Printf("  - Calculator info: http://localhost:%s/calculator", port)
		}),
	); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
