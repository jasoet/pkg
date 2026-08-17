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
		runtime.WithMetadata(gatewayMetadataAnnotator),
	)
}

// gatewayMetadataAnnotator maps incoming HTTP request headers onto the gRPC
// metadata forwarded to the backend. Besides common headers, it propagates W3C
// Trace Context so the downstream gRPC span links to the gateway request
// instead of starting a new root trace.
func gatewayMetadataAnnotator(_ context.Context, req *http.Request) metadata.MD {
	md := metadata.MD{}

	// Forward common headers
	if userAgent := req.Header.Get("User-Agent"); userAgent != "" {
		md.Set("user-agent", userAgent)
	}
	if requestID := req.Header.Get("X-Request-ID"); requestID != "" {
		md.Set("request-id", requestID)
	}

	// First forward any inbound traceparent/tracestate headers verbatim (covers
	// pass-through when no local span is active). Then inject the active span
	// from the request context: when the Echo tracing middleware ran ahead of
	// the gateway it replaced the request context with one carrying the HTTP
	// server span, and Inject writes a traceparent for that span, linking the
	// HTTP and gRPC spans.
	if tp := req.Header.Get("traceparent"); tp != "" {
		md.Set("traceparent", tp)
	}
	if ts := req.Header.Get("tracestate"); ts != "" {
		md.Set("tracestate", ts)
	}
	grpcPropagator.Inject(req.Context(), metadataCarrier(md))

	return md
}
