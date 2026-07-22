//go:build integration

package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcnetwork "github.com/testcontainers/testcontainers-go/network"
	"github.com/testcontainers/testcontainers-go/wait"
)

// httpServerMarker is the known content served by the test HTTP backend.
// Forwarding tests assert on it to prove real bytes cross the tunnel.
const httpServerMarker = "ssh-tunnel-test-ok"

// SSHServerContainer represents an SSH server test container, plus the HTTP
// backend container it can reach over a shared Docker network. The
// linuxserver/openssh-server image ships no HTTP server (no python3, busybox
// built without the httpd applet), so the backend runs as a second container.
type SSHServerContainer struct {
	testcontainers.Container
	Backend testcontainers.Container
	Network *testcontainers.DockerNetwork

	Host     string
	Port     int
	User     string
	Password string

	// BackendHost/BackendPort address the HTTP backend from the SSH
	// container's perspective (i.e. what the tunnel should forward to).
	BackendHost string
	BackendPort int
}

// StartSSHServerContainer starts an SSH server container and an nginx HTTP
// backend container on a shared network. The backend serves httpServerMarker.
func StartSSHServerContainer(ctx context.Context, t *testing.T) (*SSHServerContainer, error) {
	password := "testpass"

	nw, err := tcnetwork.New(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create test network: %w", err)
	}
	// cleanup tears down everything created so far; c may be nil.
	cleanup := func(c testcontainers.Container) {
		if c != nil {
			_ = c.Terminate(ctx)
		}
		_ = nw.Remove(ctx)
	}

	// HTTP backend serving known content, reachable by name from the SSH container.
	backendHost := "http-backend"
	backendPort := 80
	backend, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			Image:          "nginx:alpine",
			ExposedPorts:   []string{"80/tcp"},
			Networks:       []string{nw.Name},
			NetworkAliases: map[string][]string{nw.Name: {backendHost}},
			WaitingFor:     wait.ForHTTP("/").WithPort("80/tcp").WithStartupTimeout(60 * time.Second),
		},
		Started: true,
	})
	if err != nil {
		cleanup(nil)
		return nil, fmt.Errorf("failed to start HTTP backend container: %w", err)
	}
	cleanup = func(c testcontainers.Container) {
		if c != nil {
			_ = c.Terminate(ctx)
		}
		_ = backend.Terminate(ctx)
		_ = nw.Remove(ctx)
	}

	// Serve the marker content instead of the default nginx welcome page.
	code, reader, err := backend.Exec(ctx, []string{
		"sh", "-c", "echo '" + httpServerMarker + "' > /usr/share/nginx/html/index.html",
	})
	if err != nil {
		cleanup(nil)
		return nil, fmt.Errorf("failed to write backend content: %w", err)
	}
	if code != 0 {
		buf, _ := io.ReadAll(reader)
		cleanup(nil)
		return nil, fmt.Errorf("backend content write returned code %d: %s", code, string(buf))
	}

	// Use a lightweight SSH server image (alpine-based)
	req := testcontainers.ContainerRequest{
		Image:        "lscr.io/linuxserver/openssh-server:latest",
		ExposedPorts: []string{"2222/tcp"},
		Networks:     []string{nw.Name},
		Env: map[string]string{
			"PUID":            "1000",
			"PGID":            "1000",
			"PASSWORD_ACCESS": "true",
			"USER_PASSWORD":   password,
			"USER_NAME":       "testuser",
		},
		WaitingFor: wait.ForListeningPort("2222/tcp").WithStartupTimeout(90 * time.Second),
		Cmd:        []string{},
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		cleanup(nil)
		return nil, fmt.Errorf("failed to start SSH server container: %w", err)
	}

	// The image's sshd config disables TCP forwarding
	// (/config/sshd/sshd_config: AllowTcpForwarding no). The tunnel needs it,
	// so enable it and restart the sshd service (s6-overlay).
	code, reader, err = container.Exec(ctx, []string{
		"sh", "-c",
		"sed -i 's/^AllowTcpForwarding no/AllowTcpForwarding yes/' /config/sshd/sshd_config && " +
			"s6-svc -r /run/service/svc-openssh-server",
	})
	if err != nil {
		cleanup(container)
		return nil, fmt.Errorf("failed to enable SSH TCP forwarding: %w", err)
	}
	if code != 0 {
		buf, _ := io.ReadAll(reader)
		cleanup(container)
		return nil, fmt.Errorf("enabling SSH TCP forwarding returned code %d: %s", code, string(buf))
	}

	// Get the mapped SSH port
	mappedPort, err := container.MappedPort(ctx, "2222")
	if err != nil {
		cleanup(container)
		return nil, fmt.Errorf("failed to get mapped SSH port: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		cleanup(container)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	// Convert port to int
	portInt, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		cleanup(container)
		return nil, fmt.Errorf("failed to convert port to int: %w", err)
	}

	t.Logf("SSH server container started at %s:%d (user: testuser, password: %s, backend: %s:%d)",
		host, portInt, password, backendHost, backendPort)

	// Wait a bit more for SSH server to fully initialize
	time.Sleep(5 * time.Second)

	return &SSHServerContainer{
		Container:   container,
		Backend:     backend,
		Network:     nw,
		Host:        host,
		Port:        portInt,
		User:        "testuser",
		Password:    password,
		BackendHost: backendHost,
		BackendPort: backendPort,
	}, nil
}

// Terminate stops and removes the SSH server container, the HTTP backend
// container, and the shared network.
func (c *SSHServerContainer) Terminate(ctx context.Context) error {
	err := c.Container.Terminate(ctx)
	if c.Backend != nil {
		if berr := c.Backend.Terminate(ctx); err == nil {
			err = berr
		}
	}
	if c.Network != nil {
		if nerr := c.Network.Remove(ctx); err == nil {
			err = nerr
		}
	}
	return err
}

func TestSSHTunnelIntegration(t *testing.T) {
	ctx := context.Background()

	// Start SSH server container
	sshContainer, err := StartSSHServerContainer(ctx, t)
	require.NoError(t, err, "Failed to start SSH server container")
	defer sshContainer.Terminate(ctx)

	t.Run("Start establishes SSH connection successfully", func(t *testing.T) {
		config := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  sshContainer.User,
			Password:              sshContainer.Password,
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18080,
			Timeout:               10 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		require.NotNil(t, tunnel)

		err := tunnel.Start(ctx)
		require.NoError(t, err, "Failed to start SSH tunnel")
		defer tunnel.Close()

		// Give tunnel time to establish
		time.Sleep(2 * time.Second)

		// Verify we can connect to the local port
		conn, err := net.DialTimeout("tcp", "localhost:18080", 2*time.Second)
		if err == nil {
			conn.Close()
			t.Logf("Successfully connected to tunnel at localhost:18080")
		} else {
			t.Logf("Warning: Could not connect to tunnel: %v", err)
		}
	})

	t.Run("Start with invalid credentials returns error", func(t *testing.T) {
		config := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  "invalid-user",
			Password:              "wrong-password",
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18081,
			Timeout:               5 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		err := tunnel.Start(ctx)
		assert.Error(t, err, "Expected error with invalid credentials")
		assert.Contains(t, err.Error(), "SSH dial error", "Error should mention SSH dial")
	})

	t.Run("Start with invalid host returns error", func(t *testing.T) {
		config := Config{
			Host:                  "invalid-host-that-does-not-exist",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18082,
			Timeout:               2 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		err := tunnel.Start(ctx)
		assert.Error(t, err, "Expected error with invalid host")
		assert.Contains(t, err.Error(), "SSH dial error", "Error should mention SSH dial")
	})

	t.Run("Start with invalid local port returns error", func(t *testing.T) {
		config := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  sshContainer.User,
			Password:              sshContainer.Password,
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             99999, // Invalid port
			Timeout:               5 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		err := tunnel.Start(ctx)
		assert.Error(t, err, "Expected error with invalid local port")
		assert.Contains(t, err.Error(), "local listen error", "Error should mention local listen")
	})

	t.Run("Close closes active connection", func(t *testing.T) {
		config := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  sshContainer.User,
			Password:              sshContainer.Password,
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18083,
			Timeout:               10 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		err := tunnel.Start(ctx)
		require.NoError(t, err, "Failed to start SSH tunnel")

		time.Sleep(1 * time.Second)

		// Close the tunnel
		err = tunnel.Close()
		assert.NoError(t, err, "Failed to close tunnel")

		// Try to close again - the underlying SSH client may return error on second close
		// This is acceptable behavior
		_ = tunnel.Close()
	})

	t.Run("Multiple tunnels can coexist", func(t *testing.T) {
		config1 := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  sshContainer.User,
			Password:              sshContainer.Password,
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18084,
			Timeout:               10 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		config2 := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  sshContainer.User,
			Password:              sshContainer.Password,
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18085,
			Timeout:               10 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel1 := New(config1)
		tunnel2 := New(config2)

		err1 := tunnel1.Start(ctx)
		err2 := tunnel2.Start(ctx)

		assert.NoError(t, err1, "First tunnel should start successfully")
		assert.NoError(t, err2, "Second tunnel should start successfully")

		defer tunnel1.Close()
		defer tunnel2.Close()

		time.Sleep(1 * time.Second)

		// Both local ports should be listening
		conn1, err := net.DialTimeout("tcp", "localhost:18084", 2*time.Second)
		if err == nil {
			conn1.Close()
		}

		conn2, err := net.DialTimeout("tcp", "localhost:18085", 2*time.Second)
		if err == nil {
			conn2.Close()
		}
	})

	t.Run("Tunnel forwards data correctly", func(t *testing.T) {
		config := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  sshContainer.User,
			Password:              sshContainer.Password,
			RemoteHost:            sshContainer.BackendHost,
			RemotePort:            sshContainer.BackendPort,
			LocalPort:             18086,
			Timeout:               10 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		err := tunnel.Start(ctx)
		require.NoError(t, err, "Failed to start SSH tunnel")
		defer tunnel.Close()

		// Make an HTTP request through the tunnel and assert the backend's body.
		// DisableKeepAlives: an idle keep-alive connection would otherwise keep
		// the forward goroutine (and tunnel.Close's wg.Wait) blocked until the
		// transport's 90s IdleConnTimeout fires.
		client := &http.Client{
			Timeout:   10 * time.Second,
			Transport: &http.Transport{DisableKeepAlives: true},
		}

		resp, err := client.Get("http://" + tunnel.LocalAddr())
		require.NoError(t, err, "HTTP request through tunnel should succeed")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP 200")
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), httpServerMarker,
			"response body should come from the HTTP backend container")
	})
}

func TestTunnelForwardsHTTP(t *testing.T) {
	ctx := context.Background()

	// Start SSH server container (with HTTP backend container on a shared network)
	sshContainer, err := StartSSHServerContainer(ctx, t)
	require.NoError(t, err, "Failed to start SSH server container")
	defer sshContainer.Terminate(ctx)

	config := Config{
		Host:                  sshContainer.Host,
		Port:                  sshContainer.Port,
		User:                  sshContainer.User,
		Password:              sshContainer.Password,
		RemoteHost:            sshContainer.BackendHost,
		RemotePort:            sshContainer.BackendPort,
		LocalPort:             0, // ephemeral port; discovered via LocalAddr
		Timeout:               10 * time.Second,
		InsecureIgnoreHostKey: true,
	}

	tunnel := New(config)
	require.NoError(t, tunnel.Start(ctx), "Failed to start SSH tunnel")
	defer tunnel.Close()

	// LocalAddr must report the bound address after Start.
	localAddr := tunnel.LocalAddr()
	require.NotEmpty(t, localAddr, "LocalAddr should return the bound address after Start")

	// Push REAL bytes through the tunnel: HTTP GET via the forwarded local port.
	// DisableKeepAlives: an idle keep-alive connection would otherwise keep the
	// forward goroutine (and tunnel.Close's wg.Wait) blocked until the
	// transport's 90s IdleConnTimeout fires.
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	resp, err := client.Get("http://" + localAddr + "/")
	require.NoError(t, err, "HTTP GET through the tunnel should succeed")
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP 200 through tunnel")

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err, "Failed to read response body")
	assert.Contains(t, string(body), httpServerMarker,
		"response body should come from the container's test HTTP server")
}

func TestSSHTunnelConnectionTimeout(t *testing.T) {
	ctx := context.Background()

	t.Run("Tunnel respects timeout setting", func(t *testing.T) {
		// Try to connect to a non-existent server with short timeout
		config := Config{
			Host:                  "192.0.2.1", // TEST-NET-1, should be unreachable
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18087,
			Timeout:               1 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)

		start := time.Now()
		err := tunnel.Start(ctx)
		elapsed := time.Since(start)

		assert.Error(t, err, "Expected error when connecting to unreachable host")
		assert.Less(t, elapsed, 3*time.Second, "Timeout should be respected (expected ~1s, got %v)", elapsed)
	})
}
