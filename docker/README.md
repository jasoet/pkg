# Docker Executor

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/docker.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/docker)

Simple, flexible Docker container executor inspired by testcontainers API. Run Docker containers with maximum configurability and easy log/status gathering.

## Overview

The `docker` package provides production-ready Docker container management with two API styles:
- **Functional options** - Flexible, chainable configuration
- **Struct-based** - Testcontainers-compatible declarative style

## Features

- **Dual API Design**: Choose between functional options or struct-based configuration
- **Lifecycle Management**: Start, Stop, Restart, Terminate, Wait
- **Wait Strategies**: Log patterns, port listening, HTTP health checks, custom functions — all against a library-owned `ContainerTarget`, no Docker client import needed
- **Log Streaming**: Real-time log access with filtering and following
- **Status Monitoring**: Container state, health checks, resource stats
- **Network Helpers**: Easy access to host, ports, endpoints
- **OpenTelemetry v2**: Built-in observability with traces and metrics
- **Simple & Powerful**: Easy for simple cases, flexible for complex scenarios

## Installation

```bash
go get github.com/jasoet/pkg/v3/docker
```

## Quick Start

### Functional Options Style

```go
package main

import (
    "context"

    "github.com/jasoet/pkg/v3/docker"
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
    _ = endpoint
}
```

### Struct-Based Style (Testcontainers-like)

```go
package main

import (
    "context"
    "time"

    "github.com/jasoet/pkg/v3/docker"
)

func main() {
    // Create executor with ContainerRequest
    req := docker.ContainerRequest{
        Image:        "postgres:18-alpine",
        ExposedPorts: []string{"5432/tcp"},
        // Publish to an auto-assigned host port so Endpoint/ConnectionString work
        PortBindings: map[string]string{"5432/tcp": ""},
        Env: map[string]string{
            "POSTGRES_PASSWORD": "secret",
            "POSTGRES_USER":     "testuser",
            "POSTGRES_DB":       "testdb",
        },
        // Postgres logs "ready to accept connections" twice (init server, then
        // the real one); "listening on IPv4" only appears once TCP is bound.
        WaitingFor: docker.WaitForLog(`listening on IPv4`).
            WithStartupTimeout(60 * time.Second),
    }

    exec, _ := docker.NewFromRequest(req)

    ctx := context.Background()
    exec.Start(ctx)
    defer exec.Terminate(ctx)

    // Connection string helper — note the {{endpoint}} placeholder
    connStr, _ := exec.ConnectionString(ctx, "5432/tcp",
        "postgres://testuser:secret@{{endpoint}}/testdb")
    _ = connStr
}
```

### Hybrid Style (Mix Both)

`NewFromRequest(req, opts...)` is sugar over `New(WithRequest(req), opts...)`: it prepends the struct as the first option so that any additional options override or extend the struct fields.

**1. Struct within options:**
```go
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

**2. Options after struct:**
```go
req := docker.ContainerRequest{
    Image: "postgres:18-alpine",
    Env: map[string]string{
        "POSTGRES_PASSWORD": "secret",
    },
}

// Add additional options that override/extend the struct
exec, _ := docker.NewFromRequest(req,
    docker.WithName("my-postgres"),        // Add name
    docker.WithPorts("5432:15432"),        // Add port mapping
    docker.WithOTelConfig(otelCfg),        // Add observability
)
```

**Note:** When using both, later options override earlier ones:
```go
req := docker.ContainerRequest{
    Image: "nginx:latest",
    Name:  "default-name",
}

exec, _ := docker.NewFromRequest(req,
    docker.WithName("override-name"),  // ← This wins!
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
    docker.WaitForFunc(func(ctx context.Context, target docker.ContainerTarget) error {
        // Custom readiness check
        return nil
    }),
)

// Combine several strategies (ALL must pass)
docker.WithWaitStrategy(
    docker.WaitForAll(
        docker.WaitForPort("5432/tcp"),
        docker.WaitForLog("ready to accept connections"),
    ),
)
```

Every strategy exposes `WithStartupTimeout(d)` (default 60s; 120s for `WaitForAll`).

### Observability

```go
docker.WithOTelConfig(otelCfg)                // OpenTelemetry
docker.WithTimeout(30 * time.Second)          // Operation timeout
```

## Wait Strategies and ContainerTarget

A wait strategy decides when a started container is *ready*; `Start` blocks until it passes (or its timeout fires, in which case the container is cleaned up and `Start` fails).

The strategy contract is:

```go
type WaitStrategy interface {
    WaitUntilReady(ctx context.Context, target ContainerTarget) error
}
```

Strategies never touch the Docker client. Instead they receive a `ContainerTarget` — a library-owned, value-type view of the running container:

```go
target.ID()                  // container ID
target.Logs(ctx)             // io.ReadCloser streaming stdout+stderr (follow mode)
target.State(ctx)            // ContainerState{Running, HealthStatus, Ports}
```

`ContainerState` is projected from Docker inspect into plain Go types:

```go
type ContainerState struct {
    Running      bool
    HealthStatus string              // "" when no healthcheck is defined
    Ports        map[string][]string // container port ("80/tcp") → host ports
}
```

Custom strategies implement `WaitUntilReady` directly, or use `WaitForFunc` for one-off checks:

```go
docker.WaitForFunc(func(ctx context.Context, target docker.ContainerTarget) error {
    state, err := target.State(ctx)
    if err != nil {
        return err
    }
    if !state.Running {
        return fmt.Errorf("container %s not running", target.ID())
    }
    return nil
})
```

A `ContainerTarget` is only usable when constructed by the Executor during `Start`; the zero value holds a nil client and will panic on `Logs` and `State`.

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

### Close

```go
err := exec.Close()
// - Closes the Docker client connection
// - Does NOT terminate the container — call Terminate() first if needed
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
    fmt.Println(log.Content) // LogEntry{Stream, Content}
}
```

`LogEntry` carries the stream name (`stdout`/`stderr`) and the frame content. To get timestamps, enable `WithTimestamps()` — they are embedded as a prefix in `Content`.

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
// Returns *container.InspectResponse with all details
```

### Resource Stats

```go
stats, err := exec.GetStats(ctx)
// CPU, memory, network, disk I/O
```

### Wait for State

```go
err := exec.WaitForState(ctx, "running", 30*time.Second)
err := exec.WaitHealthy(ctx, 60*time.Second)
```

Note: the executor method is `WaitHealthy` (verb phrase); the wait *strategy* constructor remains `docker.WaitForHealthy()`.

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
    "postgres://user:pass@{{endpoint}}/db")
// "postgres://user:pass@localhost:15432/db"
```

The template placeholder is `{{endpoint}}`, substituted via plain string replacement — **not** a `fmt.Sprintf` verb. `%s` in the template is left untouched (and would break the DSN), and passwords containing `%` are safe.

## Use Cases

### Database Testing

```go
req := docker.ContainerRequest{
    Image: "postgres:18-alpine",
    ExposedPorts: []string{"5432/tcp"},
    PortBindings: map[string]string{"5432/tcp": ""}, // auto-assigned host port
    Env: map[string]string{
        "POSTGRES_PASSWORD": "test",
        "POSTGRES_USER":     "test",
        "POSTGRES_DB":       "test",
    },
    WaitingFor: docker.WaitForLog(`listening on IPv4`), // see note below
}

exec, _ := docker.NewFromRequest(req)
exec.Start(ctx)
defer exec.Terminate(ctx)

endpoint, _ := exec.Endpoint(ctx, "5432/tcp")
db, _ := sql.Open("postgres", "postgres://test:test@"+endpoint+"/test")
```

> **Postgres wait pattern:** the official image logs `database system is ready
> to accept connections` twice — first for the temporary init server (Unix
> socket only), then for the real server. Waiting on that line can return
> before TCP 5432 is bound. `listening on IPv4` appears only when the real
> server binds TCP, so it is the safer pattern.

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
    docker.WithImage("postgres:18-alpine"),
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

The docker package includes full OpenTelemetry v2 instrumentation for observability.

```go
import (
    "github.com/jasoet/pkg/v3/otel"
    sdktrace "go.opentelemetry.io/otel/sdk/trace"
    sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// Initialize OTel providers
tp := sdktrace.NewTracerProvider(...)
mp := sdkmetric.NewMeterProvider(...)

otelCfg := &otel.Config{
    TracerProvider: tp,
    MeterProvider:  mp,
}

// Use with executor (functional option or struct field — both work)
exec, _ := docker.New(
    docker.WithImage("nginx:latest"),
    docker.WithOTelConfig(otelCfg),
)

req := docker.ContainerRequest{
    Image:      "nginx:latest",
    OTelConfig: otelCfg, // excluded from yaml/mapstructure decoding
}

// Automatic instrumentation:
// - Traces: docker.Start, docker.Stop, docker.Terminate, docker.Restart, docker.Wait
// - Metrics:
//   - docker.containers.started
//   - docker.containers.stopped
//   - docker.containers.terminated
//   - docker.containers.restarted
//   - docker.container.errors
// - Error tracking: Errors recorded in both traces and metrics with attributes
```

## Migrating from v2

Breaking changes in v3:

- **Import path**: `github.com/jasoet/pkg/v3/docker` (was `/v2/docker`).
- **`WaitStrategy` contract**: `WaitUntilReady(ctx, target ContainerTarget)` — strategies no longer receive the Docker `*client.Client` and container ID. Use `ContainerTarget.ID()`, `.Logs(ctx)`, and `.State(ctx)` instead. `WaitForFunc` signatures change accordingly.
- **`Executor.WaitForHealthy` → `Executor.WaitHealthy`**: the method was renamed; the strategy constructor `docker.WaitForHealthy()` is unchanged.
- **Removed helpers**: `NatPort`, `PortBindings`, and `ExposedPorts` (thin wrappers over `github.com/docker/go-connections/nat`) are gone; port strings like `"8080/tcp"` are parsed internally.
- **`LogEntry.Timestamp` removed**: the field was never populated. Enable `WithTimestamps()` and read the prefix from `Content` instead.
- **`ConnectionString` placeholder**: templates use `{{endpoint}}`, not `%s` (plain string replacement, safe for passwords containing `%`).

## Testing

The package has comprehensive unit and integration tests.

```bash
# Run all tests (requires Docker)
go test ./docker -v

# With coverage
go test ./docker -cover

# Run specific test
go test ./docker -run TestExecutor_FunctionalOptions -v

# Run benchmarks
go test ./docker -bench=. -benchmem
```

**Test Requirements:**
- Docker daemon running
- Docker API accessible
- Internet access (for pulling images)

## Examples

See the [examples/docker/](../examples/docker/) directory for complete, runnable examples:

- **[basic](../examples/docker/basic/)** - Functional options, struct-based, and hybrid styles
- **[database](../examples/docker/database/)** - PostgreSQL container with real database operations
- **[logs](../examples/docker/logs/)** - Log streaming, filtering, and following
- **[multi_container](../examples/docker/multi_container/)** - Running multiple containers (Nginx + Redis)

Run examples:
```bash
go run -tags=example ./examples/docker/basic
go run -tags=example ./examples/docker/database
go run -tags=example ./examples/docker/logs
go run -tags=example ./examples/docker/multi_container
```

## Comparison with Testcontainers

| Feature | Docker Executor | Testcontainers-go |
|---------|----------------|-------------------|
| API Style | Functional options + Struct | Struct-based |
| Simplicity | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| Flexibility | ⭐⭐⭐⭐⭐ | ⭐⭐⭐⭐ |
| Dependencies | Minimal | Many |
| OTel Support | Built-in v2 | No |
| Learning Curve | Low | Medium |
| Use Case | General purpose | Testing focus |

## Architecture

### Key Components

- **Executor** - Main container lifecycle manager
- **Config** - Container configuration with functional options
- **Wait Strategies** - Readiness checking against a `ContainerTarget`
- **Network** - Port mapping and endpoint resolution
- **Logs** - Log streaming and filtering
- **Status** - Container state monitoring
- **OTel** - OpenTelemetry v2 instrumentation

### Design Principles

1. **Simple by default, powerful when needed** - Easy basic usage, advanced features available
2. **Two API styles** - Functional options for Go idioms, structs for testcontainers compatibility
3. **No client leakage** - Public API never exposes the Docker client; strategies work against `ContainerTarget`
4. **Context-aware** - All operations respect context cancellation and timeouts
5. **Observable** - Built-in OpenTelemetry v2 support for production monitoring

## Troubleshooting

### Container fails to start

```go
if err := exec.Start(ctx); err != nil {
    // Check logs for startup errors
    logs, _ := exec.GetStderr(ctx)
    fmt.Println("Error logs:", logs)

    // Check container status
    status, _ := exec.Status(ctx)
    fmt.Printf("State: %s, Error: %s\n", status.State, status.Error)
}
```

### Port already in use

```go
// Use random port (0)
docker.WithPorts("80:0")  // Host port auto-assigned
```

### Wait strategy timeout

```go
// Increase timeout
docker.WithWaitStrategy(
    docker.WaitForLog("ready").
        WithStartupTimeout(120 * time.Second),  // 2 minutes
)
```

If a log-based wait never succeeds, check that the pattern is a *regex* that matches a single log line as the container actually prints it — `docker logs <container>` shows the ground truth.

### Image pull fails

```go
// Pull manually first
exec, _ := docker.New(docker.WithImage("myregistry.com/image:tag"))

// Or check pull errors
if err := exec.Start(ctx); err != nil {
    if strings.Contains(err.Error(), "pull") {
        fmt.Println("Image pull failed - check registry credentials")
    }
}
```

## Related Packages

- **[otel](../otel/)** - OpenTelemetry v2 configuration and utilities
- **[config](../config/)** - Configuration management with validation
- **[logging](../logging/)** - Structured logging with context
