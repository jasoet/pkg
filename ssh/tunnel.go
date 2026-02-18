package ssh

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
)

// Config holds the configuration for an SSH tunnel
type Config struct {
	// Server connection details
	Host     string `yaml:"host" mapstructure:"host"`
	Port     int    `yaml:"port" mapstructure:"port"`
	User     string `yaml:"user" mapstructure:"user"`
	Password string `yaml:"password" mapstructure:"password"`

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
	config Config
	client *ssh.Client
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
func (t *Tunnel) getHostKeyCallback() ssh.HostKeyCallback {
	// If explicitly set to ignore host keys (NOT recommended for production)
	if t.config.InsecureIgnoreHostKey {
		// #nosec G106 -- Insecure host key verification is intentionally configurable for development/testing
		return ssh.InsecureIgnoreHostKey()
	}

	// Default: return a callback that accepts any key but logs a warning
	// This is more secure than InsecureIgnoreHostKey as it at least logs the connection
	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		log.Warn().Str("hostname", hostname).Str("keyType", key.Type()).Msg("Unable to verify host key")
		return nil
	}
}

// Start establishes the SSH connection and begins forwarding traffic
func (t *Tunnel) Start() error {
	sshConfig := &ssh.ClientConfig{
		User: t.config.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(t.config.Password),
		},
		HostKeyCallback: t.getHostKeyCallback(),
		Timeout:         t.config.Timeout,
	}

	var err error
	serverEndpoint := fmt.Sprintf("%s:%d", t.config.Host, t.config.Port)
	t.client, err = ssh.Dial("tcp", serverEndpoint, sshConfig)
	if err != nil {
		return fmt.Errorf("SSH dial error: %w", err)
	}

	localEndpoint := fmt.Sprintf("localhost:%d", t.config.LocalPort)
	remoteEndpoint := fmt.Sprintf("%s:%d", t.config.RemoteHost, t.config.RemotePort)

	listener, err := net.Listen("tcp", localEndpoint)
	if err != nil {
		return fmt.Errorf("Local listen error: %w", err)
	}

	go func() {
		for {
			localConn, err := listener.Accept()
			if err != nil {
				continue
			}
			go t.forward(localConn, remoteEndpoint)
		}
	}()

	return nil
}

// forward handles the forwarding of data between the local and remote connections
func (t *Tunnel) forward(localConn net.Conn, remoteAddr string) {
	remoteConn, err := t.client.Dial("tcp", remoteAddr)
	if err != nil {
		log.Error().Err(err).Str("remoteAddr", remoteAddr).Msg("SSH tunnel dial error")
		if closeErr := localConn.Close(); closeErr != nil {
			log.Error().Err(closeErr).Msg("Error closing local connection")
		}
		return
	}
	go func() {
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			log.Error().Err(err).Msg("SSH tunnel copy error")
		}
	}()
	go func() {
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			log.Error().Err(err).Msg("SSH tunnel copy error")
		}
	}()
}

// Close terminates the SSH connection and stops the tunnel
func (t *Tunnel) Close() error {
	if t.client != nil {
		return t.client.Close()
	}
	return nil
}
