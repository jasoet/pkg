# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

This project uses [Mage](https://magefile.org/) for build automation. Common commands:

### Basic Commands
```bash
# Run unit tests
mage test

# Run integration tests (starts Docker services automatically)
mage integrationTest

# Run linter (installs golangci-lint if not present)
mage lint

# Clean build artifacts
mage clean
```

### Development Tools & Quality Checks
```bash
# Install all development tools (golangci-lint, gosec, nancy, etc.)
mage tools

# Run security analysis with gosec
mage security

# Check dependencies for known vulnerabilities
mage dependencies

# Generate test coverage report (creates coverage.html)
mage coverage

# Generate API documentation (if swagger annotations exist)
mage docs

# Run all quality checks (test, lint, security, dependencies, coverage)
mage checkall
```

### Docker Service Management
```bash
mage docker:up        # Start PostgreSQL and other services
mage docker:down      # Stop services and remove volumes
mage docker:logs      # View service logs
mage docker:restart   # Restart all services
```

## Development Environment

- **PostgreSQL**: localhost:5439 (user: jasoet, password: localhost, database: pkg_db)
- **Docker Compose**: Services defined in `scripts/compose/docker-compose.yml`
- **Integration Tests**: Use `AUTOMATION=true` environment variable and `-tags=integration`

## Architecture Overview

This is a Go utility library providing reusable infrastructure components. The packages are designed to work together while remaining modular:

### Core Packages

- **config**: YAML configuration loading with environment variable overrides using Viper
- **logging**: Structured logging with zerolog, provides centralized setup and context-aware loggers
- **concurrent**: Type-safe concurrent execution utilities using Go generics
- **db**: Multi-database support (MySQL, PostgreSQL, MSSQL) with GORM and migrations
- **rest**: HTTP client framework with middleware support built on Resty
- **server**: Echo-based HTTP server with health checks, metrics, and graceful shutdown
- **temporal**: Temporal workflow engine integration with workers and scheduling
- **ssh**: SSH tunneling utilities for secure remote connections
- **compress**: File compression and archive utilities with security validations

### Key Patterns

- **Configuration**: YAML-first with environment variable overrides, validation via struct tags
- **Logging**: All packages integrate with the central logging package for consistency
- **Generics**: Extensive use of Go generics for type safety (config loading, concurrent execution)
- **Lifecycle Management**: Consistent Start/Stop patterns with context-based cancellation
- **Error Handling**: Custom error types with context information and error wrapping

### Dependencies

- **logging** package is used by db, temporal, and server packages
- External integrations: GORM (databases), Temporal (workflows), Prometheus (metrics)
- Docker Compose provides PostgreSQL for integration testing

## Testing

- Unit tests: `go test ./...` or `mage test`
- Integration tests: `go test -tags=integration ./...` or `mage integrationTest`
- Integration tests automatically start required Docker services
- Test database: Uses the same PostgreSQL configuration as development

## Code Conventions

- Follow standard Go conventions and idioms
- Use struct tags for configuration validation
- Implement graceful shutdown patterns for services
- Use context-aware logging with component identification
- Prefer composition over inheritance for middleware and configuration
- Use generics for type-safe APIs where appropriate

## Integration Guide for Consuming Projects

### Quick Integration Checklist

When integrating this utility library into a new project with Claude Code:

1. **Initialize Project Structure**:
   ```bash
   go mod init your-project-name
   go get github.com/jasoet/pkg
   ```

2. **Set up Basic Structure**:
   ```
   your-project/
   ├── cmd/                    # Application entrypoints
   ├── internal/               # Private application code
   ├── configs/                # Configuration files
   ├── scripts/                # Build and deployment scripts
   ├── .claude.md              # Claude Code project guidance
   └── main.go                 # Main application
   ```

3. **Initialize Logging First** (Critical):
   ```go
   import "github.com/jasoet/pkg/logging"
   
   func main() {
       logging.Initialize("your-service-name", true)
       // ... rest of your application
   }
   ```

4. **Use Configuration Loading**:
   ```go
   import "github.com/jasoet/pkg/config"
   
   type AppConfig struct {
       Server   ServerConfig   `yaml:"server" mapstructure:"server"`
       Database DatabaseConfig `yaml:"database" mapstructure:"database"`
   }
   
   config, err := config.LoadString[AppConfig](yamlString)
   ```

### Common Integration Patterns

#### Pattern 1: Web Service
```go
// Standard web service integration
func main() {
    logging.Initialize("my-web-service", os.Getenv("DEBUG") == "true")
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    // Load configuration
    appConfig, err := config.LoadString[AppConfig](configYAML)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to load configuration")
    }
    
    // Setup database
    database, err := appConfig.Database.Pool()
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to connect to database")
    }
    
    // Setup server with custom routes
    serverConfig := &server.Config{
        Port: appConfig.Server.Port,
        EchoConfigurer: setupRoutes(database),
    }
    
    srv := server.New(serverConfig)
    logger.Info().Int("port", appConfig.Server.Port).Msg("Starting server")
    srv.Start(ctx)
}
```

#### Pattern 2: Background Worker
```go
// Background worker with concurrent processing
func main() {
    logging.Initialize("my-worker", true)
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "worker")
    
    // Setup dependencies
    database, err := setupDatabase(ctx)
    apiClient := rest.NewClient()
    
    // Process jobs concurrently
    for {
        jobs := fetchPendingJobs(database)
        if len(jobs) == 0 {
            time.Sleep(30 * time.Second)
            continue
        }
        
        err := processJobsConcurrently(ctx, jobs, apiClient, database)
        if err != nil {
            logger.Error().Err(err).Msg("Batch processing failed")
        }
    }
}
```

#### Pattern 3: CLI Tool
```go
// Command-line tool with database operations
func main() {
    logging.Initialize("my-cli", true)
    
    ctx := context.Background()
    
    if len(os.Args) < 2 {
        fmt.Println("Usage: my-cli <command>")
        os.Exit(1)
    }
    
    // Setup database for CLI operations
    database, err := setupDatabase(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    switch os.Args[1] {
    case "migrate":
        runMigrations(ctx, database)
    case "seed":
        seedData(ctx, database)
    case "backup":
        backupData(ctx, database)
    }
}
```

### Integration Anti-Patterns (Avoid These)

❌ **Don't**: Initialize logging multiple times
```go
// WRONG - will cause conflicts
logging.Initialize("service1", true)
logging.Initialize("service2", false)  // This overwrites the first
```

✅ **Do**: Initialize once at application start
```go
// CORRECT
func main() {
    logging.Initialize("my-service", isDebugMode())
    // Use context loggers for components
    logger := logging.ContextLogger(ctx, "component-name")
}
```

❌ **Don't**: Create multiple database connections unnecessarily
```go
// WRONG - creates multiple connection pools
db1, _ := config1.Pool()
db2, _ := config2.Pool()  // If same database, reuse db1
```

✅ **Do**: Reuse database connections or use dependency injection
```go
// CORRECT
type Services struct {
    DB        *gorm.DB
    APIClient *rest.Client
    Logger    zerolog.Logger
}

func NewServices(ctx context.Context) (*Services, error) {
    db, err := setupDatabase(ctx)
    // ... setup other services
    return &Services{DB: db, ...}, nil
}
```

❌ **Don't**: Mix error handling approaches
```go
// WRONG - inconsistent error handling
if err != nil {
    log.Println(err)  // Using standard library
}
if err2 != nil {
    logger.Error().Err(err2).Msg("Failed")  // Using zerolog
}
```

✅ **Do**: Use consistent logging throughout
```go
// CORRECT
logger := logging.ContextLogger(ctx, "service")
if err != nil {
    logger.Error().Err(err).Msg("Operation failed")
}
```

### Environment-Specific Configuration

Create environment-specific configurations:

```yaml
# config/development.yaml
debug: true
server:
  port: 8080
database:
  dbType: POSTGRES
  host: localhost
  port: 5432

# config/production.yaml  
debug: false
server:
  port: ${PORT:8080}
database:
  dbType: POSTGRES
  host: ${DB_HOST}
  port: ${DB_PORT:5432}
```

Load based on environment:
```go
configFile := fmt.Sprintf("config/%s.yaml", getEnvironment())
configData, err := ioutil.ReadFile(configFile)
config, err := config.LoadString[AppConfig](string(configData))
```

### Dependency Management

Always specify version constraints in go.mod:
```go
require (
    github.com/jasoet/pkg v1.1.0  // Use specific version
    // Avoid: github.com/jasoet/pkg latest
)
```

### Testing Integration

Create integration tests for your usage:
```go
func TestServiceIntegration(t *testing.T) {
    // Use test-specific configuration
    testConfig := &db.ConnectionConfig{
        DbType: db.Postgresql,
        Host:   "localhost",
        Port:   5432,
        DbName: "test_db",
    }
    
    db, err := testConfig.Pool()
    require.NoError(t, err)
    
    // Test your service integration
    service := NewService(db)
    result, err := service.ProcessData(context.Background(), testData)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

### Performance Considerations for Integration

1. **Database Connection Pooling**:
   ```go
   config := &db.ConnectionConfig{
       MaxIdleConns: 10,   // Environment-specific
       MaxOpenConns: 100,  // Adjust based on load
       // ... other config
   }
   ```

2. **HTTP Client Reuse**:
   ```go
   // Create once, reuse everywhere
   var apiClient = rest.NewClient(rest.WithRestConfig(rest.Config{
       Timeout: 30 * time.Second,
       RetryCount: 3,
   }))
   ```

3. **Concurrent Operations**:
   ```go
   // Use for I/O-bound operations
   results, err := concurrent.ExecuteConcurrently(ctx, ioOperations)
   
   // Don't use for CPU-bound operations in excess of CPU cores
   ```

### Troubleshooting Integration Issues

#### Issue: Import Path Errors
```bash
# Solution: Ensure correct module path
go get github.com/jasoet/pkg@latest
go mod tidy
```

#### Issue: Database Connection Failures
- Check network connectivity: `telnet host port`
- Verify credentials and permissions
- Check connection pool settings
- Enable debug logging: `logging.Initialize("app", true)`

#### Issue: Context Cancellation
- Always pass context through call chains
- Use `context.WithTimeout` for operations with deadlines
- Handle `ctx.Done()` in long-running operations

#### Issue: Memory Leaks
- Close HTTP response bodies: `defer response.Body.Close()`
- Close database connections properly
- Use connection pooling instead of creating new connections

### Version Compatibility

| Utility Library Version | Go Version | Key Changes |
|------------------------|------------|-------------|
| v1.1.x                 | 1.23+      | Current stable, full feature set |
| v1.0.x                 | 1.22+      | Initial release, basic features |

### Migration Between Versions

When upgrading versions:
1. Check CHANGELOG.md for breaking changes
2. Update import statements if package structure changed
3. Run integration tests after upgrade
4. Update configuration if new options available

This integration guide ensures Claude Code can seamlessly understand and work with this utility library across different projects and use cases.