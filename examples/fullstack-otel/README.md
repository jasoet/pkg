# Full-Stack OpenTelemetry Integration Example

**This is a standalone, independent Go module** demonstrating end-to-end distributed tracing, metrics, and logging across:
- **gRPC Server** with HTTP Gateway (`github.com/jasoet/pkg/v2/grpc`)
- **REST Client** making HTTP calls (`github.com/jasoet/pkg/v2/rest`)
- **Database** operations with GORM (`github.com/jasoet/pkg/v2/db`)
- **Structured Logging** with trace correlation (`github.com/jasoet/pkg/v2/logging`)

This example can be copied and run independently without cloning the entire `pkg/v2` repository.

## Architecture

```
User Request
    ↓
[gRPC Server] → Traces, Metrics, Logs
    ↓
[REST Client] → Traces, Metrics, Logs (propagates trace context)
    ↓
[Database] → Traces, Metrics (connection pool)
    ↓
Response with full trace context
```

## What Gets Traced

### Trace Propagation Flow

1. **HTTP Gateway Request** (trace starts)
   - Creates root span with trace_id
   - Logs include trace_id and span_id

2. **gRPC Handler** (child span)
   - Inherits trace_id from HTTP request
   - Creates child span for business logic

3. **REST Client Call** (child span)
   - Injects trace_id into HTTP headers (W3C Trace Context)
   - Downstream service can continue the trace

4. **Database Query** (child span)
   - SQL queries traced with duration
   - Connection pool metrics

5. **Response** (spans close in order)
   - All logs linked by same trace_id
   - Click a span → See all related logs

## Quick Start

### Prerequisites

- Go 1.25.1+
- Docker and Docker Compose
- protoc (protocol buffer compiler)
- protoc-gen-go and protoc-gen-go-grpc plugins

Install protoc plugins:
```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
```

### Installation

This example is a standalone module. You can run it directly:

```bash
# Clone or copy this directory
cd fullstack-otel-example

# Dependencies are already in go.mod - no need to clone pkg/v2
go mod download
```

### Running the Example

```bash
# 1. Start Jaeger and PostgreSQL
make docker-up

# 2. Run the example (generates proto and starts server)
make run

# 3. In another terminal, make some requests:
curl http://localhost:50051/api/v1/users/1

# 4. View traces in Jaeger UI
open http://localhost:16686
```

### Stopping

```bash
# Stop the application (Ctrl+C in terminal)

# Stop Docker containers
make docker-down

# Clean generated files
make clean
```

## Setup

### 1. Start OpenTelemetry Collector (Using Docker Compose)

The example includes a `docker-compose.yml` that starts:
- **Jaeger** (all-in-one) for distributed tracing
- **PostgreSQL** (optional) for production-like database

```bash
# Start all dependencies
docker-compose up -d

# Or use the Makefile
make docker-up
```

Visit Jaeger UI: http://localhost:16686

### 2. Example Code Structure

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/jasoet/pkg/v2/db"
    "github.com/jasoet/pkg/v2/grpc"
    "github.com/jasoet/pkg/v2/logging"
    "github.com/jasoet/pkg/v2/otel"
    "github.com/jasoet/pkg/v2/rest"
    "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
    "go.opentelemetry.io/otel/sdk/metric"
    "go.opentelemetry.io/otel/sdk/resource"
    "go.opentelemetry.io/otel/sdk/trace"
    semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

func main() {
    ctx := context.Background()

    // =========================================================================
    // Step 1: Setup OpenTelemetry Providers
    // =========================================================================

    // Resource with service information
    res, err := resource.New(ctx,
        resource.WithAttributes(
            semconv.ServiceNameKey.String("fullstack-example"),
            semconv.ServiceVersionKey.String("1.0.0"),
        ),
    )
    if err != nil {
        log.Fatal(err)
    }

    // TracerProvider with OTLP exporter
    traceExporter, err := otlptracehttp.New(ctx,
        otlptracehttp.WithEndpoint("localhost:4318"),
        otlptracehttp.WithInsecure(),
    )
    if err != nil {
        log.Fatal(err)
    }

    tracerProvider := trace.NewTracerProvider(
        trace.WithBatcher(traceExporter),
        trace.WithResource(res),
        trace.WithSampler(trace.AlwaysSample()), // Sample all for demo
    )

    // MeterProvider for metrics
    meterProvider := metric.NewMeterProvider(
        metric.WithResource(res),
    )

    // LoggerProvider with zerolog backend (automatic trace correlation)
    loggerProvider := logging.NewLoggerProvider("fullstack-example", true)

    // Create OTel config
    otelCfg := &otel.Config{
        ServiceName:    "fullstack-example",
        ServiceVersion: "1.0.0",
        TracerProvider: tracerProvider,
        MeterProvider:  meterProvider,
        LoggerProvider: loggerProvider,
    }

    // =========================================================================
    // Step 2: Setup Database with OTel
    // =========================================================================

    dbConfig := &db.ConnectionConfig{
        DbType:       db.Postgresql,
        Host:         "localhost",
        Port:         5432,
        Username:     "user",
        Password:     "password",
        DbName:       "testdb",
        Timeout:      30 * time.Second,
        MaxIdleConns: 5,
        MaxOpenConns: 10,
        OTelConfig:   otelCfg, // Enable OTel tracing and metrics
    }

    database, err := dbConfig.Pool()
    if err != nil {
        log.Fatalf("Failed to connect to database: %v", err)
    }
    log.Println("✓ Database connected with OTel instrumentation")

    // =========================================================================
    // Step 3: Setup REST Client with OTel
    // =========================================================================

    restConfig := rest.Config{
        RetryCount:       2,
        RetryWaitTime:    5 * time.Second,
        RetryMaxWaitTime: 15 * time.Second,
        Timeout:          30 * time.Second,
        OTelConfig:       otelCfg, // Enable OTel tracing and metrics
    }

    restClient := rest.NewClient(rest.WithRestConfig(restConfig))
    log.Println("✓ REST client configured with OTel instrumentation")

    // =========================================================================
    // Step 4: Setup gRPC Server with OTel
    // =========================================================================

    server, err := grpc.New(
        grpc.WithGRPCPort("50051"),
        grpc.WithOTelConfig(otelCfg), // Enable OTel for gRPC and HTTP gateway
        grpc.WithServiceRegistrar(func(s *grpc.Server) {
            // Register your gRPC services here
            // Example: pb.RegisterYourServiceServer(s, &YourServiceImpl{
            //     db:         database,
            //     restClient: restClient,
            // })
        }),
        grpc.WithShutdownHandler(func() error {
            // Shutdown OTel providers
            shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
            defer cancel()

            if err := tracerProvider.Shutdown(shutdownCtx); err != nil {
                log.Printf("Error shutting down tracer provider: %v", err)
            }
            if err := meterProvider.Shutdown(shutdownCtx); err != nil {
                log.Printf("Error shutting down meter provider: %v", err)
            }
            return nil
        }),
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Println("✓ gRPC server configured with OTel instrumentation")
    log.Println("\nServer starting with full observability:")
    log.Println("  - gRPC:          localhost:50051")
    log.Println("  - HTTP Gateway:  http://localhost:50051/api/v1")
    log.Println("  - Jaeger UI:     http://localhost:16686")
    log.Println("\nAll components instrumented with OpenTelemetry:")
    log.Println("  ✓ Distributed tracing across all components")
    log.Println("  ✓ Metrics (HTTP, gRPC, DB pool, REST client)")
    log.Println("  ✓ Structured logs with trace correlation")

    // Start server
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## Observability Features

### 1. Distributed Tracing

Every request creates a trace that spans:
- HTTP Gateway request
- gRPC service handler
- REST client calls to external services
- Database queries

**Example trace:**
```
Trace ID: 4bf92f3577b34da6a3ce929d0e0e4736

├─ HTTP GET /api/v1/users/123 (200ms)
│  └─ gRPC GetUser (180ms)
│     ├─ REST GET external-api/user-details (100ms)
│     │  └─ (propagates to external service)
│     └─ DB Query SELECT * FROM users WHERE id=$1 (5ms)
```

### 2. Metrics Collection

**gRPC Server:**
- `rpc.server.request.count` - Total requests by method and status
- `rpc.server.duration` - Request duration histogram
- `rpc.server.active_requests` - Concurrent requests

**HTTP Gateway:**
- `http.server.request.count` - Total HTTP requests
- `http.server.request.duration` - Request duration
- `http.server.active_requests` - Concurrent HTTP requests

**REST Client:**
- `http.client.request.count` - Outbound HTTP requests
- `http.client.request.duration` - Request duration
- `http.client.retry.count` - Retry attempts

**Database:**
- `db.client.connections.idle` - Idle connections
- `db.client.connections.active` - Active connections
- `db.client.connections.max` - Max connections

### 3. Log-Span Correlation

All logs automatically include trace context:

```json
{
  "level": "info",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "trace_flags": "01",
  "scope": "grpc.server",
  "rpc.method": "/users.UserService/GetUser",
  "message": "Processing user request"
}
```

In Grafana/Jaeger:
1. Click a span → See all related logs
2. Click a log → Jump to the trace

## Testing the Integration

### 1. Make a Request

```bash
# Via HTTP Gateway
curl http://localhost:50051/api/v1/users/123

# Via gRPC
grpcurl -plaintext -d '{"id": "123"}' localhost:50051 users.UserService/GetUser
```

### 2. View Traces in Jaeger

1. Open http://localhost:16686
2. Select service: `fullstack-example`
3. Click "Find Traces"
4. Click a trace to see the full request flow

### 3. Verify Log Correlation

Check logs for trace_id:
```bash
# All logs from same request will share the trace_id
grep "4bf92f3577b34da6a3ce929d0e0e4736" logs.json
```

## Benefits of This Integration

### For Development
- **Debug faster**: See exact request flow across services
- **Find bottlenecks**: Identify slow queries or API calls
- **Root cause analysis**: Trace errors to their source

### For Operations
- **SLA monitoring**: Track p50, p95, p99 latencies
- **Capacity planning**: Monitor connection pools and resource usage
- **Incident response**: Quickly isolate failing components

### For Business
- **User experience**: Measure real user request latencies
- **API performance**: Track external API call success rates
- **Database health**: Monitor query patterns and connection usage

## Advanced: Custom Spans

Add custom business logic spans:

```go
func (s *Service) ProcessOrder(ctx context.Context, orderID string) error {
    // Get tracer from OTel config
    tracer := s.otelCfg.GetTracer("business-logic")

    // Create custom span
    ctx, span := tracer.Start(ctx, "ProcessOrder")
    defer span.End()

    span.SetAttributes(
        attribute.String("order.id", orderID),
        attribute.String("user.id", "123"),
    )

    // Business logic here
    // All DB queries and REST calls will be child spans

    if err := s.validateOrder(ctx, orderID); err != nil {
        span.RecordError(err)
        return err
    }

    return s.fulfillOrder(ctx, orderID)
}
```

## Integration with Grafana

1. Configure Grafana Tempo for traces
2. Configure Grafana Loki for logs
3. Configure Grafana for metrics
4. Link them together via trace_id

Result: Click any metric/log/trace → Jump to related data

## Best Practices

1. **Always propagate context**: Pass `context.Context` to all functions
2. **Name spans descriptively**: Use operation names like "ProcessPayment", not "func1"
3. **Add relevant attributes**: Include business-relevant data (user_id, order_id, etc.)
4. **Sample strategically**: In production, use sampling to reduce overhead
5. **Monitor metrics**: Set up alerts on p99 latencies and error rates
6. **Correlate logs**: Use the logging package for automatic trace correlation

## Troubleshooting

### Traces not appearing in Jaeger

1. Check OTel collector is running: `docker ps`
2. Verify endpoint: `otlptracehttp.WithEndpoint("localhost:4318")`
3. Check sampling: Use `trace.AlwaysSample()` for testing

### Logs missing trace_id

1. Ensure you're using `logging.NewLoggerProvider()`
2. Verify context is passed to all functions
3. Check that TracerProvider is configured

### Metrics not collected

1. Verify MeterProvider is set in OTelConfig
2. Check metric names match examples above
3. Ensure context is passed to instrumented code

## Resources

- [OpenTelemetry Go SDK](https://opentelemetry.io/docs/languages/go/)
- [OTel Semantic Conventions](https://opentelemetry.io/docs/specs/semconv/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [W3C Trace Context](https://www.w3.org/TR/trace-context/)
