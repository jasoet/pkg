package ssh

import (
	"fmt"
	"io"
	"net"
	"time"

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

// Start establishes the SSH connection and begins forwarding traffic
func (t *Tunnel) Start() error {
	sshConfig := &ssh.ClientConfig{
		User: t.config.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(t.config.Password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
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
		fmt.Println("SSH tunnel dial error:", err)
		localConn.Close()
		return
	}
	go func() {
		_, err := io.Copy(remoteConn, localConn)
		if err != nil {
			fmt.Println("SSH tunnel copy error:", err)
		}
	}()
	go func() {
		_, err := io.Copy(localConn, remoteConn)
		if err != nil {
			fmt.Println("SSH tunnel copy error:", err)
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
