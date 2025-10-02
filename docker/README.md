# Docker Executor

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/docker.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/docker)

Simple, flexible Docker container executor inspired by testcontainers API. Run Docker containers with maximum configurability and easy log/status gathering.

## Overview

The `docker` package provides production-ready Docker container management with two API styles:
- **Functional options** - Flexible, chainable configuration
- **Struct-based** - Testcontainers-compatible declarative style

## Features

- **Dual API Design**: Choose between functional options or struct-based configuration
- **Lifecycle Management**: Start, Stop, Restart, Terminate, Wait
- **Wait Strategies**: Log patterns, port listening, HTTP health checks, custom functions
- **Log Streaming**: Real-time log access with filtering and following
- **Status Monitoring**: Container state, health checks, resource stats
- **Network Helpers**: Easy access to host, ports, endpoints
- **OpenTelemetry**: Built-in observability support
- **Simple & Powerful**: Easy for simple cases, flexible for complex scenarios

## Installation

```bash
go get github.com/jasoet/pkg/v2/docker
```

## Quick Start

### Functional Options Style

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/v2/docker"
)

func main() {
    // Create executor with functional options
    exec, _ := docker.New(
        docker.WithImage("nginx:latest"),
        docker.WithPorts("80:8080"),
        docker.WithEnv("ENV=production"),
        docker.WithAutoRemove(true),
    )

    // Start container
    ctx := context.Background()
    exec.Start(ctx)
    defer exec.Terminate(ctx)

    // Get endpoint
    endpoint, _ := exec.Endpoint(ctx, "80/tcp")
    // Use: http://localhost:8080
}
```

### Struct-Based Style (Testcontainers-like)

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/v2/docker"
    "time"
)

func main() {
    // Create executor with ContainerRequest
    req := docker.ContainerRequest{
        Image:        "postgres:16-alpine",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_PASSWORD": "secret",
            "POSTGRES_USER":     "testuser",
            "POSTGRES_DB":       "testdb",
        },
        WaitingFor: docker.WaitForLog("ready to accept connections").
            WithStartupTimeout(60 * time.Second),
    }

    exec, _ := docker.NewFromRequest(req)

    ctx := context.Background()
    exec.Start(ctx)
    defer exec.Terminate(ctx)

    // Connection string helper
    connStr, _ := exec.ConnectionString(ctx, "5432/tcp",
        "postgres://testuser:secret@%s/testdb")
}
```

### Hybrid Style (Mix Both)

```go
// Start with struct, add functional options
req := docker.ContainerRequest{
    Image: "nginx:latest",
    ExposedPorts: []string{"80/tcp"},
}

exec, _ := docker.New(
    docker.WithRequest(req),
    docker.WithName("my-nginx"),
    docker.WithOTelConfig(otelCfg), // Add observability
)
```

## API Styles

### Functional Options

**Advantages:**
- Chainable and composable
- Clear and explicit
- Easy to add/remove options
- Type-safe with IDE autocomplete

**Example:**
```go
exec, _ := docker.New(
    docker.WithImage("redis:7-alpine"),
    docker.WithPorts("6379:16379"),
    docker.WithVolume("/data", "/data"),
    docker.WithWaitStrategy(docker.WaitForPort("6379/tcp")),
)
```

### Struct-Based (ContainerRequest)

**Advantages:**
- Familiar to testcontainers users
- Easy to build programmatically
- Good for configuration files (YAML/JSON)
- Compact for complex configs

**Example:**
```go
req := docker.ContainerRequest{
    Image:        "mysql:8",
    ExposedPorts: []string{"3306/tcp"},
    Env: map[string]string{
        "MYSQL_ROOT_PASSWORD": "root",
        "MYSQL_DATABASE":      "app",
    },
    WaitingFor: docker.WaitForLog("ready for connections"),
}
exec, _ := docker.NewFromRequest(req)
```

## Configuration Options

### Image & Container

```go
docker.WithImage("nginx:latest")              // Container image
docker.WithName("my-container")               // Container name
docker.WithHostname("app-server")             // Hostname
docker.WithCmd("--verbose", "--debug")        // Override CMD
docker.WithEntrypoint("/bin/sh", "-c")       // Override ENTRYPOINT
docker.WithWorkDir("/app")                    // Working directory
docker.WithUser("1000:1000")                  // User (UID:GID)
```

### Environment

```go
docker.WithEnv("KEY=value")                   // Single env var
docker.WithEnvMap(map[string]string{          // Multiple env vars
    "DB_HOST": "localhost",
    "DB_PORT": "5432",
})
```

### Ports

```go
docker.WithPorts("80:8080")                   // Simple port mapping
docker.WithPorts("443:8443/tcp")              // With protocol
docker.WithPortBindings(map[string]string{    // Multiple ports
    "80/tcp":  "8080",
    "443/tcp": "8443",
})
docker.WithExposedPorts("8080", "9090")       // Expose without binding
```

### Volumes

```go
docker.WithVolume("/host/path", "/container/path")
docker.WithVolumeRO("/host/path", "/container/path") // Read-only
docker.WithVolumes(map[string]string{
    "/host/data": "/data",
    "/host/logs": "/var/log",
})
```

### Network

```go
docker.WithNetwork("my-network")              // Attach to network
docker.WithNetworks("net1", "net2")           // Multiple networks
docker.WithNetworkMode("bridge")              // Network mode
docker.WithNetworkMode("host")                // Host network
```

### Security

```go
docker.WithPrivileged(true)                   // Privileged mode
docker.WithCapAdd("NET_ADMIN", "SYS_TIME")   // Add capabilities
docker.WithCapDrop("CHOWN", "SETUID")        // Drop capabilities
```

### Resources

```go
docker.WithShmSize(67108864)                  // /dev/shm size (64MB)
docker.WithTmpfs("/tmp", "size=64m")         // tmpfs mount
```

### Cleanup

```go
docker.WithAutoRemove(true)                   // Auto-remove on stop
```

### Wait Strategies

```go
docker.WithWaitStrategy(
    docker.WaitForLog("started successfully").
        WithStartupTimeout(60 * time.Second),
)

docker.WithWaitStrategy(
    docker.WaitForPort("8080/tcp"),
)

docker.WithWaitStrategy(
    docker.WaitForHTTP("8080", "/health", 200),
)

docker.WithWaitStrategy(
    docker.WaitForHealthy(),
)

docker.WithWaitStrategy(
    docker.WaitForFunc(func(ctx context.Context, cli *client.Client, id string) error {
        // Custom readiness check
        return nil
    }),
)
```

### Observability

```go
docker.WithOTelConfig(otelCfg)                // OpenTelemetry
docker.WithTimeout(30 * time.Second)          // Operation timeout
```

## Lifecycle Methods

### Start

```go
err := exec.Start(ctx)
// - Pulls image if needed
// - Creates container
// - Starts container
// - Waits for readiness (if strategy configured)
```

### Stop

```go
err := exec.Stop(ctx)
// - Sends SIGTERM
// - Waits for graceful shutdown
// - Container can be restarted
```

### Terminate

```go
err := exec.Terminate(ctx)
// - Force stops container
// - Removes container
// - Cannot be restarted
```

### Restart

```go
err := exec.Restart(ctx)
// - Restarts running container
```

### Wait

```go
exitCode, err := exec.Wait(ctx)
// - Blocks until container exits
// - Returns exit code
```

## Logs

### Get All Logs

```go
logs, err := exec.Logs(ctx)
```

### Stream Logs

```go
logCh, errCh := exec.StreamLogs(ctx, docker.WithFollow())
for log := range logCh {
    fmt.Println(log.Content)
}
```

### Follow Logs to Writer

```go
err := exec.FollowLogs(ctx, os.Stdout)
```

### Advanced Log Options

```go
logs, err := exec.Logs(ctx,
    docker.WithStdout(true),
    docker.WithStderr(true),
    docker.WithTimestamps(),
    docker.WithTail("100"),        // Last 100 lines
    docker.WithSince("10m"),        // Last 10 minutes
)
```

### Convenience Methods

```go
logs, _ := exec.GetLogsSince(ctx, "5m")
logs, _ := exec.GetLastNLines(ctx, 50)
stdout, _ := exec.GetStdout(ctx)
stderr, _ := exec.GetStderr(ctx)
```

## Status & Monitoring

### Get Status

```go
status, err := exec.Status(ctx)
fmt.Println(status.Running)      // true/false
fmt.Println(status.State)        // "running", "exited", etc.
fmt.Println(status.ExitCode)     // Exit code if stopped
fmt.Println(status.Health.Status) // "healthy", "unhealthy", "starting"
```

### Check Running

```go
running, err := exec.IsRunning(ctx)
```

### Get Exit Code

```go
exitCode, err := exec.ExitCode(ctx)
```

### Health Check

```go
health, err := exec.HealthCheck(ctx)
fmt.Println(health.Status)
fmt.Println(health.FailingStreak)
```

### Full Inspection

```go
inspect, err := exec.Inspect(ctx)
// Returns *types.ContainerJSON with all details
```

### Resource Stats

```go
stats, err := exec.GetStats(ctx)
// CPU, memory, network, disk I/O
```

### Wait for State

```go
err := exec.WaitForState(ctx, "running", 30*time.Second)
err := exec.WaitForHealthy(ctx, 60*time.Second)
```

## Network Helpers

### Get Host

```go
host, err := exec.Host(ctx)
// Returns "localhost" for local Docker
```

### Get Mapped Port

```go
port, err := exec.MappedPort(ctx, "8080/tcp")
// Returns "32768" (example randomly assigned port)
```

### Get Endpoint

```go
endpoint, err := exec.Endpoint(ctx, "8080/tcp")
// Returns "localhost:32768"

// Use directly
resp, _ := http.Get("http://" + endpoint + "/health")
```

### Get All Ports

```go
ports, err := exec.GetAllPorts(ctx)
// map[string]string{
//     "80/tcp": "8080",
//     "443/tcp": "8443",
// }
```

### Get Networks

```go
networks, err := exec.GetNetworks(ctx)
// []string{"bridge", "my-network"}
```

### Get IP Address

```go
ip, err := exec.GetIPAddress(ctx, "bridge")
// "172.17.0.2"
```

### Connection String

```go
connStr, err := exec.ConnectionString(ctx, "5432/tcp",
    "postgres://user:pass@%s/db")
// "postgres://user:pass@localhost:15432/db"
```

## Use Cases

### Database Testing

```go
req := docker.ContainerRequest{
    Image: "postgres:16-alpine",
    ExposedPorts: []string{"5432/tcp"},
    Env: map[string]string{
        "POSTGRES_PASSWORD": "test",
        "POSTGRES_USER":     "test",
        "POSTGRES_DB":       "test",
    },
    WaitingFor: docker.WaitForLog("ready to accept connections"),
}

exec, _ := docker.NewFromRequest(req)
exec.Start(ctx)
defer exec.Terminate(ctx)

endpoint, _ := exec.Endpoint(ctx, "5432/tcp")
db, _ := sql.Open("postgres", "postgres://test:test@"+endpoint+"/test")
```

### Web Service Testing

```go
exec, _ := docker.New(
    docker.WithImage("nginx:latest"),
    docker.WithPorts("80:0"), // Random host port
    docker.WithWaitStrategy(
        docker.WaitForHTTP("80", "/", 200),
    ),
)

exec.Start(ctx)
defer exec.Terminate(ctx)

endpoint, _ := exec.Endpoint(ctx, "80/tcp")
resp, _ := http.Get("http://" + endpoint)
```

### Message Queue

```go
exec, _ := docker.New(
    docker.WithImage("rabbitmq:3-management"),
    docker.WithPorts("5672:15672"),
    docker.WithPorts("15672:25672"),
    docker.WithEnvMap(map[string]string{
        "RABBITMQ_DEFAULT_USER": "guest",
        "RABBITMQ_DEFAULT_PASS": "guest",
    }),
    docker.WithWaitStrategy(
        docker.WaitForLog("Server startup complete"),
    ),
)
```

### CI/CD Build Container

```go
exec, _ := docker.New(
    docker.WithImage("golang:1.23"),
    docker.WithVolume(pwd, "/app"),
    docker.WithWorkDir("/app"),
    docker.WithCmd("go", "test", "./..."),
    docker.WithAutoRemove(true),
)

exec.Start(ctx)
exitCode, _ := exec.Wait(ctx)
if exitCode != 0 {
    logs, _ := exec.GetStderr(ctx)
    fmt.Println("Tests failed:", logs)
}
```

### Development Environment

```go
// Redis
redis, _ := docker.New(
    docker.WithImage("redis:7-alpine"),
    docker.WithPorts("6379:6379"),
    docker.WithName("dev-redis"),
)

// PostgreSQL
postgres, _ := docker.New(
    docker.WithImage("postgres:16-alpine"),
    docker.WithPorts("5432:5432"),
    docker.WithName("dev-postgres"),
    docker.WithEnvMap(map[string]string{
        "POSTGRES_PASSWORD": "dev",
    }),
)

redis.Start(ctx)
postgres.Start(ctx)

defer redis.Terminate(ctx)
defer postgres.Terminate(ctx)
```

## Best Practices

### 1. Always Use defer for Cleanup

```go
exec, _ := docker.New(...)
exec.Start(ctx)
defer exec.Terminate(ctx) // Ensures cleanup
```

### 2. Use Wait Strategies

```go
// ✅ Good: Wait for readiness
docker.WithWaitStrategy(
    docker.WaitForLog("ready").WithStartupTimeout(30*time.Second),
)

// ❌ Bad: No wait strategy (race conditions)
```

### 3. Set Timeouts

```go
// ✅ Good: Reasonable timeout
docker.WithTimeout(30 * time.Second)

// ❌ Bad: No timeout (hangs forever)
```

### 4. Use AutoRemove for Tests

```go
docker.WithAutoRemove(true) // Clean up automatically
```

### 5. Handle Errors

```go
if err := exec.Start(ctx); err != nil {
    logs, _ := exec.GetStderr(ctx)
    log.Fatalf("Failed to start: %v\nLogs: %s", err, logs)
}
```

### 6. Use Context for Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

exec.Start(ctx) // Will respect context timeout
```

### 7. Check Container Health

```go
exec.Start(ctx)

// Verify it's actually working
running, _ := exec.IsRunning(ctx)
if !running {
    status, _ := exec.Status(ctx)
    log.Fatalf("Container failed: %s", status.Error)
}
```

## OpenTelemetry Integration

```go
import "github.com/jasoet/pkg/v2/otel"

// Initialize OTel
otelCfg := &otel.Config{
    TracerProvider: tp,
    MeterProvider:  mp,
}

// Use with executor
exec, _ := docker.New(
    docker.WithImage("nginx:latest"),
    docker.WithOTelConfig(otelCfg),
)

// Automatic instrumentation:
// - Traces: Start, Stop, Terminate, Restart
// - Metrics: containers_started, containers_stopped, etc.
// - Errors: Recorded in traces and metrics
```

## Testing

```bash
# Run tests
go test ./docker -v

# With coverage
go test ./docker -cover

# Integration tests (requires Docker)
go test ./docker -tags=integration -v
```

## Examples

See [examples/](./examples/) directory for:
- Basic usage
- Database containers
- Multi-container setups
- Custom wait strategies
- Log streaming
- Health monitoring

## Comparison with Testcontainers

| Feature | Docker Executor | Testcontainers-go |
|---------|----------------|-------------------|
| API Style | Functional options + Struct | Struct-based |
| Simplicity | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Flexibility | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Dependencies | Minimal | Many |
| OTel Support | Built-in | No |
| Learning Curve | Low | Medium |
| Use Case | General purpose | Testing focus |

## Related Packages

- **[otel](../otel/)** - OpenTelemetry configuration
- **[config](../config/)** - Configuration management

## License

MIT License - see [LICENSE](../LICENSE) for details.
