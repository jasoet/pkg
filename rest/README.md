# REST Client

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v2/rest.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v2/rest)

Resilient HTTP client with automatic retries, OpenTelemetry instrumentation, and middleware support built on Resty.

## Overview

The `rest` package provides a production-ready HTTP client with built-in resilience patterns, observability, and extensibility through middleware. Built on top of go-resty, it adds OpenTelemetry tracing, metrics, and customizable request/response processing.

## Features

- **Automatic Retries**: Configurable retry logic with exponential backoff
- **OpenTelemetry Integration**: Distributed tracing and metrics
- **Middleware System**: Extensible request/response processing
- **Timeout Management**: Request-level timeout configuration
- **Thread-Safe**: Concurrent-safe middleware management
- **Flexible API**: Support for all HTTP methods

## Installation

```bash
go get github.com/jasoet/pkg/v2/rest
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/v2/rest"
)

func main() {
    // Create client with default config
    client := rest.NewClient()

    // Make request
    ctx := context.Background()
    response, err := client.MakeRequestWithTrace(
        ctx,
        "GET",
        "https://api.example.com/users",
        "",
        nil,
    )

    if err != nil {
        panic(err)
    }

    fmt.Println(response.String())
}
```

### With Custom Configuration

```go
import (
    "time"
    "github.com/jasoet/pkg/v2/rest"
)

config := rest.Config{
    RetryCount:       3,
    RetryWaitTime:    1 * time.Second,
    RetryMaxWaitTime: 5 * time.Second,
    Timeout:          30 * time.Second,
}

client := rest.NewClient(
    rest.WithRestConfig(config),
)
```

### With OpenTelemetry

```go
import (
    "github.com/jasoet/pkg/v2/rest"
    "github.com/jasoet/pkg/v2/otel"
)

// Setup OTel
otelConfig := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

// Create client with OTel
client := rest.NewClient(
    rest.WithOTelConfig(otelConfig),
)

// All requests are automatically traced
response, err := client.MakeRequestWithTrace(ctx, "GET", url, "", nil)
```

## Configuration

### Config Struct

```go
type Config struct {
    RetryCount       int           // Number of retry attempts
    RetryWaitTime    time.Duration // Initial retry wait time
    RetryMaxWaitTime time.Duration // Maximum retry wait time
    Timeout          time.Duration // Request timeout

    // Optional: Enable OpenTelemetry (nil = disabled)
    OTelConfig       *otel.Config
}
```

### Default Configuration

```go
DefaultRestConfig() returns:
- RetryCount:       1
- RetryWaitTime:    20 seconds
- RetryMaxWaitTime: 30 seconds
- Timeout:          50 seconds
```

## Client API

### Client Options

```go
// Set configuration
WithRestConfig(config Config)

// Add single middleware
WithMiddleware(middleware Middleware)

// Set multiple middlewares
WithMiddlewares(middlewares ...Middleware)

// Enable OpenTelemetry
WithOTelConfig(cfg *otel.Config)
```

### Methods

```go
// Make HTTP request with tracing
MakeRequestWithTrace(
    ctx context.Context,
    method string,
    url string,
    body string,
    headers map[string]string,
) (*resty.Response, error)

// Get underlying Resty client
GetRestClient() *resty.Client

// Get current configuration
GetRestConfig() *Config

// Middleware management
AddMiddleware(middleware Middleware)
SetMiddlewares(middlewares ...Middleware)
GetMiddlewares() []Middleware
```

## Middleware System

### Built-in Middleware

#### LoggingMiddleware

Logs request and response details:

```go
client := rest.NewClient(
    rest.WithMiddleware(rest.NewLoggingMiddleware()),
)

// Logs:
// - Method, URL
// - Status code
// - Duration
// - Errors
```

#### NoOpMiddleware

Placeholder middleware for testing:

```go
client := rest.NewClient(
    rest.WithMiddleware(rest.NewNoOpMiddleware()),
)
```

#### OpenTelemetry Middlewares

Automatically added when `OTelConfig` is provided:

1. **OTelTracingMiddleware** - Distributed tracing
2. **OTelMetricsMiddleware** - HTTP client metrics
3. **OTelLoggingMiddleware** - Structured logging

### Custom Middleware

Implement the `Middleware` interface:

```go
type Middleware interface {
    BeforeRequest(
        ctx context.Context,
        method string,
        url string,
        body string,
        headers map[string]string,
    ) context.Context

    AfterRequest(ctx context.Context, info RequestInfo)
}
```

**Example:**

```go
type AuthMiddleware struct {
    apiKey string
}

func (m *AuthMiddleware) BeforeRequest(
    ctx context.Context,
    method string,
    url string,
    body string,
    headers map[string]string,
) context.Context {
    headers["Authorization"] = "Bearer " + m.apiKey
    return ctx
}

func (m *AuthMiddleware) AfterRequest(
    ctx context.Context,
    info RequestInfo,
) {
    // Process response
}

// Usage
client := rest.NewClient(
    rest.WithMiddleware(&AuthMiddleware{apiKey: "secret"}),
)
```

## OpenTelemetry Integration

### Automatic Tracing

When `OTelConfig` is provided, all requests are traced:

```go
otelConfig := otel.NewConfig("my-client").
    WithTracerProvider(tracerProvider)

client := rest.NewClient(
    rest.WithOTelConfig(otelConfig),
)

// Creates span for each request
response, _ := client.MakeRequestWithTrace(ctx, "GET", url, "", nil)
```

### Span Attributes

Each HTTP request span includes:

```yaml
Span Attributes:
  http.method: "GET" | "POST" | "PUT" | "DELETE" | ...
  http.url: "https://api.example.com/users"
  http.status_code: 200
  http.duration_ms: 150
  pkg.rest.client.name: "my-client"
  pkg.rest.retry.max_count: 3
  pkg.rest.timeout_ms: 30000
```

### Metrics Collection

Automatic HTTP client metrics:

```yaml
Metrics:
  http.client.request.duration: Histogram of request durations
  http.client.request.count: Counter of total requests
  http.client.request.active: Gauge of active requests

Attributes:
  http.method: "GET"
  http.status_code: 200
  service.name: "my-client"
```

## Advanced Usage

### All HTTP Methods

```go
// GET
response, _ := client.MakeRequestWithTrace(ctx, "GET", url, "", headers)

// POST
response, _ := client.MakeRequestWithTrace(ctx, "POST", url, `{"key":"value"}`, headers)

// PUT
response, _ := client.MakeRequestWithTrace(ctx, "PUT", url, body, headers)

// DELETE
response, _ := client.MakeRequestWithTrace(ctx, "DELETE", url, "", headers)

// PATCH
response, _ := client.MakeRequestWithTrace(ctx, "PATCH", url, body, headers)

// HEAD
response, _ := client.MakeRequestWithTrace(ctx, "HEAD", url, "", headers)

// OPTIONS
response, _ := client.MakeRequestWithTrace(ctx, "OPTIONS", url, "", headers)
```

### Custom Headers

```go
headers := map[string]string{
    "Authorization": "Bearer token",
    "Content-Type":  "application/json",
    "X-API-Key":     "secret",
}

response, _ := client.MakeRequestWithTrace(ctx, "GET", url, "", headers)
```

### Request Body

```go
body := `{
    "name": "John Doe",
    "email": "john@example.com"
}`

response, _ := client.MakeRequestWithTrace(ctx, "POST", url, body, headers)
```

### Configuration from YAML

```go
import (
    "github.com/jasoet/pkg/v2/config"
    "github.com/jasoet/pkg/v2/rest"
)

type AppConfig struct {
    REST rest.Config `yaml:"rest"`
}

yamlConfig := `
rest:
  retryCount: 3
  retryWaitTime: 1s
  retryMaxWaitTime: 5s
  timeout: 30s
`

cfg, _ := config.LoadString[AppConfig](yamlConfig)
client := rest.NewClient(rest.WithRestConfig(cfg.REST))
```

### Access Underlying Resty Client

For advanced Resty features:

```go
client := rest.NewClient()

// Get Resty client
restyClient := client.GetRestClient()

// Use Resty directly
restyClient.R().
    SetHeader("X-Custom", "value").
    SetQueryParam("page", "1").
    Get("https://api.example.com/users")
```

## Error Handling

```go
response, err := client.MakeRequestWithTrace(ctx, "GET", url, "", nil)

if err != nil {
    // Network error, timeout, or other client error
    log.Printf("Request failed: %v", err)
    return
}

// Check HTTP status
if response.StatusCode() != 200 {
    log.Printf("HTTP error: %d - %s", response.StatusCode(), response.String())
    return
}

// Process response
fmt.Println(response.String())
```

## Best Practices

### 1. Use Context for Cancellation

```go
// ✅ Good: Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.MakeRequestWithTrace(ctx, "GET", url, "", nil)
```

### 2. Configure Retries Appropriately

```go
// ✅ Good: Reasonable retry config
config := rest.Config{
    RetryCount:       3,              // Retry up to 3 times
    RetryWaitTime:    1 * time.Second, // Start with 1s
    RetryMaxWaitTime: 10 * time.Second, // Cap at 10s
    Timeout:          30 * time.Second,
}
```

### 3. Always Enable OTel in Production

```go
// ✅ Good: Observability enabled
client := rest.NewClient(
    rest.WithOTelConfig(otelConfig),
)

// ❌ Bad: No observability
client := rest.NewClient()
```

### 4. Reuse Client Instances

```go
// ✅ Good: Singleton client
var httpClient = rest.NewClient(/* config */)

func fetchUser(id string) {
    httpClient.MakeRequestWithTrace(/* ... */)
}

// ❌ Bad: New client per request
func fetchUser(id string) {
    client := rest.NewClient() // Creates new connection pool
    client.MakeRequestWithTrace(/* ... */)
}
```

### 5. Use Middleware for Cross-Cutting Concerns

```go
// ✅ Good: Centralized auth
type AuthMiddleware struct { /* ... */ }

client := rest.NewClient(
    rest.WithMiddleware(&AuthMiddleware{}),
    rest.WithMiddleware(&RateLimitMiddleware{}),
)

// All requests get auth + rate limiting
```

## Testing

The package includes comprehensive tests with 93% coverage:

```bash
# Run tests
go test ./rest -v

# With coverage
go test ./rest -cover
```

### Test Utilities

```go
import (
    "github.com/jasoet/pkg/v2/rest"
    "net/http/httptest"
)

func TestMyCode(t *testing.T) {
    // Mock server
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.WriteHeader(200)
        w.Write([]byte(`{"status":"ok"}`))
    }))
    defer server.Close()

    // Use no-op middleware for testing
    client := rest.NewClient(
        rest.WithMiddleware(rest.NewNoOpMiddleware()),
    )

    response, err := client.MakeRequestWithTrace(
        context.Background(),
        "GET",
        server.URL,
        "",
        nil,
    )

    assert.NoError(t, err)
    assert.Equal(t, 200, response.StatusCode())
}
```

## Troubleshooting

### Timeout Errors

**Problem**: Requests timing out

**Solutions:**
```go
// 1. Increase timeout
config := rest.Config{
    Timeout: 60 * time.Second, // Longer timeout
    // ...
}

// 2. Use context timeout
ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
defer cancel()
```

### Retry Not Working

**Problem**: Client not retrying failed requests

**Solutions:**
```go
// 1. Check retry configuration
config := rest.Config{
    RetryCount:       3,              // Must be > 0
    RetryWaitTime:    1 * time.Second,
    RetryMaxWaitTime: 5 * time.Second,
}

// 2. Verify error is retryable
// Resty retries on network errors and 5xx status codes
// Does NOT retry on 4xx client errors
```

### OTel Not Tracing

**Problem**: No spans appearing

**Solutions:**
```go
// 1. Verify OTel config is provided
client := rest.NewClient(
    rest.WithOTelConfig(otelConfig), // Must be set
)

// 2. Check tracer provider
if otelConfig.IsTracingEnabled() {
    // Tracing is enabled
}

// 3. Ensure context propagation
ctx, span := tracer.Start(ctx, "parent-span")
defer span.End()

client.MakeRequestWithTrace(ctx, /* ... */) // Propagates context
```

## Performance

- **Connection Pooling**: Reuses HTTP connections via Resty
- **Low Overhead**: Minimal middleware overhead (~microseconds)
- **Efficient Retries**: Exponential backoff prevents thundering herd

**Benchmark (typical request):**
```
BenchmarkRequest-8         1000    ~1ms/op (including network)
BenchmarkMiddleware-8     10000    ~5µs/op (middleware overhead)
```

## Examples

See [examples/](./examples/) directory for:
- Basic HTTP requests
- OpenTelemetry integration
- Custom middleware
- Error handling
- Retry configuration
- Authentication patterns

## Related Packages

- **[otel](../otel/)** - OpenTelemetry configuration
- **[config](../config/)** - Configuration management
- **[server](../server/)** - HTTP server

## License

MIT License - see [LICENSE](../LICENSE) for details.
