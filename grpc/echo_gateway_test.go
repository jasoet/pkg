package grpc

import (
	"testing"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
)

func TestMountGatewayOnEcho(t *testing.T) {
	e := echo.New()
	mux := runtime.NewServeMux()

	MountGatewayOnEcho(e, mux, "/api/v1")

	// Verify that routes were registered
	routes := e.Routes()
	found := false
	for _, route := range routes {
		if route.Path == "/api/v1/*" {
			found = true
			break
		}
	}
	assert.True(t, found, "Expected gateway route to be registered")
}

func TestCreateGatewayMux(t *testing.T) {
	mux := CreateGatewayMux()
	assert.NotNil(t, mux)

	// The mux should be properly configured
	// We can't easily test the internal configuration, but we can verify it was created
}

func TestCreateGatewayMuxMetadata(t *testing.T) {
	// Test that CreateGatewayMux creates a mux with metadata forwarding
	// We can't easily test the internal metadata function directly,
	// but we can verify the mux is created
	mux := CreateGatewayMux()
	assert.NotNil(t, mux)
}
