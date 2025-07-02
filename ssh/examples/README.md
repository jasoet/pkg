# SSH Package Examples

This directory contains examples demonstrating how to use the `ssh` package for creating SSH tunnels in Go applications.

## Overview

The `ssh` package provides utilities for:
- Creating SSH tunnels for secure port forwarding
- Connecting to remote services through SSH bastions
- Establishing secure connections to databases and other services
- Managing SSH tunnel lifecycle (start/stop)

## Running the Examples

To run the examples, use the following command from the `ssh/examples` directory:

```bash
go run example.go
```

**Note**: The examples require a working SSH server and appropriate credentials. Update the configuration in the examples to match your environment.

## Example Descriptions

The example.go file demonstrates several use cases:

### 1. Basic SSH Tunnel

Create a simple SSH tunnel to forward local traffic to a remote service:

```go
config := ssh.Config{
    Host:       "ssh-server.example.com",
    Port:       22,
    User:       "username",
    Password:   "password",
    RemoteHost: "database.internal.com",
    RemotePort: 5432,
    LocalPort:  5433,
}

tunnel := ssh.New(config)
err := tunnel.Start()
```

### 2. Database Connection Through SSH Tunnel

Connect to a PostgreSQL database through an SSH tunnel:

```go
// SSH tunnel configuration
tunnelConfig := ssh.Config{
    Host:       "bastion.example.com",
    Port:       22,
    User:       "deploy",
    Password:   "secure-password",
    RemoteHost: "postgres.internal.example.com",
    RemotePort: 5432,
    LocalPort:  5433,
    Timeout:    10 * time.Second,
}

// Start tunnel
tunnel := ssh.New(tunnelConfig)
err := tunnel.Start()
if err != nil {
    log.Fatal(err)
}
defer tunnel.Close()

// Connect to database through tunnel
db, err := sql.Open("postgres", "host=localhost port=5433 user=dbuser password=dbpass dbname=mydb sslmode=disable")
```

### 3. Multiple Tunnels Management

Manage multiple SSH tunnels for different services:

```go
// Database tunnel
dbTunnel := ssh.New(ssh.Config{
    Host: "bastion.example.com", Port: 22,
    User: "deploy", Password: "password",
    RemoteHost: "postgres.internal.com", RemotePort: 5432,
    LocalPort: 5433,
})

// Redis tunnel
redisTunnel := ssh.New(ssh.Config{
    Host: "bastion.example.com", Port: 22,
    User: "deploy", Password: "password",
    RemoteHost: "redis.internal.com", RemotePort: 6379,
    LocalPort: 6380,
})

// Start both tunnels
err := dbTunnel.Start()
err = redisTunnel.Start()

// Cleanup
defer dbTunnel.Close()
defer redisTunnel.Close()
```

### 4. Configuration from YAML

Load SSH tunnel configuration from YAML:

```go
yamlConfig := `
host: bastion.example.com
port: 22
user: deploy
password: secure-password
remoteHost: service.internal.com
remotePort: 8080
localPort: 8081
timeout: 30s
`

var config ssh.Config
err := yaml.Unmarshal([]byte(yamlConfig), &config)
tunnel := ssh.New(config)
```

### 5. Error Handling and Connection Management

Proper error handling for SSH tunnel operations:

```go
tunnel := ssh.New(config)

// Start tunnel with error handling
if err := tunnel.Start(); err != nil {
    if strings.Contains(err.Error(), "connection refused") {
        log.Fatal("SSH server is not accessible")
    } else if strings.Contains(err.Error(), "authentication failed") {
        log.Fatal("Invalid SSH credentials")
    } else {
        log.Fatal("SSH tunnel failed:", err)
    }
}

// Graceful shutdown
c := make(chan os.Signal, 1)
signal.Notify(c, os.Interrupt, syscall.SIGTERM)
go func() {
    <-c
    log.Println("Shutting down SSH tunnel...")
    tunnel.Close()
    os.Exit(0)
}()
```

## Configuration Options

The `ssh.Config` struct supports the following options:

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `Host` | string | SSH server hostname or IP address | Yes |
| `Port` | int | SSH server port (typically 22) | Yes |
| `User` | string | SSH username | Yes |
| `Password` | string | SSH password | Yes |
| `RemoteHost` | string | Remote service hostname/IP to connect to | Yes |
| `RemotePort` | int | Remote service port | Yes |
| `LocalPort` | int | Local port to bind tunnel to | Yes |
| `Timeout` | time.Duration | Connection timeout (default: 5s) | No |

## Security Considerations

### Authentication
- The current implementation uses password authentication
- Consider implementing SSH key-based authentication for production use
- Store credentials securely (environment variables, secrets management)

### Host Key Verification
- The current implementation uses `ssh.InsecureIgnoreHostKey()` for simplicity
- For production use, implement proper host key verification

### Connection Security
- Tunnels create encrypted connections between local and remote endpoints
- All traffic through the tunnel is encrypted with SSH
- Use SSH tunnels to secure connections to services that don't support encryption

## Common Use Cases

### 1. Database Access
Connect to databases behind firewalls or in private networks:
- PostgreSQL, MySQL, MongoDB through SSH bastions
- Secure database administration and development access

### 2. Service Access
Access internal services from external networks:
- Web services, APIs, microservices
- Development and testing environments

### 3. Network Troubleshooting
- Port forwarding for debugging network issues
- Accessing services for monitoring and diagnostics

### 4. Development Workflows
- Local development against remote services
- Testing applications with production-like infrastructure

## Best Practices

1. **Connection Management**: Always close tunnels when done to free resources
2. **Error Handling**: Implement proper error handling for connection failures
3. **Security**: Use SSH key authentication in production environments
4. **Timeouts**: Configure appropriate timeouts for your use case
5. **Monitoring**: Log tunnel status and connection health
6. **Resource Cleanup**: Use defer statements to ensure proper cleanup

## Troubleshooting

### Common Issues

1. **Connection Refused**: SSH server is not accessible or port is incorrect
2. **Authentication Failed**: Invalid username/password or SSH keys
3. **Port Already In Use**: Local port is already bound to another service
4. **Timeout**: Network connectivity issues or incorrect remote endpoint

### Debug Tips

- Test SSH connection manually: `ssh user@host -p port`
- Verify remote service is accessible from SSH server
- Check firewall rules and network connectivity
- Use verbose SSH logging for debugging connection issues