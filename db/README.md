# Database Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/db.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/db)

Multi-database support with GORM, automated migrations, and OpenTelemetry instrumentation.

## Overview

The `db` package provides a unified interface for connecting to multiple database systems with optional OpenTelemetry tracing and metrics collection. Built on GORM and golang-migrate, it simplifies database operations while providing production-ready observability.

## Features

- **Multi-Database Support**: PostgreSQL, MySQL, MSSQL
- **GORM Integration**: Full ORM capabilities with GORM v2
- **Automatic Tracing**: Query-level distributed tracing via otelgorm
- **Connection Pool Metrics**: Real-time pool health monitoring
- **Schema Migrations**: Embedded PostgreSQL migrations with golang-migrate
- **Validated Configuration**: `Validate()` called automatically by `NewPool`
- **Optional OTel**: Tracing and metrics gated independently

## Installation

```bash
go get github.com/jasoet/pkg/v3/db
```

## Quick Start

### Basic Connection

```go
package main

import (
    "time"

    "github.com/jasoet/pkg/v3/db"
)

func main() {
    pool, err := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
        DBType:       db.Postgresql,
        Host:         "localhost",
        Port:         5432,
        Username:     "admin",
        Password:     "${DB_PASSWORD}",
        DBName:       "myapp",
        Timeout:      5 * time.Second,
        MaxIdleConns: 5,
        MaxOpenConns: 10,
    }))
    if err != nil {
        panic(err)
    }

    // Use GORM
    var users []User
    pool.Find(&users)
}
```

### With OpenTelemetry

```go
import (
    "github.com/jasoet/pkg/v3/db"
    "github.com/jasoet/pkg/v3/otel"
)

// Setup OTel with functional options
otelConfig := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider),
    otel.WithMeterProvider(meterProvider))

pool, err := db.NewPool(
    db.WithConnectionConfig(db.ConnectionConfig{
        DBType:       db.Postgresql,
        Host:         "localhost",
        Port:         5432,
        Username:     "admin",
        Password:     "${DB_PASSWORD}",
        DBName:       "myapp",
        Timeout:      5 * time.Second,
        MaxIdleConns: 5,
        MaxOpenConns: 10,
    }),
    db.WithOTelConfig(otelConfig), // Enable tracing & metrics
)

// All queries are automatically traced
pool.Find(&users)  // Creates span "db.SELECT"
pool.Create(&user) // Creates span "db.INSERT"
```

### Independent Tracing/Metrics Gates

Tracing and metrics are enabled independently:

- The **otelgorm query-tracing plugin** is installed only when `OTelConfig` is non-nil **and** tracing is enabled (i.e. a `TracerProvider` is set; see `otel.WithoutTracing()`). If tracing is on but metrics are off, the plugin is installed with `otelgorm.WithoutMetrics()`.
- **Pool metrics** (`db.client.connections.*`) are registered whenever `OTelConfig` is non-nil **and** metrics are enabled — regardless of whether tracing is on. A metrics-only setup therefore needs only `otel.WithMeterProvider(mp)`.

## Database Types

### PostgreSQL

```go
db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
    DBType: db.Postgresql,
    Host:   "localhost",
    Port:   5432,
    // ...
}))
```

**DSN Format:** `user=admin password=*** host=localhost port=5432 dbname=myapp sslmode=require connect_timeout=5`

### MySQL

```go
db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
    DBType: db.Mysql,
    Host:   "localhost",
    Port:   3306,
    // ...
}))
```

**DSN Format:** `admin:***@tcp(localhost:3306)/myapp?parseTime=true&timeout=5s`

> **Note:** `SSLMode` is ignored for MySQL — TLS is configured via DSN parameters, which this package does not expose.

### SQL Server (MSSQL)

```go
db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
    DBType: db.MSSQL,
    Host:   "localhost",
    Port:   1433,
    // ...
}))
```

**DSN Format:** `sqlserver://admin:***@localhost:1433?database=myapp&connectTimeout=5s&encrypt=require`

## Configuration

### ConnectionConfig

```go
type ConnectionConfig struct {
    DBType       DatabaseType  `yaml:"dbType" validate:"required,oneof=MYSQL POSTGRES MSSQL" mapstructure:"dbType"`
    Host         string        `yaml:"host" validate:"required,min=1" mapstructure:"host"`
    Port         int           `yaml:"port" validate:"required,min=1,max=65535" mapstructure:"port"`
    Username     string        `yaml:"username" validate:"required,min=1" mapstructure:"username"`
    Password     string        `yaml:"password" mapstructure:"password"`
    DBName       string        `yaml:"dbName" validate:"required,min=1" mapstructure:"dbName"`
    Timeout      time.Duration `yaml:"timeout" mapstructure:"timeout"`
    MaxIdleConns int           `yaml:"maxIdleConns" validate:"min=1" mapstructure:"maxIdleConns"`
    MaxOpenConns int           `yaml:"maxOpenConns" validate:"min=2" mapstructure:"maxOpenConns"`

    // Max connection reuse/idle durations (zero = unlimited)
    ConnMaxLifetime time.Duration `yaml:"connMaxLifetime" mapstructure:"connMaxLifetime"`
    ConnMaxIdleTime time.Duration `yaml:"connMaxIdleTime" mapstructure:"connMaxIdleTime"`

    // TLS mode (PostgreSQL/MSSQL only; ignored for MySQL)
    SSLMode string `yaml:"sslMode" mapstructure:"sslMode"`

    // GORM logger verbosity: 1=Silent, 2=Error, 3=Warn, 4=Info (default: 1)
    GormLogLevel int `yaml:"gormLogLevel" mapstructure:"gormLogLevel"`

    // Optional: Enable OpenTelemetry (nil = disabled)
    OTelConfig *otel.Config `yaml:"-" mapstructure:"-"`
}
```

> **TLS default:** `SSLMode` defaults to `"require"` for PostgreSQL and MSSQL. For local dev or test databases without TLS, set `SSLMode: "disable"` explicitly. MySQL ignores `SSLMode`.
>
> **Timeout default:** a zero `Timeout` falls back to 30 seconds.

### Functions and Methods

| Function/Method | Description |
|-----------------|-------------|
| `NewPool(opts ...Option)` | Creates a GORM pool from options; validates config, opens, configures pool, pings |
| `WithConnectionConfig(cfg)` | Option that seeds the pool configuration |
| `WithOTelConfig(cfg)` | Option that attaches OTel instrumentation (nil = no-op) |
| `Validate()` | Checks required fields and value ranges (called by `NewPool`) |
| `RedactedDsn()` | DSN with the password masked as `***`, safe for logging |
| `SQLDB()` | Opens a **new** pool and returns the raw `*sql.DB`; caller must close it |

## OpenTelemetry Integration

### Automatic Tracing

When `OTelConfig` is provided with tracing enabled, all database operations are automatically traced:

```go
pool, _ := db.NewPool(
    db.WithConnectionConfig(cfg),
    db.WithOTelConfig(otelConfig),
)

// Each operation creates a span
pool.Create(&user)           // Span: "db.INSERT"
pool.Find(&users)            // Span: "db.SELECT"
pool.Where("age > ?", 18).Find(&users)  // Span: "db.SELECT"
pool.Update("name", "John")  // Span: "db.UPDATE"
pool.Delete(&user)           // Span: "db.DELETE"
```

### Span Attributes

Each span includes:

```yaml
Span Attributes:
  db.system: "POSTGRES" | "MYSQL" | "MSSQL"
  db.name: "myapp"
  server.address: "localhost"
  server.port: 5432
```

### Metrics Collection

Connection pool metrics are collected whenever metrics are enabled (independent of tracing):

```yaml
Metrics:
  db.client.connections.idle:    # Number of idle connections
  db.client.connections.active:  # Number of active connections
  db.client.connections.max:     # Maximum connections allowed

Attributes:
  db.system: "POSTGRES"
  db.name: "myapp"
  server.address: "localhost"
  server.port: 5432
```

## Database Migrations

Only PostgreSQL is supported. The migration API works on a raw `*sql.DB`; GORM users obtain one via `gormDB.DB()` at the call site.

Both functions are instrumented through `otel.Layers.StartOperations`, producing a span named `db.RunPostgresMigrations` (or `db.RunPostgresMigrationsDown`) under the `operations.db` scope, with structured success/error logging.

### Using Embedded SQL Files

```go
import (
    "context"
    "embed"

    "github.com/jasoet/pkg/v3/db"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
    pool, err := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{/* ... */}))
    if err != nil {
        panic(err)
    }

    sqlDB, err := pool.DB()
    if err != nil {
        panic(err)
    }

    // Run migrations UP
    if err := db.RunPostgresMigrations(context.Background(), sqlDB, migrationsFS, "migrations"); err != nil {
        panic(err)
    }
}
```

### Migration File Structure

```
migrations/
├── 001_create_users.up.sql
├── 001_create_users.down.sql
├── 002_add_email_index.up.sql
└── 002_add_email_index.down.sql
```

**Example Migration:**
```sql
-- 001_create_users.up.sql
CREATE TABLE users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- 001_create_users.down.sql
DROP TABLE IF EXISTS users;
```

### Migration Functions

| Function | Description |
|----------|-------------|
| `RunPostgresMigrations(ctx, sqlDB, fs, path)` | Apply pending UP migrations |
| `RunPostgresMigrationsDown(ctx, sqlDB, fs, path)` | Roll back all migrations |

## Advanced Usage

### Raw SQL Access

```go
pool, _ := db.NewPool(db.WithConnectionConfig(cfg))

// Get the pool's raw *sql.DB (shared with GORM)
sqlDB, err := pool.DB()
if err != nil {
    panic(err)
}

// Or open a separate pool with SQLDB() — you own it, so close it
sqlDB2, err := cfg.SQLDB()
if err != nil {
    panic(err)
}
defer sqlDB2.Close()

// Use standard database/sql
rows, err := sqlDB.Query("SELECT * FROM users WHERE age > $1", 18)
```

### Connection Pooling

```go
pool, _ := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
    // Connection pool settings
    MaxIdleConns:    10,               // Max idle connections
    MaxOpenConns:    100,              // Max open connections
    ConnMaxLifetime: 30 * time.Minute, // Recycle aged connections
    ConnMaxIdleTime: 5 * time.Minute,  // Close long-idle connections
    Timeout:         30 * time.Second,
    // ...
}))

// Pool is automatically managed; connections are reused efficiently
```

### Transaction Support

```go
// GORM transactions
err := pool.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err
    }

    if err := tx.Create(&profile).Error; err != nil {
        return err
    }

    return nil
})

// Each query in transaction is traced separately
```

### Configuration from YAML

```go
import (
    "github.com/jasoet/pkg/v3/config"
    "github.com/jasoet/pkg/v3/db"
)

type AppConfig struct {
    Database db.ConnectionConfig `yaml:"database"`
}

yamlConfig := `
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: admin
  password: ${DB_PASSWORD}
  dbName: myapp
  timeout: 5s
  maxIdleConns: 5
  maxOpenConns: 10
`

cfg, _ := config.LoadString[AppConfig](yamlConfig)
pool, _ := db.NewPool(db.WithConnectionConfig(cfg.Database))
```

## Error Handling

```go
pool, err := db.NewPool(db.WithConnectionConfig(cfg))
if err != nil {
    switch {
    case strings.Contains(err.Error(), "invalid config"):
        // Invalid configuration (Validate failed)
    case strings.Contains(err.Error(), "connection refused"):
        // Database not reachable
    case strings.Contains(err.Error(), "authentication failed"):
        // Invalid credentials
    default:
        // Other errors
    }
}

// GORM errors
result := pool.Find(&users)
if result.Error != nil {
    if errors.Is(result.Error, gorm.ErrRecordNotFound) {
        // No records found
    }
}
```

## Best Practices

### 1. Use Environment Variables for Secrets

```go
import (
    "github.com/jasoet/pkg/v3/config"
    "github.com/jasoet/pkg/v3/db"
)

type AppConfig struct {
    Database db.ConnectionConfig `yaml:"database"`
}

yamlConfig := `
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  # username and password from env vars
  dbName: myapp
  timeout: 5s
  maxIdleConns: 5
  maxOpenConns: 10
`

// Set via environment:
// ENV_DATABASE_USERNAME=admin
// ENV_DATABASE_PASSWORD=secret123

cfg, _ := config.LoadString[AppConfig](yamlConfig)
pool, _ := db.NewPool(db.WithConnectionConfig(cfg.Database))
```

### 2. Connection Pool Sizing

```go
import "runtime"

db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
    // Rule of thumb: 2-3x number of CPU cores
    MaxOpenConns: runtime.NumCPU() * 3,
    // Keep some idle connections ready
    MaxIdleConns: runtime.NumCPU(),
    // ...
}))
```

### 3. Always Enable OTel in Production

```go
// ✅ Good: Observability enabled
pool, _ := db.NewPool(
    db.WithConnectionConfig(cfg),
    db.WithOTelConfig(otelConfig), // Tracing + Metrics
)

// ❌ Bad: No observability
pool, _ := db.NewPool(db.WithConnectionConfig(cfg)) // No tracing, no metrics
```

### 4. Use Context for Tracing

```go
// ✅ Good: Context propagates trace
ctx := context.Background()
ctx, span := tracer.Start(ctx, "user-service")
defer span.End()

pool.WithContext(ctx).Find(&users)  // Trace linked

// ❌ Bad: Trace not propagated
pool.Find(&users)  // New root span
```

### 5. Validate Configuration Early

```go
cfg := db.ConnectionConfig{
    DBType:       db.Postgresql,
    Host:         "localhost",
    Port:         5432,
    Username:     "admin",
    DBName:       "myapp",
    MaxIdleConns: 5,
    MaxOpenConns: 10,
}

// NewPool calls Validate() internally; calling it yourself surfaces
// config errors at startup before any dial attempt.
if err := cfg.Validate(); err != nil {
    panic(fmt.Sprintf("invalid config: %v", err))
}

pool, _ := db.NewPool(db.WithConnectionConfig(cfg))
```

## Testing

```bash
# Unit tests
go test ./db -v

# Integration tests (requires Docker)
go test ./db -tags=integration -v

# With coverage
go test ./db -tags=integration -cover
```

### Test Utilities

```go
import (
    "github.com/jasoet/pkg/v3/db"
    "github.com/jasoet/pkg/v3/otel"
    noopt "go.opentelemetry.io/otel/trace/noop"
    noopm "go.opentelemetry.io/otel/metric/noop"
)

func TestWithTestcontainer(t *testing.T) {
    // Use testcontainers for integration tests
    ctx := context.Background()
    container, _ := setupPostgresContainer(ctx)
    defer container.Terminate(ctx)

    pool, err := db.NewPool(
        db.WithConnectionConfig(db.ConnectionConfig{
            DBType:   db.Postgresql,
            Host:     container.Host(ctx),
            Port:     container.MappedPort(ctx, "5432").Int(),
            Username: "test",
            Password: "test",
            DBName:   "testdb",
        }),
        db.WithOTelConfig(otel.NewConfig("test",
            otel.WithTracerProvider(noopt.NewTracerProvider()),
            otel.WithMeterProvider(noopm.NewMeterProvider()))),
    )
    assert.NoError(t, err)

    // Test your code
}
```

## Troubleshooting

### Connection Refused

**Problem**: `connection refused` error

**Solutions:**
```go
// 1. Check database is running
// docker ps | grep postgres

// 2. Verify host and port
cfg := db.ConnectionConfig{
    Host: "localhost",  // or "127.0.0.1"
    Port: 5432,         // default PostgreSQL port
    // ...
}

// 3. Check timeout
cfg.Timeout = 30 * time.Second  // Increase timeout
```

### TLS Required by Default

**Problem**: connection fails with an SSL/TLS error against a local dev database

**Solution:** `SSLMode` defaults to `"require"`; set `SSLMode: "disable"` (PostgreSQL) or `SSLMode: "disable"`/`"false"` (MSSQL) for servers without TLS.

### Authentication Failed

**Problem**: `authentication failed` error

**Solutions:**
```go
// 1. Verify credentials
cfg := db.ConnectionConfig{
    Username: "correct_username",
    Password: "correct_password",
    // ...
}

// 2. Check database exists
// psql -U admin -l

// 3. Verify user permissions
// GRANT ALL PRIVILEGES ON DATABASE myapp TO admin;
```

### Too Many Connections

**Problem**: `sorry, too many clients already` error

**Solutions:**
```go
// 1. Reduce max connections
cfg := db.ConnectionConfig{
    MaxOpenConns: 20,  // Lower value
    MaxIdleConns: 5,
    // ...
}

// 2. Check pool metrics (if OTel metrics enabled)
// Look at db.client.connections.active metric

// 3. Increase database max_connections
// ALTER SYSTEM SET max_connections = 200;
```

### Migrations Not Running

**Problem**: Migrations not applying

**Solutions:**
```go
// 1. Check migration files exist
//go:embed migrations/*.sql
var migrationsFS embed.FS

// 2. Verify path
sqlDB, _ := pool.DB()
err := db.RunPostgresMigrations(
    ctx,
    sqlDB,
    migrationsFS,
    "migrations",  // Correct path
)

// 3. Check migration version table
// SELECT * FROM schema_migrations;
```

## Performance

- **Connection Pooling**: Efficiently reuses connections
- **Prepared Statements**: GORM uses prepared statements by default
- **Query Optimization**: Use indexes and EXPLAIN ANALYZE
- **Batch Operations**: Use GORM's batch features for bulk inserts

## Version Compatibility

- **GORM**: v1.31.0+
- **golang-migrate**: v4.19.0+
- **PostgreSQL**: 12+
- **MySQL**: 8.0+
- **SQL Server**: 2019+
- **Go**: 1.25+
- **pkg library**: v3.0.0+

## Examples

See the [examples/db/](../examples/db/) directory for:
- Basic database connection
- Multi-database setup
- OpenTelemetry integration
- Migration management
- Transaction handling
- Connection pooling
- Error handling

## Related Packages

- **[otel](../otel/)** - OpenTelemetry configuration
- **[config](../config/)** - Configuration management

## License

MIT License - see [LICENSE](../LICENSE) for details.
