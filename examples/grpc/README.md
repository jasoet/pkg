# gRPC Calculator Example

This example demonstrates how to use the reusable `pkg/grpc` component to build a complete gRPC service with different types of RPC calls.

## What This Example Demonstrates

- **Unary RPC**: Simple request-response operations (Add, Subtract, Multiply, Divide)
- **Server Streaming RPC**: Server sends multiple responses to a single client request (Factorial calculation)
- **Client Streaming RPC**: Client sends multiple requests and gets a single response (Sum calculation)
- **Bidirectional Streaming RPC**: Both client and server send streams of messages (Running Average)
- **Error Handling**: Proper gRPC error codes and messages
- **Production Features**: Health checks, metrics, reflection using `pkg/grpc`

## Project Structure

```
examples/
├── api/calculator/v1/         # Protocol Buffer definitions
│   └── calculator.proto       # Service and message definitions
├── gen/calculator/v1/         # Generated protobuf code
│   ├── calculator.pb.go       # Generated message types
│   └── calculator_grpc.pb.go  # Generated service interfaces
├── internal/service/          # Business logic implementation
│   └── calculator_service.go  # Calculator service implementation
├── cmd/                       # Executables
│   ├── server/                # gRPC server
│   │   └── main.go           # Server main with pkg/grpc integration
│   └── client/               # gRPC client for testing
│       └── main.go           # Client implementation and tests
└── README.md                 # This documentation
```

## Prerequisites

### Required Tools

1. **Go 1.21+**
   ```bash
   go version
   ```

2. **Protocol Buffers Compiler (protoc)**
   ```bash
   # macOS
   brew install protobuf

   # Ubuntu/Debian
   sudo apt install protobuf-compiler

   # Check installation
   protoc --version
   ```

3. **Go Protocol Buffers Plugins**
   ```bash
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
   ```

### Required Go Dependencies

The example uses the following dependencies (already included in the main project):

```go
// Core gRPC
google.golang.org/grpc
google.golang.org/protobuf

// Our reusable gRPC package
github.com/jasoet/grpc-learn/pkg/grpc
```

## Quick Start

### 1. Generate Protocol Buffer Code

From the examples directory, regenerate the protobuf code:

```bash
cd pkg/grpc/examples

protoc \
  --proto_path=api \
  --go_out=gen \
  --go_opt=paths=source_relative \
  --go-grpc_out=gen \
  --go-grpc_opt=paths=source_relative \
  api/calculator/v1/calculator.proto
```

### 2. Start the Server

```bash
# From the project root
go run -tags=examples pkg/grpc/examples/cmd/server/main.go

# Or with custom port
PORT=8080 go run -tags=examples pkg/grpc/examples/cmd/server/main.go
```

The server will start and show:
```
Starting Calculator gRPC server example on port 50051
This example demonstrates:
  - Unary RPC (Add, Subtract, Multiply, Divide)
  - Server streaming RPC (Factorial)
  - Client streaming RPC (Sum)
  - Bidirectional streaming RPC (RunningAverage)
Mixed gRPC+HTTP server starting on port 50051 (H2C mode)
gRPC endpoints available on port 50051
gRPC reflection enabled
Health checks available at http://localhost:50051/health
Metrics available at http://localhost:50051/metrics
Calculator service registered
```

### 3. Run the Client

In another terminal:

```bash
# From the project root
go run -tags=examples pkg/grpc/examples/cmd/client/main.go

# Or with custom server address
SERVER_ADDR=localhost:8080 go run -tags=examples pkg/grpc/examples/cmd/client/main.go
```

## Understanding the Code

### 1. Protocol Buffer Definition

The `api/calculator/v1/calculator.proto` file defines:

- **Service**: `CalculatorService` with various RPC methods
- **Messages**: Request/response pairs for each operation
- **Streaming Types**: Different streaming patterns (server, client, bidirectional)

### 2. Service Implementation

The `internal/service/calculator_service.go` implements:

- **Business Logic**: Actual calculation operations
- **Error Handling**: Proper gRPC status codes
- **Streaming**: Handling different stream types
- **Logging**: Request/response logging for debugging

### 3. Server Setup

The `cmd/server/main.go` demonstrates:

- **pkg/grpc Integration**: Using our reusable server component
- **Configuration**: Setting up H2C mode, reflection, health checks
- **Service Registration**: Registering the calculator service
- **Production Features**: Built-in metrics and monitoring

### 4. Client Implementation

The `cmd/client/main.go` shows:

- **Connection Setup**: Establishing gRPC connection
- **Unary Calls**: Simple request-response patterns
- **Streaming Calls**: Handling different streaming patterns
- **Error Handling**: Dealing with gRPC errors

## Testing Different RPC Types

### Unary RPC (Request-Response)

```go
// Simple addition
resp, err := client.Add(ctx, &calculatorv1.AddRequest{A: 10, B: 5})
if err != nil {
    log.Printf("Error: %v", err)
} else {
    fmt.Printf("Result: %.2f", resp.Result)
}
```

### Server Streaming RPC

```go
// Get factorial steps
stream, err := client.Factorial(ctx, &calculatorv1.FactorialRequest{Number: 5})
for {
    resp, err := stream.Recv()
    if err == io.EOF {
        break // Stream ended
    }
    fmt.Printf("Step %d: %d", resp.Step, resp.Result)
}
```

### Client Streaming RPC

```go
// Send multiple numbers, get sum
stream, err := client.Sum(ctx)
for _, num := range numbers {
    stream.Send(&calculatorv1.SumRequest{Number: num})
}
resp, err := stream.CloseAndRecv()
fmt.Printf("Total: %.2f", resp.Total)
```

### Bidirectional Streaming RPC

```go
// Send numbers and receive running averages
stream, err := client.RunningAverage(ctx)

// Start goroutine to receive responses
go func() {
    for {
        resp, err := stream.Recv()
        if err == io.EOF { return }
        fmt.Printf("Average: %.2f", resp.Average)
    }
}()

// Send numbers
for _, num := range numbers {
    stream.Send(&calculatorv1.RunningAverageRequest{Number: num})
}
stream.CloseSend()
```

## Advanced Features

### Health Checks

The server exposes health check endpoints:

```bash
# Check overall health
curl http://localhost:50051/health

# Check readiness
curl http://localhost:50051/health/ready

# Check liveness
curl http://localhost:50051/health/live
```

### Metrics

Prometheus metrics are available at:

```bash
curl http://localhost:50051/metrics
```

### gRPC Reflection

The server supports gRPC reflection for tools like `grpcurl`:

```bash
# List services
grpcurl -plaintext localhost:50051 list

# List methods
grpcurl -plaintext localhost:50051 list calculator.v1.CalculatorService

# Call a method
grpcurl -plaintext -d '{"a": 10, "b": 5}' \
  localhost:50051 calculator.v1.CalculatorService/Add
```

## Build Tags

This example uses Go build tags to avoid compilation conflicts:

- All Go files have `//go:build examples` at the top
- Build with `-tags=examples` to include the example code
- Regular builds (`go build`) will ignore the example files

## Error Handling Examples

The example demonstrates proper gRPC error handling:

```go
// Division by zero returns InvalidArgument
_, err := client.Divide(ctx, &calculatorv1.DivideRequest{A: 10, B: 0})
if err != nil {
    // Will show: rpc error: code = InvalidArgument desc = division by zero is not allowed
    fmt.Printf("Error: %v", err)
}
```

## Performance Considerations

- **H2C Mode**: Enables HTTP/2 without TLS for development
- **Connection Pooling**: Reuse gRPC connections when possible
- **Streaming**: Use streaming for large datasets or real-time updates
- **Context**: Always pass context for timeout and cancellation

## Troubleshooting

### Common Issues

1. **Port Already in Use**
   ```bash
   # Check what's using the port
   lsof -i :50051

   # Use a different port
   PORT=8080 go run -tags=examples pkg/grpc/examples/cmd/server/main.go
   ```

2. **Protobuf Generation Issues**
   ```bash
   # Make sure protoc plugins are in PATH
   which protoc-gen-go
   which protoc-gen-go-grpc

   # Regenerate code
   go generate ./...
   ```

3. **Connection Refused**
   ```bash
   # Make sure server is running first
   go run -tags=examples pkg/grpc/examples/cmd/server/main.go

   # Then run client in another terminal
   go run -tags=examples pkg/grpc/examples/cmd/client/main.go
   ```

### Debug Mode

Enable verbose gRPC logging:

```bash
GRPC_GO_LOG_VERBOSITY_LEVEL=2 GRPC_GO_LOG_SEVERITY_LEVEL=info \
  go run -tags=examples pkg/grpc/examples/cmd/server/main.go
```

## Next Steps

1. **Add Authentication**: Implement gRPC interceptors for auth
2. **Add Validation**: Use protobuf validation rules
3. **Add Database**: Persist calculations to a database
4. **Add REST Gateway**: Use grpc-gateway for REST API
5. **Add Load Balancing**: Deploy multiple server instances

## References

- [gRPC Go Documentation](https://grpc.io/docs/languages/go/)
- [Protocol Buffers Guide](https://developers.google.com/protocol-buffers)
- [gRPC Streaming Guide](https://grpc.io/docs/what-is-grpc/core-concepts/#server-streaming-rpc)
- [pkg/grpc Package](../README.md) - Our reusable gRPC component