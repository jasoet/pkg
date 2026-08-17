# Config Examples

This directory contains examples demonstrating the config package functionality.

## Running Examples

```bash
# From the repository root
go run -tags=example ./examples/config

# Or by module path
go run -tags=example github.com/jasoet/pkg/v3/examples/config
```

## Examples Included

### 1. Basic Configuration Loading
Loads a YAML string into a strongly-typed struct with `config.LoadString[AppConfig](yamlConfig)`.

### 2. Environment Variable Overrides
Environment variables automatically override YAML values: `ENV_DATABASE_HOST` overrides `database.host` (the default prefix is `ENV`).

### 3. Custom Environment Prefix
`config.LoadString[AppConfig](yamlConfig, "CUSTOM")` switches the env prefix, so `CUSTOM_DATABASE_HOST` applies instead.

### 4. Custom Option
`config.LoadStringWithOptions` accepts any `func(*viper.Viper)` as an `Option` — here a custom function that sets values directly on the Viper instance.

### 5. Nested Environment Variables
`config.WithNestedEnvVars(prefix, keyDepth, configPath)` maps prefixed env vars onto a map-typed config section. `keyDepth` is **prefix-relative**: the prefix is stripped first, then `keyDepth` indexes the remaining underscore-split tokens to locate the entity name. With prefix `APP_GOERS_ACCOUNTS_`, env `APP_GOERS_ACCOUNTS_USER_NAME` yields tokens `[USER, NAME]`, so `keyDepth=0` treats `USER` as the entity and `NAME` as the field under config path `goers.accounts`.

## Important: CamelCase Convention

YAML field names are CamelCase, and environment variables preserve that casing in uppercase — they are NOT converted to snake_case:

- `checkInterval` → `PREFIX_CHECKINTERVAL` (NOT `PREFIX_CHECK_INTERVAL`)
- `database.connectionTimeout` → `PREFIX_DATABASE_CONNECTIONTIMEOUT`

Underscores in env var names separate nesting levels only.

## Expected Output

```
Example 1: Basic configuration loading
App Name: my-app
Version: 1.0.0
Database Host: localhost
Auth Service URL: http://auth-service:8080

Example 2: Using environment variables to override configuration
App Name (from env): env-app
Database Host (from env): db.example.com
Auth Service URL (from env): https://auth.example.com

Example 3: Using custom environment prefix
App Name (from custom env): custom-app
Database Host (from custom env): custom-db.example.com

Example 4: Using a custom option
App Name (from custom option): custom-function-app
Database Host (from custom function): custom-function-db.example.com
Payment Service Enabled: false

Example 5: Using WithNestedEnvVars for complex environment variable handling
Nested App Name: env-app
User Name: john
User Email: john@example.com
Admin Name: admin
Admin Email: admin@example.com
```

## Learn More

- [Config Package Documentation](../../config/README.md)
- [API Reference](https://pkg.go.dev/github.com/jasoet/pkg/v3/config)
- [Example source](./example.go)
