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

- [x] Integration with GitHub Actions for CI/CD
- [x] Automated versioning using semantic-release
- [ ] Unit testing coverage improvements
- [ ] Documentation improvements
- [ ] Additional database drivers support
- [ ] More comprehensive examples

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
