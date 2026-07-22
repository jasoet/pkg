# DB Package Examples

This directory contains examples demonstrating how to use the `db` package for database connection management and migrations in Go applications.

## 📍 Example Code Location

**Full example implementation:** [example.go](./example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/v3/db"

// Create database connection
config := db.ConnectionConfig{
    DBType:       db.Postgresql, // or db.Mysql, db.MSSQL
    Host:         "localhost",
    Port:         5432,
    Username:     "user",
    Password:     "pass",
    DBName:       "mydb",
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}

// Get GORM database instance
database, err := db.NewPool(db.WithConnectionConfig(config))

// Run migrations (PostgreSQL only; pass the pool's raw *sql.DB)
sqlDB, _ := database.DB()
err = db.RunPostgresMigrations(ctx, sqlDB, migrationFS, "migrations")

// Check connection
err = database.Exec("SELECT 1").Error
```

**Critical notes:**
- Initialize observability with `otel.Initialize("my-app", true)` before database operations
- Connection strings are built automatically based on DBType; use `config.RedactedDsn()` for safe logging
- Migrations use golang-migrate library format and only support PostgreSQL

## Overview

The `db` package provides utilities for:
- Multi-database support (PostgreSQL, MySQL, SQL Server)
- Connection pooling configuration with GORM
- Database migrations with golang-migrate
- OpenTelemetry tracing, metrics, and structured logging
- Connection validation and health checks

## Running the Examples

The example program is gated behind the `example` build tag. Run it from the repository root:

```bash
go run -tags=example ./examples/db
```

**Note**: The examples require a working database server. Override the defaults with environment variables if needed: `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`.

## Database Setup

For testing, use the Docker Compose configuration in the repository:

```bash
# From the repository root
docker compose -f scripts/compose/docker-compose.yml up -d
```

This starts PostgreSQL on `localhost:5439` with:
- Username: `jasoet`
- Password: `localhost`
- Database: `pkg_db`

(MySQL on `3309` and MSSQL on `1439` are also included.)

## Example Descriptions

The [example.go](./example.go) file demonstrates several use cases:

### 1. Basic Database Connection

Connect to different database types with proper configuration:

```go
// PostgreSQL connection
config := db.ConnectionConfig{
    DBType:       db.Postgresql,
    Host:         "localhost",
    Port:         5439,
    Username:     "jasoet",
    Password:     "localhost",
    DBName:       "pkg_db",
    Timeout:      10 * time.Second,
    MaxIdleConns: 5,
    MaxOpenConns: 25,
}

database, err := db.NewPool(db.WithConnectionConfig(config))
if err != nil {
    log.Fatal("Failed to connect:", err)
}
```

### 2. Connection Pool Configuration

Configure connection pools for optimal performance:

```go
config := db.ConnectionConfig{
    DBType:       db.Postgresql,
    Host:         "localhost",
    Port:         5432,
    Username:     "user",
    Password:     "password",
    DBName:       "myapp",
    Timeout:      30 * time.Second,
    MaxIdleConns: 10,  // Number of idle connections
    MaxOpenConns: 100, // Maximum open connections
}
```

### 3. Database Migrations

Run database migrations using embedded SQL files:

```go
//go:embed migrations/*.sql
var migrationFS embed.FS

sqlDB, err := database.DB()
if err != nil {
    log.Fatal("Failed to get SQL DB:", err)
}

// Run migrations up
err = db.RunPostgresMigrations(ctx, sqlDB, migrationFS, "migrations")
if err != nil {
    log.Fatal("Migration failed:", err)
}

// Run migrations down (rollback)
err = db.RunPostgresMigrationsDown(ctx, sqlDB, migrationFS, "migrations")
```

### 4. Multiple Database Connections

Manage connections to multiple databases:

```go
// Primary database
primaryDB, err := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
    DBType: db.Postgresql,
    Host: "primary.db.com", Port: 5432,
    Username: "app", Password: "secret",
    DBName: "primary_db",
    MaxIdleConns: 5, MaxOpenConns: 25,
}))

// Analytics database
analyticsDB, err := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
    DBType: db.Mysql,
    Host: "analytics.db.com", Port: 3306,
    Username: "analytics", Password: "secret",
    DBName: "analytics_db",
    MaxIdleConns: 3, MaxOpenConns: 15,
}))
```

### 5. GORM Model Operations

Perform CRUD operations with GORM models:

```go
type User struct {
    ID    uint   `gorm:"primaryKey"`
    Name  string `gorm:"not null"`
    Email string `gorm:"uniqueIndex"`
}

// Auto-migrate the schema
database.AutoMigrate(&User{})

// Create
user := User{Name: "John Doe", Email: "john@example.com"}
result := database.Create(&user)

// Read
var users []User
database.Find(&users)

// Update
database.Model(&user).Update("Email", "newemail@example.com")

// Delete
database.Delete(&user)
```

### 6. Raw SQL with Connection Pool

Execute raw SQL queries using a dedicated connection pool:

```go
// SQLDB() opens a new pool; the caller must close it.
sqlDB, err := config.SQLDB()
if err != nil {
    log.Fatal("Failed to get SQL DB:", err)
}
defer sqlDB.Close()

rows, err := sqlDB.Query("SELECT id, name FROM users WHERE active = $1", true)
if err != nil {
    log.Fatal("Query failed:", err)
}
defer rows.Close()

for rows.Next() {
    var id int
    var name string
    err := rows.Scan(&id, &name)
    if err != nil {
        log.Fatal("Scan failed:", err)
    }
    fmt.Printf("User: %d - %s\n", id, name)
}
```

### 7. Transaction Management

Handle database transactions properly:

```go
tx := database.Begin()
defer func() {
    if r := recover(); r != nil {
        tx.Rollback()
    }
}()

if err := tx.Error; err != nil {
    return err
}

// Perform multiple operations
if err := tx.Create(&user1).Error; err != nil {
    tx.Rollback()
    return err
}

if err := tx.Create(&user2).Error; err != nil {
    tx.Rollback()
    return err
}

// Commit the transaction
tx.Commit()
```

### 8. Health Checks and Monitoring

Implement database health checks:

```go
func checkDatabaseHealth(db *gorm.DB) error {
    sqlDB, err := db.DB()
    if err != nil {
        return fmt.Errorf("failed to get SQL DB: %w", err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    if err := sqlDB.PingContext(ctx); err != nil {
        return fmt.Errorf("database ping failed: %w", err)
    }

    return nil
}
```

## Configuration Options

The `ConnectionConfig` struct supports the following options:

| Field | Type | Description | Required |
|-------|------|-------------|----------|
| `DBType` | DatabaseType | Database type (MYSQL, POSTGRES, MSSQL) | Yes |
| `Host` | string | Database server hostname or IP | Yes |
| `Port` | int | Database server port | Yes |
| `Username` | string | Database username | Yes |
| `Password` | string | Database password | No |
| `DBName` | string | Database name | Yes |
| `Timeout` | time.Duration | Connection timeout (default: 30s) | No |
| `MaxIdleConns` | int | Maximum idle connections (min: 1) | No |
| `MaxOpenConns` | int | Maximum open connections (min: 2) | No |
| `ConnMaxLifetime` | time.Duration | Max connection reuse time (0 = unlimited) | No |
| `ConnMaxIdleTime` | time.Duration | Max connection idle time (0 = unlimited) | No |
| `SSLMode` | string | TLS mode for PostgreSQL/MSSQL (default: "require"; ignored for MySQL) | No |
| `GormLogLevel` | int | GORM logger verbosity (1=Silent … 4=Info; default: 1) | No |

### Database Type Constants

```go
const (
    Mysql      DatabaseType = "MYSQL"
    Postgresql DatabaseType = "POSTGRES"
    MSSQL      DatabaseType = "MSSQL"
)
```

## Migration File Structure

Migration files should follow the golang-migrate naming convention:

```
migrations/
├── 001_initial_schema.up.sql
├── 001_initial_schema.down.sql
├── 002_add_users_table.up.sql
├── 002_add_users_table.down.sql
└── 003_add_indexes.up.sql
└── 003_add_indexes.down.sql
```

### Example Migration Files

**001_initial_schema.up.sql**:
```sql
CREATE TABLE IF NOT EXISTS users (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

**001_initial_schema.down.sql**:
```sql
DROP TABLE IF EXISTS users;
```

## Database-Specific Configurations

### PostgreSQL

```go
config := db.ConnectionConfig{
    DBType:       db.Postgresql,
    Host:         "localhost",
    Port:         5432,
    Username:     "postgres",
    Password:     "password",
    DBName:       "myapp",
    Timeout:      30 * time.Second,
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}
```

**Connection String Format**: `user=username password=password host=host port=5432 dbname=database sslmode=require connect_timeout=30`

### MySQL

```go
config := db.ConnectionConfig{
    DBType:       db.Mysql,
    Host:         "localhost",
    Port:         3306,
    Username:     "root",
    Password:     "password",
    DBName:       "myapp",
    Timeout:      30 * time.Second,
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}
```

**Connection String Format**: `username:password@tcp(host:3306)/database?parseTime=true&timeout=30s`

Note: `SSLMode` is ignored for MySQL.

### SQL Server

```go
config := db.ConnectionConfig{
    DBType:       db.MSSQL,
    Host:         "localhost",
    Port:         1433,
    Username:     "sa",
    Password:     "password",
    DBName:       "myapp",
    Timeout:      30 * time.Second,
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}
```

**Connection String Format**: `sqlserver://username:password@host:1433?database=myapp&connectTimeout=30s&encrypt=require`

## Integration with OTel Logging

The db package emits structured logs through the `otel` package's zerolog-based logger:

```go
ctx := context.Background()
logger := otel.ContextLogger(ctx, "database")

// Migration logging is automatic (via otel.Layers spans)
sqlDB, _ := database.DB()
err := db.RunPostgresMigrations(ctx, sqlDB, migrationFS, "migrations")

// Custom database logging
logger.Info().Msg("Database operation started")
result := database.Create(&user)
if result.Error != nil {
    logger.Error().Err(result.Error).Msg("Failed to create user")
} else {
    logger.Info().Int64("rows_affected", result.RowsAffected).Msg("User created successfully")
}
```

## Best Practices

### 1. Connection Pool Sizing

```go
// For web applications
config.MaxIdleConns = 10     // Keep some connections warm
config.MaxOpenConns = 100    // Limit total connections

// For background workers
config.MaxIdleConns = 5      // Fewer idle connections
config.MaxOpenConns = 50     // Lower concurrency
```

### 2. Context Usage

```go
// Always use context for database operations
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

// GORM with context
database.WithContext(ctx).Find(&users)

// Raw SQL with context
sqlDB.QueryContext(ctx, "SELECT * FROM users")
```

### 3. Error Handling

```go
// Check for specific GORM errors
if errors.Is(result.Error, gorm.ErrRecordNotFound) {
    // Handle not found
}

// Check for database connection errors
if err := database.Error; err != nil {
    logger.Error().Err(err).Msg("Database error")
}
```

### 4. Migration Management

```go
// Always use embedded migrations for deployment
//go:embed migrations/*.sql
var migrationFS embed.FS

// Run migrations in a separate function
func runMigrations(ctx context.Context, database *gorm.DB) error {
    sqlDB, err := database.DB()
    if err != nil {
        return err
    }
    return db.RunPostgresMigrations(ctx, sqlDB, migrationFS, "migrations")
}
```

### 5. Testing

```go
// Use test databases for testing
func setupTestDB() *gorm.DB {
    database, err := db.NewPool(db.WithConnectionConfig(db.ConnectionConfig{
        DBType:   db.Postgresql,
        Host:     "localhost",
        Port:     5432,
        Username: "test",
        Password: "test",
        DBName:   "test_db",
        MaxIdleConns: 2,
        MaxOpenConns: 10,
    }))
    if err != nil {
        panic(err)
    }

    return database
}
```

## Performance Considerations

### Connection Pool Tuning

- **MaxOpenConns**: Should not exceed database's max connections
- **MaxIdleConns**: Balance between resource usage and connection overhead
- **Connection lifetime**: Set `ConnMaxLifetime`/`ConnMaxIdleTime` for long-running applications

### Query Optimization

- Use prepared statements for repeated queries
- Implement proper indexing in migrations
- Use GORM's `Select()` to limit fields
- Consider pagination for large result sets

## Troubleshooting

### Common Issues

1. **Connection Refused**: Database server not running or wrong host/port
2. **Authentication Failed**: Invalid username/password
3. **Database Not Found**: Database doesn't exist or wrong name
4. **TLS errors against local databases**: `SSLMode` defaults to `"require"` — set `SSLMode: "disable"` for dev databases without TLS
5. **Connection Pool Exhausted**: Too many concurrent connections
6. **Migration Conflicts**: Conflicting migration files or database state

### Debug Tips

- Enable GORM logging: set `GormLogLevel: 4` (Info) on the `ConnectionConfig`
- Check database logs for detailed error messages
- Verify network connectivity and firewall rules
- Test connection with database client tools first
- Monitor connection pool metrics in production
