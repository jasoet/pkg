# SSH Tunnel

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/ssh.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/ssh)

Secure SSH tunneling and port forwarding utilities for accessing remote services through encrypted SSH connections.

## Overview

The `ssh` package provides SSH tunneling functionality for secure port forwarding. It allows you to access remote services (like databases) through an SSH server, encrypting all traffic and bypassing firewalls.

## Features

- **Port Forwarding**: Forward a local port to a remote endpoint via SSH
- **Password Authentication**: Simple password-based auth
- **Key-Based Authentication**: SSH private key (Ed25519, RSA, etc.) with optional passphrase
- **Configurable Timeout**: Control connection timeouts
- **Host Key Verification**: known_hosts checking, or explicit opt-out for development
- **Concurrent Connections**: Handles multiple simultaneous connections
- **OpenTelemetry**: Optional tracing, metrics, and logging via `WithOTelConfig`

## Installation

```bash
go get github.com/jasoet/pkg/v3/ssh
```

## Quick Start

### Basic Tunnel

```go
package main

import (
    "context"
    "os"
    "time"

    "github.com/jasoet/pkg/v3/ssh"
)

func main() {
    config := ssh.Config{
        // SSH server
        Host:     "bastion.example.com",
        Port:     22,
        User:     "admin",
        Password: os.Getenv("SSH_PASSWORD"), // secrets come from env/code, never YAML (see below)

        // Remote service to access
        RemoteHost: "database.internal",
        RemotePort: 5432,

        // Local port to listen on
        LocalPort: 15432,

        // Optional
        Timeout: 10 * time.Second,
    }

    tunnel := ssh.New(config)

    ctx := context.Background()
    if err := tunnel.Start(ctx); err != nil {
        panic(err)
    }
    defer tunnel.Close()

    // Now connect to localhost:15432 to access database.internal:5432.
    // tunnel.LocalAddr() returns the bound address, e.g. "127.0.0.1:15432"
    // db, _ := sql.Open("postgres", "host=localhost port=15432 ...")
}
```

### Database Access

```go
import (
    "context"
    "database/sql"

    "github.com/jasoet/pkg/v3/ssh"
)

// Start SSH tunnel
config := ssh.Config{
    Host:       "bastion.example.com",
    Port:       22,
    User:       "admin",
    Password:   os.Getenv("SSH_PASSWORD"),
    RemoteHost: "mysql.internal",
    RemotePort: 3306,
    LocalPort:  13306,
}

tunnel := ssh.New(config)
if err := tunnel.Start(ctx); err != nil {
    return err
}
defer tunnel.Close()

// Connect to database through tunnel
db, _ := sql.Open("mysql", "user:pass@tcp(localhost:13306)/database")
defer db.Close()

// Use database normally
db.Ping()
```

## Configuration

### Config Struct

```go
type Config struct {
    // SSH Server
    Host                 string // SSH server hostname        (yaml: host)
    Port                 int    // SSH server port (usually 22) (yaml: port)
    User                 string // SSH username                (yaml: user)
    Password             string // SSH password                (yaml:"-" — code/env only)
    PrivateKey           []byte // PEM-encoded private key     (yaml:"-" — code/env only)
    PrivateKeyPassphrase string // Private key passphrase      (yaml:"-" — code/env only)

    // Remote Endpoint
    RemoteHost string // Remote service hostname (yaml: remoteHost)
    RemotePort int    // Remote service port     (yaml: remotePort)

    // Local Settings
    LocalPort int // Local port to listen on (yaml: localPort)

    // Optional
    Timeout               time.Duration // Connection timeout (default: 5s)
    KnownHostsFile        string        // Path to known_hosts file
    InsecureIgnoreHostKey bool          // Skip host key verification (NOT recommended)
    OTelConfig            *otel.Config  // OpenTelemetry config (yaml:"-" — code only)
}
```

### Secrets Are Not Loadable from YAML

`Password`, `PrivateKey`, `PrivateKeyPassphrase`, and `OTelConfig` are tagged
`yaml:"-"` / `mapstructure:"-"`. This is deliberate: secrets must not sit in
config files. A `password:` key in YAML is **silently dropped** — inject
secrets from the environment (or a secret manager) after loading:

```go
import (
    "os"

    "github.com/jasoet/pkg/v3/config"
    "github.com/jasoet/pkg/v3/ssh"
)

type AppConfig struct {
    Tunnel ssh.Config `yaml:"tunnel"`
}

// Only non-secret fields belong in the file:
yamlConfig := `
tunnel:
  host: bastion.example.com
  port: 22
  user: admin
  remoteHost: database.internal
  remotePort: 5432
  localPort: 15432
  timeout: 10s
`

cfg, _ := config.LoadString[AppConfig](yamlConfig)

// Inject secrets after loading:
cfg.Tunnel.Password = os.Getenv("SSH_PASSWORD")
// or key-based:
// key, _ := os.ReadFile(os.Getenv("SSH_KEY_PATH"))
// cfg.Tunnel.PrivateKey = key
// cfg.Tunnel.PrivateKeyPassphrase = os.Getenv("SSH_KEY_PASSPHRASE")

tunnel := ssh.New(cfg.Tunnel)
```

### OpenTelemetry

Pass an `otel.Config` via the functional option to instrument `Start`/`Close`
with spans, metrics, and correlated logs (scope `operations.ssh`):

```go
import (
    "github.com/jasoet/pkg/v3/otel"
    "github.com/jasoet/pkg/v3/ssh"
)

otelCfg := otel.NewConfig("my-service")
tunnel := ssh.New(config, ssh.WithOTelConfig(otelCfg))
```

## Use Cases

### Access Internal Database

```go
// Production database behind firewall
config := ssh.Config{
    Host:       "bastion-prod.example.com",
    Port:       22,
    User:       "devops",
    Password:   os.Getenv("SSH_PASSWORD"),
    RemoteHost: "postgres-prod.internal",
    RemotePort: 5432,
    LocalPort:  15432,
}

tunnel := ssh.New(config)
if err := tunnel.Start(ctx); err != nil {
    return err
}
defer tunnel.Close()

// Connect to production DB securely
db, _ := sql.Open("postgres", "host=localhost port=15432 ...")
```

### Access Multiple Services

```go
// Database tunnel
dbTunnel := ssh.New(ssh.Config{
    Host:       "bastion.example.com",
    Port:       22,
    User:       "admin",
    Password:   os.Getenv("SSH_PASSWORD"),
    RemoteHost: "db.internal",
    RemotePort: 5432,
    LocalPort:  15432,
})

// Redis tunnel
redisTunnel := ssh.New(ssh.Config{
    Host:       "bastion.example.com",
    Port:       22,
    User:       "admin",
    Password:   os.Getenv("SSH_PASSWORD"),
    RemoteHost: "redis.internal",
    RemotePort: 6379,
    LocalPort:  16379,
})

if err := dbTunnel.Start(ctx); err != nil {
    return err
}
if err := redisTunnel.Start(ctx); err != nil {
    return err
}

defer dbTunnel.Close()
defer redisTunnel.Close()

// Access both services through local ports
```

### Temporary Access

```go
// Start tunnel for specific operation
tunnel := ssh.New(config)
if err := tunnel.Start(ctx); err != nil {
    return err
}

// Perform operation
db, _ := sql.Open("postgres", "host=localhost port=15432 ...")
db.Ping()
db.Close()

// Close tunnel when done
tunnel.Close()
```

## Security

### Host Key Verification

Host key verification is **required by default**: with neither
`KnownHostsFile` nor `InsecureIgnoreHostKey` set, `Start` fails with
`host key verification required`. The two options are mutually exclusive —
setting both is an error.

**Production (Recommended):**

```go
config := ssh.Config{
    // ...
    KnownHostsFile: "/home/user/.ssh/known_hosts",
}
```

**Development Only:**

```go
config := ssh.Config{
    // ...
    InsecureIgnoreHostKey: true, // ⚠️ Skip verification (NOT for production)
}
```

### Password Management

```go
// ✅ Good: Use environment variables
config := ssh.Config{
    Password: os.Getenv("SSH_PASSWORD"),
    // ...
}

// ❌ Bad: Hardcoded password
config := ssh.Config{
    Password: "hardcoded-secret", // Never do this!
    // ...
}
```

### Connection Timeout

```go
// ✅ Good: Set reasonable timeout
config := ssh.Config{
    Timeout: 10 * time.Second, // Fail fast
    // ...
}

// Note: Timeout 0 uses the default of 5s — it never means "no timeout".
config := ssh.Config{
    Timeout: 0, // Will use default 5s
    // ...
}
```

## Error Handling

`Start` and `Close` return errors wrapped with `fmt.Errorf("...: %w", err)`.
There are currently **no exported sentinel errors**, so `errors.Is`/`errors.As`
cannot match stable package-level targets; match on the stable message
prefixes instead:

```go
tunnel := ssh.New(config)

if err := tunnel.Start(ctx); err != nil {
    switch {
    case strings.Contains(err.Error(), "SSH dial error"):
        // Cannot reach SSH server, or the SSH handshake/auth failed
        // (server-side auth rejection surfaces here as "unable to authenticate")
        log.Printf("SSH server unreachable or handshake failed: %v", err)

    case strings.Contains(err.Error(), "authentication error"):
        // Client-side auth setup failed, e.g. unparsable private key or
        // no auth method configured
        log.Printf("Invalid SSH credentials configuration: %v", err)

    case strings.Contains(err.Error(), "host key callback error"):
        // known_hosts file unreadable, both host-key options set, or
        // neither set (verification is required by default)
        log.Printf("Host key verification misconfigured: %v", err)

    case strings.Contains(err.Error(), "local listen error"):
        // Local port already in use
        log.Printf("Local port %d already in use", config.LocalPort)

    default:
        log.Printf("Tunnel start failed: %v", err)
    }
    return
}
defer tunnel.Close()
```

## Advanced Usage

### With Context

`Start(ctx)` uses the context for the local listener and logger creation; the
SSH dial itself is bounded by `Config.Timeout`. Cancelling the context does
**not** stop a running tunnel — call `Close`:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

tunnel := ssh.New(config)
if err := tunnel.Start(ctx); err != nil {
    return err
}

// Close tunnel when context cancelled
go func() {
    <-ctx.Done()
    tunnel.Close()
}()
```

### Retry Logic

The package does not reconnect automatically. If you need resilience, restart
the tunnel yourself — create a fresh `Tunnel` per attempt, since a failed
`Start` may leave internal state behind:

```go
func startTunnelWithRetry(ctx context.Context, config ssh.Config, maxRetries int) (*ssh.Tunnel, error) {
    var err error
    for i := 0; i < maxRetries; i++ {
        tunnel := ssh.New(config)
        if err = tunnel.Start(ctx); err == nil {
            return tunnel, nil
        }
        log.Printf("Tunnel start failed (attempt %d/%d): %v", i+1, maxRetries, err)
        time.Sleep(time.Second * time.Duration(i+1))
    }
    return nil, fmt.Errorf("failed to start tunnel after %d retries: %w", maxRetries, err)
}
```

### Health Check

```go
func checkTunnelHealth(tunnel *ssh.Tunnel) error {
    addr := tunnel.LocalAddr() // "" if the tunnel is not started
    if addr == "" {
        return fmt.Errorf("tunnel not started")
    }
    conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
    if err != nil {
        return fmt.Errorf("tunnel not responsive: %w", err)
    }
    conn.Close()
    return nil
}

// Usage
if err := tunnel.Start(ctx); err != nil {
    log.Fatal(err)
}
if err := checkTunnelHealth(tunnel); err != nil {
    log.Fatal(err)
}
```

## Best Practices

### 1. Always Close Tunnels

```go
// ✅ Good: Use defer
tunnel := ssh.New(config)
if err := tunnel.Start(ctx); err != nil {
    return err
}
defer tunnel.Close()

// ❌ Bad: Forget to close
tunnel := ssh.New(config)
tunnel.Start(ctx)
// Tunnel leaks!
```

### 2. Unique Local Ports

```go
// ✅ Good: Different local ports
dbTunnel := ssh.New(ssh.Config{LocalPort: 15432, ...})
redisTunnel := ssh.New(ssh.Config{LocalPort: 16379, ...})

// ❌ Bad: Same local port
dbTunnel := ssh.New(ssh.Config{LocalPort: 15000, ...})
redisTunnel := ssh.New(ssh.Config{LocalPort: 15000, ...}) // Conflict!
```

### 3. Verify Connectivity

```go
// ✅ Good: Test before using
if err := tunnel.Start(ctx); err != nil {
    return err
}

conn, err := net.DialTimeout("tcp", tunnel.LocalAddr(), 5*time.Second)
if err != nil {
    return fmt.Errorf("tunnel not ready: %w", err)
}
conn.Close()

// Now use tunnel
```

### 4. Use Known Hosts in Production

```go
// ✅ Good: Verify host keys
config := ssh.Config{
    KnownHostsFile: "/etc/ssh/known_hosts",
    // ...
}

// ❌ Bad: Ignore host keys
config := ssh.Config{
    InsecureIgnoreHostKey: true, // Vulnerable to MITM attacks
    // ...
}
```

### 5. Set Timeouts

```go
// ✅ Good: Reasonable timeout
config := ssh.Config{
    Timeout: 10 * time.Second,
    // ...
}

// Avoid: Very long timeout (hangs on issues)
config := ssh.Config{
    Timeout: 5 * time.Minute, // Too long
    // ...
}
```

## Testing

The package includes unit tests plus integration tests that run a real SSH
server and assert end-to-end forwarding via testcontainers:

```bash
# Run unit tests
go test ./ssh -v

# Integration tests (requires Docker)
go test ./ssh -tags=integration -v

# With coverage
go test ./ssh -tags=integration -cover
```

### Test Utilities

```go
import (
    "github.com/jasoet/pkg/v3/ssh"
    "github.com/testcontainers/testcontainers-go"
)

func TestSSHTunnel(t *testing.T) {
    // Start SSH server container
    ctx := context.Background()
    sshContainer, _ := testcontainers.GenericContainer(ctx, /* SSH server config */)
    defer sshContainer.Terminate(ctx)

    // Get container details
    host, _ := sshContainer.Host(ctx)
    port, _ := sshContainer.MappedPort(ctx, "22")

    // Test tunnel
    config := ssh.Config{
        Host:     host,
        Port:     port.Int(),
        User:     "testuser",
        Password: "testpass",
        // ...
    }

    tunnel := ssh.New(config)
    err := tunnel.Start(ctx)
    assert.NoError(t, err)
    defer tunnel.Close()
}
```

## Troubleshooting

### Connection Refused

**Problem**: `SSH dial error: ... connection refused`

**Solutions:**
```go
// 1. Check SSH server is running
// ssh user@bastion.example.com

// 2. Verify port
config := ssh.Config{
    Port: 22, // Standard SSH port
    // ...
}

// 3. Check firewall
// telnet bastion.example.com 22
```

### Authentication Failed

**Problem**: `SSH dial error: ssh: handshake failed: ssh: unable to authenticate`
(server rejected credentials), or `authentication error: ...` (client-side
setup, e.g. unparsable private key)

**Solutions:**
```go
// 1. Verify credentials are actually set — remember Password/PrivateKey are
//    yaml:"-", so loading from YAML leaves them empty
config := ssh.Config{
    User:     "correct-username",
    Password: os.Getenv("SSH_PASSWORD"),
    // ...
}

// 2. Check SSH server config
// grep PasswordAuthentication /etc/ssh/sshd_config
```

### Port Already in Use

**Problem**: `local listen error: ... address already in use`

**Solutions:**
```go
// 1. Use different local port
config := ssh.Config{
    LocalPort: 15433, // Different port
    // ...
}

// 2. Find and kill process using port
// lsof -ti:15432 | xargs kill -9
```

### Tunnel Not Responding

**Problem**: Tunnel starts but doesn't forward traffic

**Solutions:**
```go
// 1. Verify remote endpoint
config := ssh.Config{
    RemoteHost: "database.internal", // Correct hostname
    RemotePort: 5432,                // Correct port
    // ...
}

// 2. Test from SSH server
// ssh bastion.example.com
// telnet database.internal 5432
```

## Limitations

1. **TCP Only**: Only TCP port forwarding (no UDP)
2. **Single SSH Server**: One SSH server per tunnel
3. **No Auto-Reconnection**: A dropped SSH connection is not re-established; restart the tunnel yourself (see Retry Logic)
4. **No half-close**: Half-close (CloseWrite) is not implemented and may affect streaming protocols

## Examples

See [examples/ssh/](../examples/ssh/) directory for:
- Basic SSH tunneling
- Database access through tunnel
- Multiple concurrent tunnels
- Retry logic
- Health checking

## Related Packages

- **[db](../db/)** - Database package (often used with SSH tunnels)
- **[config](../config/)** - Configuration management
- **[otel](../otel/)** - OpenTelemetry instrumentation

## License

MIT License - see [LICENSE](../LICENSE) for details.
