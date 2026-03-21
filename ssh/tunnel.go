package ssh

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"

	"github.com/jasoet/pkg/v2/otel"
)

// Config holds the configuration for an SSH tunnel
type Config struct {
	// Server connection details
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	User     string `yaml:"user" mapstructure:"user"`
	Password string `yaml:"-" mapstructure:"-"`

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
	stopCh   chan struct{}
	wg       sync.WaitGroup
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
	// InsecureIgnoreHostKey and KnownHostsFile are mutually exclusive
	if t.config.InsecureIgnoreHostKey && t.config.KnownHostsFile != "" {
		return nil, fmt.Errorf("InsecureIgnoreHostKey and KnownHostsFile cannot both be set")
	}

	// If explicitly set to ignore host keys (NOT recommended for production)
	if t.config.InsecureIgnoreHostKey {
		// #nosec G106 -- Insecure host key verification is intentionally configurable for development/testing
		return ssh.InsecureIgnoreHostKey(), nil
	}

	// If a known hosts file is specified, use it for verification
	if t.config.KnownHostsFile != "" {
		callback, err := knownhosts.New(t.config.KnownHostsFile)
		if err != nil {
			return nil, fmt.Errorf("cannot load known hosts file %q: %w", t.config.KnownHostsFile, err)
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

// Start establishes the SSH connection and begins forwarding traffic.
// The provided ctx is used for logger creation and SSH dial operations.
func (t *Tunnel) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.client != nil {
		t.mu.Unlock()
		return fmt.Errorf("tunnel already started")
	}
	t.stopCh = make(chan struct{})
	t.mu.Unlock()

	// Input validation
	if t.config.Host == "" {
		return fmt.Errorf("SSH host is required")
	}
	if t.config.Port <= 0 || t.config.Port > 65535 {
		return fmt.Errorf("invalid SSH port: %d", t.config.Port)
	}
	if t.config.User == "" {
		return fmt.Errorf("SSH user is required")
	}
	if t.config.RemoteHost == "" {
		return fmt.Errorf("remote host is required")
	}
	if t.config.RemotePort <= 0 || t.config.RemotePort > 65535 {
		return fmt.Errorf("invalid remote port: %d", t.config.RemotePort)
	}

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

	if t.config.InsecureIgnoreHostKey {
		logger.Warn("InsecureIgnoreHostKey is enabled - SSH host key verification is disabled")
	}

	serverEndpoint := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	logger.Debug("Connecting to SSH server", otel.F("endpoint", serverEndpoint))

	client, err := ssh.Dial("tcp", serverEndpoint, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH dial error: %w", err)
	}

	t.mu.Lock()
	t.client = client
	t.mu.Unlock()

	localEndpoint := fmt.Sprintf("localhost:%d", t.config.LocalPort)
	remoteEndpoint := fmt.Sprintf("%s:%d", t.config.RemoteHost, t.config.RemotePort)

	listener, err := net.Listen("tcp", localEndpoint)
	if err != nil {
		t.mu.Lock()
		t.client = nil
		t.mu.Unlock()
		client.Close()
		return fmt.Errorf("local listen error: %w", err)
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
				select {
				case <-t.stopCh:
					return
				default:
					continue
				}
			}
			t.wg.Add(1)
			go func() {
				defer t.wg.Done()
				t.forward(localConn, remoteEndpoint)
			}()
		}
	}()

	return nil
}

// LocalAddr returns the local address the tunnel listener is bound to.
// Returns an empty string if the tunnel is not started.
func (t *Tunnel) LocalAddr() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.listener != nil {
		return t.listener.Addr().String()
	}
	return ""
}

// forward handles the forwarding of data between the local and remote connections.
//
// Note: half-close (CloseWrite) is not implemented here. Both directions are
// copied concurrently and both connections are closed once both copies finish.
// This may affect streaming protocols that rely on half-close semantics.
func (t *Tunnel) forward(localConn net.Conn, remoteAddr string) {
	ctx := context.Background()
	logger := otel.NewLogHelper(ctx, nil, "github.com/jasoet/pkg/v2/ssh", "ssh.Tunnel.forward")

	t.mu.Lock()
	client := t.client
	t.mu.Unlock()

	if client == nil {
		localConn.Close()
		return
	}

	remoteConn, err := client.Dial("tcp", remoteAddr)
	if err != nil {
		logger.Error(err, "SSH tunnel dial error", otel.F("remoteAddr", remoteAddr))
		localConn.Close()
		return
	}

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		if _, err := io.Copy(remoteConn, localConn); err != nil {
			logger.Debug("copy local->remote ended", otel.F("err", err.Error()))
		}
	}()

	go func() {
		defer wg.Done()
		if _, err := io.Copy(localConn, remoteConn); err != nil {
			logger.Debug("copy remote->local ended", otel.F("err", err.Error()))
		}
	}()

	wg.Wait()
	localConn.Close()
	remoteConn.Close()
}

// Close terminates the SSH connection and stops the tunnel
func (t *Tunnel) Close() error {
	t.mu.Lock()
	if t.client == nil {
		t.mu.Unlock()
		return nil
	}

	if t.stopCh != nil {
		select {
		case <-t.stopCh:
		default:
			close(t.stopCh)
		}
	}

	if t.listener != nil {
		_ = t.listener.Close()
		t.listener = nil
	}

	client := t.client
	t.client = nil
	t.mu.Unlock()

	t.wg.Wait()
	return client.Close()
}
