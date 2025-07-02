# REST Package Examples

This directory contains examples demonstrating how to use the `rest` package for HTTP client operations in Go applications.

## Overview

The `rest` package provides utilities for:
- HTTP client creation with retry and timeout configuration
- Middleware support for request/response interception
- Structured error handling with custom error types
- Built-in logging and tracing capabilities
- Context-aware request handling

## Running the Examples

To run the examples, use the following command from the `rest/examples` directory:

```bash
go run example.go
```

**Note**: Some examples make actual HTTP requests to public APIs. Ensure you have internet connectivity for full demonstration.

## Example Descriptions

The example.go file demonstrates several use cases:

### 1. Basic HTTP Client

Create a simple HTTP client with default configuration:

```go
// Create client with default configuration
client := rest.NewClient()

// Make a GET request
response, err := client.MakeRequest(ctx, "GET", "https://api.example.com/users", "", nil)
if err != nil {
    log.Fatal(err)
}
```

### 2. Client with Custom Configuration

Configure retry behavior, timeouts, and other settings:

```go
config := &rest.Config{
    RetryCount:       3,
    RetryWaitTime:    1 * time.Second,
    RetryMaxWaitTime: 5 * time.Second,
    Timeout:          30 * time.Second,
}

client := rest.NewClient(rest.WithRestConfig(*config))
```

### 3. Middleware Integration

Add custom middleware for logging, authentication, or other cross-cutting concerns:

```go
// Built-in logging middleware
loggingMiddleware := rest.NewLoggingMiddleware()

// Custom authentication middleware
authMiddleware := &AuthMiddleware{token: "your-api-token"}

client := rest.NewClient(
    rest.WithMiddlewares(loggingMiddleware, authMiddleware),
)
```

### 4. Error Handling

Handle different types of HTTP errors:

```go
response, err := client.MakeRequest(ctx, "GET", url, "", headers)
if err != nil {
    switch e := err.(type) {
    case *rest.UnauthorizedError:
        log.Printf("Authentication failed: %s", e.Error())
    case *rest.ServerError:
        log.Printf("Server error: %s", e.Error())
    case *rest.ResponseError:
        log.Printf("Response error: %s", e.Error())
    default:
        log.Printf("Request failed: %s", err.Error())
    }
}
```

### 5. JSON API Interactions

Work with JSON APIs using built-in JSON support:

```go
// GET request with JSON response
response, err := client.MakeRequest(ctx, "GET", "https://api.example.com/users", "", nil)
if err == nil {
    var users []User
    json.Unmarshal(response.Body(), &users)
}

// POST request with JSON body
userData := User{Name: "John Doe", Email: "john@example.com"}
jsonBody, _ := json.Marshal(userData)
headers := map[string]string{"Content-Type": "application/json"}
response, err := client.MakeRequest(ctx, "POST", "https://api.example.com/users", string(jsonBody), headers)
```

### 6. Retry and Circuit Breaker Patterns

Handle transient failures with retry logic:

```go
config := &rest.Config{
    RetryCount:       5,
    RetryWaitTime:    500 * time.Millisecond,
    RetryMaxWaitTime: 10 * time.Second,
    Timeout:          30 * time.Second,
}

client := rest.NewClient(rest.WithRestConfig(*config))

// Requests will automatically retry on transient failures
response, err := client.MakeRequest(ctx, "GET", unreliableAPI, "", nil)
```

### 7. Request Tracing and Performance Monitoring

Monitor request performance with built-in tracing:

```go
client := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))

// Request will be automatically traced and logged
response, err := client.MakeRequest(ctx, "GET", url, "", nil)

// Access trace information
if response != nil {
    traceInfo := response.Request.TraceInfo()
    fmt.Printf("DNS lookup: %v\n", traceInfo.DNSLookup)
    fmt.Printf("TCP connection: %v\n", traceInfo.TCPConnTime)
    fmt.Printf("TLS handshake: %v\n", traceInfo.TLSHandshake)
}
```

### 8. Advanced Resty Client Usage

Access the underlying Resty client for advanced features:

```go
client := rest.NewClient()
restyClient := client.GetRestyClient()

// Use Resty-specific features
response, err := restyClient.R().
    SetHeader("Authorization", "Bearer token").
    SetQueryParam("limit", "10").
    SetResult(&users). // Automatic JSON unmarshaling
    Get("https://api.example.com/users")
```

## Configuration Options

The `Config` struct supports the following options:

| Field | Type | Description | Default |
|-------|------|-------------|---------|
| `RetryCount` | int | Number of retry attempts | 1 |
| `RetryWaitTime` | time.Duration | Initial wait time between retries | 20s |
| `RetryMaxWaitTime` | time.Duration | Maximum wait time between retries | 30s |
| `Timeout` | time.Duration | Request timeout | 50s |

### Configuration Examples

**Development Configuration**:
```go
config := &rest.Config{
    RetryCount:       1,
    RetryWaitTime:    1 * time.Second,
    RetryMaxWaitTime: 5 * time.Second,
    Timeout:          10 * time.Second,
}
```

**Production Configuration**:
```go
config := &rest.Config{
    RetryCount:       3,
    RetryWaitTime:    500 * time.Millisecond,
    RetryMaxWaitTime: 30 * time.Second,
    Timeout:          60 * time.Second,
}
```

**High-Performance Configuration**:
```go
config := &rest.Config{
    RetryCount:       2,
    RetryWaitTime:    100 * time.Millisecond,
    RetryMaxWaitTime: 2 * time.Second,
    Timeout:          5 * time.Second,
}
```

## Middleware System

### Built-in Middleware

#### LoggingMiddleware
Automatically logs all requests and responses:

```go
client := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))
```

#### NoOpMiddleware
Does nothing - useful for testing:

```go
client := rest.NewClient(rest.WithMiddleware(rest.NewNoOpMiddleware()))
```

#### DatabaseLoggingMiddleware
Example middleware for database logging:

```go
client := rest.NewClient(rest.WithMiddleware(rest.NewDatabaseLoggingMiddleware()))
```

### Custom Middleware

Implement the `Middleware` interface:

```go
type CustomMiddleware struct {
    name string
}

func (m *CustomMiddleware) BeforeRequest(ctx context.Context, method, url, body string, headers map[string]string) context.Context {
    // Add custom headers, modify request, etc.
    headers["X-Custom-Header"] = "custom-value"
    return ctx
}

func (m *CustomMiddleware) AfterRequest(ctx context.Context, info rest.RequestInfo) {
    // Log metrics, update counters, etc.
    fmt.Printf("Request to %s took %v\n", info.URL, info.Duration)
}
```

### Middleware Examples

#### Authentication Middleware
```go
type AuthMiddleware struct {
    token string
}

func (m *AuthMiddleware) BeforeRequest(ctx context.Context, method, url, body string, headers map[string]string) context.Context {
    headers["Authorization"] = "Bearer " + m.token
    return ctx
}

func (m *AuthMiddleware) AfterRequest(ctx context.Context, info rest.RequestInfo) {
    if info.StatusCode == 401 {
        log.Println("Authentication failed, token may be expired")
    }
}
```

#### Rate Limiting Middleware
```go
type RateLimitMiddleware struct {
    limiter *rate.Limiter
}

func (m *RateLimitMiddleware) BeforeRequest(ctx context.Context, method, url, body string, headers map[string]string) context.Context {
    m.limiter.Wait(ctx) // Block until rate limit allows
    return ctx
}
```

#### Metrics Middleware
```go
type MetricsMiddleware struct {
    requestCounter  prometheus.Counter
    durationHistogram prometheus.Histogram
}

func (m *MetricsMiddleware) AfterRequest(ctx context.Context, info rest.RequestInfo) {
    m.requestCounter.Inc()
    m.durationHistogram.Observe(info.Duration.Seconds())
}
```

## Error Types and Handling

### Built-in Error Types

#### UnauthorizedError (401/403)
```go
if unauthorizedErr, ok := err.(*rest.UnauthorizedError); ok {
    fmt.Printf("Status: %d, Message: %s\n", unauthorizedErr.StatusCode, unauthorizedErr.Error())
    // Handle re-authentication
}
```

#### ServerError (5xx)
```go
if serverErr, ok := err.(*rest.ServerError); ok {
    fmt.Printf("Server error: %s\n", serverErr.Error())
    // Implement retry or failover logic
}
```

#### ResponseError (4xx)
```go
if responseErr, ok := err.(*rest.ResponseError); ok {
    fmt.Printf("Client error: %s\n", responseErr.Error())
    // Handle client-side errors
}
```

#### ResourceNotFoundError (404)
```go
if notFoundErr, ok := err.(*rest.ResourceNotFoundError); ok {
    fmt.Printf("Resource not found: %s\n", notFoundErr.Error())
    // Handle missing resources
}
```

#### ExecutionError
```go
if execErr, ok := err.(*rest.ExecutionError); ok {
    fmt.Printf("Execution failed: %s\n", execErr.Error())
    // Handle network or execution failures
}
```

## Integration with Other Packages

### With Logging Package

```go
import (
    "github.com/jasoet/pkg/logging"
    "github.com/jasoet/pkg/rest"
)

func makeAPICall(ctx context.Context) {
    logger := logging.ContextLogger(ctx, "api-client")
    
    client := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))
    
    logger.Info().Str("endpoint", "/users").Msg("Making API call")
    
    response, err := client.MakeRequest(ctx, "GET", "https://api.example.com/users", "", nil)
    if err != nil {
        logger.Error().Err(err).Msg("API call failed")
        return
    }
    
    logger.Info().Int("status", response.StatusCode()).Msg("API call successful")
}
```

### With Concurrent Package

```go
import (
    "github.com/jasoet/pkg/concurrent"
    "github.com/jasoet/pkg/rest"
)

func makeParallelAPICalls(ctx context.Context) {
    client := rest.NewClient()
    
    apiFunctions := map[string]concurrent.Func[*resty.Response]{
        "users": func(ctx context.Context) (*resty.Response, error) {
            return client.MakeRequest(ctx, "GET", "https://api.example.com/users", "", nil)
        },
        "posts": func(ctx context.Context) (*resty.Response, error) {
            return client.MakeRequest(ctx, "GET", "https://api.example.com/posts", "", nil)
        },
    }
    
    results, err := concurrent.ExecuteConcurrently(ctx, apiFunctions)
    // Handle results...
}
```

## Best Practices

### 1. Client Configuration

```go
// Use environment-specific configurations
func createAPIClient(env string) *rest.Client {
    var config *rest.Config
    
    switch env {
    case "production":
        config = &rest.Config{
            RetryCount: 3,
            Timeout:    30 * time.Second,
        }
    case "development":
        config = &rest.Config{
            RetryCount: 1,
            Timeout:    10 * time.Second,
        }
    }
    
    return rest.NewClient(rest.WithRestConfig(*config))
}
```

### 2. Error Handling

```go
func handleAPIError(err error) {
    switch e := err.(type) {
    case *rest.UnauthorizedError:
        // Refresh token and retry
        refreshAuthToken()
    case *rest.ServerError:
        // Implement circuit breaker
        if e.StatusCode >= 500 {
            markServiceUnhealthy()
        }
    case *rest.ResponseError:
        // Log client errors for debugging
        log.Printf("Client error: %s", e.Error())
    }
}
```

### 3. Context Usage

```go
// Always use context with timeout
func makeAPICallWithTimeout(baseCtx context.Context) {
    ctx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
    defer cancel()
    
    client := rest.NewClient()
    response, err := client.MakeRequest(ctx, "GET", url, "", nil)
    // Handle response...
}
```

### 4. Resource Management

```go
// Reuse clients for better performance
var apiClient *rest.Client
var clientOnce sync.Once

func getAPIClient() *rest.Client {
    clientOnce.Do(func() {
        apiClient = rest.NewClient(
            rest.WithRestConfig(getAPIConfig()),
            rest.WithMiddleware(rest.NewLoggingMiddleware()),
        )
    })
    return apiClient
}
```

### 5. Testing

```go
func TestAPICall(t *testing.T) {
    // Use NoOpMiddleware for testing
    client := rest.NewClient(rest.WithMiddleware(rest.NewNoOpMiddleware()))
    
    // Mock HTTP server for testing
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status":"success"}`))
    }))
    defer server.Close()
    
    response, err := client.MakeRequest(context.Background(), "GET", server.URL, "", nil)
    assert.NoError(t, err)
    assert.Equal(t, 200, response.StatusCode())
}
```

## Performance Considerations

### Connection Pooling
The underlying Resty client automatically handles connection pooling. For high-throughput applications:

```go
client := rest.NewClient()
restyClient := client.GetRestyClient()

// Configure connection pool
restyClient.GetClient().Transport = &http.Transport{
    MaxIdleConns:       100,
    MaxIdleConnsPerHost: 100,
}
```

### Request Timeouts
Configure appropriate timeouts based on your use case:

```go
// Fast APIs
fastConfig := &rest.Config{Timeout: 5 * time.Second}

// Slow APIs (file uploads, reports)
slowConfig := &rest.Config{Timeout: 300 * time.Second}
```

### Retry Strategy
Balance between reliability and performance:

```go
// High reliability, slower
reliableConfig := &rest.Config{
    RetryCount: 5,
    RetryWaitTime: 1 * time.Second,
    RetryMaxWaitTime: 10 * time.Second,
}

// Fast failure, better performance
fastFailConfig := &rest.Config{
    RetryCount: 1,
    RetryWaitTime: 100 * time.Millisecond,
    RetryMaxWaitTime: 1 * time.Second,
}
```

## Troubleshooting

### Common Issues

1. **Timeout Errors**: Increase timeout or check network connectivity
2. **Retry Exhausted**: Check API availability and retry configuration
3. **Authentication Failures**: Verify credentials and token expiration
4. **Rate Limiting**: Implement backoff strategy or reduce request rate

### Debug Tips

- Enable verbose logging with `LoggingMiddleware`
- Use trace information for performance analysis
- Check response headers for API-specific error codes
- Monitor middleware execution order
- Test with different timeout configurations