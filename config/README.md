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

> **Important — env vars can only *override*, never *introduce* keys.** Viper's `AutomaticEnv` only overrides keys it already knows about, i.e. keys present in the YAML **or** registered with `WithDefaults`. An environment variable for a key that is absent from both is **silently ignored** (`ENV_PORT=9090` with no `port` in the YAML and no default leaves `Port == 0`). To make a key env-overridable without requiring it in the YAML, register a default for it:
>
> ```go
> cfg, err := config.LoadStringWithOptions[AppConfig](yamlConfig,
>     config.WithDefaults(map[string]any{"server.port": 8080}), // now ENV_SERVER_PORT applies even if absent from YAML
> )
> ```
>
> This limitation is proved by `TestAutomaticEnv_AbsentKeyIgnored` in `config_test.go`.

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

- `prefix`: prefix of the environment variables to process (e.g. `"APP"`). The match is **anchored at an underscore boundary**: the prefix is normalized to end with a single `_`, so `"APP"` (or `"APP_"`) matches `APP_FOO` but never `APPLE_FOO`. An **empty prefix is rejected** (it would otherwise sweep the entire environment into your config) and processes nothing.
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

Note: unlike the flat `ENV_` override mechanism (which overrides YAML), `WithNestedEnvVars` never overrides YAML keys. "Absent from the YAML" is judged by YAML presence only (`InConfig`), **not** by defaults — so a nested env var still fills (and therefore beats) a key that was only set via `WithDefaults`, preserving `env > default` precedence.

**Migrating from v2 `NestedEnvVars`:** `keyDepth` is now prefix-relative — subtract the number of prefix tokens from your old `keyDepth` value (e.g. old `2` with prefix `"MY_APP_"` becomes `1`; old `1` with prefix `"APP"` becomes `0`).

## Struct Tags

**Use `mapstructure` tags (or `mapstructure` alongside `yaml`) whenever a config key differs from the lowercased field name.** `LoadString`/`LoadStringWithOptions` decode with `viper.Unmarshal`, which matches keys using the **`mapstructure`** tag only — the `yaml` tag is never consulted during unmarshaling. Without a `mapstructure` tag, a field falls back to matching on its Go field name (case-insensitively).

This matters because matching is case-insensitive but **not** separator-insensitive:

```go
// YAML: max_size: 42

// WRONG — yaml tag alone: field name "MaxSize" (→ "maxsize") never matches
// the key "max_size", so MaxSize is silently left at its zero value (0).
type Bad struct {
    MaxSize int `yaml:"max_size"`
}

// RIGHT — a mapstructure tag binds the snake_case key.
type Good struct {
    MaxSize int `mapstructure:"max_size"`
}
```

A plain `yaml` tag only *appears* to work when the key equals the lowercased field name (e.g. field `Name` with key `name`, as in the examples above). Any key with underscores, hyphens, or other punctuation — or that otherwise differs from the lowercased field name — requires a `mapstructure` tag. Carrying both `yaml` and `mapstructure` tags is a safe habit if the same structs are also (un)marshaled as YAML elsewhere.

This behavior is proved by `TestSnakeCaseKeyRequiresMapstructureTag` in `config_test.go`.

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
