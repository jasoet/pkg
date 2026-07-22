# Configuration Management

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/config.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/config)

Type-safe YAML configuration with environment variable overrides using Viper and Go generics.

## Overview

The `config` package provides a simple, type-safe way to load configuration from YAML strings with automatic environment variable support. Built on top of Viper, it leverages Go generics for compile-time type safety.

## Features

- **Type-Safe**: Generic functions ensure compile-time type checking
- **Environment Overrides**: Automatic environment variable support with configurable prefix
- **Functional Options**: `LoadStringWithOptions` accepts `Option` values (`WithEnvPrefix`, `WithDefaults`, `WithNestedEnvVars`) for advanced configuration
- **Nested Configuration**: Map environment variables onto map-typed config sections with `WithNestedEnvVars`
- **Simple API**: Load configuration in one function call

## Installation

```bash
go get github.com/jasoet/pkg/v3/config
```

## Quick Start

Compile-checked versions of these snippets live in [`example_test.go`](example_test.go); a runnable end-to-end program lives in [`examples/config/`](../examples/config/).

### Basic Usage

```go
package main

import (
    "fmt"

    "github.com/jasoet/pkg/v3/config"
)

type AppConfig struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
    Server  struct {
        Host string `yaml:"host"`
        Port int    `yaml:"port"`
    } `yaml:"server"`
}

func main() {
    yamlConfig := `
name: my-app
version: 1.0.0
server:
  host: localhost
  port: 8080
`

    cfg, err := config.LoadString[AppConfig](yamlConfig)
    if err != nil {
        panic(err)
    }

    fmt.Printf("%s v%s on %s:%d\n",
        cfg.Name, cfg.Version, cfg.Server.Host, cfg.Server.Port)
}
```

### Environment Variable Overrides

By default, environment variables with the `ENV_` prefix override YAML values. Dots in nested keys become underscores (`server.port` → `ENV_SERVER_PORT`):

```go
os.Setenv("ENV_SERVER_PORT", "9090")

cfg, err := config.LoadString[AppConfig](yamlConfig)
// cfg.Server.Port == 9090 (from env), other fields from YAML
```

### Custom Environment Prefix

Pass a prefix as the second argument to `LoadString` (only the first value is used; additional values are ignored):

```go
cfg, err := config.LoadString[AppConfig](yamlConfig, "MYAPP")
// Now MYAPP_* environment variables apply, e.g. MYAPP_SERVER_PORT=9090
```

## Options API

`LoadStringWithOptions` applies functional options after the YAML has been parsed and before unmarshaling:

```go
func LoadStringWithOptions[T any](configString string, opts ...Option) (*T, error)
```

An `Option` is a `func(*viper.Viper)`, so besides the provided constructors you can pass any custom function that mutates the underlying Viper instance (see `examples/config/` for a custom-option example).

### WithDefaults

Sets default values for keys absent from the YAML:

```go
cfg, err := config.LoadStringWithOptions[AppConfig](`server: {port: 8080}`,
    config.WithDefaults(map[string]any{"debug": true}),
    config.WithEnvPrefix("APP"),
)
// cfg.Debug == true (default), cfg.Server.Port == 8080 (YAML),
// or 9090 if APP_SERVER_PORT=9090 is set (env override)
```

### WithNestedEnvVars

Maps prefixed environment variables onto a map-typed config section:

```go
func WithNestedEnvVars(prefix string, keyDepth int, configPath string) Option
```

- `prefix`: prefix of the environment variables to process (e.g. `"APP"`).
- `keyDepth`: **prefix-relative** — the prefix is stripped first, then `keyDepth` indexes the remaining underscore-split tokens to locate the entity name; everything after it forms the field name.
- `configPath`: base path in the configuration where values are set.

```go
type Config struct {
    Users map[string]map[string]string `yaml:"users"`
}

// APP_USERS_ADMIN_NAME: strip "APP" -> ["USERS", "ADMIN", "NAME"];
// keyDepth 1 -> entity "admin", field "name" under path "users".
os.Setenv("APP_USERS_ADMIN_NAME", "alice")

cfg, err := config.LoadStringWithOptions[Config](``,
    config.WithNestedEnvVars("APP", 1, "users"),
)
// cfg.Users["admin"]["name"] == "alice"
```

**Precedence contract:** nested env vars fill only keys that are *absent* from the YAML. If the YAML already sets a key, the environment variable is ignored:

```go
os.Setenv("APP_USERS_ADMIN_NAME", "alice")
os.Setenv("APP_USERS_ADMIN_EMAIL", "alice@example.com")

cfg, _ := config.LoadStringWithOptions[Config](`users: {admin: {name: bob}}`,
    config.WithNestedEnvVars("APP", 1, "users"),
)
// cfg.Users["admin"]["name"]  == "bob"              (YAML wins)
// cfg.Users["admin"]["email"] == "alice@example.com" (filled from env)
```

Note: unlike the flat `ENV_` override mechanism (which overrides YAML), `WithNestedEnvVars` never overrides YAML keys.

**Migrating from v2 `NestedEnvVars`:** `keyDepth` is now prefix-relative — subtract the number of prefix tokens from your old `keyDepth` value (e.g. old `2` with prefix `"MY_APP_"` becomes `1`; old `1` with prefix `"APP"` becomes `0`).

## Struct Tags

Decoding is case-insensitive via mapstructure, so plain `yaml` tags (as used throughout these examples) are sufficient. Adding matching `mapstructure` tags is harmless but not required.

## Testing

```bash
go test ./config/ -v
```

The package's tests use `t.Setenv` to isolate environment variable fixtures.

## Examples

See [`examples/config/`](../examples/config/) for a runnable program covering:

- Basic configuration loading
- Environment variable overrides
- Custom environment prefix
- Custom `Option` functions
- Nested environment variables with `WithNestedEnvVars`

## Related Packages

- **[otel](../otel/)** - OpenTelemetry configuration
- **[db](../db/)** - Database configuration
- **[server](../server/)** - HTTP server configuration
- **[grpc](../grpc/)** - gRPC server configuration

## License

MIT License - see [LICENSE](../LICENSE) for details.
