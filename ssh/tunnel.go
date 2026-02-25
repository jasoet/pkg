package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/jasoet/pkg/v2/otel"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Config holds the configuration for an SSH tunnel
type Config struct {
	// Server connection details
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	User     string `yaml:"user" mapstructure:"user"`
	Password string `yaml:"password" mapstructure:"password"`

	// SSH private key for key-based authentication (PEM-encoded)
	PrivateKey []byte `yaml:"-" mapstructure:"-"`

	// SSH private key passphrase (if the key is encrypted)
	PrivateKeyPassphrase string `yaml:"-" mapstructure:"-"`

	// Remote endpoint to connect to through the tunnel
	RemoteHost string `yaml:"remoteHost" mapstructure:"remoteHost"`
	RemotePort int    `yaml:"remotePort" mapstructure:"remotePort"`

	// Local port to listen on
	LocalPort int `yaml:"localPort" mapstructure:"localPort"`

	// Optional connection timeout (defaults to 5 seconds if not specified)
	Timeout time.Duration `yaml:"timeout" mapstructure:"timeout"`

	// Optional known hosts file path for host key verification
	KnownHostsFile string `yaml:"knownHostsFile" mapstructure:"knownHostsFile"`

	// Optional flag to disable host key checking (NOT recommended for production)
	InsecureIgnoreHostKey bool `yaml:"insecureIgnoreHostKey" mapstructure:"insecureIgnoreHostKey"`
}

// Tunnel represents an SSH tunnel that forwards traffic from a local port to a remote endpoint
type Tunnel struct {
	config   Config
	client   *ssh.Client
	listener net.Listener
	mu       sync.Mutex
}

// New creates a new SSH tunnel with the given configuration
func New(config Config) *Tunnel {
	// Set default timeout if not specified
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}

	return &Tunnel{
		config: config,
	}
}

// getHostKeyCallback returns the appropriate host key callback based on configuration
func (t *Tunnel) getHostKeyCallback() (ssh.HostKeyCallback, error) {
	// If explicitly set to ignore host keys (NOT recommended for production)
	if t.config.InsecureIgnoreHostKey {
		// #nosec G106 -- Insecure host key verification is intentionally configurable for development/testing
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// If a known hosts file is specified, use it for verification
	if t.config.KnownHostsFile != "" {
		callback, err := knownhosts.New(t.config.KnownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load known hosts file %s: %w", t.config.KnownHostsFile, err)
		}
		return callback, nil
	}

	// Default: reject unknown hosts
	return nil, fmt.Errorf("host key verification required: set InsecureIgnoreHostKey=true or provide KnownHostsFile")
}

// getAuthMethods returns the configured authentication methods
func (t *Tunnel) getAuthMethods() ([]ssh.AuthMethod, error) {
	var methods []ssh.AuthMethod

	// Add key-based auth if private key is provided
	if len(t.config.PrivateKey) > 0 {
		var signer ssh.Signer
		var err error
		if t.config.PrivateKeyPassphrase != "" {
			signer, err = ssh.ParsePrivateKeyWithPassphrase(t.config.PrivateKey, []byte(t.config.PrivateKeyPassphrase))
		} else {
			signer, err = ssh.ParsePrivateKey(t.config.PrivateKey)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH private key: %w", err)
		}
		methods = append(methods, ssh.PublicKeys(signer))
	}

	// Add password auth if password is provided
	if t.config.Password != "" {
		methods = append(methods, ssh.Password(t.config.Password))
	}

	if len(methods) == 0 {
		return nil, fmt.Errorf("no authentication method configured: provide Password or PrivateKey")
	}

	return methods, nil
}

// Start establishes the SSH connection and begins forwarding traffic
func (t *Tunnel) Start() error {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/ssh", "ssh.Tunnel.Start")

	hostKeyCallback, err := t.getHostKeyCallback()
	if err != nil {
		return fmt.Errorf("host key callback error: %w", err)
	}

	authMethods, err := t.getAuthMethods()
	if err != nil {
		return fmt.Errorf("authentication error: %w", err)
	}

	sshConfig := &ssh.ClientConfig{
		User:            t.config.User,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         t.config.Timeout,
	}

	serverEndpoint := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	logger.Debug("Connecting to SSH server", otel.F("endpoint", serverEndpoint))

	t.client, err = ssh.Dial("tcp", serverEndpoint, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH dial error: %w", err)
	}

	localEndpoint := fmt.Sprintf("localhost:%d", t.config.LocalPort)
	remoteEndpoint := fmt.Sprintf("%s:%d", t.config.RemoteHost, t.config.RemotePort)

	listener, err := net.Listen("tcp", localEndpoint)
	if err != nil {
		t.client.Close()
		t.client = nil
		return fmt.Errorf("Local listen error: %w", err)
	}

	t.mu.Lock()
	t.listener = listener
	t.mu.Unlock()

	logger.Debug("SSH tunnel listening",
		otel.F("local", localEndpoint),
		otel.F("remote", remoteEndpoint))

	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				// Listener was closed, exit goroutine
				return
			}
			go t.forward(localConn, remoteEndpoint)
		}
	}()

	return nil
}

// forward handles the forwarding of data between the local and remote connections
func (t *Tunnel) forward(localConn net.Conn, remoteAddr string) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/ssh", "ssh.Tunnel.forward")

	remoteConn, err := t.client.Dial("tcp", remoteAddr)
	if err != nil {
		logger.Error(err, "SSH tunnel dial error", otel.F("remoteAddr", remoteAddr))
		localConn.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		_, _ = io.Copy(remoteConn, localConn)
	}()

	go func() {
		defer wg.Done()
		_, _ = io.Copy(localConn, remoteConn)
	}()

	wg.Wait()
	localConn.Close()
	remoteConn.Close()
}

// Close terminates the SSH connection and stops the tunnel
func (t *Tunnel) Close() error {
	t.mu.Lock()
	listener := t.listener
	t.listener = nil
	t.mu.Unlock()

	if listener != nil {
		listener.Close()
	}
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}
