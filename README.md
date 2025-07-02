# Go Utility Packages

A collection of commonly used Go packages designed to reduce boilerplate and standardize common patterns across Go applications. These battle-tested, reusable components can be imported into new Go applications with minimal setup.

## Installation

```bash
go get github.com/jasoet/pkg
```

## Usage

Import the specific package you need:

```go
import (
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    // ... other packages as needed
)
```

### Example: Setting up logging

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/logging"
)

func main() {
    // Initialize global logger
    logging.Initialize("my-service", true) // service name and debug mode

    // Create a context-aware logger for a component
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "user-service")

    logger.Info().Msg("Service started successfully")
}
```

### Example: Database Connection

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/db"
    "github.com/rs/zerolog/log"
)

func main() {
    config := &db.ConnectionConfig{
        DbType:       db.Postgresql,
        Host:         "localhost",
        Port:         5439,
        Username:     "jasoet",
        Password:     "localhost",
        DbName:       "pkg_db",
        MaxIdleConns: 5,
        MaxOpenConns: 10,
    }

    // Connect to database
    database, err := db.Connect(config)
    if err != nil {
        log.Fatal().Err(err).Msg("Failed to connect to database")
    }

    // Use the database connection
    // ...
}
```

## Package Integration Examples

The packages are designed to work seamlessly together. Here are some real-world integration patterns:

### Complete Web Service Example

```go
package main

import (
    "context"
    "time"

    "github.com/jasoet/pkg/config"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/server"
)

type AppConfig struct {
    Server struct {
        Port int `yaml:"port" mapstructure:"port"`
    } `yaml:"server" mapstructure:"server"`
    Database db.ConnectionConfig `yaml:"database" mapstructure:"database"`
}

func main() {
    // 1. Initialize logging first
    logging.Initialize("my-web-service", true)
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    // 2. Load configuration
    yamlConfig := `
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
    
    appConfig, err := config.LoadString[AppConfig](yamlConfig)
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to load configuration")
    }
    
    // 3. Setup database
    database, err := appConfig.Database.Pool()
    if err != nil {
        logger.Fatal().Err(err).Msg("Failed to connect to database")
    }
    
    // 4. Start HTTP server with integrated components
    serverConfig := &server.Config{
        Port: appConfig.Server.Port,
    }
    
    srv := server.New(serverConfig)
    
    // Server automatically includes logging middleware and health checks
    logger.Info().Int("port", appConfig.Server.Port).Msg("Starting server")
    srv.Start(ctx)
}
```

### Microservice with External APIs

```go
package main

import (
    "context"
    "encoding/json"
    "time"

    "github.com/jasoet/pkg/concurrent"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/rest"
)

type UserService struct {
    apiClient *rest.Client
    logger    zerolog.Logger
}

func (s *UserService) GetDashboardData(ctx context.Context, userID int) (*DashboardData, error) {
    // Use concurrent package to fetch data from multiple APIs in parallel
    apiFunctions := map[string]concurrent.Func[*resty.Response]{
        "profile": func(ctx context.Context) (*resty.Response, error) {
            return s.apiClient.MakeRequest(ctx, "GET", 
                fmt.Sprintf("/users/%d", userID), "", nil)
        },
        "orders": func(ctx context.Context) (*resty.Response, error) {
            return s.apiClient.MakeRequest(ctx, "GET", 
                fmt.Sprintf("/users/%d/orders", userID), "", nil)
        },
        "preferences": func(ctx context.Context) (*resty.Response, error) {
            return s.apiClient.MakeRequest(ctx, "GET", 
                fmt.Sprintf("/users/%d/preferences", userID), "", nil)
        },
    }
    
    s.logger.Info().Int("user_id", userID).Msg("Fetching dashboard data")
    
    results, err := concurrent.ExecuteConcurrently(ctx, apiFunctions)
    if err != nil {
        s.logger.Error().Err(err).Int("user_id", userID).Msg("Failed to fetch dashboard data")
        return nil, err
    }
    
    // Process results...
    dashboard := &DashboardData{}
    // ... marshal JSON responses into dashboard struct
    
    s.logger.Info().Int("user_id", userID).Msg("Dashboard data retrieved successfully")
    return dashboard, nil
}
```

### Database Operations with SSH Tunneling

```go
package main

import (
    "context"
    "time"

    "github.com/jasoet/pkg/config"
    "github.com/jasoet/pkg/db"
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/ssh"
)

type SecureDBConfig struct {
    SSH      ssh.Config            `yaml:"ssh" mapstructure:"ssh"`
    Database db.ConnectionConfig `yaml:"database" mapstructure:"database"`
}

func connectToSecureDatabase(ctx context.Context) (*gorm.DB, error) {
    logger := logging.ContextLogger(ctx, "secure-db")
    
    // Load configuration
    configYAML := `
ssh:
  host: bastion.example.com
  port: 22
  user: deploy
  password: ssh-password
  remoteHost: internal-db.example.com
  remotePort: 5432
  localPort: 5433
database:
  dbType: POSTGRES
  host: localhost
  port: 5433  # Local port from SSH tunnel
  username: dbuser
  password: dbpass
  dbName: production_db
`
    
    config, err := config.LoadString[SecureDBConfig](configYAML)
    if err != nil {
        return nil, err
    }
    
    // 1. Establish SSH tunnel
    logger.Info().Msg("Establishing SSH tunnel to database")
    tunnel := ssh.New(config.SSH)
    err = tunnel.Start()
    if err != nil {
        logger.Error().Err(err).Msg("Failed to start SSH tunnel")
        return nil, err
    }
    
    // 2. Connect to database through tunnel
    logger.Info().Msg("Connecting to database through SSH tunnel")
    database, err := config.Database.Pool()
    if err != nil {
        tunnel.Close()
        logger.Error().Err(err).Msg("Failed to connect to database")
        return nil, err
    }
    
    logger.Info().Msg("Secure database connection established")
    return database, nil
}
```

### Background Worker with All Components

```go
package main

import (
    "context"
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

func (w *WorkerService) ProcessBatch(ctx context.Context, jobIDs []int) error {
    w.logger.Info().Int("job_count", len(jobIDs)).Msg("Starting batch processing")
    
    // Process jobs concurrently
    jobFunctions := make(map[string]concurrent.Func[ProcessResult])
    
    for _, jobID := range jobIDs {
        jobKey := fmt.Sprintf("job_%d", jobID)
        jobFunctions[jobKey] = w.createJobProcessor(jobID)
    }
    
    results, err := concurrent.ExecuteConcurrently(ctx, jobFunctions)
    if err != nil {
        w.logger.Error().Err(err).Msg("Batch processing failed")
        return err
    }
    
    // Update database with results
    for jobKey, result := range results {
        err := w.updateJobStatus(ctx, result.JobID, result.Status)
        if err != nil {
            w.logger.Error().
                Err(err).
                Int("job_id", result.JobID).
                Msg("Failed to update job status")
        }
    }
    
    w.logger.Info().
        Int("total_jobs", len(jobIDs)).
        Int("successful", len(results)).
        Msg("Batch processing completed")
    
    return nil
}

func (w *WorkerService) createJobProcessor(jobID int) concurrent.Func[ProcessResult] {
    return func(ctx context.Context) (ProcessResult, error) {
        jobLogger := logging.ContextLogger(ctx, "job-processor")
        
        // 1. Fetch job details from database
        var job Job
        err := w.db.WithContext(ctx).First(&job, jobID).Error
        if err != nil {
            jobLogger.Error().Err(err).Int("job_id", jobID).Msg("Failed to fetch job")
            return ProcessResult{}, err
        }
        
        // 2. Call external API to process
        response, err := w.apiClient.MakeRequest(ctx, "POST", 
            "/process", job.Data, map[string]string{
                "Content-Type": "application/json",
            })
        if err != nil {
            jobLogger.Error().Err(err).Int("job_id", jobID).Msg("API call failed")
            return ProcessResult{JobID: jobID, Status: "failed"}, err
        }
        
        jobLogger.Info().
            Int("job_id", jobID).
            Int("status_code", response.StatusCode()).
            Msg("Job processed successfully")
        
        return ProcessResult{JobID: jobID, Status: "completed"}, nil
    }
}
```

## Quick Start Guide

### 1. Basic Setup

```go
// main.go
package main

import (
    "context"
    "github.com/jasoet/pkg/logging"
)

func main() {
    // Always start with logging initialization
    logging.Initialize("my-app", true)
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    logger.Info().Msg("Application started")
    // Your application logic here...
}
```

### 2. Add Configuration

```go
import "github.com/jasoet/pkg/config"

type AppConfig struct {
    Port     int    `yaml:"port" mapstructure:"port"`
    LogLevel string `yaml:"logLevel" mapstructure:"logLevel"`
}

func loadConfig() (*AppConfig, error) {
    return config.LoadString[AppConfig](`
port: 8080
logLevel: info
`)
}
```

### 3. Add Database

```go
import "github.com/jasoet/pkg/db"

func setupDatabase() (*gorm.DB, error) {
    config := &db.ConnectionConfig{
        DbType:   db.Postgresql,
        Host:     "localhost",
        Port:     5432,
        Username: "user",
        Password: "pass",
        DbName:   "myapp",
    }
    
    return config.Pool()
}
```

### 4. Add HTTP Server

```go
import "github.com/jasoet/pkg/server"

func startServer(port int) {
    config := &server.Config{Port: port}
    srv := server.New(config)
    srv.Start(context.Background())
}
```

## Packages Overview

### concurrent

Utilities for executing functions concurrently with proper error handling and context cancellation.

### db

Database connection utilities supporting MySQL, PostgreSQL, and SQL Server with connection pooling and migrations.

### logging

Standardized logging setup using zerolog with context-aware logging capabilities.

### rest

HTTP client utilities with retry mechanisms, timeouts, and standardized error handling.

### server

HTTP server utilities using Echo framework with built-in metrics, logging, and health checks.

### ssh

SSH tunnel utilities for securely connecting to remote services.

### temporal

Utilities for working with Temporal workflow engine, including client creation, metrics reporting, and logging integration.

## Troubleshooting

### Common Issues

#### Package Import Errors
```bash
# Ensure you're using the correct import path
go mod tidy
```

#### Database Connection Issues
```go
// Check connection configuration and network access
config := &db.ConnectionConfig{
    DbType: db.Postgresql, // Ensure correct database type
    Host:   "localhost",   // Verify host is accessible
    Port:   5432,          // Check port is correct
    // ... other config
}
```

#### SSH Tunnel Connection Problems
- Verify SSH server is accessible
- Check SSH credentials and permissions
- Ensure remote service is running and accessible from SSH server
- Verify local port is not already in use

#### HTTP Client Timeout Issues
```go
// Adjust timeout configuration based on your needs
config := &rest.Config{
    Timeout: 30 * time.Second, // Increase for slow APIs
    RetryCount: 3,             // Adjust retry behavior
}
```

### Performance Tips

1. **Database Connections**: Configure connection pools appropriately
2. **HTTP Clients**: Reuse clients instead of creating new ones for each request
3. **Concurrent Operations**: Use the concurrent package for I/O-bound operations
4. **Logging**: Use appropriate log levels in production (avoid debug)

### Getting Help

- Check package-specific README files in each `examples/` directory
- Review the comprehensive examples provided
- Use the logging package to debug issues with structured logging

## Roadmap

- [x] Integration with GitHub Actions for CI/CD
- [x] Automated versioning using semantic-release
- [x] Comprehensive examples and documentation
- [x] Package integration patterns
- [ ] Unit testing coverage improvements
- [ ] Additional database drivers support
- [ ] Performance benchmarks and optimization guides
- [ ] Advanced middleware examples
- [ ] Distributed tracing integration

## Semantic Versioning

This project uses [semantic-release](https://github.com/semantic-release/semantic-release) for automated versioning and CHANGELOG generation. The versioning process is triggered automatically when commits are pushed to the main branch.

### How it works

1. When code is pushed to the main branch, GitHub Actions runs the tests
2. If tests pass, semantic-release analyzes the commit messages
3. Based on the commit messages, it determines the next version number
4. A new GitHub release is created with release notes
5. The CHANGELOG.md file is updated with the release notes
6. Changes are committed back to the repository

### Commit Message Format

This project follows the [Conventional Commits](https://www.conventionalcommits.org/) specification. Your commit messages should be structured as follows:

```
<type>(<optional scope>): <description>

[optional body]

[optional footer(s)]
```

Types that trigger version updates:
- `feat`: A new feature (minor version bump)
- `fix`: A bug fix (patch version bump)
- `perf`: A performance improvement (patch version bump)
- `docs`: Documentation changes (no version bump unless scope is README)
- `style`: Changes that do not affect the meaning of the code (no version bump)
- `refactor`: Code changes that neither fix a bug nor add a feature (patch version bump)
- `test`: Adding or correcting tests (patch version bump)
- `build`: Changes to the build system or dependencies (no version bump)
- `ci`: Changes to CI configuration files and scripts (no version bump)
- `chore`: Other changes that don't modify src or test files (patch version bump)

Breaking changes (major version bump) are indicated by:
- Adding `BREAKING CHANGE:` in the commit message body
- Adding a `!` after the type/scope (e.g., `feat!: introduce breaking API change`)

### GitHub Repository Settings

For semantic-release to work properly, ensure:

1. The repository has a `GITHUB_TOKEN` secret (automatically provided by GitHub Actions)
2. Branch protection rules are set up for the main branch (optional but recommended)
3. The GitHub Actions workflow has permission to write to the repository

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Development Guidelines

1. **Branching Strategy**
   - `main` branch is the stable release branch
   - Create feature branches from `main` using the format `feature/your-feature-name`
   - Create bugfix branches using the format `bugfix/issue-description`

2. **Code Style**
   - Run `mage lint` to ensure code follows the project's style guidelines
   - All code should be properly documented with comments

3. **Testing**
   - Run `mage test` to run unit tests
   - Run `mage integrationTest` to run integration tests (requires Docker)
   - Aim for high test coverage for all new code

4. **Commit Messages**
   - Follow [Conventional Commits](https://www.conventionalcommits.org/) format
   - This will be used for automated versioning in the future

### Development Setup

The project uses [Mage](https://magefile.org/) for build automation and Docker Compose for local development services, including PostgreSQL.

#### Setting up the development environment:

1. Start the Docker Compose services:
   ```bash
   mage docker:up
   ```
   This will start PostgreSQL and any other services defined in the docker-compose.yml file.

2. Common Mage commands:
   ```bash
   mage test           # Run unit tests
   mage lint           # Run linter
   mage docker:logs    # View logs from Docker services
   mage docker:down    # Stop Docker services
   mage docker:restart # Restart Docker services
   mage integrationTest # Run integration tests (automatically starts Docker services)
   ```

#### PostgreSQL Configuration:
- Host: localhost
- Port: 5439 (mapped from container's 5432)
- Username: jasoet
- Password: localhost
- Database: pkg_db

The PostgreSQL container is configured to automatically load SQL files from the `scripts/compose/pg/backup` directory during initialization.

## License

This project is licensed under the MIT License - see the LICENSE file for details.
