# Claude Code Integration Guide for github.com/jasoet/pkg

This file provides Claude Code with specific guidance for integrating this utility library into other projects.

## Library Overview

This is a comprehensive Go utility library providing production-ready infrastructure components:
- **config**: YAML configuration with environment overrides
- **logging**: Structured logging with zerolog
- **db**: Multi-database support with GORM and migrations
- **server**: Echo-based HTTP server with observability
- **rest**: HTTP client with middleware and retry logic
- **concurrent**: Type-safe parallel execution
- **ssh**: SSH tunneling for secure connections
- **temporal**: Temporal workflow engine integration
- **compress**: File compression and archiving

## Integration Priorities

When helping users integrate this library, follow this priority order:

1. **Always start with logging initialization**
2. **Set up configuration loading**
3. **Initialize required services (database, HTTP client, etc.)**
4. **Set up application-specific logic**
5. **Add observability and health checks**

## Common Use Cases and Templates

### Web Service Integration
```go
// Template for web services using this library
package main

import (
    "context"
    "os"
    
    "github.com/jasoet/pkg/config"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/server"
)

type AppConfig struct {
    Debug    bool              `yaml:"debug" mapstructure:"debug"`
    Server   server.Config     `yaml:"server" mapstructure:"server"`
    Database db.ConnectionConfig `yaml:"database" mapstructure:"database"`
}

func main() {
    // 1. Initialize logging first (CRITICAL)
    logging.Initialize("service-name", os.Getenv("DEBUG") == "true")
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    // 2. Load configuration
    configYAML := `
debug: true
server:
  port: 8080
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: user
  password: pass
  dbName: myapp
  maxIdleConns: 10
  maxOpenConns: 100
`
    
    appConfig, err := config.LoadString[AppConfig](configYAML)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to load configuration")
    }
    
    // 3. Setup database
    database, err := appConfig.Database.Pool()
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to connect to database")
    }
    
    // 4. Setup server with routes
    srv := server.New(&appConfig.Server)
    
    logger.Info().Int("port", appConfig.Server.Port).Msg("Starting server")
    srv.Start(ctx)
}
```

### Background Worker Integration
```go
// Template for background workers
package main

import (
    "context"
    "time"
    
    "github.com/jasoet/pkg/concurrent"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/rest"
)

func main() {
    logging.Initialize("worker-service", true)
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "worker")
    
    // Setup dependencies
    database, err := setupDatabase(ctx)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to setup database")
    }
    
    apiClient := rest.NewClient()
    
    // Main worker loop
    for {
        jobs := fetchPendingJobs(database)
        if len(jobs) == 0 {
            time.Sleep(30 * time.Second)
            continue
        }
        
        logger.Info().Int("job_count", len(jobs)).Msg("Processing jobs")
        
        // Process jobs concurrently
        jobFunctions := make(map[string]concurrent.Func[JobResult])
        for _, job := range jobs {
            jobFunctions[job.ID] = createJobProcessor(job, apiClient, database)
        }
        
        results, err := concurrent.ExecuteConcurrently(ctx, jobFunctions)
        if err != nil {
            logger.Error().Err(err).Msg("Batch processing failed")
            continue
        }
        
        logger.Info().Int("completed", len(results)).Msg("Jobs completed")
    }
}
```

### CLI Tool Integration
```go
// Template for CLI tools
package main

import (
    "context"
    "flag"
    "fmt"
    "os"
    
    "github.com/jasoet/pkg/config"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
)

func main() {
    var (
        configFile = flag.String("config", "config.yaml", "Configuration file")
        command    = flag.String("cmd", "", "Command to execute")
    )
    flag.Parse()
    
    logging.Initialize("cli-tool", true)
    ctx := context.Background()
    
    // Load configuration from file
    configData, err := os.ReadFile(*configFile)
    if err != nil {
        fmt.Printf("Failed to read config file: %v\n", err)
        os.Exit(1)
    }
    
    appConfig, err := config.LoadString[AppConfig](string(configData))
    if err != nil {
        fmt.Printf("Failed to parse config: %v\n", err)
        os.Exit(1)
    }
    
    // Setup database
    database, err := appConfig.Database.Pool()
    if err != nil {
        fmt.Printf("Failed to connect to database: %v\n", err)
        os.Exit(1)
    }
    
    // Execute command
    switch *command {
    case "migrate":
        runMigrations(ctx, database)
    case "seed":
        seedData(ctx, database)
    case "backup":
        backupData(ctx, database)
    default:
        fmt.Println("Usage: cli-tool -cmd <migrate|seed|backup>")
        os.Exit(1)
    }
}
```

## Integration Validation

When integrating this library, validate these key points:

1. **Logging is initialized before any other operations**
2. **Context is properly propagated through all function calls**
3. **Database connections are reused, not created per operation**
4. **HTTP clients are reused across requests**
5. **Configuration includes all required fields with proper validation tags**
6. **Error handling is consistent using the logging package**

## Common Integration Mistakes

### ❌ Mistake 1: Multiple logging initializations
```go
// WRONG
func serviceA() {
    logging.Initialize("serviceA", true)
}
func serviceB() {
    logging.Initialize("serviceB", false)
}
```

### ✅ Correct approach:
```go
// CORRECT
func main() {
    logging.Initialize("my-app", isDebugMode())
}
func serviceA(ctx context.Context) {
    logger := logging.ContextLogger(ctx, "serviceA")
}
```

### ❌ Mistake 2: Creating multiple database connections
```go
// WRONG
func handlerA() {
    db, _ := dbConfig.Pool()  // New connection
}
func handlerB() {
    db, _ := dbConfig.Pool()  // Another new connection
}
```

### ✅ Correct approach:
```go
// CORRECT
type App struct {
    DB *gorm.DB
}
func (a *App) handlerA() {
    // Use a.DB
}
```

### ❌ Mistake 3: Not using context
```go
// WRONG
func processData() error {
    // No context propagation
    result, err := apiClient.MakeRequest("GET", "/data", "", nil)
}
```

### ✅ Correct approach:
```go
// CORRECT
func processData(ctx context.Context) error {
    result, err := apiClient.MakeRequest(ctx, "GET", "/data", "", nil)
}
```

## Performance Guidelines

1. **Connection Pooling**: Configure appropriate pool sizes based on load
   ```go
   config := &db.ConnectionConfig{
       MaxIdleConns: 10,   // Keep connections warm
       MaxOpenConns: 100,  // Limit total connections
   }
   ```

2. **HTTP Client Reuse**: Create once, use everywhere
   ```go
   var globalAPIClient = rest.NewClient(rest.WithRestConfig(restConfig))
   ```

3. **Concurrent Processing**: Use for I/O-bound operations
   ```go
   // Good for API calls, database queries
   results, err := concurrent.ExecuteConcurrently(ctx, ioOperations)
   ```

## Testing Integration

Always include integration tests:

```go
func TestIntegration(t *testing.T) {
    // Setup test environment
    testConfig := &db.ConnectionConfig{
        DbType: db.Postgresql,
        Host:   "localhost",
        Port:   5432,
        DbName: "test_db",
    }
    
    db, err := testConfig.Pool()
    require.NoError(t, err)
    defer db.Close()
    
    // Test your integration
    service := NewService(db)
    result, err := service.Process(context.Background(), testInput)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Environment Configuration

Create environment-specific configurations:

```yaml
# Development
debug: true
server:
  port: 8080
database:
  host: localhost
  port: 5432

# Production  
debug: false
server:
  port: ${PORT:8080}
database:
  host: ${DB_HOST}
  port: ${DB_PORT:5432}
  username: ${DB_USER}
  password: ${DB_PASSWORD}
```

## Deployment Considerations

1. **Health Checks**: Always enable health endpoints
2. **Metrics**: Configure Prometheus metrics for observability
3. **Graceful Shutdown**: Implement proper shutdown handlers
4. **Resource Limits**: Set appropriate connection pool limits
5. **Security**: Never log sensitive information

This guide ensures Claude Code can effectively help users integrate this utility library following best practices and avoiding common pitfalls.