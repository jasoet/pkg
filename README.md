# 🚀 Go Utility Packages

[![Go Version](https://img.shields.io/badge/Go-1.23+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Build Status](https://github.com/jasoet/pkg/workflows/CI/badge.svg)](https://github.com/jasoet/pkg/actions)

A comprehensive collection of production-ready Go utility packages designed to eliminate boilerplate and standardize common patterns across Go applications. These battle-tested components integrate seamlessly to accelerate development while maintaining best practices.

## 📦 What's Inside

| Package | Description | Key Features |
|---------|-------------|--------------|
| **[config](./config/)** | YAML configuration with env overrides | Type-safe, validation, hot-reload |
| **[logging](./logging/)** | Structured logging with zerolog | Context-aware, performance optimized |
| **[db](./db/)** | Multi-database support | PostgreSQL, MySQL, MSSQL, migrations |
| **[server](./server/)** | HTTP server with Echo | Health checks, metrics, graceful shutdown |
| **[rest](./rest/)** | HTTP client framework | Retries, timeouts, middleware support |
| **[concurrent](./concurrent/)** | Type-safe concurrent execution | Generics, error handling, cancellation |
| **[temporal](./temporal/)** | Temporal workflow integration | Workers, scheduling, monitoring |
| **[ssh](./ssh/)** | SSH tunneling utilities | Secure connections, port forwarding |
| **[compress](./compress/)** | File compression utilities | ZIP, tar.gz with security validation |

## 🎯 Quick Start

### Installation

```bash
go get github.com/jasoet/pkg
```

### Hello World Example

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/server"
)

func main() {
    // 1. Initialize logging (always first!)
    logging.Initialize("my-app", true)
    
    // 2. Get a context logger
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    // 3. Start a web server with built-in health checks
    logger.Info().Msg("Starting server on :8080")
    
    config := server.Config{Port: 8080}
    server.StartWithConfig(config)
}
```

Visit `http://localhost:8080/health` to see your server running! 🎉

## 🔧 Usage Patterns

### 🌐 Complete Web Service

Build a production-ready web service in minutes:

```go
package main

import (
    "context"
    "os"
    
    "github.com/jasoet/pkg/config"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/server"
    "github.com/labstack/echo/v4"
)

type AppConfig struct {
    Server   ServerConfig        `yaml:"server" mapstructure:"server"`
    Database db.ConnectionConfig `yaml:"database" mapstructure:"database"`
}

type ServerConfig struct {
    Port int `yaml:"port" mapstructure:"port"`
}

func main() {
    // 1. Initialize logging first
    logging.Initialize("web-service", os.Getenv("DEBUG") == "true")
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    // 2. Load configuration
    appConfig, err := config.LoadString[AppConfig](`
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
`)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to load configuration")
    }
    
    // 3. Setup database
    database, err := appConfig.Database.Pool()
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to connect to database")
    }
    
    // 4. Setup routes
    serverConfig := server.Config{
        Port: appConfig.Server.Port,
        EchoConfigurer: func(e *echo.Echo) {
            // Add your routes here
            e.GET("/api/users", getUsersHandler)
            e.POST("/api/users", createUserHandler)
        },
    }
    
    // 5. Start server with automatic graceful shutdown
    logger.Info().Int("port", appConfig.Server.Port).Msg("Starting server")
    server.StartWithConfig(serverConfig)
}

func getUsersHandler(c echo.Context) error {
    // Your handler logic
    return c.JSON(200, map[string]string{"message": "Users endpoint"})
}

func createUserHandler(c echo.Context) error {
    // Your handler logic  
    return c.JSON(201, map[string]string{"message": "User created"})
}
```

### 🔄 Background Worker

Process jobs concurrently with full observability:

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/jasoet/pkg/concurrent"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/rest"
)

type WorkerService struct {
    db        *gorm.DB
    apiClient *rest.Client
    logger    zerolog.Logger
}

func (w *WorkerService) ProcessJobsBatch(ctx context.Context, jobIDs []int) error {
    w.logger.Info().Int("job_count", len(jobIDs)).Msg("Starting batch processing")
    
    // Create concurrent job processors
    jobFunctions := make(map[string]concurrent.Func[JobResult])
    
    for _, jobID := range jobIDs {
        jobKey := fmt.Sprintf("job_%d", jobID)
        jobFunctions[jobKey] = w.createJobProcessor(jobID)
    }
    
    // Execute all jobs concurrently
    results, err := concurrent.ExecuteConcurrently(ctx, jobFunctions)
    if err != nil {
        w.logger.Error().Err(err).Msg("Batch processing failed")
        return err
    }
    
    // Process results
    successCount := 0
    for jobKey, result := range results {
        if result.Success {
            successCount++
        }
        w.logger.Info().
            Str("job_key", jobKey).
            Bool("success", result.Success).
            Msg("Job completed")
    }
    
    w.logger.Info().
        Int("total", len(jobIDs)).
        Int("successful", successCount).
        Msg("Batch processing completed")
    
    return nil
}

func (w *WorkerService) createJobProcessor(jobID int) concurrent.Func[JobResult] {
    return func(ctx context.Context) (JobResult, error) {
        // 1. Fetch job from database
        var job Job
        err := w.db.WithContext(ctx).First(&job, jobID).Error
        if err != nil {
            return JobResult{Success: false}, err
        }
        
        // 2. Process via external API
        response, err := w.apiClient.MakeRequest(ctx, "POST", 
            "/process", job.Data, map[string]string{
                "Content-Type": "application/json",
            })
        if err != nil {
            return JobResult{Success: false}, err
        }
        
        // 3. Update job status
        job.Status = "completed"
        job.ProcessedAt = time.Now()
        w.db.WithContext(ctx).Save(&job)
        
        return JobResult{Success: true, JobID: jobID}, nil
    }
}

type JobResult struct {
    Success bool
    JobID   int
}

type Job struct {
    ID          int       `gorm:"primaryKey"`
    Data        string    `gorm:"type:jsonb"`
    Status      string    `gorm:"default:'pending'"`
    ProcessedAt time.Time
}
```

### 🔐 Secure Database Access

Connect to databases through SSH tunnels:

```go
package main

import (
    "context"
    
    "github.com/jasoet/pkg/config"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/ssh"
)

type SecureDBConfig struct {
    SSH      ssh.Config            `yaml:"ssh" mapstructure:"ssh"`
    Database db.ConnectionConfig   `yaml:"database" mapstructure:"database"`
}

func connectSecurely(ctx context.Context) (*gorm.DB, error) {
    logger := logging.ContextLogger(ctx, "secure-db")
    
    // Load configuration
    config, err := config.LoadString[SecureDBConfig](`
ssh:
  host: bastion.example.com
  port: 22
  user: deploy
  privateKeyPath: ~/.ssh/id_rsa
  remoteHost: internal-db.example.com
  remotePort: 5432
  localPort: 5433
database:
  dbType: POSTGRES
  host: localhost
  port: 5433  # Local tunnel port
  username: dbuser
  password: dbpass
  dbName: production_db
`)
    if err != nil {
        return nil, err
    }
    
    // 1. Start SSH tunnel
    logger.Info().Msg("Establishing SSH tunnel")
    tunnel := ssh.New(config.SSH)
    if err := tunnel.Start(); err != nil {
        return nil, fmt.Errorf("failed to start SSH tunnel: %w", err)
    }
    
    // 2. Connect to database through tunnel
    logger.Info().Msg("Connecting to database through tunnel")
    database, err := config.Database.Pool()
    if err != nil {
        tunnel.Close()
        return nil, fmt.Errorf("failed to connect to database: %w", err)
    }
    
    logger.Info().Msg("Secure database connection established")
    return database, nil
}
```

### ⚡ Temporal Workflows

Build robust, durable workflows:

```go
package main

import (
    "context"
    "fmt"
    "time"
    
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/temporal"
    "go.temporal.io/sdk/workflow"
    "go.temporal.io/sdk/activity"
)

// Order processing workflow
func OrderProcessingWorkflow(ctx workflow.Context, orderID string) error {
    logger := workflow.GetLogger(ctx)
    logger.Info("Starting order processing", "orderID", orderID)
    
    // Configure activity options
    activityOptions := workflow.ActivityOptions{
        StartToCloseTimeout: 30 * time.Second,
        RetryPolicy: &temporal.RetryPolicy{
            MaximumAttempts: 3,
        },
    }
    ctx = workflow.WithActivityOptions(ctx, activityOptions)
    
    // Step 1: Validate order
    err := workflow.ExecuteActivity(ctx, ValidateOrderActivity, orderID).Get(ctx, nil)
    if err != nil {
        logger.Error("Order validation failed", "orderID", orderID, "error", err)
        return err
    }
    
    // Step 2: Process payment
    var paymentID string
    err = workflow.ExecuteActivity(ctx, ProcessPaymentActivity, orderID).Get(ctx, &paymentID)
    if err != nil {
        logger.Error("Payment processing failed", "orderID", orderID, "error", err)
        return err
    }
    
    // Step 3: Update inventory
    err = workflow.ExecuteActivity(ctx, UpdateInventoryActivity, orderID).Get(ctx, nil)
    if err != nil {
        // Compensate: refund payment
        workflow.ExecuteActivity(ctx, RefundPaymentActivity, paymentID)
        return err
    }
    
    // Step 4: Ship order
    err = workflow.ExecuteActivity(ctx, ShipOrderActivity, orderID).Get(ctx, nil)
    if err != nil {
        // Compensate: restore inventory and refund
        workflow.ExecuteActivity(ctx, RestoreInventoryActivity, orderID)
        workflow.ExecuteActivity(ctx, RefundPaymentActivity, paymentID)
        return err
    }
    
    logger.Info("Order processing completed successfully", "orderID", orderID)
    return nil
}

// Activities
func ValidateOrderActivity(ctx context.Context, orderID string) error {
    logger := activity.GetLogger(ctx)
    logger.Info("Validating order", "orderID", orderID)
    
    // Your validation logic here
    time.Sleep(1 * time.Second) // Simulate work
    
    return nil
}

func ProcessPaymentActivity(ctx context.Context, orderID string) (string, error) {
    logger := activity.GetLogger(ctx)
    logger.Info("Processing payment", "orderID", orderID)
    
    // Your payment processing logic here
    time.Sleep(2 * time.Second) // Simulate work
    
    return fmt.Sprintf("payment_%s", orderID), nil
}

func UpdateInventoryActivity(ctx context.Context, orderID string) error {
    logger := activity.GetLogger(ctx)
    logger.Info("Updating inventory", "orderID", orderID)
    
    // Your inventory update logic here
    time.Sleep(1 * time.Second) // Simulate work
    
    return nil
}

func ShipOrderActivity(ctx context.Context, orderID string) error {
    logger := activity.GetLogger(ctx)
    logger.Info("Shipping order", "orderID", orderID)
    
    // Your shipping logic here
    time.Sleep(3 * time.Second) // Simulate work
    
    return nil
}

func RefundPaymentActivity(ctx context.Context, paymentID string) error {
    logger := activity.GetLogger(ctx)
    logger.Info("Refunding payment", "paymentID", paymentID)
    
    // Your refund logic here
    return nil
}

func RestoreInventoryActivity(ctx context.Context, orderID string) error {
    logger := activity.GetLogger(ctx)
    logger.Info("Restoring inventory", "orderID", orderID)
    
    // Your inventory restoration logic here
    return nil
}

// Worker setup
func main() {
    logging.Initialize("temporal-worker", true)
    
    // Create Temporal client
    client, err := temporal.NewClient(temporal.ClientConfig{
        HostPort: "localhost:7233",
    })
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to create Temporal client")
    }
    defer client.Close()
    
    // Create and start worker
    worker := temporal.NewWorker(client, "order-processing-queue")
    
    // Register workflows and activities
    worker.RegisterWorkflow(OrderProcessingWorkflow)
    worker.RegisterActivity(ValidateOrderActivity)
    worker.RegisterActivity(ProcessPaymentActivity)
    worker.RegisterActivity(UpdateInventoryActivity)
    worker.RegisterActivity(ShipOrderActivity)
    worker.RegisterActivity(RefundPaymentActivity)
    worker.RegisterActivity(RestoreInventoryActivity)
    
    // Start worker
    err = worker.Run(context.Background())
    if err != nil {
        log.Fatal().Err(err).Msg("Worker failed")
    }
}
```

## 📖 Package Documentation

### Core Packages

#### 🔧 [config](./config/)
Type-safe YAML configuration with environment variable overrides.

```go
type Config struct {
    Port     int    `yaml:"port" mapstructure:"port" validate:"min=1,max=65535"`
    LogLevel string `yaml:"logLevel" mapstructure:"logLevel" validate:"oneof=debug info warn error"`
}

config, err := config.LoadString[Config](yamlString)
```

#### 📝 [logging](./logging/)
Structured logging with context awareness and performance optimization.

```go
logging.Initialize("service-name", true) // debug mode
logger := logging.ContextLogger(ctx, "component")
logger.Info().Str("user_id", "123").Msg("User created")
```

#### 🗄️ [db](./db/)
Multi-database support with connection pooling and migrations.

```go
config := &db.ConnectionConfig{
    DbType: db.Postgresql,
    Host:   "localhost",
    Port:   5432,
    // ... other config
}
database, err := config.Pool()
```

#### 🌐 [server](./server/)
HTTP server with built-in health checks, metrics, and graceful shutdown.

```go
config := server.Config{
    Port: 8080,
    EchoConfigurer: func(e *echo.Echo) {
        e.GET("/api/hello", helloHandler)
    },
}
server.StartWithConfig(config)
```

#### 🌍 [rest](./rest/)
HTTP client with retries, timeouts, and middleware support.

```go
client := rest.NewClient()
response, err := client.MakeRequest(ctx, "GET", "https://api.example.com/users", "", nil)
```

#### ⚡ [concurrent](./concurrent/)
Type-safe concurrent execution with generics.

```go
functions := map[string]concurrent.Func[string]{
    "api1": func(ctx context.Context) (string, error) { /* ... */ },
    "api2": func(ctx context.Context) (string, error) { /* ... */ },
}
results, err := concurrent.ExecuteConcurrently(ctx, functions)
```

#### 🔄 [temporal](./temporal/)
Temporal workflow integration with monitoring and logging.

```go
client, err := temporal.NewClient(temporal.ClientConfig{HostPort: "localhost:7233"})
worker := temporal.NewWorker(client, "task-queue")
worker.RegisterWorkflow(MyWorkflow)
```

#### 🔐 [ssh](./ssh/)
SSH tunneling for secure remote connections.

```go
config := ssh.Config{
    Host: "bastion.example.com",
    Port: 22,
    User: "deploy",
    // ... other config
}
tunnel := ssh.New(config)
err := tunnel.Start()
```

#### 📦 [compress](./compress/)
File compression with security validation.

```go
err := compress.CreateZip("output.zip", []string{"file1.txt", "file2.txt"})
err := compress.ExtractZip("archive.zip", "output/directory")
```

## 🎭 Examples & Templates

### Running Examples

Examples are isolated with build tags. To run them:

```bash
# Run specific examples
go run -tags=example ./logging/examples
go run -tags=example ./db/examples
go run -tags=example ./server/examples

# Build all examples
go build -tags=example ./...
```

### Project Templates

Bootstrap new projects with our templates:

```bash
# Copy a template to start a new project
cp -r templates/web-service my-new-service
cd my-new-service

# Initialize as new module
go mod init my-new-service
go mod tidy

# Build and run
go build -tags=template .
./my-new-service
```

Available templates:
- **web-service**: Complete REST API with database
- **worker**: Background job processor
- **cli-app**: Command-line application

## 🔧 Development

### Prerequisites

- Go 1.23+
- [Mage](https://magefile.org/) for build automation
- Docker & Docker Compose for services

### Development Commands

```bash
# Development environment
mage docker:up          # Start PostgreSQL and other services
mage test              # Run unit tests
mage integrationTest   # Run integration tests
mage lint              # Run linter
mage security          # Security analysis
mage coverage          # Generate coverage report

# Docker services management
mage docker:down       # Stop services
mage docker:restart    # Restart services
mage docker:logs       # View service logs

# Quality checks
mage checkall          # Run all quality checks
mage dependencies      # Check for vulnerabilities
mage docs              # Generate documentation
```

### Database Configuration

PostgreSQL is available for testing:
- **Host**: localhost:5439
- **Username**: jasoet
- **Password**: localhost
- **Database**: pkg_db

## 🤝 Contributing

We welcome contributions! Please read our [Contributing Guide](CONTRIBUTING.md) for details.

### Quick Contribution Guide

1. **Fork & Clone**
   ```bash
   git clone https://github.com/your-username/pkg.git
   cd pkg
   ```

2. **Setup Development Environment**
   ```bash
   mage docker:up
   mage test
   ```

3. **Create Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

4. **Make Changes & Test**
   ```bash
   mage test
   mage lint
   mage integrationTest
   ```

5. **Commit with Conventional Commits**
   ```bash
   git commit -m "feat: add new feature"
   git commit -m "fix: resolve issue"
   git commit -m "docs: update README"
   ```

6. **Push & Create PR**
   ```bash
   git push origin feature/your-feature-name
   # Create pull request on GitHub
   ```

### Commit Message Format

We use [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

**Types**: `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`
**Breaking Changes**: Add `!` after type or `BREAKING CHANGE:` in footer

## 📈 Roadmap

- [x] **Core Packages**: All essential utilities implemented
- [x] **Integration Examples**: Real-world usage patterns
- [x] **Build Automation**: Mage-based development workflow
- [x] **CI/CD Pipeline**: Automated testing and releases
- [x] **Comprehensive Documentation**: Examples and guides
- [ ] **Performance Benchmarks**: Optimization guides and metrics
- [ ] **Distributed Tracing**: OpenTelemetry integration
- [ ] **Additional Database Drivers**: MongoDB, Redis support
- [ ] **Cloud Provider Integrations**: AWS, GCP, Azure utilities
- [ ] **Kubernetes Helpers**: Service discovery, health checks

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**[⬆ Back to Top](#-go-utility-packages)**

Made with ❤️ by [Jasoet](https://github.com/jasoet)

</div>