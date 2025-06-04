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

## Roadmap

- [ ] Integration with GitHub Actions for CI/CD
- [ ] Automated versioning using semantic-release
- [ ] Unit testing coverage improvements
- [ ] Documentation improvements
- [ ] Additional database drivers support
- [ ] More comprehensive examples

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
