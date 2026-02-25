package ssh

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

func TestNew(t *testing.T) {
	t.Run("creates tunnel with provided config", func(t *testing.T) {
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			RemoteHost:            "remote.example.com",
			RemotePort:            3306,
			LocalPort:             3307,
			Timeout:               10 * time.Second,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		require.NotNil(t, tunnel)
		assert.Equal(t, config.Host, tunnel.config.Host)
		assert.Equal(t, config.Timeout, tunnel.config.Timeout)
	})

	t.Run("sets default timeout when not specified", func(t *testing.T) {
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			RemoteHost:            "remote.example.com",
			RemotePort:            3306,
			LocalPort:             3307,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		require.NotNil(t, tunnel)
		assert.Equal(t, 5*time.Second, tunnel.config.Timeout)
	})

	t.Run("does not override non-zero timeout", func(t *testing.T) {
		customTimeout := 15 * time.Second
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			RemoteHost:            "remote.example.com",
			RemotePort:            3306,
			LocalPort:             3307,
			Timeout:               customTimeout,
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		assert.Equal(t, customTimeout, tunnel.config.Timeout)
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
		assert.Equal(t, config.Host, tunnel.config.Host)
		assert.Equal(t, config.Port, tunnel.config.Port)
		assert.Equal(t, config.User, tunnel.config.User)
		assert.Equal(t, config.Password, tunnel.config.Password)
		assert.Equal(t, config.RemoteHost, tunnel.config.RemoteHost)
		assert.Equal(t, config.RemotePort, tunnel.config.RemotePort)
		assert.Equal(t, config.LocalPort, tunnel.config.LocalPort)
		assert.Equal(t, config.KnownHostsFile, tunnel.config.KnownHostsFile)
		assert.Equal(t, config.InsecureIgnoreHostKey, tunnel.config.InsecureIgnoreHostKey)
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
		callback, err := tunnel.getHostKeyCallback()
		require.NoError(t, err)
		require.NotNil(t, callback)

		// The callback should be ssh.InsecureIgnoreHostKey which always returns nil
		err = callback("example.com", &net.TCPAddr{}, &mockPublicKey{})
		assert.NoError(t, err)
	})

	t.Run("returns error when no host key verification configured", func(t *testing.T) {
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			InsecureIgnoreHostKey: false,
		}

		tunnel := New(config)
		callback, err := tunnel.getHostKeyCallback()
		assert.Error(t, err)
		assert.Nil(t, callback)
		assert.Contains(t, err.Error(), "host key verification required")
	})

	t.Run("returns error for non-existent known hosts file", func(t *testing.T) {
		config := Config{
			Host:           "example.com",
			Port:           22,
			User:           "testuser",
			Password:       "testpass",
			KnownHostsFile: "/nonexistent/known_hosts",
		}

		tunnel := New(config)
		callback, err := tunnel.getHostKeyCallback()
		assert.Error(t, err)
		assert.Nil(t, callback)
		assert.Contains(t, err.Error(), "failed to load known hosts file")
	})

	t.Run("InsecureIgnoreHostKey works with different hosts", func(t *testing.T) {
		config := Config{
			Host:                  "example.com",
			Port:                  22,
			User:                  "testuser",
			Password:              "testpass",
			InsecureIgnoreHostKey: true,
		}

		tunnel := New(config)
		callback, err := tunnel.getHostKeyCallback()
		require.NoError(t, err)

		testHosts := []string{"localhost", "example.com", "192.168.1.1", "ssh.example.org"}
		for _, host := range testHosts {
			err := callback(host, &net.TCPAddr{}, &mockPublicKey{})
			assert.NoError(t, err, "callback should accept host %s", host)
		}
	})
}

func TestGetAuthMethods(t *testing.T) {
	t.Run("returns password auth when password is set", func(t *testing.T) {
		tunnel := New(Config{
			Password:              "testpass",
			InsecureIgnoreHostKey: true,
		})
		methods, err := tunnel.getAuthMethods()
		require.NoError(t, err)
		assert.Len(t, methods, 1)
	})

	t.Run("returns error when no auth method configured", func(t *testing.T) {
		tunnel := New(Config{
			InsecureIgnoreHostKey: true,
		})
		methods, err := tunnel.getAuthMethods()
		assert.Error(t, err)
		assert.Nil(t, methods)
		assert.Contains(t, err.Error(), "no authentication method configured")
	})

	t.Run("returns key auth when private key is set", func(t *testing.T) {
		// Use a test RSA key (generated for testing only)
		testKey := generateTestKey(t)
		tunnel := New(Config{
			PrivateKey:            testKey,
			InsecureIgnoreHostKey: true,
		})
		methods, err := tunnel.getAuthMethods()
		require.NoError(t, err)
		assert.Len(t, methods, 1)
	})

	t.Run("returns both auth methods when both configured", func(t *testing.T) {
		testKey := generateTestKey(t)
		tunnel := New(Config{
			Password:              "testpass",
			PrivateKey:            testKey,
			InsecureIgnoreHostKey: true,
		})
		methods, err := tunnel.getAuthMethods()
		require.NoError(t, err)
		assert.Len(t, methods, 2)
	})

	t.Run("returns error for invalid private key", func(t *testing.T) {
		tunnel := New(Config{
			PrivateKey:            []byte("invalid-key-data"),
			InsecureIgnoreHostKey: true,
		})
		methods, err := tunnel.getAuthMethods()
		assert.Error(t, err)
		assert.Nil(t, methods)
		assert.Contains(t, err.Error(), "failed to parse SSH private key")
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
		assert.NoError(t, err)
	})

	t.Run("is safe to call multiple times", func(t *testing.T) {
		tunnel := &Tunnel{
			config: Config{
				Host: "example.com",
				Port: 22,
			},
			client: nil,
		}

		assert.NoError(t, tunnel.Close())
		assert.NoError(t, tunnel.Close())
		assert.NoError(t, tunnel.Close())
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

		assert.Equal(t, "ssh.example.com", config.Host)
		assert.Equal(t, 2222, config.Port)
		assert.Equal(t, "admin", config.User)
		assert.Equal(t, "secret", config.Password)
		assert.Equal(t, "db.internal", config.RemoteHost)
		assert.Equal(t, 3306, config.RemotePort)
		assert.Equal(t, 3307, config.LocalPort)
		assert.Equal(t, 10*time.Second, config.Timeout)
		assert.Equal(t, "/home/user/.ssh/known_hosts", config.KnownHostsFile)
		assert.False(t, config.InsecureIgnoreHostKey)
	})

	t.Run("Config with zero values", func(t *testing.T) {
		config := Config{}
		assert.Empty(t, config.Host)
		assert.Zero(t, config.Port)
		assert.Zero(t, config.Timeout)
		assert.False(t, config.InsecureIgnoreHostKey)
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

		assert.Equal(t, "example.com", tunnel.config.Host)
		assert.Nil(t, tunnel.client)
		assert.Nil(t, tunnel.listener)
	})
}

// ============================================================================
// Helper functions for testing
// ============================================================================

// generateTestKey generates a test Ed25519 SSH private key
func generateTestKey(t *testing.T) []byte {
	t.Helper()
	// Pre-generated Ed25519 test key (not used in production)
	key := `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
QyNTUxOQAAACA08n7gs4YIb2GUXZimqMUHm8XhtSFnaDjtyxNZGHVcDAAAAKgDzd9hA83f
YQAAAAtzc2gtZWQyNTUxOQAAACA08n7gs4YIb2GUXZimqMUHm8XhtSFnaDjtyxNZGHVcDA
AAAEDNaMtMq/J8fWuoxBg9hFGUKGRkpSNkLAfHFkjERLyg/TTyfuCzhghvYZRdmKaoxQeb
xeG1IWdoOO3LE1kYdVwMAAAAIGphc29ldEBKYXNvZXRzLU1hY0Jvb2stQWlyLmxvY2FsAQ
IDBAU=
-----END OPENSSH PRIVATE KEY-----`
	return []byte(key)
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
// - Authentication method configuration (getAuthMethods)
// - Close behavior with nil client
//
// Integration tests would be needed to fully test:
// - Start() - establishing SSH connection
// - forward() - bidirectional port forwarding
// - Close() - with actual SSH client and listener
