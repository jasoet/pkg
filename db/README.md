# Database Package

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/db.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/db)

Multi-database support with GORM, automated migrations, and OpenTelemetry instrumentation.

## Overview

The `db` package provides a unified interface for connecting to multiple database systems with automatic OpenTelemetry tracing and metrics collection. Built on GORM and golang-migrate, it simplifies database operations while providing production-ready observability.

## Features

- **Multi-Database Support**: PostgreSQL, MySQL, MSSQL
- **GORM Integration**: Full ORM capabilities with GORM v2
- **Automatic Tracing**: Query-level distributed tracing
- **Connection Pool Metrics**: Real-time pool health monitoring
- **Schema Migrations**: Embedded migrations with golang-migrate
- **Type-Safe Configuration**: Validation with struct tags
- **Zero Configuration OTel**: Optional but seamless observability

## Installation

```bash
go get github.com/jasoet/pkg/v2/db
```

## Quick Start

### Basic Connection

```go
package main

import (
    "github.com/jasoet/pkg/v2/db"
    "time"
)

func main() {
    config := db.ConnectionConfig{
        DbType:       db.Postgresql,
        Host:         "localhost",
        Port:         5432,
        Username:     "admin",
        Password:     "secret",
        DbName:       "myapp",
        Timeout:      5 * time.Second,
        MaxIdleConns: 5,
        MaxOpenConns: 10,
    }

    pool, err := config.Pool()
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
    "github.com/jasoet/pkg/v2/db"
    "github.com/jasoet/pkg/v2/otel"
)

// Setup OTel
otelConfig := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

// Configure database with OTel
config := db.ConnectionConfig{
    DbType:       db.Postgresql,
    Host:         "localhost",
    Port:         5432,
    Username:     "admin",
    Password:     "secret",
    DbName:       "myapp",
    Timeout:      5 * time.Second,
    MaxIdleConns: 5,
    MaxOpenConns: 10,
    OTelConfig:   otelConfig,  // Enable tracing & metrics
}

pool, _ := config.Pool()

// All queries are automatically traced
pool.Find(&users)  // Creates span "db.SELECT"
pool.Create(&user) // Creates span "db.INSERT"
```

## Database Types

### PostgreSQL

```go
config := db.ConnectionConfig{
    DbType: db.Postgresql,
    Host:   "localhost",
    Port:   5432,
    // ...
}
```

**DSN Format:** `user=admin password=secret host=localhost port=5432 dbname=myapp sslmode=disable connect_timeout=5`

### MySQL

```go
config := db.ConnectionConfig{
    DbType: db.Mysql,
    Host:   "localhost",
    Port:   3306,
    // ...
}
```

**DSN Format:** `admin:secret@tcp(localhost:3306)/myapp?parseTime=true&timeout=5s`

### SQL Server (MSSQL)

```go
config := db.ConnectionConfig{
    DbType: db.MSSQL,
    Host:   "localhost",
    Port:   1433,
    // ...
}
```

**DSN Format:** `sqlserver://admin:secret@localhost:1433?database=myapp&connectTimeout=5s&encrypt=disable`

## Configuration

### ConnectionConfig

```go
type ConnectionConfig struct {
    DbType       DatabaseType  `yaml:"dbType" validate:"required,oneof=MYSQL POSTGRES MSSQL"`
    Host         string        `yaml:"host" validate:"required,min=1"`
    Port         int           `yaml:"port"`
    Username     string        `yaml:"username" validate:"required,min=1"`
    Password     string        `yaml:"password"`
    DbName       string        `yaml:"dbName" validate:"required,min=1"`
    Timeout      time.Duration `yaml:"timeout" validate:"min=3s"`
    MaxIdleConns int           `yaml:"maxIdleConns" validate:"min=1"`
    MaxOpenConns int           `yaml:"maxOpenConns" validate:"min=2"`

    // Optional: Enable OpenTelemetry (nil = disabled)
    OTelConfig   *otel.Config  `yaml:"-"`
}
```

### Methods

| Method | Description |
|--------|-------------|
| `Pool()` | Returns GORM DB instance with connection pooling |
| `SQLDB()` | Returns raw `*sql.DB` for direct SQL access |
| `Dsn()` | Generates database connection string |

## OpenTelemetry Integration

### Automatic Tracing

When `OTelConfig` is provided, all database operations are automatically traced:

```go
config := db.ConnectionConfig{
    // ... database config
    OTelConfig: otelConfig,
}

pool, _ := config.Pool()

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
  db.statement: "SELECT * FROM users WHERE age > 18"
  db.collection.name: "users"
  db.rows_affected: 42
  db.duration_ms: 15
  server.address: "localhost"
  server.port: 5432
```

### Metrics Collection

Connection pool metrics are automatically collected:

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

### Using Embedded SQL Files

```go
import (
    "context"
    "embed"
    "github.com/jasoet/pkg/v2/db"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func main() {
    config := db.ConnectionConfig{/* ... */}
    pool, _ := config.Pool()

    ctx := context.Background()

    // Run migrations UP
    err := db.RunPostgresMigrationsWithGorm(
        ctx,
        pool,
        migrationsFS,
        "migrations",
    )
    if err != nil {
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
| `RunPostgresMigrationsWithGorm(ctx, gormDB, fs, path)` | Run migrations UP with GORM |
| `RunPostgresMigrationsDownWithGorm(ctx, gormDB, fs, path)` | Roll back migrations with GORM |
| `RunPostgresMigrations(ctx, sqlDB, fs, path)` | Run migrations UP with raw SQL DB |
| `RunPostgresMigrationsDown(ctx, sqlDB, fs, path)` | Roll back migrations with raw SQL DB |

## Advanced Usage

### Raw SQL Access

```go
pool, _ := config.Pool()

// Get raw *sql.DB
sqlDB, err := pool.DB()
if err != nil {
    panic(err)
}

// Or use SQLDB() directly
sqlDB, err := config.SQLDB()

// Use standard database/sql
rows, err := sqlDB.Query("SELECT * FROM users WHERE age > ?", 18)
```

### Connection Pooling

```go
config := db.ConnectionConfig{
    // Connection pool settings
    MaxIdleConns: 10,   // Max idle connections
    MaxOpenConns: 100,  // Max open connections
    Timeout:      30 * time.Second,
    // ...
}

pool, _ := config.Pool()

// Pool is automatically managed
// Connections are reused efficiently
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
    "github.com/jasoet/pkg/v2/config"
    "github.com/jasoet/pkg/v2/db"
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
  password: secret
  dbName: myapp
  timeout: 5s
  maxIdleConns: 5
  maxOpenConns: 10
`

cfg, _ := config.LoadString[AppConfig](yamlConfig)
pool, _ := cfg.Database.Pool()
```

## Error Handling

```go
pool, err := config.Pool()
if err != nil {
    switch {
    case strings.Contains(err.Error(), "dsn is empty"):
        // Invalid configuration
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
    "github.com/jasoet/pkg/v2/config"
    "github.com/jasoet/pkg/v2/db"
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
pool, _ := cfg.Database.Pool()
```

### 2. Connection Pool Sizing

```go
import "runtime"

config := db.ConnectionConfig{
    // Rule of thumb: 2-3x number of CPU cores
    MaxOpenConns: runtime.NumCPU() * 3,
    // Keep some idle connections ready
    MaxIdleConns: runtime.NumCPU(),
    // ...
}
```

### 3. Always Enable OTel in Production

```go
// ✅ Good: Observability enabled
config := db.ConnectionConfig{
    // ... database config
    OTelConfig: otelConfig,  // Tracing + Metrics
}

// ❌ Bad: No observability
config := db.ConnectionConfig{
    // ... database config
    OTelConfig: nil,  // No tracing, no metrics
}
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

### 5. Validate Configuration

```go
import "github.com/go-playground/validator/v10"

config := db.ConnectionConfig{
    DbType:       db.Postgresql,
    Host:         "localhost",
    Port:         5432,
    Username:     "admin",
    DbName:       "myapp",
    Timeout:      5 * time.Second,
    MaxIdleConns: 5,
    MaxOpenConns: 10,
}

validate := validator.New()
if err := validate.Struct(config); err != nil {
    panic(fmt.Sprintf("invalid config: %v", err))
}

pool, _ := config.Pool()
```

## Testing

The package includes comprehensive tests with 79.1% coverage:

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
    "github.com/jasoet/pkg/v2/db"
    "github.com/jasoet/pkg/v2/otel"
    noopt "go.opentelemetry.io/otel/trace/noop"
    noopm "go.opentelemetry.io/otel/metric/noop"
)

func TestWithTestcontainer(t *testing.T) {
    // Use testcontainers for integration tests
    ctx := context.Background()
    container, _ := setupPostgresContainer(ctx)
    defer container.Terminate(ctx)

    config := db.ConnectionConfig{
        DbType:   db.Postgresql,
        Host:     container.Host(ctx),
        Port:     container.MappedPort(ctx, "5432").Int(),
        Username: "test",
        Password: "test",
        DbName:   "testdb",
        OTelConfig: otel.NewConfig("test").
            WithTracerProvider(noopt.NewTracerProvider()).
            WithMeterProvider(noopm.NewMeterProvider()),
    }

    pool, err := config.Pool()
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
config := db.ConnectionConfig{
    Host: "localhost",  // or "127.0.0.1"
    Port: 5432,         // default PostgreSQL port
    // ...
}

// 3. Check timeout
config.Timeout = 30 * time.Second  // Increase timeout
```

### Authentication Failed

**Problem**: `authentication failed` error

**Solutions:**
```go
// 1. Verify credentials
config := db.ConnectionConfig{
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
config := db.ConnectionConfig{
    MaxOpenConns: 20,  // Lower value
    MaxIdleConns: 5,
    // ...
}

// 2. Check pool metrics (if OTel enabled)
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
err := db.RunPostgresMigrationsWithGorm(
    ctx,
    pool,
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

**Benchmark (typical operations):**
```
BenchmarkQuery-8         10000    ~500 µs/op
BenchmarkInsert-8         5000    ~800 µs/op
BenchmarkUpdate-8         8000    ~600 µs/op
```

## Version Compatibility

- **GORM**: v1.31.0+
- **golang-migrate**: v4.19.0+
- **PostgreSQL**: 12+
- **MySQL**: 8.0+
- **SQL Server**: 2019+
- **Go**: 1.25+
- **pkg library**: v2.0.0+

## Examples

See [examples/](./examples/) directory for:
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
- **[logging](../logging/)** - Structured logging

## License

MIT License - see [LICENSE](../LICENSE) for details.
