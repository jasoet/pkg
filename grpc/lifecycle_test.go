package grpc

import (
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// freePort returns an available TCP port on localhost.
func freePort(t *testing.T) string {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer l.Close()
	return fmt.Sprintf("%d", l.Addr().(*net.TCPAddr).Port)
}

// waitForPort polls until the given port accepts TCP connections or the
// timeout elapses, failing the test in the latter case.
func waitForPort(t *testing.T, port string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:"+port, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("port %s did not become ready within %v", port, timeout)
}

// recvWithTimeout receives an error from ch, failing the test on timeout so a
// regression hangs fail fast instead of blocking the suite.
func recvWithTimeout(t *testing.T, ch chan error, timeout time.Duration) error {
	t.Helper()
	select {
	case err := <-ch:
		return err
	case <-time.After(timeout):
		t.Fatalf("timed out after %v waiting for Start to return", timeout)
		return nil // unreachable
	}
}

// TestServerRestartStoppable verifies that a server can go through full
// Start -> Stop -> Start -> Stop cycles: after a Stop, a second Start must
// work (gRPC server rebuilt) and a second Stop must actually shut everything
// down (shutdownOnce re-armed), leaving IsRunning()==false.
func TestServerRestartStoppable(t *testing.T) {
	grpcPort := freePort(t)
	httpPort := freePort(t)

	server, err := New(
		WithSeparateMode(grpcPort, httpPort),
		WithShutdownTimeout(5*time.Second),
	)
	require.NoError(t, err)

	startErr := make(chan error, 2)

	for cycle := 1; cycle <= 2; cycle++ {
		t.Run(fmt.Sprintf("cycle%d", cycle), func(t *testing.T) {
			go func() { startErr <- server.Start() }()

			waitForPort(t, grpcPort, 5*time.Second)
			waitForPort(t, httpPort, 5*time.Second)
			assert.True(t, server.IsRunning(), "server should report running after Start (cycle %d)", cycle)

			require.NoError(t, server.Stop(), "Stop must succeed (cycle %d)", cycle)

			err := recvWithTimeout(t, startErr, 10*time.Second)
			assert.NoError(t, err, "Start must return nil after graceful Stop (cycle %d)", cycle)
			assert.False(t, server.IsRunning(), "server must not report running after Stop (cycle %d)", cycle)

			// Both ports must actually be released after Stop.
			deadline := time.Now().Add(5 * time.Second)
			for time.Now().Before(deadline) {
				conn, dialErr := net.DialTimeout("tcp", "127.0.0.1:"+httpPort, 100*time.Millisecond)
				if dialErr != nil {
					break
				}
				conn.Close()
				time.Sleep(20 * time.Millisecond)
			}
			conn, dialErr := net.DialTimeout("tcp", "127.0.0.1:"+httpPort, 100*time.Millisecond)
			if dialErr == nil {
				conn.Close()
				t.Fatalf("HTTP port %s still accepting connections after Stop (cycle %d)", httpPort, cycle)
			}
		})
	}
}

// TestServerFailedStartNotRunning verifies that a failed Start (e.g. busy
// gRPC port) rolls back the running flag: IsRunning() must be false after
// Start returns an error.
func TestServerFailedStartNotRunning(t *testing.T) {
	// Occupy the gRPC port on the wildcard address so the server's own
	// wildcard bind fails with "address already in use".
	busy, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer busy.Close()
	busyPort := fmt.Sprintf("%d", busy.Addr().(*net.TCPAddr).Port)

	server, err := New(
		WithSeparateMode(busyPort, freePort(t)),
		WithShutdownTimeout(2*time.Second),
	)
	require.NoError(t, err)

	startErr := make(chan error, 1)
	go func() { startErr <- server.Start() }()

	err = recvWithTimeout(t, startErr, 5*time.Second)
	require.Error(t, err, "Start must fail on a busy gRPC port")
	assert.False(t, server.IsRunning(), "failed Start must not leave the server marked running")
}

// TestServerCleanShutdownReturnsNil verifies that a graceful Stop causes the
// blocking Start call to return nil (not http.ErrServerClosed) in both
// Separate and H2C modes.
func TestServerCleanShutdownReturnsNil(t *testing.T) {
	modes := []struct {
		name string
		opts []Option
	}{
		{"Separate", []Option{WithSeparateMode(freePort(t), freePort(t))}},
		{"H2C", []Option{WithH2CMode(), WithGRPCPort(freePort(t))}},
	}

	for _, m := range modes {
		t.Run(m.name, func(t *testing.T) {
			opts := append(m.opts, WithShutdownTimeout(5*time.Second))
			server, err := New(opts...)
			require.NoError(t, err)

			startErr := make(chan error, 1)
			go func() { startErr <- server.Start() }()

			waitForPort(t, server.config.grpcPort, 5*time.Second)

			require.NoError(t, server.Stop())

			err = recvWithTimeout(t, startErr, 10*time.Second)
			assert.NoError(t, err, "Start must return nil on clean graceful shutdown, not http.ErrServerClosed")
			assert.False(t, server.IsRunning())
		})
	}
}
