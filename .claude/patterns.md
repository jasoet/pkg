# Common Patterns and Anti-Patterns for github.com/jasoet/pkg

This file helps Claude Code recognize and suggest appropriate usage patterns when integrating this utility library.

## Initialization Patterns

### ✅ Correct Initialization Order
```go
func main() {
    // 1. Logging first (always)
    logging.Initialize("service-name", isDebugMode())
    
    // 2. Context setup
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    // 3. Configuration loading
    config, err := loadConfiguration()
    if err != nil {
        logger.Fatal().Err(err).Msg("Configuration failed")
    }
    
    // 4. Infrastructure setup (database, clients)
    db, err := setupDatabase(ctx, config.Database)
    if err != nil {
        logger.Fatal().Err(err).Msg("Database setup failed")
    }
    
    // 5. Application services
    services := setupServices(ctx, db, config)
    
    // 6. Server/application start
    startApplication(ctx, services, config)
}
```

### ❌ Anti-Pattern: Wrong Initialization Order
```go
func main() {
    // WRONG: Setting up services before logging
    db := setupDatabase()
    logging.Initialize("service", true)  // Too late!
}
```

## Configuration Patterns

### ✅ Correct Configuration Structure
```go
type AppConfig struct {
    Environment string                `yaml:"environment" mapstructure:"environment" validate:"required,oneof=development staging production"`
    Debug       bool                  `yaml:"debug" mapstructure:"debug"`
    Server      server.Config         `yaml:"server" mapstructure:"server" validate:"required"`
    Database    db.ConnectionConfig   `yaml:"database" mapstructure:"database" validate:"required"`
    Redis       RedisConfig           `yaml:"redis" mapstructure:"redis"`
    ExternalAPI ExternalAPIConfig     `yaml:"externalApi" mapstructure:"externalApi"`
}

type ExternalAPIConfig struct {
    BaseURL     string        `yaml:"baseUrl" mapstructure:"baseUrl" validate:"required,url"`
    Timeout     time.Duration `yaml:"timeout" mapstructure:"timeout" validate:"min=1s"`
    RetryCount  int           `yaml:"retryCount" mapstructure:"retryCount" validate:"min=0,max=10"`
    APIKey      string        `yaml:"apiKey" mapstructure:"apiKey" validate:"required"`
}
```

### ✅ Environment-Specific Configuration Loading
```go
func loadConfiguration() (*AppConfig, error) {
    env := getEnvironment() // development, staging, production
    
    // Load base configuration
    configFile := fmt.Sprintf("configs/%s.yaml", env)
    configData, err := os.ReadFile(configFile)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file %s: %w", configFile, err)
    }
    
    // Load with environment variable overrides
    config, err := config.LoadString[AppConfig](string(configData), "APP")
    if err != nil {
        return nil, fmt.Errorf("failed to parse configuration: %w", err)
    }
    
    // Validate configuration
    if err := validate.Struct(config); err != nil {
        return nil, fmt.Errorf("configuration validation failed: %w", err)
    }
    
    return config, nil
}
```

### ❌ Anti-Pattern: Hardcoded Configuration
```go
// WRONG: Hardcoded values without environment support
func setupDatabase() *gorm.DB {
    config := &db.ConnectionConfig{
        Host:     "localhost",  // Should be configurable
        Port:     5432,         // Should be configurable
        Username: "postgres",   // Should be from env
        Password: "password",   // Should be from env/secrets
    }
    db, _ := config.Pool()
    return db
}
```

## Dependency Injection Patterns

### ✅ Correct Service Container Pattern
```go
type Services struct {
    DB        *gorm.DB
    APIClient *rest.Client
    Logger    zerolog.Logger
    Config    *AppConfig
}

func NewServices(ctx context.Context, config *AppConfig) (*Services, error) {
    logger := logging.ContextLogger(ctx, "services")
    
    // Database
    database, err := config.Database.Pool()
    if err != nil {
        return nil, fmt.Errorf("database setup failed: %w", err)
    }
    
    // HTTP Client
    restConfig := &rest.Config{
        Timeout:     config.ExternalAPI.Timeout,
        RetryCount:  config.ExternalAPI.RetryCount,
    }
    apiClient := rest.NewClient(rest.WithRestConfig(*restConfig))
    
    return &Services{
        DB:        database,
        APIClient: apiClient,
        Logger:    logger,
        Config:    config,
    }, nil
}

// Business logic services
type UserService struct {
    services *Services
}

func NewUserService(services *Services) *UserService {
    return &UserService{services: services}
}

func (s *UserService) CreateUser(ctx context.Context, userData UserData) (*User, error) {
    logger := logging.ContextLogger(ctx, "user-service")
    
    // Use s.services.DB for database operations
    // Use s.services.APIClient for external calls
    // Use logger for logging
}
```

### ❌ Anti-Pattern: Global Variables
```go
// WRONG: Global variables make testing difficult
var globalDB *gorm.DB
var globalAPIClient *rest.Client

func init() {
    globalDB, _ = setupDatabase()
    globalAPIClient = rest.NewClient()
}

func CreateUser(userData UserData) (*User, error) {
    // Hard to test, hard to configure
    return globalDB.Create(&userData)
}
```

## Error Handling Patterns

### ✅ Correct Error Handling with Context
```go
func (s *UserService) ProcessUser(ctx context.Context, userID int) error {
    logger := logging.ContextLogger(ctx, "user-service")
    
    // Database operation with error context
    var user User
    err := s.db.WithContext(ctx).First(&user, userID).Error
    if err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            logger.Warn().Int("user_id", userID).Msg("User not found")
            return NewUserNotFoundError(userID)
        }
        logger.Error().Err(err).Int("user_id", userID).Msg("Database query failed")
        return fmt.Errorf("failed to fetch user %d: %w", userID, err)
    }
    
    // External API call with error context
    response, err := s.apiClient.MakeRequest(ctx, "POST", "/process", userData, headers)
    if err != nil {
        // Handle different error types
        switch e := err.(type) {
        case *rest.UnauthorizedError:
            logger.Error().Int("status", e.StatusCode).Msg("API authentication failed")
            return fmt.Errorf("authentication failed: %w", err)
        case *rest.ServerError:
            logger.Error().Int("status", e.StatusCode).Msg("API server error")
            return fmt.Errorf("API server error: %w", err)
        default:
            logger.Error().Err(err).Msg("API call failed")
            return fmt.Errorf("external API call failed: %w", err)
        }
    }
    
    logger.Info().Int("user_id", userID).Msg("User processed successfully")
    return nil
}
```

### ❌ Anti-Pattern: Swallowing Errors
```go
// WRONG: Swallowing errors without logging
func ProcessUser(userID int) {
    user, err := db.First(&user, userID)
    if err != nil {
        return  // Error lost!
    }
    
    response, err := apiClient.MakeRequest("POST", "/process", userData, nil)
    if err != nil {
        log.Println("API failed")  // Not enough context
        return
    }
}
```

## Concurrent Processing Patterns

### ✅ Correct Concurrent Processing
```go
func (s *UserService) ProcessUsersBatch(ctx context.Context, userIDs []int) error {
    logger := logging.ContextLogger(ctx, "user-batch-processor")
    
    logger.Info().Int("user_count", len(userIDs)).Msg("Starting batch processing")
    
    // Create processing functions
    processingFuncs := make(map[string]concurrent.Func[ProcessResult])
    for _, userID := range userIDs {
        key := fmt.Sprintf("user_%d", userID)
        processingFuncs[key] = s.createUserProcessor(userID)
    }
    
    // Execute concurrently with timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()
    
    results, err := concurrent.ExecuteConcurrently(ctx, processingFuncs)
    if err != nil {
        logger.Error().Err(err).Msg("Batch processing failed")
        return fmt.Errorf("batch processing failed: %w", err)
    }
    
    // Process results
    successCount := 0
    for key, result := range results {
        if result.Success {
            successCount++
        } else {
            logger.Warn().Str("key", key).Str("error", result.Error).Msg("User processing failed")
        }
    }
    
    logger.Info().
        Int("total", len(userIDs)).
        Int("successful", successCount).
        Int("failed", len(userIDs)-successCount).
        Msg("Batch processing completed")
    
    return nil
}

func (s *UserService) createUserProcessor(userID int) concurrent.Func[ProcessResult] {
    return func(ctx context.Context) (ProcessResult, error) {
        logger := logging.ContextLogger(ctx, "user-processor")
        
        // Individual user processing logic
        err := s.ProcessUser(ctx, userID)
        if err != nil {
            logger.Error().Err(err).Int("user_id", userID).Msg("User processing failed")
            return ProcessResult{Success: false, Error: err.Error()}, nil
        }
        
        return ProcessResult{Success: true}, nil
    }
}
```

### ❌ Anti-Pattern: Uncontrolled Goroutines
```go
// WRONG: Uncontrolled goroutines without proper error handling
func ProcessUsers(userIDs []int) {
    for _, userID := range userIDs {
        go func(id int) {
            ProcessUser(id)  // No error handling, no context
        }(userID)
    }
    // No way to know when processing is complete
}
```

## HTTP Server Patterns

### ✅ Correct Server Setup with Middleware
```go
func setupServer(services *Services) *server.Server {
    config := &server.Config{
        Port:                services.Config.Server.Port,
        ReadTimeout:         30 * time.Second,
        WriteTimeout:        30 * time.Second,
        EnableHealthChecks:  true,
        EnableMetrics:       true,
        EchoConfigurer:      setupRoutes(services),
    }
    
    return server.New(config)
}

func setupRoutes(services *Services) func(*echo.Echo) {
    return func(e *echo.Echo) {
        // Add custom middleware
        e.Use(middleware.RequestID())
        e.Use(authMiddleware(services))
        
        // API routes
        api := e.Group("/api/v1")
        api.Use(rateLimitMiddleware())
        
        // User endpoints
        userHandler := NewUserHandler(services)
        users := api.Group("/users")
        users.GET("", userHandler.ListUsers)
        users.POST("", userHandler.CreateUser)
        users.GET("/:id", userHandler.GetUser)
        users.PUT("/:id", userHandler.UpdateUser)
        users.DELETE("/:id", userHandler.DeleteUser)
        
        // Health check with custom checks
        api.GET("/health", healthHandler(services))
    }
}

func authMiddleware(services *Services) echo.MiddlewareFunc {
    return func(next echo.HandlerFunc) echo.HandlerFunc {
        return func(c echo.Context) error {
            logger := logging.ContextLogger(c.Request().Context(), "auth-middleware")
            
            token := c.Request().Header.Get("Authorization")
            if token == "" {
                logger.Warn().Str("path", c.Request().URL.Path).Msg("Missing authorization header")
                return echo.NewHTTPError(http.StatusUnauthorized, "Authorization required")
            }
            
            // Validate token logic here
            userInfo, err := validateToken(token)
            if err != nil {
                logger.Error().Err(err).Msg("Token validation failed")
                return echo.NewHTTPError(http.StatusUnauthorized, "Invalid token")
            }
            
            // Store user info in context
            c.Set("user", userInfo)
            return next(c)
        }
    }
}
```

### ❌ Anti-Pattern: Minimal Server Setup
```go
// WRONG: No middleware, no error handling, no observability
func main() {
    e := echo.New()
    e.GET("/users", func(c echo.Context) error {
        // Direct database access, no error handling
        users := getUsers()
        return c.JSON(200, users)
    })
    e.Start(":8080")  // No graceful shutdown
}
```

## Testing Patterns

### ✅ Correct Integration Testing
```go
func TestUserService_Integration(t *testing.T) {
    // Setup test environment
    ctx := context.Background()
    
    // Test database configuration
    testDBConfig := &db.ConnectionConfig{
        DbType:   db.Postgresql,
        Host:     "localhost",
        Port:     5432,
        Username: "test",
        Password: "test",
        DbName:   "test_db",
    }
    
    testDB, err := testDBConfig.Pool()
    require.NoError(t, err)
    defer testDB.Close()
    
    // Run migrations for test
    err = runTestMigrations(ctx, testDB)
    require.NoError(t, err)
    
    // Setup test services
    testServices := &Services{
        DB:        testDB,
        APIClient: createMockAPIClient(),
        Logger:    logging.ContextLogger(ctx, "test"),
    }
    
    userService := NewUserService(testServices)
    
    // Test cases
    t.Run("CreateUser", func(t *testing.T) {
        userData := UserData{
            Name:  "Test User",
            Email: "test@example.com",
        }
        
        user, err := userService.CreateUser(ctx, userData)
        assert.NoError(t, err)
        assert.NotNil(t, user)
        assert.Equal(t, userData.Name, user.Name)
        assert.NotZero(t, user.ID)
    })
    
    t.Run("GetUser", func(t *testing.T) {
        // Create test user first
        testUser := createTestUser(t, testDB)
        
        user, err := userService.GetUser(ctx, testUser.ID)
        assert.NoError(t, err)
        assert.Equal(t, testUser.ID, user.ID)
    })
}

func createMockAPIClient() *rest.Client {
    // Create mock server for testing
    mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status": "success"}`))
    }))
    
    config := &rest.Config{
        Timeout:    5 * time.Second,
        RetryCount: 1,
    }
    
    client := rest.NewClient(rest.WithRestConfig(*config))
    // Configure client to use mock server
    return client
}
```

### ❌ Anti-Pattern: No Integration Testing
```go
// WRONG: Only unit tests without integration testing
func TestUserService(t *testing.T) {
    // No real database, no real HTTP client
    service := &UserService{}
    
    // Can't test real integration
    result := service.SomeMethod()
    assert.NotNil(t, result)
}
```

## Deployment Patterns

### ✅ Correct Production Configuration
```go
// Production-ready main function
func main() {
    // Initialize logging with environment-based debug mode
    debug := os.Getenv("DEBUG") == "true"
    logging.Initialize("my-service", debug)
    
    ctx := context.Background()
    logger := logging.ContextLogger(ctx, "main")
    
    // Load configuration from environment
    config, err := loadConfiguration()
    if err != nil {
        logger.Fatal().Err(err).Msg("Configuration loading failed")
    }
    
    // Setup services with proper error handling
    services, err := NewServices(ctx, config)
    if err != nil {
        logger.Fatal().Err(err).Msg("Service initialization failed")
    }
    
    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(ctx)
    defer cancel()
    
    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    
    go func() {
        sig := <-sigChan
        logger.Info().Str("signal", sig.String()).Msg("Received shutdown signal")
        cancel()
    }()
    
    // Start server
    srv := setupServer(services)
    
    logger.Info().
        Str("environment", config.Environment).
        Int("port", config.Server.Port).
        Msg("Starting server")
    
    if err := srv.Start(ctx); err != nil {
        logger.Error().Err(err).Msg("Server failed to start")
    }
    
    logger.Info().Msg("Server shutdown completed")
}
```

This pattern guide helps Claude Code suggest appropriate patterns and avoid anti-patterns when integrating this utility library.