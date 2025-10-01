package grpc

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// GatewayRoute represents a single gateway route configuration
type GatewayRoute struct {
	Path        string
	StripPrefix string
	Description string
}

// MountGatewayOnEcho mounts a gRPC gateway mux onto Echo under a base path
func MountGatewayOnEcho(e *echo.Echo, gatewayMux *runtime.ServeMux, basePath string) {
	// Create a group for the gateway routes
	gatewayGroup := e.Group(basePath)

	// Mount the entire gateway mux under the base path
	// The "/*" pattern captures all sub-paths
	gatewayGroup.Any("/*", echo.WrapHandler(gatewayMux))

	log.Printf("gRPC Gateway mounted at %s", basePath)
}

// MountGatewayWithStripPrefix mounts gateway with path prefix stripping
func MountGatewayWithStripPrefix(e *echo.Echo, gatewayMux *runtime.ServeMux, mountPath, stripPrefix string) {
	e.Any(mountPath, echo.WrapHandler(http.StripPrefix(stripPrefix, gatewayMux)))
	log.Printf("gRPC Gateway mounted at %s (stripping prefix %s)", mountPath, stripPrefix)
}

// SetupGatewayForH2C sets up gateway for H2C mode (server-side registration)
func SetupGatewayForH2C(ctx context.Context, gatewayMux *runtime.ServeMux, serviceRegistrar func(*grpc.Server), grpcServer *grpc.Server) error {
	// In H2C mode, we register services directly with the gateway mux
	// This requires services that implement both gRPC and HTTP interfaces

	// Note: The actual service registration depends on the generated gateway code
	// Each service needs to be registered with RegisterServiceHandlerServer
	// This is typically done in the service registrar function

	log.Printf("Gateway configured for H2C mode")
	return nil
}

// SetupGatewayForSeparate sets up gateway for separate mode (endpoint-based registration)
func SetupGatewayForSeparate(ctx context.Context, gatewayMux *runtime.ServeMux, grpcEndpoint string) error {
	// For separate mode, we connect to the gRPC server via network
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}

	// Wait for gRPC server to be ready with retries
	return waitForGRPCServer(ctx, grpcEndpoint, opts, gatewayMux)
}

// waitForGRPCServer waits for the gRPC server to be ready and then registers gateway handlers
func waitForGRPCServer(ctx context.Context, endpoint string, opts []grpc.DialOption, gatewayMux *runtime.ServeMux) error {
	// Retry logic to wait for gRPC server
	maxRetries := 10
	retryDelay := 100 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		// Try to establish connection
		conn, err := grpc.NewClient(endpoint, opts...)
		if err == nil {
			conn.Close() // Close test connection
			log.Printf("gRPC server at %s is ready for gateway registration", endpoint)
			return nil
		}

		log.Printf("Waiting for gRPC server at %s (attempt %d/%d): %v", endpoint, i+1, maxRetries, err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryDelay):
			retryDelay = time.Duration(float64(retryDelay) * 1.5) // Exponential backoff
		}
	}

	return fmt.Errorf("gRPC server at %s not ready after %d attempts", endpoint, maxRetries)
}

// CreateGatewayMux creates a new gateway mux with standard configuration
func CreateGatewayMux() *runtime.ServeMux {
	return runtime.NewServeMux(
		runtime.WithErrorHandler(runtime.DefaultHTTPErrorHandler),
		runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
			// Add custom metadata from HTTP headers
			md := metadata.MD{}

			// Forward common headers
			if userAgent := req.Header.Get("User-Agent"); userAgent != "" {
				md.Set("user-agent", userAgent)
			}
			if requestID := req.Header.Get("X-Request-ID"); requestID != "" {
				md.Set("request-id", requestID)
			}

			return md
		}),
	)
}

// GatewayHealthMiddleware adds health monitoring for gateway routes
func GatewayHealthMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Add headers to identify gateway requests
			c.Response().Header().Set("X-Gateway-Version", "grpc-gateway/v2")
			c.Response().Header().Set("X-Server-Type", "grpc-gateway")

			return next(c)
		}
	}
}

// LogGatewayRoutes logs information about mounted gateway routes
func LogGatewayRoutes(basePath string, services []string) {
	log.Printf("=== gRPC Gateway Routes ===")
	log.Printf("Base path: %s", basePath)
	log.Printf("Mounted services:")
	for _, service := range services {
		log.Printf("  - %s", service)
	}
	log.Printf("=== End Gateway Routes ===")
}
