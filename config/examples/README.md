# Config Package Examples

This directory contains examples demonstrating how to use the `config` package for configuration management in Go applications.

## Overview

The `config` package provides flexible configuration loading from YAML strings with support for:
- Environment variable overrides
- Custom environment variable prefixes
- Custom configuration functions
- Nested environment variable processing

## Running the Examples

To run the examples, use the following command from the `config/examples` directory:

```bash
go run example.go
```

## Example Descriptions

The example.go file demonstrates several use cases:

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