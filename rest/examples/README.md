# REST Package Examples

This directory contains examples demonstrating how to use the `rest` package for HTTP client operations in Go applications.

## 📍 Example Code Location

**Full example implementation:** [/rest/examples/example.go](https://github.com/jasoet/pkg/blob/main/rest/examples/example.go)

## 🚀 Quick Reference for LLMs/Coding Agents

```go
// Basic usage pattern
import (
    "net/http"
    "github.com/jasoet/pkg/rest"
)

// Create client with defaults
client := rest.NewClient()

// Make requests with all HTTP methods supported
response, err := client.MakeRequest(ctx, http.MethodGet, "https://api.example.com/users", "", nil)
response, err = client.MakeRequest(ctx, http.MethodPost, "https://api.example.com/users", jsonBody, headers)
response, err = client.MakeRequest(ctx, http.MethodPut, "https://api.example.com/users/1", jsonBody, headers)
response, err = client.MakeRequest(ctx, http.MethodDelete, "https://api.example.com/users/1", "", nil)
response, err = client.MakeRequest(ctx, http.MethodPatch, "https://api.example.com/users/1", patchBody, headers)
response, err = client.MakeRequest(ctx, http.MethodHead, "https://api.example.com/users", "", nil)
response, err = client.MakeRequest(ctx, http.MethodOptions, "https://api.example.com/users", "", nil)

// Custom methods are also supported via fallback
response, err = client.MakeRequest(ctx, "CUSTOM", "https://api.example.com/special", "", nil)

// With custom configuration
config := &rest.Config{
    RetryCount:    3,
    RetryWaitTime: 2 * time.Second,
    Timeout:       30 * time.Second,
}
client = rest.NewClient(rest.WithRestConfig(*config))

// Add middleware
authMiddleware := rest.NewLoggingMiddleware()
client = rest.NewClient(rest.WithMiddleware(authMiddleware))
```

**Key features:**
- **Full HTTP method support** - GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS + custom methods
- **Built-in retry logic** with exponential backoff
- **Middleware support** for auth, logging, etc.
- **Context-aware** with proper cancellation
- **Comprehensive error handling** - separate execution errors from HTTP response errors
- **Type-safe error categorization** - UnauthorizedError, ServerError, ResourceNotFoundError, etc.

## Overview

The `rest` package provides utilities for:
- **HTTP client creation** with retry and timeout configuration
- **Full HTTP method support** - all standard methods (GET, POST, PUT, DELETE, PATCH, HEAD, OPTIONS) + custom methods
- **Middleware support** for request/response interception
- **Comprehensive error handling** with distinct error types for execution vs HTTP response errors
- **Built-in logging and tracing** capabilities
- **Context-aware request handling** with proper cancellation support

## Running the Examples

To run the examples, use the following command from the `rest/examples` directory:

```bash
go run example.go
```

**Note**: Some examples make actual HTTP requests to public APIs. Ensure you have internet connectivity for full demonstration.

## Example Descriptions

The [example.go](https://github.com/jasoet/pkg/blob/main/rest/examples/example.go) file demonstrates several use cases:

### 1. Basic HTTP Client

Create a simple HTTP client with default configuration:

```go
// Create client with default configuration
client := rest.NewClient()

// Make requests using HTTP method constants
response, err := client.MakeRequest(ctx, http.MethodGet, "https://api.example.com/users", "", nil)
if err != nil {
    log.Fatal(err)
}

// Other HTTP methods
response, err = client.MakeRequest(ctx, http.MethodPost, "https://api.example.com/users", jsonBody, headers)
response, err = client.MakeRequest(ctx, http.MethodPut, "https://api.example.com/users/1", jsonBody, headers)
response, err = client.MakeRequest(ctx, http.MethodDelete, "https://api.example.com/users/1", "", nil)
response, err = client.MakeRequest(ctx, http.MethodPatch, "https://api.example.com/users/1", patchBody, headers)
response, err = client.MakeRequest(ctx, http.MethodHead, "https://api.example.com/users", "", nil)
response, err = client.MakeRequest(ctx, http.MethodOptions, "https://api.example.com/users", "", nil)
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

Handle different types of HTTP errors with comprehensive error categorization:

```go
response, err := client.MakeRequest(ctx, http.MethodGet, url, "", headers)
if err != nil {
    switch e := err.(type) {
    case *rest.UnauthorizedError:
        // HTTP 401/403 - Authentication/Authorization errors
        log.Printf("Auth failed (Status %d): %s", e.StatusCode, e.Error())
        // Handle token refresh, re-authentication, etc.
        
    case *rest.ResourceNotFoundError:
        // HTTP 404 - Resource not found
        log.Printf("Resource not found (Status %d): %s", e.StatusCode, e.Error())
        // Handle missing resources, redirect to creation page, etc.
        
    case *rest.ServerError:
        // HTTP 5xx - Server-side errors
        log.Printf("Server error (Status %d): %s", e.StatusCode, e.Error())
        // Implement retry logic, circuit breaker, failover, etc.
        
    case *rest.ResponseError:
        // HTTP 4xx (except 401/403/404) - Client errors
        log.Printf("Client error (Status %d): %s", e.StatusCode, e.Error())
        // Handle validation errors, bad requests, etc.
        
    case *rest.ExecutionError:
        // Network, DNS, timeout, connection errors (not HTTP response errors)
        log.Printf("Execution failed: %s", e.Error())
        if e.Unwrap() != nil {
            log.Printf("Underlying error: %s", e.Unwrap().Error())
        }
        // Handle network issues, DNS problems, timeouts, etc.
        
    default:
        log.Printf("Unknown error: %s", err.Error())
    }
}
```

### 5. JSON API Interactions

Work with JSON APIs using built-in JSON support:

```go
// GET request with JSON response
response, err := client.MakeRequest(ctx, http.MethodGet, "https://api.example.com/users", "", nil)
if err == nil {
    var users []User
    json.Unmarshal(response.Body(), &users)
}

// POST request with JSON body
userData := User{Name: "John Doe", Email: "john@example.com"}
jsonBody, _ := json.Marshal(userData)
headers := map[string]string{"Content-Type": "application/json"}
response, err := client.MakeRequest(ctx, http.MethodPost, "https://api.example.com/users", string(jsonBody), headers)

// PUT request for updates
response, err = client.MakeRequest(ctx, http.MethodPut, "https://api.example.com/users/1", string(jsonBody), headers)

// PATCH request for partial updates
patchData := map[string]interface{}{"email": "newemail@example.com"}
patchBody, _ := json.Marshal(patchData)
response, err = client.MakeRequest(ctx, http.MethodPatch, "https://api.example.com/users/1", string(patchBody), headers)

// DELETE request
response, err = client.MakeRequest(ctx, http.MethodDelete, "https://api.example.com/users/1", "", nil)
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
response, err := client.MakeRequest(ctx, http.MethodGet, unreliableAPI, "", nil)
```

### 7. Request Tracing and Performance Monitoring

Monitor request performance with built-in tracing:

```go
client := rest.NewClient(rest.WithMiddleware(rest.NewLoggingMiddleware()))

// Request will be automatically traced and logged
response, err := client.MakeRequest(ctx, http.MethodGet, url, "", nil)

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
restyClient := client.GetRestClient()

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

The REST client provides comprehensive error categorization to help you handle different failure scenarios appropriately.

### Error Categories

#### 1. Execution Errors vs Response Errors

**Execution Errors** (`*rest.ExecutionError`):
- Network connectivity issues (DNS resolution, connection refused, etc.)
- Request timeouts before reaching the server
- Invalid URLs or malformed requests
- Any error that prevents the HTTP request from being sent or completed

**Response Errors** (HTTP status-based errors):
- Server successfully received and processed the request but returned an error status
- Can be categorized by HTTP status code ranges

### Built-in Error Types

#### ExecutionError - Network/Connection Issues
```go
if execErr, ok := err.(*rest.ExecutionError); ok {
    fmt.Printf("Execution failed: %s\n", execErr.Error())
    if execErr.Unwrap() != nil {
        fmt.Printf("Underlying error: %s\n", execErr.Unwrap().Error())
    }
    // Handle network issues, DNS problems, timeouts
    // Implement connection retry, fallback endpoints, etc.
}
```

#### UnauthorizedError - HTTP 401/403
```go
if unauthorizedErr, ok := err.(*rest.UnauthorizedError); ok {
    fmt.Printf("Auth failed (Status %d): %s\n", unauthorizedErr.StatusCode, unauthorizedErr.Error())
    // Handle re-authentication, token refresh, permission issues
}
```

#### ResourceNotFoundError - HTTP 404
```go
if notFoundErr, ok := err.(*rest.ResourceNotFoundError); ok {
    fmt.Printf("Resource not found (Status %d): %s\n", notFoundErr.StatusCode, notFoundErr.Error())
    // Handle missing resources, redirect to creation, suggest alternatives
}
```

#### ServerError - HTTP 5xx
```go
if serverErr, ok := err.(*rest.ServerError); ok {
    fmt.Printf("Server error (Status %d): %s\n", serverErr.StatusCode, serverErr.Error())
    // Implement retry logic, circuit breaker, failover to backup services
}
```

#### ResponseError - Other HTTP 4xx
```go
if responseErr, ok := err.(*rest.ResponseError); ok {
    fmt.Printf("Client error (Status %d): %s\n", responseErr.StatusCode, responseErr.Error())
    // Handle validation errors, bad requests, rate limiting
}
```

### Complete Error Handling Pattern

```go
response, err := client.MakeRequest(ctx, http.MethodPost, apiURL, jsonBody, headers)
if err != nil {
    switch e := err.(type) {
    case *rest.ExecutionError:
        // Network/DNS/Connection issues - not HTTP response errors
        logger.Error().Err(e.Unwrap()).Msg("Network connectivity issue")
        return retryWithBackoff() // or switch to fallback endpoint
        
    case *rest.UnauthorizedError:
        // HTTP 401/403 - Authentication/Authorization
        logger.Warn().Int("status", e.StatusCode).Msg("Authentication required")
        return refreshTokenAndRetry()
        
    case *rest.ResourceNotFoundError:
        // HTTP 404 - Resource doesn't exist
        logger.Info().Int("status", e.StatusCode).Msg("Resource not found")
        return createResourceFirst()
        
    case *rest.ServerError:
        // HTTP 5xx - Server-side issues
        logger.Error().Int("status", e.StatusCode).Msg("Server error")
        return useCircuitBreaker() // or failover
        
    case *rest.ResponseError:
        // HTTP 4xx (except 401/403/404) - Client errors
        logger.Warn().Int("status", e.StatusCode).Msg("Request validation failed")
        return handleValidationErrors()
        
    default:
        logger.Error().Err(err).Msg("Unexpected error type")
        return err
    }
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
    
    response, err := client.MakeRequest(ctx, http.MethodGet, "https://api.example.com/users", "", nil)
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
            return client.MakeRequest(ctx, http.MethodGet, "https://api.example.com/users", "", nil)
        },
        "posts": func(ctx context.Context) (*resty.Response, error) {
            return client.MakeRequest(ctx, http.MethodGet, "https://api.example.com/posts", "", nil)
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
    response, err := client.MakeRequest(ctx, http.MethodGet, url, "", nil)
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
    
    response, err := client.MakeRequest(context.Background(), http.MethodGet, server.URL, "", nil)
    assert.NoError(t, err)
    assert.Equal(t, 200, response.StatusCode())
}
```

## Performance Considerations

### Connection Pooling
The underlying Resty client automatically handles connection pooling. For high-throughput applications:

```go
client := rest.NewClient()
restyClient := client.GetRestClient()

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