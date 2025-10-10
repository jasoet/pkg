# Configuration Management

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/config.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/config)

Type-safe YAML configuration with environment variable overrides using Viper and Go generics.

## Overview

The `config` package provides a simple, type-safe way to load configuration from YAML strings with automatic environment variable support. Built on top of Viper, it leverages Go generics for compile-time type safety.

## Features

- **Type-Safe**: Generic functions ensure compile-time type checking
- **Environment Overrides**: Automatic environment variable support with configurable prefix
- **Nested Configuration**: Support for complex nested structures
- **Custom Processing**: Hook into Viper for advanced configuration
- **Zero Dependencies**: Only requires Viper (already used in most Go projects)
- **Simple API**: Load configuration in one function call

## Installation

```bash
go get github.com/jasoet/pkg/v2/config
```

## Quick Start

### Basic Usage

```go
package main

import (
    "github.com/jasoet/pkg/v2/config"
)

type AppConfig struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
    Server  struct {
        Port int    `yaml:"port"`
        Host string `yaml:"host"`
    } `yaml:"server"`
}

func main() {
    yamlConfig := `
name: my-app
version: 1.0.0
server:
  port: 8080
  host: localhost
`

    cfg, err := config.LoadString[AppConfig](yamlConfig)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Starting %s v%s on %s:%d\n",
        cfg.Name, cfg.Version, cfg.Server.Host, cfg.Server.Port)
}
```

### Environment Variable Overrides

By default, environment variables with `ENV_` prefix override YAML values:

```go
// YAML config
yamlConfig := `
name: my-app
version: 1.0.0
`

// Environment variables
// ENV_NAME=prod-app
// ENV_VERSION=2.0.0

cfg, err := config.LoadString[AppConfig](yamlConfig)
// cfg.Name = "prod-app" (from env)
// cfg.Version = "2.0.0" (from env)
```

**Nested keys** use underscores:
```bash
# Override server.port
export ENV_SERVER_PORT=9090

# Override database.host
export ENV_DATABASE_HOST=prod-db.example.com
```

### Custom Environment Prefix

```go
// Use custom prefix
cfg, err := config.LoadString[AppConfig](yamlConfig, "MYAPP")

// Now use MYAPP_* environment variables
// MYAPP_NAME=prod-app
// MYAPP_SERVER_PORT=9090
```

## API Reference

### LoadString

Load configuration from YAML string with environment variable support:

```go
func LoadString[T any](configString string, envPrefix ...string) (*T, error)
```

**Parameters:**
- `configString`: YAML configuration string
- `envPrefix`: Optional environment variable prefix (default: `"ENV"`)

**Returns:**
- `*T`: Pointer to populated configuration struct
- `error`: Error if parsing or unmarshaling fails

**Example:**
```go
cfg, err := config.LoadString[AppConfig](yamlString)
cfg, err := config.LoadString[AppConfig](yamlString, "CUSTOM")
```

### LoadStringWithConfig

Advanced loading with custom Viper configuration:

```go
func LoadStringWithConfig[T any](
    configString string,
    configFn func(*viper.Viper),
    envPrefix ...string,
) (*T, error)
```

**Parameters:**
- `configString`: YAML configuration string
- `configFn`: Custom function to modify Viper before unmarshaling
- `envPrefix`: Optional environment variable prefix (default: `"ENV"`)

**Example:**
```go
customFn := func(v *viper.Viper) {
    v.Set("defaults.timeout", 30)
    v.SetDefault("debug", false)
}

cfg, err := config.LoadStringWithConfig[AppConfig](yamlString, customFn)
```

### NestedEnvVars

Process nested environment variables for dynamic configuration:

```go
func NestedEnvVars(
    prefix string,
    keyDepth int,
    configPath string,
    viperConfig *viper.Viper,
)
```

**Use Case**: Load entity-specific configuration from environment variables.

**Example:**
```go
// Environment variables:
// TEST_GOERS_ACCOUNTS_USER_NAME=john
// TEST_GOERS_ACCOUNTS_USER_EMAIL=john@example.com
// TEST_GOERS_ACCOUNTS_ADMIN_NAME=admin
// TEST_GOERS_ACCOUNTS_ADMIN_EMAIL=admin@example.com

customFn := func(v *viper.Viper) {
    config.NestedEnvVars("TEST_GOERS_ACCOUNTS_", 3, "goers.accounts", v)
}

type Config struct {
    Goers struct {
        Accounts map[string]map[string]string `yaml:"accounts"`
    } `yaml:"goers"`
}

cfg, _ := config.LoadStringWithConfig[Config](yamlString, customFn)

// Access nested values
userName := cfg.Goers.Accounts["user"]["name"]  // "john"
adminEmail := cfg.Goers.Accounts["admin"]["email"]  // "admin@example.com"
```

## Advanced Examples

### Database Configuration

```go
type DatabaseConfig struct {
    Type     string `yaml:"type"`
    Host     string `yaml:"host"`
    Port     int    `yaml:"port"`
    Username string `yaml:"username"`
    Password string `yaml:"password"`
    Database string `yaml:"database"`
}

yamlConfig := `
type: postgresql
host: localhost
port: 5432
username: admin
database: myapp
`

// Override sensitive data via env vars
// ENV_PASSWORD=secret123
// ENV_HOST=prod-db.example.com

cfg, err := config.LoadString[DatabaseConfig](yamlConfig)
// cfg.Host = "prod-db.example.com"
// cfg.Password = "secret123"
```

### Multi-Environment Setup

```go
type Environment struct {
    Name     string
    Database DatabaseConfig
    Server   ServerConfig
}

// Development
devYaml := `
name: development
database:
  host: localhost
server:
  port: 8080
`

// Production (override with env vars)
// ENV_DATABASE_HOST=prod-db.example.com
// ENV_SERVER_PORT=443

cfg, err := config.LoadString[Environment](devYaml)
```

### Slice Configuration

```go
type FeatureConfig struct {
    Name     string   `yaml:"name" mapstructure:"name"`
    Tags     []string `yaml:"tags" mapstructure:"tags"`
    Features []string `yaml:"features" mapstructure:"features"`
}

yamlConfig := `
name: my-service
tags:
  - api
  - grpc
  - rest
features:
  - auth
  - logging
`

cfg, err := config.LoadString[FeatureConfig](yamlConfig)
// cfg.Tags = []string{"api", "grpc", "rest"}

// Override with env (comma-separated)
// ENV_TAGS=production,kubernetes,ha
// cfg.Tags = []string{"production", "kubernetes", "ha"}
```

### Integration with OTel Config

```go
import (
    "github.com/jasoet/pkg/v2/config"
    "github.com/jasoet/pkg/v2/otel"
)

type AppConfig struct {
    Service struct {
        Name    string `yaml:"name"`
        Version string `yaml:"version"`
    } `yaml:"service"`
    OTel struct {
        Endpoint string `yaml:"endpoint"`
        Insecure bool   `yaml:"insecure"`
    } `yaml:"otel"`
}

yamlConfig := `
service:
  name: my-service
  version: 1.0.0
otel:
  endpoint: localhost:4317
  insecure: true
`

cfg, _ := config.LoadString[AppConfig](yamlConfig)

// Use in OTel setup
otelConfig := otel.NewConfig(cfg.Service.Name).
    WithServiceVersion(cfg.Service.Version)
```

## Best Practices

### 1. Define Struct Tags

```go
// ✅ Good: Use both yaml and mapstructure tags
type Config struct {
    Port int `yaml:"port" mapstructure:"port"`
}

// ⚠️ May cause issues with env override
type Config struct {
    Port int `yaml:"port"` // missing mapstructure
}
```

### 2. Use Pointers for Optional Fields

```go
// ✅ Good: Optional fields are pointers
type Config struct {
    Required string  `yaml:"required"`
    Optional *string `yaml:"optional"`
}

// Check before using
if cfg.Optional != nil {
    fmt.Println(*cfg.Optional)
}
```

### 3. Validate After Loading

```go
import "github.com/go-playground/validator/v10"

type Config struct {
    Port int    `yaml:"port" validate:"required,min=1,max=65535"`
    Host string `yaml:"host" validate:"required,hostname"`
}

cfg, err := config.LoadString[Config](yamlString)
if err != nil {
    return err
}

validate := validator.New()
if err := validate.Struct(cfg); err != nil {
    return fmt.Errorf("invalid config: %w", err)
}
```

### 4. Environment-Specific Defaults

```go
customFn := func(v *viper.Viper) {
    // Set defaults for production
    if os.Getenv("APP_ENV") == "production" {
        v.SetDefault("server.timeout", 30)
        v.SetDefault("logging.level", "info")
    } else {
        v.SetDefault("server.timeout", 60)
        v.SetDefault("logging.level", "debug")
    }
}

cfg, _ := config.LoadStringWithConfig[AppConfig](yamlString, customFn)
```

### 5. Secrets Management

```go
// ✅ Good: Never commit secrets to YAML
yamlConfig := `
database:
  host: localhost
  port: 5432
  # username and password from env vars only
`

// Set via environment
// ENV_DATABASE_USERNAME=admin
// ENV_DATABASE_PASSWORD=secret123

cfg, _ := config.LoadString[DatabaseConfig](yamlConfig)
```

## Testing

The package includes comprehensive tests with 94.7% coverage:

```bash
# Run tests
go test ./config -v

# With coverage
go test ./config -cover
```

### Test Utilities

```go
func TestMyConfig(t *testing.T) {
    yamlConfig := `
    name: test-app
    version: 1.0.0
    `

    // Set test env vars
    t.Setenv("ENV_NAME", "test-override")

    cfg, err := config.LoadString[TestConfig](yamlConfig)
    assert.NoError(t, err)
    assert.Equal(t, "test-override", cfg.Name)
}
```

## Troubleshooting

### Environment Variables Not Working

**Problem**: Env vars not overriding YAML values

**Solution**:
```go
// 1. Check the prefix
cfg, _ := config.LoadString[T](yaml, "MYAPP")  // Use MYAPP_*

// 2. Check the key format (dots become underscores)
// YAML: server.port -> ENV: ENV_SERVER_PORT

// 3. Ensure mapstructure tags exist
type Config struct {
    Port int `yaml:"port" mapstructure:"port"`  // Both tags needed
}
```

### Nested Config Not Loading

**Problem**: Nested environment variables not working

**Solution**:
```go
// Use NestedEnvVars for dynamic nested structures
customFn := func(v *viper.Viper) {
    config.NestedEnvVars("PREFIX_", keyDepth, "config.path", v)
}

cfg, _ := config.LoadStringWithConfig[T](yaml, customFn)
```

### Type Mismatch Errors

**Problem**: Viper can't unmarshal to struct

**Solution**:
```go
// Ensure types match YAML values
type Config struct {
    Port int `yaml:"port"`  // ✅ Use int for numbers
    // Port string `yaml:"port"`  // ❌ Will fail if YAML has number
}
```

## Performance

- **Lightweight**: Minimal overhead over direct Viper usage
- **Type-Safe**: No reflection at runtime (only during unmarshal)
- **Efficient**: Viper caches parsed values

Benchmark (typical config load):
```
BenchmarkLoadString-8    50000    ~30 µs/op
```

## Version Compatibility

- **Viper**: v1.21.0+
- **Go**: 1.25+ (generics required)
- **pkg library**: v2.0.0+

## Migration from Direct Viper

```go
// Before (direct Viper)
v := viper.New()
v.SetConfigType("yaml")
v.ReadConfig(strings.NewReader(yamlString))
var cfg AppConfig
v.Unmarshal(&cfg)

// After (this package)
cfg, err := config.LoadString[AppConfig](yamlString)
```

## Examples

See [examples/](.../examples/config/config/) directory for:
- Basic configuration loading
- Environment variable overrides
- Custom Viper configuration
- Nested configuration handling
- Integration with other packages

## Related Packages

- **[otel](../otel/)** - OpenTelemetry configuration
- **[db](../db/)** - Database configuration
- **[server](../server/)** - HTTP server configuration
- **[grpc](../grpc/)** - gRPC server configuration

## License

MIT License - see [LICENSE](../LICENSE) for details.
