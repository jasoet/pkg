# Project Templates for github.com/jasoet/pkg

This directory contains production-ready project templates that demonstrate how to properly integrate and use the [github.com/jasoet/pkg](https://github.com/jasoet/pkg) utility library.

## Available Templates

### 🌐 [Web Service](./web-service/)
Complete web service implementation with REST API, database integration, and observability.

**Features:**
- Echo-based HTTP server with middleware
- PostgreSQL database with GORM
- Structured logging and health checks
- Docker support with multi-stage builds
- Graceful shutdown and signal handling

**Use Cases:**
- REST APIs and microservices
- CRUD applications
- Services requiring HTTP endpoints
- Applications with database persistence

### ⚙️ [Worker Service](./worker/)
Background worker implementation for job queue processing with concurrent execution.

**Features:**
- Database-backed job queue with retry logic
- Concurrent job processing
- External API integration
- Configurable batch processing
- Job status tracking and monitoring

**Use Cases:**
- Background job processing
- Data pipeline workers
- Email/notification services
- Batch processing applications

### 💻 [CLI Application](./cli-app/)
Command-line application with database operations and management commands.

**Features:**
- Full-featured CLI with subcommands
- Database migrations and seeding
- User management operations
- Data export/import functionality
- Comprehensive help system

**Use Cases:**
- Database management tools
- Admin utilities
- Data migration scripts
- System administration tools

## Quick Start

1. **Choose a Template**
   ```bash
   # Copy the template you need
   cp -r templates/web-service my-new-project
   cd my-new-project
   ```

2. **Customize Module Name**
   ```bash
   # Update go.mod with your module name
   go mod edit -module github.com/yourusername/my-new-project
   ```

3. **Install Dependencies**
   ```bash
   go mod tidy
   ```

4. **Configure Application**
   ```bash
   # Edit config.yaml with your settings
   vim config.yaml
   ```

5. **Run Application**
   ```bash
   go run main.go
   ```

## Template Features

All templates follow the same architectural patterns and best practices:

### 🏗️ **Proper Initialization Order**
1. Logging initialization (always first)
2. Configuration loading
3. Database setup
4. Service initialization
5. Application start

### 📋 **Configuration Management**
- YAML-based configuration files
- Environment variable overrides
- Validation with struct tags
- Environment-specific configs

### 🗄️ **Database Integration**
- GORM with connection pooling
- Automatic migrations
- Proper error handling
- Transaction support

### 📊 **Structured Logging**
- Context-aware logging with zerolog
- Consistent log formatting
- Debug and production modes
- Request/operation tracing

### 🔧 **Error Handling**
- Structured error wrapping
- Proper error logging
- Graceful failure handling
- User-friendly error messages

### 🐳 **Production Ready**
- Docker support
- Health checks
- Graceful shutdown
- Signal handling

## Integration with Claude Code

These templates are specifically designed to work seamlessly with Claude Code:

### 📖 **Comprehensive Documentation**
- Detailed README files for each template
- Code comments explaining patterns
- Usage examples and best practices
- Troubleshooting guides

### 🎯 **Claude Code Patterns**
- Follows patterns from [.claude/patterns.md](../.claude/patterns.md)
- Implements guidance from [.claude/integration-guide.md](../.claude/integration-guide.md)
- Uses consistent code organization
- Includes testing examples

### 🚀 **Easy Customization**
- Clear separation of concerns
- Modular architecture
- Extensible design patterns
- Well-documented extension points

## Template Comparison

| Feature | Web Service | Worker Service | CLI Application |
|---------|-------------|----------------|-----------------|
| HTTP Server | ✅ Echo | ❌ | ❌ |
| Database | ✅ PostgreSQL | ✅ PostgreSQL | ✅ PostgreSQL |
| Background Jobs | ❌ | ✅ Queue-based | ❌ |
| REST API | ✅ Full CRUD | ❌ | ❌ |
| Command Line | ❌ | ❌ | ✅ Full CLI |
| External APIs | ✅ REST client | ✅ REST client | ✅ REST client |
| Concurrent Processing | ❌ | ✅ Batch jobs | ❌ |
| Docker Support | ✅ Multi-stage | ✅ | ✅ |
| Health Checks | ✅ HTTP endpoint | ❌ | ❌ |
| Graceful Shutdown | ✅ | ✅ | ❌ |

## Environment Configuration

All templates support environment-specific configuration:

### Development
```yaml
environment: development
debug: true
database:
  host: localhost
  port: 5432
```

### Production
```yaml
environment: production
debug: false
database:
  host: ${DB_HOST}
  password: ${DB_PASSWORD}
```

### Environment Variables
```bash
# Web Service
export APP_DATABASE_HOST=prod-db.example.com
export APP_SERVER_PORT=3000

# Worker Service  
export WORKER_DATABASE_HOST=prod-db.example.com
export WORKER_WORKER_BATCHSIZE=20

# CLI Application
export CLI_DATABASE_HOST=prod-db.example.com
export CLI_DEBUG=false
```

## Development Workflow

### 1. Local Development
```bash
# Start dependencies
docker-compose up -d postgres redis

# Run application
go run main.go

# Run tests
go test ./...
```

### 2. Production Build
```bash
# Build optimized binary
CGO_ENABLED=0 go build -ldflags="-w -s" -o app

# Build Docker image
docker build -t my-app .
```

### 3. Deployment
```bash
# Docker deployment
docker run -d --name my-app \
  -e APP_DATABASE_HOST=prod-db \
  -e APP_DATABASE_PASSWORD=secret \
  -p 8080:8080 \
  my-app
```

## Testing Strategy

All templates include comprehensive testing approaches:

### Unit Tests
- Individual function testing
- Mock external dependencies
- Database model validation
- Configuration parsing tests

### Integration Tests
- Database connectivity
- External API integration
- End-to-end workflows
- Error handling scenarios

### Example Test Structure
```
my-project/
├── handlers_test.go     # HTTP handler tests
├── services_test.go     # Business logic tests
├── models_test.go       # Database model tests
└── integration_test.go  # End-to-end tests
```

## Extension Patterns

### Adding New Features

1. **New HTTP Endpoints** (Web Service)
   ```go
   func setupProductRoutes(g *echo.Group, services *Services) {
       products := g.Group("/products")
       handler := NewProductHandler(services)
       products.GET("", handler.ListProducts)
       products.POST("", handler.CreateProducts)
   }
   ```

2. **New Job Types** (Worker Service)
   ```go
   func processCustomJob(ctx context.Context, services *Services, job *Job) error {
       // Custom job processing logic
       return nil
   }
   ```

3. **New Commands** (CLI Application)
   ```go
   func customCommand(ctx context.Context, services *Services) error {
       // Custom command logic
       return nil
   }
   ```

### Adding External Integrations
```go
// Add to Services struct
type Services struct {
    DB           *gorm.DB
    APIClient    *rest.Client
    CacheClient  *redis.Client  // New integration
    Logger       zerolog.Logger
}
```

## Best Practices

### 🎯 **Configuration**
- Use environment variables for secrets
- Validate configuration on startup
- Provide sensible defaults
- Document all configuration options

### 🔒 **Security**
- Never log sensitive information
- Use proper authentication/authorization
- Validate all inputs
- Use HTTPS in production

### 📈 **Performance**
- Configure appropriate connection pools
- Use proper indexing for database queries
- Implement caching where appropriate
- Monitor resource usage

### 🔍 **Observability**
- Structure all log messages
- Include request/operation IDs
- Monitor key metrics
- Implement health checks

## Troubleshooting

### Common Issues

**Module resolution errors**
```bash
# Ensure proper module name
go mod edit -module github.com/yourusername/project-name
go mod tidy
```

**Database connection failures**
```bash
# Check database connectivity
docker ps | grep postgres
telnet localhost 5432
```

**Configuration not loading**
```bash
# Enable debug logging
DEBUG=true go run main.go
```

### Getting Help

1. **Template Issues**: Check individual template README files
2. **Library Issues**: https://github.com/jasoet/pkg/issues
3. **Integration Help**: See [.claude/integration-guide.md](../.claude/integration-guide.md)
4. **Pattern Reference**: See [.claude/patterns.md](../.claude/patterns.md)

## Contributing

When adding new templates:

1. Follow the established patterns from existing templates
2. Include comprehensive README documentation
3. Add configuration examples
4. Include Docker support
5. Provide testing examples
6. Update this main README

---

These templates provide a solid foundation for building production-ready applications with the github.com/jasoet/pkg utility library. Each template is designed to be a starting point that can be customized and extended based on your specific requirements.