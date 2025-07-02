# CLI Application Template

This template provides a complete command-line application implementation using [github.com/jasoet/pkg](https://github.com/jasoet/pkg).

## Features

- **Command-Line Interface**: Full-featured CLI with subcommands and flags
- **Database Operations**: Database migrations, seeding, and management
- **User Management**: CRUD operations for user entities
- **Data Export/Import**: Backup and restore functionality
- **Structured Configuration**: YAML configuration with environment variable overrides
- **Database Integration**: PostgreSQL with GORM for data persistence
- **External API Integration**: HTTP client for external service calls
- **Structured Logging**: Context-aware logging for all operations
- **Error Handling**: Comprehensive error handling with proper exit codes
- **Help System**: Built-in help and usage information

## Quick Start

1. **Copy Template**
   ```bash
   cp -r templates/cli-app my-cli-tool
   cd my-cli-tool
   ```

2. **Update Module Name**
   ```bash
   go mod edit -module github.com/yourusername/my-cli-tool
   ```

3. **Install Dependencies**
   ```bash
   go mod tidy
   ```

4. **Start Database**
   ```bash
   docker run -d --name cli-postgres \
     -e POSTGRES_USER=postgres \
     -e POSTGRES_PASSWORD=password \
     -e POSTGRES_DB=cli_app \
     -p 5432:5432 \
     postgres:15-alpine
   ```

5. **Build and Run**
   ```bash
   go build -o cli-tool
   ./cli-tool -cmd help
   ```

## Configuration

The CLI uses `config.yaml` for base configuration with environment variable overrides:

```yaml
environment: development
debug: true
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: postgres
  password: password
  dbName: cli_app
```

### Environment Variable Overrides

Use the `CLI_` prefix for environment variables:

```bash
export CLI_DATABASE_HOST=production-db.example.com
export CLI_DATABASE_PASSWORD=secure-password
export CLI_DEBUG=true
```

## Available Commands

### Database Management
```bash
# Run database migrations
./cli-tool -cmd migrate

# Seed database with sample data
./cli-tool -cmd seed

# Create database backup
./cli-tool -cmd backup
```

### User Management
```bash
# List all users
./cli-tool -cmd users

# Show specific user by ID
./cli-tool -cmd users -id 1

# Search user by email
./cli-tool -cmd users -email "john@example.com"

# Create new user
./cli-tool -cmd users -name "John Doe" -email "john@example.com"
```

### General Commands
```bash
# Show help
./cli-tool -cmd help

# Verbose output
./cli-tool -cmd users -v

# Custom config file
./cli-tool -config production.yaml -cmd migrate
```

## Command Line Options

| Flag | Description | Example |
|------|-------------|---------|
| `-cmd` | Command to execute | `-cmd migrate` |
| `-config` | Configuration file | `-config prod.yaml` |
| `-id` | User ID for operations | `-id 123` |
| `-name` | User name | `-name "John Doe"` |
| `-email` | User email | `-email "john@example.com"` |
| `-v` | Verbose output | `-v` |

## Adding Custom Commands

1. **Add Command Handler**
   ```go
   func customCommand(ctx context.Context, services *Services) error {
       logger := logging.ContextLogger(ctx, "custom-command")
       logger.Info().Msg("Executing custom command")
       
       // Your custom logic here
       
       return nil
   }
   ```

2. **Register in executeCommand**
   ```go
   switch strings.ToLower(command) {
   case "migrate":
       return runMigrations(ctx, services)
   case "custom":
       return customCommand(ctx, services)
   // ... other cases
   }
   ```

3. **Update Help Text**
   ```go
   fmt.Println("Commands:")
   fmt.Println("  migrate              Run database migrations")
   fmt.Println("  custom               Execute custom operation")
   ```

## Database Models

### User Model
```go
type User struct {
    ID        uint      `json:"id" gorm:"primaryKey"`
    Name      string    `json:"name" gorm:"not null"`
    Email     string    `json:"email" gorm:"unique;not null"`
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
```

### Adding New Models
```go
type Product struct {
    ID          uint      `json:"id" gorm:"primaryKey"`
    Name        string    `json:"name" gorm:"not null"`
    Description string    `json:"description"`
    Price       float64   `json:"price" gorm:"not null"`
    CreatedAt   time.Time `json:"created_at" gorm:"autoCreateTime"`
    UpdatedAt   time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// Add to migrations
func runMigrations(ctx context.Context, services *Services) error {
    if err := services.DB.AutoMigrate(&User{}, &Product{}); err != nil {
        return fmt.Errorf("failed to migrate models: %w", err)
    }
    return nil
}
```

## Error Handling

The CLI uses structured error handling with proper exit codes:

```go
func executeCommand(ctx context.Context, services *Services, command string) error {
    switch command {
    case "migrate":
        if err := runMigrations(ctx, services); err != nil {
            return fmt.Errorf("migration failed: %w", err)
        }
    default:
        return fmt.Errorf("unknown command: %s", command)
    }
    return nil
}
```

Main function handles errors and exit codes:
```go
if err := executeCommand(ctx, services, *command); err != nil {
    logger.Error().Err(err).Msg("Command execution failed")
    os.Exit(1)  // Non-zero exit code for failures
}
```

## Logging and Observability

### Structured Logging
All operations include context-aware logging:
```go
func createUser(ctx context.Context, services *Services, name, email string) error {
    logger := logging.ContextLogger(ctx, "create-user")
    
    logger.Info().
        Str("name", name).
        Str("email", email).
        Msg("Creating user")
    
    // ... operation logic
    
    logger.Info().
        Uint("user_id", user.ID).
        Msg("User created successfully")
    
    return nil
}
```

### Debug Output
Enable verbose logging:
```bash
./cli-tool -v -cmd migrate
# or
DEBUG=true ./cli-tool -cmd migrate
```

## Production Usage

### Building for Production
```bash
# Build optimized binary
CGO_ENABLED=0 go build -ldflags="-w -s" -o cli-tool

# Cross-platform builds
GOOS=linux GOARCH=amd64 go build -o cli-tool-linux
GOOS=windows GOARCH=amd64 go build -o cli-tool.exe
GOOS=darwin GOARCH=amd64 go build -o cli-tool-mac
```

### Docker Image
```dockerfile
FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o cli-tool .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/cli-tool .
COPY --from=builder /app/config.yaml .
ENTRYPOINT ["./cli-tool"]
```

### Environment Configuration
Create production configuration:
```yaml
# config.production.yaml
environment: production
debug: false
database:
  host: ${DB_HOST}
  port: ${DB_PORT:5432}
  username: ${DB_USER}
  password: ${DB_PASSWORD}
  dbName: ${DB_NAME}
```

## Testing

### Unit Tests
```go
func TestCreateUser(t *testing.T) {
    // Setup test database
    testDB := setupTestDB(t)
    defer testDB.Close()
    
    services := &Services{DB: testDB}
    ctx := context.Background()
    
    // Test user creation
    err := createUser(ctx, services, "Test User", "test@example.com")
    assert.NoError(t, err)
    
    // Verify user was created
    var user User
    err = testDB.Where("email = ?", "test@example.com").First(&user).Error
    assert.NoError(t, err)
    assert.Equal(t, "Test User", user.Name)
}
```

### Integration Tests
```bash
# Setup test database
docker run -d --name test-postgres \
  -e POSTGRES_USER=test \
  -e POSTGRES_PASSWORD=test \
  -e POSTGRES_DB=test_db \
  -p 5433:5432 \
  postgres:15-alpine

# Run tests
CLI_DATABASE_HOST=localhost \
CLI_DATABASE_PORT=5433 \
CLI_DATABASE_USERNAME=test \
CLI_DATABASE_PASSWORD=test \
CLI_DATABASE_DBNAME=test_db \
go test ./...
```

## Performance and Best Practices

### Database Connection Management
- CLI applications should use minimal connection pools
- Close connections properly after operations
- Use transactions for multi-step operations

### Memory Usage
- Process large datasets in batches
- Stream results for export operations
- Use proper cleanup for temporary resources

### Command Design
- Keep commands focused and single-purpose
- Provide clear error messages
- Include progress indicators for long operations

## Troubleshooting

### Common Issues

**Configuration not found**
```bash
# Specify config file explicitly
./cli-tool -config /path/to/config.yaml -cmd migrate
```

**Database connection failed**
```bash
# Check database connectivity
CLI_DATABASE_HOST=localhost CLI_DEBUG=true ./cli-tool -cmd migrate
```

**Permission denied**
```bash
# Make binary executable
chmod +x cli-tool
```

### Debug Commands
```bash
# Test database connection
./cli-tool -v -cmd migrate

# Check configuration loading
DEBUG=true ./cli-tool -cmd help

# Validate user operations
./cli-tool -v -cmd users
```

## Support

For issues with this template or the underlying library:
- Template issues: Create issue in your project repository
- Library issues: https://github.com/jasoet/pkg/issues  
- Documentation: Check the [integration guide](../../.claude/integration-guide.md)