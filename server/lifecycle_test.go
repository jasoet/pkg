package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForAddr polls srv.Addr until the listener is bound and returns the
// bound address (host:port). Useful with Port 0 where the OS assigns the port.
func waitForAddr(t *testing.T, srv *Server) string {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if addr := srv.Addr(); addr != "" {
			return addr
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("server did not start listening within timeout")
	return ""
}

// addrPort extracts the port from a listener address like "[::]:8080" or "0.0.0.0:8080".
func addrPort(addr string) string {
	return addr[strings.LastIndex(addr, ":")+1:]
}

func TestServerStartShutdown(t *testing.T) {
	srv, err := New(WithPort(0))
	require.NoError(t, err)
	require.NotNil(t, srv.Echo(), "Echo instance should be available before Start")

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()

	addr := waitForAddr(t, srv)

	resp, err := http.Get("http://localhost:" + addrPort(addr) + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `{"status":"UP"}`, strings.TrimSpace(string(body)))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, srv.Shutdown(ctx))

	select {
	case err := <-startErr:
		assert.NoError(t, err, "Start should return nil on clean Shutdown")
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Shutdown")
	}
}

func TestServerShutdownTimeout(t *testing.T) {
	var shutdownCalled atomic.Bool
	srv, err := New(
		WithPort(0),
		WithShutdownTimeout(5*time.Second),
		WithShutdown(func(e *echo.Echo) { shutdownCalled.Store(true) }),
	)
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()
	waitForAddr(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, srv.Shutdown(ctx))
	assert.True(t, shutdownCalled.Load(), "Shutdown callback should be invoked on Shutdown")

	select {
	case err := <-startErr:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Shutdown")
	}
}

func TestServerStartTwiceFails(t *testing.T) {
	srv, err := New(WithPort(0))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()
	waitForAddr(t, srv)

	err = srv.Start()
	require.Error(t, err, "second Start while running should return an error")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, srv.Shutdown(ctx))

	select {
	case err := <-startErr:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return after Shutdown")
	}
}

func TestNewInvalidPort(t *testing.T) {
	_, err := New(WithPort(-1))
	assert.Error(t, err, "New should reject negative port")

	_, err = New(WithPort(70000))
	assert.Error(t, err, "New should reject port above 65535")
}

func TestServerRestartAfterShutdownFails(t *testing.T) {
	srv, err := New(WithPort(0))
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- srv.Start() }()
	waitForAddr(t, srv)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	require.NoError(t, srv.Shutdown(ctx))
	require.NoError(t, <-startErr)

	err = srv.Start()
	require.Error(t, err, "Start after Shutdown must fail (no silent no-op restart)")
	assert.Contains(t, err.Error(), "cannot be restarted")

	// Addr must not report a stale listener after shutdown.
	assert.Empty(t, srv.Addr())

	// Shutdown is idempotent: callback and drain run exactly once.
	require.NoError(t, srv.Shutdown(ctx))
}
