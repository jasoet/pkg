# Integration Examples

This directory contains complete, working examples that demonstrate how to integrate [github.com/jasoet/pkg](https://github.com/jasoet/pkg) into real-world applications.

Unlike the templates (which are starting points), these examples are fully functional applications that you can run and study.

## Available Examples

### 🌐 [E-commerce API](./ecommerce-api/)
A complete e-commerce REST API with product catalog, user management, and order processing.

**Demonstrates:**
- Complex database schemas with relationships
- JWT authentication and authorization
- File upload handling
- Advanced error handling and validation
- Integration with external payment services
- Background job processing for order fulfillment

### 📊 [Analytics Dashboard](./analytics-dashboard/)
A real-time analytics dashboard with data ingestion, processing, and visualization.

**Demonstrates:**
- Time-series data processing
- Real-time WebSocket connections
- Data aggregation and reporting
- Scheduled batch processing
- External API integrations
- Performance optimization techniques

### 🔄 [Data Pipeline](./data-pipeline/)
A scalable data pipeline for processing large datasets with multiple data sources.

**Demonstrates:**
- ETL (Extract, Transform, Load) operations
- Concurrent data processing
- Multiple database integrations
- File processing and validation
- Error recovery and retry logic
- Monitoring and observability

## Running Examples

Each example includes:

1. **Complete Documentation**: README with setup instructions
2. **Docker Compose**: One-command environment setup
3. **Sample Data**: Pre-populated test data
4. **API Documentation**: OpenAPI/Swagger specifications
5. **Tests**: Unit and integration test suites
6. **Monitoring**: Health checks and metrics

### Prerequisites

- Go 1.23+
- Docker and Docker Compose
- Make (optional, for convenience scripts)

### Quick Start

```bash
# Clone the repository
git clone https://github.com/jasoet/pkg.git
cd pkg/integration-examples

# Choose an example and run it
cd ecommerce-api
docker-compose up -d    # Start dependencies
go mod download        # Install Go dependencies
go run main.go         # Start the application

# Or use the convenience script
make run
```

## What Makes These Examples Special

### 🏭 **Production-Ready Patterns**
- Proper error handling and recovery
- Graceful shutdown and signal handling
- Health checks and monitoring
- Security best practices
- Performance optimization

### 📚 **Educational Value**
- Extensive code comments explaining decisions
- Multiple implementation approaches shown
- Common pitfalls and how to avoid them
- Progressive complexity (simple → advanced)

### 🔧 **Real-World Scenarios**
- Authentication and authorization
- File uploads and processing
- External service integrations
- Background job processing
- Data validation and sanitization

### 🧪 **Testing Excellence**
- Unit tests with proper mocking
- Integration tests with test databases
- End-to-end API testing
- Performance and load testing

## Learning Path

### 1. **Start with E-commerce API** (Beginner)
Learn the fundamentals of web service development with this library.

**Key Learning Points:**
- Basic CRUD operations
- Database modeling with GORM
- HTTP routing with Echo
- Configuration management
- Structured logging

### 2. **Progress to Analytics Dashboard** (Intermediate)
Understand real-time data processing and advanced patterns.

**Key Learning Points:**
- WebSocket handling
- Concurrent data processing
- Time-series data management
- Caching strategies
- Performance monitoring

### 3. **Master Data Pipeline** (Advanced)
Learn complex data processing and system integration.

**Key Learning Points:**
- ETL pipeline design
- Multiple data source integration
- Error recovery patterns
- System observability
- Scalability considerations

## Integration Patterns Demonstrated

### Configuration Management
```go
// Environment-specific configuration loading
type AppConfig struct {
    Environment string               `yaml:"environment"`
    Server      server.Config        `yaml:"server"`
    Database    db.ConnectionConfig  `yaml:"database"`
    Redis       RedisConfig          `yaml:"redis"`
    ExternalAPI ExternalAPIConfig    `yaml:"externalApi"`
}

// Load with environment overrides
config, err := config.LoadString[AppConfig](configYAML, "APP")
```

### Service Architecture
```go
// Dependency injection pattern
type Services struct {
    DB           *gorm.DB
    Cache        *redis.Client
    APIClient    *rest.Client
    Logger       zerolog.Logger
    Config       *AppConfig
}

// Service initialization
func NewServices(ctx context.Context, cfg *AppConfig) (*Services, error) {
    // Initialize all services with proper error handling
}
```

### Background Processing
```go
// Job queue pattern with concurrent processing
type JobProcessor struct {
    services *Services
}

func (jp *JobProcessor) ProcessBatch(ctx context.Context) error {
    jobs := fetchPendingJobs(jp.services.DB)
    
    // Process concurrently
    results, err := concurrent.ExecuteConcurrently(ctx, jobFunctions)
    
    // Handle results and update job statuses
    return jp.handleResults(results)
}
```

### Error Handling
```go
// Structured error handling with context
func (s *UserService) CreateUser(ctx context.Context, userData CreateUserRequest) (*User, error) {
    logger := logging.ContextLogger(ctx, "user-service")
    
    // Validation
    if err := s.validateUserData(userData); err != nil {
        logger.Warn().Err(err).Interface("data", userData).Msg("Invalid user data")
        return nil, NewValidationError("user data validation failed", err)
    }
    
    // Database operation
    user := &User{...}
    if err := s.db.Create(user).Error; err != nil {
        logger.Error().Err(err).Msg("Failed to create user")
        return nil, fmt.Errorf("user creation failed: %w", err)
    }
    
    logger.Info().Uint("user_id", user.ID).Msg("User created successfully")
    return user, nil
}
```

## Testing Strategies

### Unit Testing
```go
func TestUserService_CreateUser(t *testing.T) {
    // Setup test dependencies
    testDB := setupTestDB(t)
    userService := NewUserService(testDB)
    
    // Test user creation
    user, err := userService.CreateUser(ctx, validUserData)
    
    // Assertions
    assert.NoError(t, err)
    assert.NotNil(t, user)
    assert.NotZero(t, user.ID)
}
```

### Integration Testing
```go
func TestAPI_CreateUser_Integration(t *testing.T) {
    // Setup test server
    app := setupTestApp(t)
    defer app.Cleanup()
    
    // Make HTTP request
    resp, err := http.Post(app.URL+"/api/users", "application/json", userDataJSON)
    
    // Verify response and database state
    assert.Equal(t, http.StatusCreated, resp.StatusCode)
    
    var user User
    err = app.DB.First(&user, "email = ?", testEmail).Error
    assert.NoError(t, err)
}
```

## Performance Considerations

### Database Optimization
- Connection pooling configuration
- Query optimization and indexing
- Transaction management
- Connection lifecycle

### HTTP Performance
- Request/response compression
- Connection keep-alive
- Request timeout handling
- Response caching

### Concurrent Processing
- Worker pool sizing
- Resource contention management
- Context-based cancellation
- Error isolation

## Security Best Practices

### Input Validation
- Request payload validation
- SQL injection prevention
- XSS protection
- File upload security

### Authentication & Authorization
- JWT token management
- Role-based access control
- API key validation
- Session management

### Data Protection
- Sensitive data masking in logs
- Encryption at rest and in transit
- Secure configuration management
- Audit logging

## Monitoring and Observability

### Structured Logging
```go
logger.Info().
    Str("operation", "user_creation").
    Str("user_id", userID).
    Dur("duration", time.Since(start)).
    Msg("User creation completed")
```

### Metrics Collection
```go
// Custom metrics
userCreationCounter.Inc()
requestDurationHistogram.Observe(duration.Seconds())
```

### Health Checks
```go
func healthHandler(services *Services) echo.HandlerFunc {
    return func(c echo.Context) error {
        // Check all critical dependencies
        if err := checkDatabase(services.DB); err != nil {
            return c.JSON(503, HealthResponse{Status: "unhealthy"})
        }
        
        return c.JSON(200, HealthResponse{Status: "healthy"})
    }
}
```

## Contributing New Examples

When adding new integration examples:

1. **Choose Real-World Scenarios**: Pick practical use cases that developers commonly face
2. **Follow Established Patterns**: Use the same architectural patterns as existing examples
3. **Include Comprehensive Tests**: Unit, integration, and end-to-end tests
4. **Document Thoroughly**: Explain design decisions and trade-offs
5. **Provide Sample Data**: Include realistic test data and scenarios
6. **Add Monitoring**: Include health checks, metrics, and logging

## Support and Feedback

- **Example Issues**: Create issues in this repository for example-specific problems
- **Library Issues**: Report library bugs at https://github.com/jasoet/pkg/issues
- **Feature Requests**: Suggest new integration examples via GitHub issues
- **Documentation**: Check [.claude/integration-guide.md](../.claude/integration-guide.md) for additional guidance

These integration examples provide practical, production-ready demonstrations of how to build robust applications using the github.com/jasoet/pkg utility library.