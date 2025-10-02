package ssh

import (
	"net"
	"testing"
	"time"

	"golang.org/x/crypto/ssh"
)

func TestNew(t *testing.T) {
	t.Run("creates tunnel with provided config", func(t *testing.T) {
		config := Config{
			Host:       "example.com",
			Port:       22,
			User:       "testuser",
			Password:   "testpass",
			RemoteHost: "remote.example.com",
			RemotePort: 3306,
			LocalPort:  3307,
			Timeout:    10 * time.Second,
		}

		tunnel := New(config)
		if tunnel == nil {
			t.Fatal("Expected non-nil tunnel")
		}
		if tunnel.config.Host != config.Host {
			t.Errorf("Expected host %s, got %s", config.Host, tunnel.config.Host)
		}
		if tunnel.config.Timeout != config.Timeout {
			t.Errorf("Expected timeout %v, got %v", config.Timeout, tunnel.config.Timeout)
		}
	})

	t.Run("sets default timeout when not specified", func(t *testing.T) {
		config := Config{
			Host:       "example.com",
			Port:       22,
			User:       "testuser",
			Password:   "testpass",
			RemoteHost: "remote.example.com",
			RemotePort: 3306,
			LocalPort:  3307,
			// Timeout not specified
		}

		tunnel := New(config)
		if tunnel == nil {
			t.Fatal("Expected non-nil tunnel")
		}
		expectedTimeout := 5 * time.Second
		if tunnel.config.Timeout != expectedTimeout {
			t.Errorf("Expected default timeout %v, got %v", expectedTimeout, tunnel.config.Timeout)
		}
	})

	t.Run("does not override non-zero timeout", func(t *testing.T) {
		customTimeout := 15 * time.Second
		config := Config{
			Host:       "example.com",
			Port:       22,
			User:       "testuser",
			Password:   "testpass",
			RemoteHost: "remote.example.com",
			RemotePort: 3306,
			LocalPort:  3307,
			Timeout:    customTimeout,
		}

		tunnel := New(config)
		if tunnel.config.Timeout != customTimeout {
			t.Errorf("Expected timeout %v, got %v", customTimeout, tunnel.config.Timeout)
		}
	})

	t.Run("preserves all config fields", func(t *testing.T) {
		config := Config{
			Host:                  "ssh.example.com",
			Port:                  2222,
			User:                  "admin",
			Password:              "secretpass",
			RemoteHost:            "db.internal",
			RemotePort:            5432,
			LocalPort:             5433,
			Timeout:               20 * time.Second,
			KnownHostsFile:        "/path/to/known_hosts",
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		if tunnel.config.Host != config.Host {
			t.Error("Host not preserved")
		}
		if tunnel.config.Port != config.Port {
			t.Error("Port not preserved")
		}
		if tunnel.config.User != config.User {
			t.Error("User not preserved")
		}
		if tunnel.config.Password != config.Password {
			t.Error("Password not preserved")
		}
		if tunnel.config.RemoteHost != config.RemoteHost {
			t.Error("RemoteHost not preserved")
		}
		if tunnel.config.RemotePort != config.RemotePort {
			t.Error("RemotePort not preserved")
		}
		if tunnel.config.LocalPort != config.LocalPort {
			t.Error("LocalPort not preserved")
		}
		if tunnel.config.KnownHostsFile != config.KnownHostsFile {
			t.Error("KnownHostsFile not preserved")
		}
		if tunnel.config.InsecureIgnoreHostKey != config.InsecureIgnoreHostKey {
			t.Error("InsecureIgnoreHostKey not preserved")
		}
	})
}

func TestGetHostKeyCallback(t *testing.T) {
	t.Run("returns InsecureIgnoreHostKey when configured", func(t *testing.T) {
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		callback := tunnel.getHostKeyCallback()

		if callback == nil {
			t.Fatal("Expected non-nil host key callback")
		}

		// The callback should be ssh.InsecureIgnoreHostKey which always returns nil
		// We can't directly compare function pointers, but we can test its behavior
		err := callback("example.com", &net.TCPAddr{}, &mockPublicKey{})
		if err != nil {
			t.Errorf("InsecureIgnoreHostKey should always return nil, got: %v", err)
		}
	})

	t.Run("returns warning callback when InsecureIgnoreHostKey is false", func(t *testing.T) {
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			InsecureIgnoreHostKey: false,
		}

		tunnel := New(config)
		callback := tunnel.getHostKeyCallback()

		if callback == nil {
			t.Fatal("Expected non-nil host key callback")
		}

		// The default callback should still accept the key but log a warning
		err := callback("example.com", &net.TCPAddr{}, &mockPublicKey{})
		if err != nil {
			t.Errorf("Default callback should accept key and return nil, got: %v", err)
		}
	})

	t.Run("callback works with different host names", func(t *testing.T) {
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			InsecureIgnoreHostKey: false,
		}

		tunnel := New(config)
		callback := tunnel.getHostKeyCallback()

		testHosts := []string{"localhost", "example.com", "192.168.1.1", "ssh.example.org"}
		for _, host := range testHosts {
			err := callback(host, &net.TCPAddr{}, &mockPublicKey{})
			if err != nil {
				t.Errorf("Callback should accept host %s, got error: %v", host, err)
			}
		}
	})
}

func TestClose(t *testing.T) {
	t.Run("returns nil when client is nil", func(t *testing.T) {
		tunnel := &Tunnel{
			config: Config{
				Host: "example.com",
				Port: 22,
			},
			client: nil,
		}

		err := tunnel.Close()
		if err != nil {
			t.Errorf("Expected nil error when client is nil, got: %v", err)
		}
	})

	t.Run("is safe to call multiple times", func(t *testing.T) {
		tunnel := &Tunnel{
			config: Config{
				Host: "example.com",
				Port: 22,
			},
			client: nil,
		}

		// Call Close multiple times
		err1 := tunnel.Close()
		err2 := tunnel.Close()
		err3 := tunnel.Close()

		if err1 != nil {
			t.Errorf("First Close returned error: %v", err1)
		}
		if err2 != nil {
			t.Errorf("Second Close returned error: %v", err2)
		}
		if err3 != nil {
			t.Errorf("Third Close returned error: %v", err3)
		}
	})
}

func TestConfig(t *testing.T) {
	t.Run("Config struct has correct fields", func(t *testing.T) {
		config := Config{
			Host:                  "ssh.example.com",
			Port:                  2222,
			User:                  "admin",
			Password:              "secret",
			RemoteHost:            "db.internal",
			RemotePort:            3306,
			LocalPort:             3307,
			Timeout:               10 * time.Second,
			KnownHostsFile:        "/home/user/.ssh/known_hosts",
			InsecureIgnoreHostKey: false,
		}

		// Verify all fields can be set and read
		if config.Host != "ssh.example.com" {
			t.Error("Host field not working")
		}
		if config.Port != 2222 {
			t.Error("Port field not working")
		}
		if config.User != "admin" {
			t.Error("User field not working")
		}
		if config.Password != "secret" {
			t.Error("Password field not working")
		}
		if config.RemoteHost != "db.internal" {
			t.Error("RemoteHost field not working")
		}
		if config.RemotePort != 3306 {
			t.Error("RemotePort field not working")
		}
		if config.LocalPort != 3307 {
			t.Error("LocalPort field not working")
		}
		if config.Timeout != 10*time.Second {
			t.Error("Timeout field not working")
		}
		if config.KnownHostsFile != "/home/user/.ssh/known_hosts" {
			t.Error("KnownHostsFile field not working")
		}
		if config.InsecureIgnoreHostKey != false {
			t.Error("InsecureIgnoreHostKey field not working")
		}
	})

	t.Run("Config with zero values", func(t *testing.T) {
		config := Config{}

		if config.Host != "" {
			t.Error("Expected empty Host")
		}
		if config.Port != 0 {
			t.Error("Expected zero Port")
		}
		if config.Timeout != 0 {
			t.Error("Expected zero Timeout")
		}
		if config.InsecureIgnoreHostKey != false {
			t.Error("Expected false InsecureIgnoreHostKey")
		}
	})
}

func TestTunnelStruct(t *testing.T) {
	t.Run("Tunnel struct has correct fields", func(t *testing.T) {
		config := Config{
			Host: "example.com",
			Port: 22,
		}

		tunnel := &Tunnel{
			config: config,
			client: nil,
		}

		if tunnel.config.Host != "example.com" {
			t.Error("Config not properly stored")
		}
		if tunnel.client != nil {
			t.Error("Expected nil client")
		}
	})
}

// ============================================================================
// Mock types for testing
// ============================================================================

// mockPublicKey implements ssh.PublicKey for testing
type mockPublicKey struct{}

func (m *mockPublicKey) Type() string {
	return "ssh-rsa"
}

func (m *mockPublicKey) Marshal() []byte {
	return []byte("mock-public-key")
}

func (m *mockPublicKey) Verify(data []byte, sig *ssh.Signature) error {
	return nil
}

// Note: Testing Close() with a non-nil client requires integration tests
// with an actual SSH server, as ssh.Client is a concrete type that cannot
// be mocked. The current tests cover:
// - Configuration and setup (New, Config struct)
// - Host key callback generation (getHostKeyCallback)
// - Close behavior with nil client
//
// Integration tests would be needed to fully test:
// - Start() - establishing SSH connection
// - forward() - bidirectional port forwarding
// - Close() - with actual SSH client
