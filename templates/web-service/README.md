# Web Service Template

This template provides a complete web service implementation using [github.com/jasoet/pkg](https://github.com/jasoet/pkg).

## Features

- **Structured Configuration**: YAML configuration with environment variable overrides
- **Database Integration**: PostgreSQL with GORM and connection pooling  
- **HTTP Server**: Echo-based server with middleware and observability
- **Structured Logging**: Zerolog with context-aware logging
- **Health Checks**: Built-in health endpoints with database connectivity checks
- **Graceful Shutdown**: Proper signal handling and resource cleanup
- **Docker Support**: Multi-stage Docker build and compose setup
- **Development Ready**: Hot-reload friendly configuration

## Quick Start

1. **Copy Template**
   ```bash
   cp -r templates/web-service my-web-service
   cd my-web-service
   ```

2. **Update Module Name**
   ```bash
   # Edit go.mod and replace 'myapp' with your module name
   go mod edit -module github.com/yourusername/my-web-service
   ```

3. **Install Dependencies**
   ```bash
   go mod tidy
   ```

4. **Start Development Services**
   ```bash
   docker-compose up -d postgres redis
   ```

5. **Run Application**
   ```bash
   go run main.go
   ```

6. **Test Health Endpoint**
   ```bash
   curl http://localhost:8080/api/v1/health
   ```

## Configuration

The application uses `config.yaml` for base configuration with environment variable overrides:

```yaml
environment: development
debug: true
server:
  port: 8080
database:
  dbType: POSTGRES
  host: localhost
  port: 5432
  username: postgres
  password: password
  dbName: myapp
```

### Environment Variable Overrides

Use the `APP_` prefix for environment variables:

```bash
export APP_DATABASE_HOST=production-db.example.com
export APP_DATABASE_PASSWORD=secure-password
export APP_SERVER_PORT=3000
```

## API Endpoints

### Health Check
- `GET /api/v1/health` - Application health status

### User Management (Example)
- `GET /api/v1/users` - List users
- `POST /api/v1/users` - Create user
- `GET /api/v1/users/:id` - Get user by ID
- `PUT /api/v1/users/:id` - Update user
- `DELETE /api/v1/users/:id` - Delete user

### Product Management (Example)
- `GET /api/v1/products` - List products
- `POST /api/v1/products` - Create product
- `GET /api/v1/products/:id` - Get product by ID
- `PUT /api/v1/products/:id` - Update product  
- `DELETE /api/v1/products/:id` - Delete product

### Admin Endpoints
- `GET /api/v1/admin/stats` - System statistics (requires authentication)

## Development

### Running Tests
```bash
go test ./...
```

### Database Migrations
```bash
# Add your migration files to migrations/ directory
# They will be automatically applied on startup
```

### Adding New Routes
```bash
# Add route handlers in main.go or separate handler files
# Follow the existing pattern for consistent structure
```

## Production Deployment

### Docker Build
```bash
docker build -t my-web-service .
docker run -p 8080:8080 my-web-service
```

### Docker Compose
```bash
docker-compose up -d
```

### Environment Configuration
Create production configuration files:
- `config.production.yaml` for production settings
- Use environment variables for secrets

## Architecture

This template follows the patterns from github.com/jasoet/pkg:

1. **Initialization Order**: Logging → Configuration → Database → Services → Server
2. **Dependency Injection**: Services container pattern for clean dependencies
3. **Context Propagation**: Proper context handling throughout the application
4. **Error Handling**: Structured error handling with logging
5. **Observability**: Health checks, metrics, and structured logging

## Customization

### Adding Database Models
```go
type User struct {
    ID       uint      `json:"id" gorm:"primaryKey"`
    Name     string    `json:"name" gorm:"not null"`
    Email    string    `json:"email" gorm:"unique;not null"`
    Created  time.Time `json:"created" gorm:"autoCreateTime"`
    Updated  time.Time `json:"updated" gorm:"autoUpdateTime"`
}
```

### Adding External API Integration
```go
import "github.com/jasoet/pkg/rest"

// In your service configuration
apiClient := rest.NewClient(rest.WithRestConfig(rest.Config{
    Timeout:    30 * time.Second,
    RetryCount: 3,
}))
```

### Adding Background Processing
```go
import "github.com/jasoet/pkg/concurrent"

// Process items concurrently
results, err := concurrent.ExecuteConcurrently(ctx, processingFunctions)
```

## Support

For issues with this template or the underlying library:
- Template issues: Create issue in your project repository
- Library issues: https://github.com/jasoet/pkg/issues
- Documentation: Check the [integration guide](../../.claude/integration-guide.md)