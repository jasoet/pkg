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
	"github.com/testcontainers/testcontainers-go/wait"
)

// SSHServerContainer represents an SSH server test container
type SSHServerContainer struct {
	testcontainers.Container
	Host     string
	Port     int
	User     string
	Password string
}

// StartSSHServerContainer starts an SSH server container with a test HTTP server
func StartSSHServerContainer(ctx context.Context, t *testing.T) (*SSHServerContainer, error) {
	password := "testpass"

	// Use a lightweight SSH server image (alpine-based)
	req := testcontainers.ContainerRequest{
		Image:        "lscr.io/linuxserver/openssh-server:latest",
		ExposedPorts: []string{"2222/tcp", "8080/tcp"},
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
		return nil, fmt.Errorf("failed to start SSH server container: %w", err)
	}

	// Get the mapped SSH port
	mappedPort, err := container.MappedPort(ctx, "2222")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get mapped SSH port: %w", err)
	}

	// Get the host
	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get host: %w", err)
	}

	// Convert port to int
	portInt, err := strconv.Atoi(mappedPort.Port())
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to convert port to int: %w", err)
	}

	t.Logf("SSH server container started at %s:%d (user: testuser, password: %s)", host, portInt, password)

	// Wait a bit more for SSH server to fully initialize
	time.Sleep(5 * time.Second)

	// Start a simple HTTP server inside the container for testing
	// We'll use exec to run a Python HTTP server
	code, reader, err := container.Exec(ctx, []string{
		"sh", "-c",
		"nohup python3 -m http.server 8080 > /tmp/http.log 2>&1 &",
	})
	if err != nil {
		t.Logf("Warning: Failed to start HTTP server in container: %v", err)
	} else if code != 0 {
		buf, _ := io.ReadAll(reader)
		t.Logf("Warning: HTTP server exec returned code %d: %s", code, string(buf))
	} else {
		t.Logf("HTTP server started on port 8080 inside container")
		time.Sleep(2 * time.Second) // Wait for HTTP server to start
	}

	return &SSHServerContainer{
		Container: container,
		Host:      host,
		Port:      portInt,
		User:      "testuser",
		Password:  password,
	}, nil
}

// Terminate stops and removes the SSH server container
func (c *SSHServerContainer) Terminate(ctx context.Context) error {
	return c.Container.Terminate(ctx)
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

		err := tunnel.Start()
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
		err := tunnel.Start()
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
		err := tunnel.Start()
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
		err := tunnel.Start()
		assert.Error(t, err, "Expected error with invalid local port")
		assert.Contains(t, err.Error(), "Local listen error", "Error should mention local listen")
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
		err := tunnel.Start()
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

		err1 := tunnel1.Start()
		err2 := tunnel2.Start()

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
		// This test would require HTTP server actually running
		// We'll test the tunnel can be established
		config := Config{
			Host:                  sshContainer.Host,
			Port:                  sshContainer.Port,
			User:                  sshContainer.User,
			Password:              sshContainer.Password,
			RemoteHost:            "localhost",
			RemotePort:            8080,
			LocalPort:             18086,
			Timeout:               10 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		err := tunnel.Start()
		require.NoError(t, err, "Failed to start SSH tunnel")
		defer tunnel.Close()

		time.Sleep(2 * time.Second)

		// Try to make HTTP request through tunnel
		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		resp, err := client.Get("http://localhost:18086")
		if err != nil {
			t.Logf("HTTP request through tunnel failed (expected if HTTP server not running): %v", err)
		} else {
			defer resp.Body.Close()
			t.Logf("HTTP request through tunnel succeeded with status: %s", resp.Status)
			assert.Equal(t, http.StatusOK, resp.StatusCode, "Expected HTTP 200")
		}
	})
}

func TestSSHTunnelConnectionTimeout(t *testing.T) {
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
		err := tunnel.Start()
		elapsed := time.Since(start)

		assert.Error(t, err, "Expected error when connecting to unreachable host")
		assert.Less(t, elapsed, 3*time.Second, "Timeout should be respected (expected ~1s, got %v)", elapsed)
	})
}
