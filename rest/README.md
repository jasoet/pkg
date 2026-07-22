# REST Client

[![Go Reference](https://pkg.go.dev/badge/github.com/jasoet/pkg/v3/rest.svg)](https://pkg.go.dev/github.com/jasoet/pkg/v3/rest)

Resilient HTTP client with automatic retries, OpenTelemetry instrumentation, and middleware support built on Resty.

## Overview

The `rest` package provides a production-ready HTTP client with built-in resilience patterns, observability, and extensibility through middleware. Built on top of go-resty, it adds OpenTelemetry tracing, metrics, and customizable request/response processing — while returning library-owned types (`rest.Response`, typed errors) so callers never depend on resty in their own code.

## Features

- **Automatic Retries**: Configurable retry logic with exponential backoff (network errors and HTTP 5xx)
- **Library-Owned Response**: `rest.Response` with status predicates — no resty types in the public API
- **Typed Errors**: `errors.As`-friendly error types for 401/403, 404, 5xx, other 4xx, and execution failures
- **OpenTelemetry Integration**: Distributed tracing and metrics
- **Middleware System**: Extensible request/response processing
- **Timeout Management**: Request-level timeout configuration
- **Thread-Safe**: Concurrent-safe middleware management

## Installation

```bash
go get github.com/jasoet/pkg/v3/rest
```

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "fmt"

    "github.com/jasoet/pkg/v3/rest"
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

    fmt.Println(response.Body)
}
```

### With Custom Configuration

```go
import (
    "time"

    "github.com/jasoet/pkg/v3/rest"
)

config := rest.Config{
    RetryCount:         3,
    RetryWaitTime:      1 * time.Second,
    RetryMaxWaitTime:   5 * time.Second,
    Timeout:            30 * time.Second,
    MaxResponseBodyLog: 2048,
}

client := rest.NewClient(
    rest.WithRestConfig(config),
)
```

### With OpenTelemetry

```go
import (
    "github.com/jasoet/pkg/v3/otel"
    "github.com/jasoet/pkg/v3/rest"
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

    // Limits bytes of response body stored in logs/errors. 0 = unlimited.
    MaxResponseBodyLog int

    // Optional: Enable OpenTelemetry (nil = disabled)
    OTelConfig       *otel.Config
}
```

### Default Configuration

`DefaultRestConfig()` returns:

- RetryCount: 1
- RetryWaitTime: 2 seconds
- RetryMaxWaitTime: 10 seconds
- Timeout: 30 seconds
- MaxResponseBodyLog: 1024

## Response Type

`MakeRequest` and `MakeRequestWithTrace` return the library-owned `*rest.Response`:

```go
type Response struct {
    StatusCode int
    Body       string
    Header     http.Header
}
```

### Status Predicates

```go
resp, err := client.MakeRequest(ctx, "GET", url, "", nil)

resp.IsSuccess()     // 2xx
resp.IsError()       // any status >= 400
resp.IsServerError() // 5xx
resp.IsClientError() // 4xx
resp.IsAuthError()   // 401 or 403
resp.IsNotFound()    // 404
```

Note: even when a request returns a typed error for a non-2xx status, the
`*Response` is still returned (non-nil) so you can inspect the status, body,
and headers.

## Error Handling

Non-2xx responses and execution failures produce typed errors. The error
types are exported for type switches / `errors.As`; construction is internal
to the package.

| Error type | Condition | Sentinel (`errors.Is`) |
|---|---|---|
| `*rest.ExecutionError` | Network/DNS/timeout failure | wraps underlying error |
| `*rest.UnauthorizedError` | HTTP 401 or 403 | `rest.ErrUnauthorized` |
| `*rest.ResourceNotFoundError` | HTTP 404 | `rest.ErrResourceNotFound` |
| `*rest.ServerError` | HTTP 5xx | `rest.ErrServer` |
| `*rest.ResponseError` | Other HTTP 4xx | `rest.ErrResponse` |

Each HTTP error type exposes `StatusCode`, `Msg`, and `RespBody` (truncated to
`MaxResponseBodyLog`).

```go
resp, err := client.MakeRequest(ctx, "GET", url, "", nil)
if err != nil {
    var authErr *rest.UnauthorizedError
    var notFound *rest.ResourceNotFoundError
    var srvErr *rest.ServerError
    var execErr *rest.ExecutionError

    switch {
    case errors.As(err, &authErr):
        log.Printf("auth failed (HTTP %d)", authErr.StatusCode)
    case errors.As(err, &notFound):
        log.Printf("missing resource (HTTP %d)", notFound.StatusCode)
    case errors.As(err, &srvErr):
        log.Printf("server error (HTTP %d): %s", srvErr.StatusCode, srvErr.RespBody)
    case errors.As(err, &execErr):
        log.Printf("request execution failed: %v", execErr.Unwrap())
    default:
        log.Printf("request failed: %v", err)
    }
    return
}

fmt.Println(resp.Body)
```

## Client API

### Client Options

```go
// Set configuration
WithRestConfig(config Config)

// Add single middleware
WithMiddleware(middleware Middleware)

// Set multiple middlewares (replaces the chain, including the default LoggingMiddleware)
WithMiddlewares(middlewares ...Middleware)

// Enable OpenTelemetry
WithOTelConfig(cfg *otel.Config)
```

### Methods

```go
// Make HTTP request
MakeRequest(
    ctx context.Context,
    method string,
    url string,
    body string,
    headers map[string]string,
) (*rest.Response, error)

// Make HTTP request with resty trace enabled (populates RequestInfo.TraceInfo for middleware)
MakeRequestWithTrace(
    ctx context.Context,
    method string,
    url string,
    body string,
    headers map[string]string,
) (*rest.Response, error)

// Get current configuration (a copy)
GetRestConfig() *Config

// Middleware management
AddMiddleware(middleware Middleware)
SetMiddlewares(middlewares ...Middleware)
GetMiddlewares() []Middleware
```

### Escape Hatch: GetRestClient

`GetRestClient()` returns the underlying `*resty.Client` for advanced use
cases the wrapper does not cover — custom TLS configuration, binary request
bodies via `SetBody(interface{})`, automatic result unmarshaling with
`SetResult`, file uploads, etc. Responses from calls made directly through
the resty client are resty types and bypass the middleware chain and the
typed-error mapping above. Note that `GetRestClient()` calls still record
retry metrics (the retry hook lives on the resty client itself) while
bypassing the other middleware telemetry (tracing, logging, request metrics).

```go
client := rest.NewClient()

// Advanced: use resty directly
restyClient := client.GetRestClient()
restyClient.R().
    SetHeader("X-Custom", "value").
    SetQueryParam("page", "1").
    Get("https://api.example.com/users")
```

Note: mutating the resty client after `NewClient` returns is not thread-safe
for concurrent use with `MakeRequest`/`MakeRequestWithTrace`.

## Middleware System

### Built-in Middleware

#### LoggingMiddleware

Logs request and response details (method, URL, status code, duration,
errors). Added by default when no middleware options are provided.

```go
client := rest.NewClient(
    rest.WithMiddleware(rest.NewLoggingMiddleware()),
)
```

#### NoOpMiddleware

Placeholder middleware for testing:

```go
client := rest.NewClient(
    rest.WithMiddleware(rest.NewNoOpMiddleware()),
)
```

#### OpenTelemetry Middlewares

Automatically prepended when `OTelConfig` is provided (the default
`LoggingMiddleware` is dropped in that case, since OTel provides logging):

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

`RequestInfo` carries method, URL, headers, body, timing, status code,
truncated response body, error, and — for `MakeRequestWithTrace` — a
`TraceInfo` with DNS/TCP/TLS/server/response/total durations.

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
    info rest.RequestInfo,
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

Each HTTP request span (named after the HTTP method, kind=client) includes:

```yaml
Span Attributes:
  http.request.method: "GET" | "POST" | "PUT" | "DELETE" | ...
  url.full: "https://api.example.com/users"
  http.request.body.size: 42
  http.response.status_code: 200
  http.response.body.size: 1024
  http.request.duration_ms: 150
```

The span status is `Error` when the request fails or returns a status >= 400,
`Ok` otherwise. Trace context (W3C TraceContext) is injected into the request
headers for distributed tracing. Bodies larger than `MaxResponseBodyLog`
(default 1024 bytes) report the truncated length in the `http.client.response.size`
metric and the `http.response.body.size` span attribute.

### Metrics Collection

Automatic HTTP client metrics:

```yaml
Metrics:
  http.client.request.count: Counter of total requests ({request})
  http.client.request.duration: Histogram of request durations (ms)
  http.client.request.size: Histogram of request body sizes (By)
  http.client.response.size: Histogram of response body sizes (By)
  http.client.retry.count: Counter of retry attempts ({retry})

Metric Attributes:
  http.request.method: "GET"
  http.response.status_code: 200
```

`http.client.retry.count` is wired into resty's retry hook, so it increments
on both transport errors and status-based (5xx) retries; it also carries an
`http.retry.attempt` attribute with the resty attempt number. The counter
counts failed retryable attempts (retries actually performed), and retries
triggered by transport errors lose trace-exemplar correlation because they
fall back to `context.Background()` when no response/request context exists.

## Advanced Usage

### All HTTP Methods

```go
// GET
response, _ := client.MakeRequest(ctx, "GET", url, "", headers)

// POST
response, _ := client.MakeRequest(ctx, "POST", url, `{"key":"value"}`, headers)

// PUT
response, _ := client.MakeRequest(ctx, "PUT", url, body, headers)

// DELETE
response, _ := client.MakeRequest(ctx, "DELETE", url, "", headers)

// PATCH
response, _ := client.MakeRequest(ctx, "PATCH", url, body, headers)

// HEAD
response, _ := client.MakeRequest(ctx, "HEAD", url, "", headers)

// OPTIONS
response, _ := client.MakeRequest(ctx, "OPTIONS", url, "", headers)

// Custom methods fall back to resty's Execute
response, _ := client.MakeRequest(ctx, "REPORT", url, body, headers)
```

### Custom Headers

```go
headers := map[string]string{
    "Authorization": "Bearer token",
    "Content-Type":  "application/json",
    "X-API-Key":     "secret",
}

response, _ := client.MakeRequest(ctx, "GET", url, "", headers)
```

### Request Body

The `body` parameter is a string. For binary payloads, use `GetRestClient()`
and build the request directly with resty's `SetBody(interface{})`.

```go
body := `{
    "name": "John Doe",
    "email": "john@example.com"
}`

response, _ := client.MakeRequest(ctx, "POST", url, body, headers)
```

### Configuration from YAML

```go
import (
    "github.com/jasoet/pkg/v3/config"
    "github.com/jasoet/pkg/v3/rest"
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
  maxResponseBodyLog: 2048
`

cfg, _ := config.LoadString[AppConfig](yamlConfig)
client := rest.NewClient(rest.WithRestConfig(cfg.REST))
```

## Best Practices

### 1. Use Context for Cancellation

```go
// ✅ Good: Context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

response, err := client.MakeRequest(ctx, "GET", url, "", nil)
```

### 2. Configure Retries Appropriately

```go
// ✅ Good: Reasonable retry config
config := rest.Config{
    RetryCount:       3,               // Retry up to 3 times
    RetryWaitTime:    1 * time.Second, // Start with 1s
    RetryMaxWaitTime: 10 * time.Second, // Cap at 10s
    Timeout:          30 * time.Second,
}
```

Retries trigger on network errors and HTTP 5xx responses — not on 4xx client
errors.

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
    httpClient.MakeRequest(/* ... */)
}

// ❌ Bad: New client per request
func fetchUser(id string) {
    client := rest.NewClient() // Creates new connection pool
    client.MakeRequest(/* ... */)
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

The package ships compile-checked examples (`example_test.go`) and unit tests
backed by `httptest` servers:

```bash
# Run tests
go test ./rest -v

# With coverage
go test ./rest -cover
```

### Test Utilities

```go
import (
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/jasoet/pkg/v3/rest"
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
        rest.WithMiddlewares(rest.NewNoOpMiddleware()),
    )

    response, err := client.MakeRequest(
        context.Background(),
        "GET",
        server.URL,
        "",
        nil,
    )

    assert.NoError(t, err)
    assert.Equal(t, 200, response.StatusCode)
    assert.True(t, response.IsSuccess())
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
    RetryCount:       3, // Must be > 0
    RetryWaitTime:    1 * time.Second,
    RetryMaxWaitTime: 5 * time.Second,
}

// 2. Verify error is retryable
// The client retries on network errors and 5xx status codes.
// It does NOT retry on 4xx client errors.
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

## Examples

See [examples/rest/](../examples/rest/) directory for:
- Basic HTTP requests
- OpenTelemetry integration
- Custom middleware
- Error handling
- Retry configuration
- Authentication patterns

Compile-checked examples also live in [`example_test.go`](./example_test.go).

## Related Packages

- **[otel](../otel/)** - OpenTelemetry configuration
- **[config](../config/)** - Configuration management
- **[server](../server/)** - HTTP server

## License

MIT License - see [LICENSE](../LICENSE) for details.
