package grpc

import (
	"context"
	"log"
	"net/http"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"google.golang.org/grpc/metadata"
)

// MountGatewayOnEcho mounts a gRPC gateway mux onto Echo under a base path.
// The base path is stripped before the request reaches the mux, so the mux
// sees proto http-rule paths verbatim (e.g. "/users", not "/api/v1/users").
func MountGatewayOnEcho(e *echo.Echo, gatewayMux *runtime.ServeMux, basePath string) {
	// Create a group for the gateway routes
	gatewayGroup := e.Group(basePath)

	// Mount the entire gateway mux under the base path
	// The "/*" pattern captures all sub-paths; StripPrefix removes the base
	// path so the mux matches proto http-rule patterns like "/users".
	gatewayGroup.Any("/*", echo.WrapHandler(http.StripPrefix(basePath, gatewayMux)))

	// Also route the bare base path ("/api/v1", no trailing slash) to the
	// gateway mux; "/*" does not match it, so without this Echo would 404.
	gatewayGroup.Any("", echo.WrapHandler(http.StripPrefix(basePath, gatewayMux)))

	log.Printf("gRPC Gateway mounted at %s", basePath)
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
