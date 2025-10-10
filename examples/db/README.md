# DB Package Examples

This directory contains examples demonstrating how to use the `db` package for database connection management and migrations in Go applications.

## 📍 Example Code Location

**Full example implementation:** [/db/examples/example.go](https://github.com/jasoet/pkg/blob/main/db/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import "github.com/jasoet/pkg/db"

// Create database connection
config := &db.ConnectionConfig{
    DbType:       db.Postgresql, // or db.Mysql, db.SqlServer
    Host:         "localhost",
    Port:         5432,
    Username:     "user",
    Password:     "pass",
    DbName:       "mydb",
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}

// Get GORM database instance
database, err := config.Pool()

// Run migrations
err = db.Migrate(database, "file://migrations")

// Check connection
err = database.Exec("SELECT 1").Error
```

**Critical notes:**
- Always use logging.Initialize() before database operations
- Connection strings are built automatically based on DbType
- Migrations use golang-migrate library format

## Overview

The `db` package provides utilities for:
- Multi-database support (PostgreSQL, MySQL, SQL Server)
- Connection pooling configuration with GORM
- Database migrations with golang-migrate
- Context-aware logging integration
- Connection validation and health checks

## Running the Examples

To run the examples, use the following command from the `db/examples` directory:

```bash
go run example.go
```

**Note**: The examples require a working database server. Update the configuration in the examples to match your environment, or use the provided Docker Compose setup.

## Database Setup

For testing, you can use the Docker Compose configuration in the repository:

```bash
# From the root directory
task docker:up
```

This starts PostgreSQL on `localhost:5439` with:
- Username: `jasoet`
- Password: `localhost` 
- Database: `pkg_db`

## Example Descriptions

The [example.go](https://github.com/jasoet/pkg/blob/main/db/examples/example.go) file demonstrates several use cases:

### 1. Basic Database Connection

Connect to different database types with proper configuration:

```go
// PostgreSQL connection
config := &db.ConnectionConfig{
    DbType:       db.Postgresql,
    Host:         "localhost",
    Port:         5439,
    Username:     "jasoet",
    Password:     "localhost",
    DbName:       "pkg_db",
    Timeout:      10 * time.Second,
    MaxIdleConns: 5,
    MaxOpenConns: 25,
}

database, err := config.Pool()
if err != nil {
    log.Fatal("Failed to connect:", err)
}
```

### 2. Connection Pool Configuration

Configure connection pools for optimal performance:

```go
config := &db.ConnectionConfig{
    DbType:       db.Postgresql,
    Host:         "localhost",
    Port:         5432,
    Username:     "user",
    Password:     "password",
    DbName:       "myapp",
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

// Run migrations up
err := db.RunPostgresMigrationsWithGorm(ctx, database, migrationFS, "migrations")
if err != nil {
    log.Fatal("Migration failed:", err)
}

// Run migrations down (rollback)
err = db.RunPostgresMigrationsDownWithGorm(ctx, database, migrationFS, "migrations")
```

### 4. Multiple Database Connections

Manage connections to multiple databases:

```go
// Primary database
primaryDB, err := (&db.ConnectionConfig{
    DbType: db.Postgresql,
    Host: "primary.db.com", Port: 5432,
    Username: "app", Password: "secret",
    DbName: "primary_db",
    MaxIdleConns: 5, MaxOpenConns: 25,
}).Pool()

// Analytics database
analyticsDB, err := (&db.ConnectionConfig{
    DbType: db.Mysql,
    Host: "analytics.db.com", Port: 3306,
    Username: "analytics", Password: "secret",
    DbName: "analytics_db",
    MaxIdleConns: 3, MaxOpenConns: 15,
}).Pool()
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

Execute raw SQL queries using the connection pool:

```go
sqlDB, err := config.SQLDB()
if err != nil {
    log.Fatal("Failed to get SQL DB:", err)
}

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
| `DbType` | DatabaseType | Database type (MYSQL, POSTGRES, MSSQL) | Yes |
| `Host` | string | Database server hostname or IP | Yes |
| `Port` | int | Database server port | Yes |
| `Username` | string | Database username | Yes |
| `Password` | string | Database password | No |
| `DbName` | string | Database name | Yes |
| `Timeout` | time.Duration | Connection timeout (min: 3s) | No |
| `MaxIdleConns` | int | Maximum idle connections (min: 1) | No |
| `MaxOpenConns` | int | Maximum open connections (min: 2) | No |

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
config := &db.ConnectionConfig{
    DbType:       db.Postgresql,
    Host:         "localhost",
    Port:         5432,
    Username:     "postgres",
    Password:     "password",
    DbName:       "myapp",
    Timeout:      30 * time.Second,
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}
```

**Connection String Format**: `user=username password=password host=host port=5432 dbname=database sslmode=disable connect_timeout=30`

### MySQL

```go
config := &db.ConnectionConfig{
    DbType:       db.Mysql,
    Host:         "localhost",
    Port:         3306,
    Username:     "root",
    Password:     "password",
    DbName:       "myapp",
    Timeout:      30 * time.Second,
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}
```

**Connection String Format**: `username:password@tcp(host:3306)/database?parseTime=true&timeout=30s`

### SQL Server

```go
config := &db.ConnectionConfig{
    DbType:       db.MSSQL,
    Host:         "localhost",
    Port:         1433,
    Username:     "sa",
    Password:     "password",
    DbName:       "myapp",
    Timeout:      30 * time.Second,
    MaxIdleConns: 10,
    MaxOpenConns: 100,
}
```

**Connection String Format**: `sqlserver://username:password@host:1433?database=myapp&connectTimeout=30s&encrypt=disable`

## Integration with Logging

The db package integrates with the logging package for structured logging:

```go
ctx := context.Background()
logger := logging.ContextLogger(ctx, "database")

// Migration logging is automatic
err := db.RunPostgresMigrationsWithGorm(ctx, database, migrationFS, "migrations")

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
func runMigrations(ctx context.Context, db *gorm.DB) error {
    return db.RunPostgresMigrationsWithGorm(ctx, db, migrationFS, "migrations")
}
```

### 5. Testing

```go
// Use test databases for testing
func setupTestDB() *gorm.DB {
    config := &db.ConnectionConfig{
        DbType:   db.Postgresql,
        Host:     "localhost",
        Port:     5432,
        Username: "test",
        Password: "test",
        DbName:   "test_db",
        MaxIdleConns: 2,
        MaxOpenConns: 10,
    }
    
    database, err := config.Pool()
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
- **Connection lifetime**: Consider setting `SetConnMaxLifetime()` for long-running applications

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
4. **Connection Pool Exhausted**: Too many concurrent connections
5. **Migration Conflicts**: Conflicting migration files or database state

### Debug Tips

- Enable GORM logging: `db.Config{Logger: logger.Default.LogMode(logger.Info)}`
- Check database logs for detailed error messages
- Verify network connectivity and firewall rules
- Test connection with database client tools first
- Monitor connection pool metrics in production