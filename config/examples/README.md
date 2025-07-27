# Config Package Examples

This directory contains examples demonstrating how to use the `config` package for configuration management in Go applications.

## 📍 Example Code Location

**Full example implementation:** [/config/examples/example.go](https://github.com/jasoet/pkg/blob/main/config/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/config"

// Load config from YAML string
config, err := config.LoadString[YourConfigType](yamlString)

// With custom ENV prefix (default is "ENV")
config, err := config.LoadString[YourConfigType](yamlString, "MYAPP")

// With custom configuration function
config, err := config.LoadStringWithConfig[YourConfigType](yamlString, func(v *viper.Viper) {
    // Custom configuration logic
})
```

**Critical naming convention:** YAML fields use CamelCase, environment variables preserve the casing:
- YAML: `checkInterval` → ENV: `PREFIX_CHECKINTERVAL` (NOT `PREFIX_CHECK_INTERVAL`)
- Nested: `database.connectionTimeout` → ENV: `PREFIX_DATABASE_CONNECTIONTIMEOUT`

## Overview

The `config` package provides flexible configuration loading from YAML strings with support for:
- Environment variable overrides
- Custom environment variable prefixes
- Custom configuration functions
- Nested environment variable processing

## Important: CamelCase Convention for YAML and Environment Variables

This package uses CamelCase convention for YAML field names to maintain consistency with environment variable naming. This is crucial to understand for proper configuration:

### YAML Field Naming

When defining your configuration struct, use CamelCase in your YAML tags:

```go
type Config struct {
    Targets       []string      `yaml:"targets"`
    CheckInterval time.Duration `yaml:"checkInterval"`  // CamelCase in YAML
    Timeout       time.Duration `yaml:"timeout"`
    ListenPort    int           `yaml:"listenPort"`     // CamelCase in YAML
    InstanceID    string        `yaml:"instanceId"`      // CamelCase in YAML
    Retries       int           `yaml:"retries"`
    LogLevel      string        `yaml:"logLevel"`        // CamelCase in YAML
}
```

### Environment Variable Naming

**Important**: CamelCase YAML fields are NOT converted to snake_case for environment variables. Instead, they are converted to UPPERCASE while preserving the casing structure:

- `checkInterval` → `PREFIX_CHECKINTERVAL` (NOT `PREFIX_CHECK_INTERVAL`)
- `listenPort` → `PREFIX_LISTENPORT` (NOT `PREFIX_LISTEN_PORT`)
- `instanceId` → `PREFIX_INSTANCEID` (NOT `PREFIX_INSTANCE_ID`)

### Nested Structures

For nested structures, underscores are used to separate the nested levels:

```go
type TestConfig struct {
    Name    string `yaml:"name"`
    Version string `yaml:"version"`
    Nested  struct {
        Value int `yaml:"value"`
        SubNested struct {
            DeepValue string `yaml:"deepValue"`
        } `yaml:"subNested"`
    } `yaml:"nested"`
}
```

Environment variable mapping:
- `nested.value` → `PREFIX_NESTED_VALUE`
- `nested.subNested.deepValue` → `PREFIX_NESTED_SUBNESTED_DEEPVALUE`

### Rationale

This convention ensures consistent environment variable naming across all configuration levels, avoiding ambiguity when dealing with nested structures or fields that already contain underscores.

## Running the Examples

To run the examples, use the following command from the `config/examples` directory:

```bash
go run example.go
```

## Example Descriptions

The [example.go](https://github.com/jasoet/pkg/blob/main/config/examples/example.go) file demonstrates several use cases:

### 1. Basic Configuration Loading

Loads a YAML configuration string into a strongly-typed struct.

```go
appConfig, err := config.LoadString[AppConfig](yamlConfig)
```

### 2. Environment Variable Overrides

Shows how environment variables automatically override configuration values.

```go
os.Setenv("ENV_NAME", "env-app")
os.Setenv("ENV_DATABASE_HOST", "db.example.com")
appConfig, err = config.LoadString[AppConfig](yamlConfig)
```

### 3. Custom Environment Prefix

Demonstrates using a custom prefix for environment variables.

```go
os.Setenv("CUSTOM_NAME", "custom-app")
appConfig, err = config.LoadString[AppConfig](yamlConfig, "CUSTOM")
```

### 4. Custom Configuration Function

Shows how to use a custom configuration function to modify the configuration.

```go
customConfigFn := func(v *viper.Viper) {
    v.Set("name", "custom-function-app")
    v.Set("database.host", "custom-function-db.example.com")
}
appConfig, err = config.LoadStringWithConfig[AppConfig](yamlConfig, customConfigFn)
```

### 5. Nested Environment Variables

Demonstrates processing nested environment variables for complex configurations.

```go
nestedConfigFn := func(v *viper.Viper) {
    nestedEnvPrefix := strings.ToUpper("APP_GOERS_ACCOUNTS_")
    config.NestedEnvVars(nestedEnvPrefix, 3, "goers.accounts", v)
}
nestedConfig, err := config.LoadStringWithConfig[NestedConfig](nestedYamlConfig, nestedConfigFn)
```

## Configuration Structs

The examples use two configuration structs:

1. `AppConfig` - A general application configuration with database and service settings
2. `NestedConfig` - A configuration with nested structures for demonstrating complex environment variable handling

## Key Features

- **Type Safety**: Using Go generics for type-safe configuration
- **Environment Variables**: Automatic binding of environment variables to configuration
- **Customization**: Flexible customization through configuration functions
- **Nested Structures**: Support for complex nested configuration structures