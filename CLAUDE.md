# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository Overview

Production-ready Go utility library (v2) with OpenTelemetry instrumentation. 16 core packages for building cloud-native applications: otel, config, logging, db, docker, server, grpc, rest, concurrent, temporal, ssh, compress, argo, retry, base32, and examples.

**Module Path:** `github.com/jasoet/pkg/v2`  
**Go Version:** 1.24+ (uses generics)  
**Test Coverage:** 85%

## Development Commands

All commands use Taskfile. Check `task --list` for full list.

### Testing

```bash
# Unit tests (no Docker required, fast)
task test

# Integration tests (requires Docker daemon, 15min timeout)
# Uses testcontainers for temporal, docker, db, ssh packages
task test:integration

# Argo tests (requires Kubernetes cluster with Argo Workflows)
task test:argo

# Complete test suite (unit + integration + argo, 20min timeout)
task test:complete

# Check infrastructure availability
task docker:check  # Verify Docker daemon running
task k8s:check     # Verify kubectl and cluster access
task argo:check    # Verify Argo Workflows installation
```

**Coverage reports:** Generated in `output/coverage*.html`

### Code Quality

```bash
task lint          # Run golangci-lint
task fmt           # Format with gofumpt
task vendor        # Update dependencies (go mod tidy + vendor)
task clean         # Remove build artifacts
```

### Running Examples

```bash
# Each package has runnable examples
go run -tags=example ./logging/examples
go run -tags=example ./db/examples
go run -tags=example ./server/examples

# Build all examples (verifies compilation)
go build -tags=example ./...
```

## Architecture Patterns

### 1. Functional Options Pattern

Dominant pattern across all packages for flexible, backward-compatible configuration:

```go
// docker package
exec, _ := docker.New(
    docker.WithImage("nginx:alpine"),
    docker.WithPorts("80:8080"),
    docker.WithOTelConfig(otelConfig),
)

// grpc package
server, _ := grpc.New(
    grpc.WithGRPCPort("9090"),
    grpc.WithOTelConfig(otelConfig),
)
```

**When creating new functionality:** Follow this pattern. Add `With*()` option functions rather than expanding struct fields directly.

### 2. OpenTelemetry Integration (Universal Pattern)

**All packages** follow this OTel integration pattern:

```go
// 1. Config struct has optional OTelConfig field
type Config struct {
    Host       string
    Port       int
    OTelConfig *otel.Config `yaml:"-" mapstructure:"-"` // Not serializable
}

// 2. Functional option for runtime injection
func WithOTelConfig(cfg *otel.Config) Option {
    return func(o *Options) { o.OTelConfig = cfg }
}

// 3. Access OTel providers
tracer := cfg.GetTracer()
meter := cfg.GetMeter()
logger := cfg.GetLogger()

// 4. Context propagation
ctx = otel.ContextWithConfig(ctx, cfg)
// Downstream:
cfg := otel.ConfigFromContext(ctx)
```

**Key files:**
- `otel/config.go` - Central OTel configuration
- `otel/instrumentation.go` - Layered span helpers
- `otel/logging.go` - OTel logger provider integration

### 3. LayerContext Pattern (Unified Span + Logger)

Use `otel.Layers.Start*()` for automatic span + logger correlation:

```go
// Available layers: Handler, Service, Repository, Operations, Middleware
lc := otel.Layers.StartService(ctx, "user", "Create", 
    otel.F("user_id", id),
    otel.F("email", email))
defer lc.End()

lc.Logger.Info("Creating user")

if err := repo.Save(lc.Context(), user); err != nil {
    return lc.Error(err, "failed to save user") // Logs + sets span error + returns error
}

return lc.Success("User created successfully") // Logs at info level
```

**Benefits:**
- Automatic trace-log correlation
- Consistent span naming: `{layer}.{component}.{operation}`
- Base fields automatically included in Error/Success logs
- Single object instead of separate span + logger management

### 4. Configuration Layers

Three-layer configuration strategy:

**Layer 1: YAML Files**
```go
cfg, err := config.LoadString[AppConfig](yamlContent)
// Or from file:
cfg, err := config.LoadString[AppConfig](yamlContent, "APP")
```
- Type-safe with generics (Go 1.24+)
- Struct tags: `yaml:"field" mapstructure:"field" validate:"required"`

**Layer 2: Environment Variable Overrides**
- Automatic via Viper: `APP_FIELD_NAME=value`
- Nested support: Use `NestedEnvVars()` for complex structures
- Convention: underscore separator, uppercase, prefix-based

**Layer 3: Functional Options (Runtime)**
```go
// Override config at runtime
db.Pool(
    db.WithConfig(dbConfig),
    db.WithOTelConfig(otelConfig), // Not in YAML
)
```

**Important:** `OTelConfig` is **never** serialized. Always use `yaml:"-" mapstructure:"-"` tags and inject via functional options.

### 5. No-Op Provider Pattern

All OTel providers gracefully default to no-op when nil:
- Zero runtime overhead when observability disabled
- No nil checks needed in application code
- Production-ready observability as opt-in

## Testing Strategy

### Test Categories (Build Tags)

**Unit Tests** (no build tag)
- No external dependencies (no Docker, no k8s)
- Fast execution, race detection enabled
- Run: `task test`

**Integration Tests** (`//go:build integration`)
- Uses testcontainers (requires Docker daemon)
- Tests against real infrastructure: PostgreSQL, MySQL, MSSQL, Temporal, etc.
- 15-minute timeout
- Run: `task test:integration`

**Argo Tests** (`//go:build argo`)
- Requires Kubernetes cluster with Argo Workflows installed
- Tests argo package against real cluster
- Run: `task test:argo`

**Example Code** (`//go:build example`)
- Excluded from normal builds/tests
- Runnable demonstrations
- Run: `go run -tags=example ./package/examples`

### Testing Patterns

**Assertion Library:** Use `github.com/stretchr/testify/assert` for all test assertions:

```go
import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestFeature(t *testing.T) {
    result, err := DoSomething()
    require.NoError(t, err)  // Fail immediately if error
    assert.Equal(t, expected, result)
    assert.NotNil(t, result.Field)
}
```

**Integration Test Pattern:**

```go
//go:build integration

package mypackage_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/testcontainers/testcontainers-go"
)

func TestIntegration_Feature(t *testing.T) {
    ctx := context.Background()
    
    // 1. Start testcontainer
    container, err := startTestContainer(ctx)
    require.NoError(t, err)
    defer container.Terminate(ctx)
    
    // 2. Test against real infrastructure
    result, err := mypackage.DoSomething(ctx, container.Endpoint())
    
    // 3. Assert expectations
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
}
```

**Testcontainer Helper Pattern:**

See `temporal/testcontainer/` for reusable pattern:
```go
// Provides: container + client + cleanup in one call
container, client, cleanup, err := testcontainer.Setup(ctx, config, options)
defer cleanup()
```

## Package-Specific Patterns

### docker Package - Dual API Pattern

Supports two API styles:

```go
// Style 1: Functional options (fluent)
exec, _ := docker.New(
    docker.WithImage("nginx:alpine"),
    docker.WithPorts("80:8080"),
    docker.WithWaitStrategy(docker.WaitForLog("ready")),
)

// Style 2: Struct-based (testcontainers-compatible)
req := docker.ContainerRequest{
    Image:        "postgres:16-alpine",
    ExposedPorts: []string{"5432/tcp"},
    Env:          map[string]string{"POSTGRES_PASSWORD": "secret"},
    WaitingFor:   docker.WaitForLog("ready to accept connections"),
}
exec, _ := docker.NewFromRequest(req)
```

Both styles support same lifecycle: `Start()`, `Stop()`, `Restart()`, `Terminate()`, `Wait()`

### retry Package - OTel Integration

```go
cfg := retry.DefaultConfig().
    WithName("db.connect").
    WithOTel(otelConfig)

err := retry.Do(ctx, cfg, func(ctx context.Context) error {
    return db.Ping(ctx)
})

// For non-retryable errors:
return retry.Permanent(errors.New("invalid config"))
```

Automatically creates spans with attempt counts and logs each retry.

### db Package - Multi-Database Support

```go
pool, _ := db.ConnectionConfig{
    DbType:     db.Postgresql, // or Mysql, MSSQL
    Host:       "localhost",
    Port:       5432,
    Username:   "user",
    Password:   "pass",
    DbName:     "mydb",
    OTelConfig: otelConfig, // Automatic query tracing
}.Pool()

// Automatic GORM + OTel tracing for all queries
pool.Find(&users)
```

Supports migrations via golang-migrate.

### config Package - Type-Safe Loading

```go
type AppConfig struct {
    Server struct {
        Port    int           `yaml:"port" validate:"required,min=1,max=65535"`
        Timeout time.Duration `yaml:"timeout" validate:"min=3s"`
    } `yaml:"server"`
    
    Database struct {
        Host string `yaml:"host" validate:"required"`
        Port int    `yaml:"port" validate:"required"`
    } `yaml:"database"`
}

// Load from string (useful for tests/examples)
cfg, err := config.LoadString[AppConfig](yamlContent, "APP")

// Load from file
cfg, err := config.LoadString[AppConfig](yamlContent, "APP")
```

Environment override: `APP_SERVER_PORT=8080`

### argo Package - Flexible Connection

```go
// Default: kubeconfig from home directory
ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())

// With functional options
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/path/to/kubeconfig"),
    argo.WithContext("production"),
    argo.WithOTelConfig(otelConfig),
)

// In-cluster mode (when running in k8s pod)
ctx, client, err := argo.NewClient(ctx, argo.InClusterConfig())

// Argo Server mode
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithArgoServer("https://argo-server:2746", "Bearer token"),
)

defer client.Close()
```

## Common Development Tasks

### Adding a New Package

1. Create package directory with standard structure:
   ```
   newpackage/
   ├── README.md           # Package documentation
   ├── config.go           # Config struct with OTelConfig field
   ├── newpackage.go       # Core implementation
   ├── newpackage_test.go  # Unit tests (no build tag)
   ├── integration_test.go # Integration tests (//go:build integration)
   └── examples/
       ├── README.md
       └── main.go         # Runnable example (//go:build example)
   ```

2. Follow patterns:
   - Functional options for configuration
   - `OTelConfig *otel.Config` field with `yaml:"-" mapstructure:"-"`
   - Use `otel.Layers.Start*()` for instrumentation
   - Use testify/assert for tests

3. Add to main README.md package table

### Adding Integration Tests

1. Tag file: `//go:build integration`
2. Use testcontainers when possible
3. Check Docker availability: Ensure `task docker:check` passes
4. Use cleanup patterns: `defer container.Terminate(ctx)`
5. 15-minute timeout is default

### Adding Instrumentation to Existing Code

```go
// Before
func ProcessUser(ctx context.Context, id string) error {
    user, err := db.FindUser(ctx, id)
    if err != nil {
        return err
    }
    // process...
    return nil
}

// After
func ProcessUser(ctx context.Context, id string) error {
    lc := otel.Layers.StartService(ctx, "user", "Process", otel.F("user_id", id))
    defer lc.End()
    
    user, err := db.FindUser(lc.Context(), id)
    if err != nil {
        return lc.Error(err, "failed to find user")
    }
    
    lc.Logger.Info("Processing user")
    // process...
    
    return lc.Success("User processed")
}
```

## Version Information

- **Current:** v2.0.0 GA (includes OpenTelemetry)
- **Stable v1:** v1.5.0 (no OpenTelemetry, maintenance only)
- **Migration Guide:** See VERSIONING_GUIDE.md in repository root

v2 adds optional OpenTelemetry support with minimal API changes from v1.

## Infrastructure Requirements

**Docker Engine:** Required for integration tests (`task test:integration`)
- Verify: `task docker:check`
- Testcontainers needs Docker daemon accessible

**Kubernetes Cluster:** Required for Argo tests (`task test:argo`)
- Verify: `task k8s:check`
- Argo Workflows must be installed: `task argo:check`

**Go 1.24+:** Repository uses generics extensively
