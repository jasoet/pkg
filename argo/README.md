# Argo Workflows Client

[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Production-ready Argo Workflows client library with flexible configuration, OpenTelemetry support, and comprehensive error handling.

## Features

- **Multiple Connection Modes**: Kubernetes API, In-Cluster, or Argo Server HTTP
- **Flexible Configuration**: Config structs and functional options
- **OpenTelemetry Integration**: Built-in tracing and observability
- **Production-Ready**: Proper error handling, no fatal errors
- **Type-Safe**: Full Go type safety with generics support
- **Well-Documented**: Comprehensive examples and documentation

## Installation

```bash
go get github.com/jasoet/pkg/v2/argo
```

## Quick Start

### Basic Usage (Default Configuration)

```go
package main

import (
    "context"
    "github.com/jasoet/pkg/v2/argo"
)

func main() {
    ctx := context.Background()

    // Create client using default kubeconfig (~/.kube/config)
    ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())
    if err != nil {
        panic(err)
    }

    // Use the client
    wfClient := client.NewWorkflowServiceClient()
    // ... interact with workflows
}
```

### Using Functional Options

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/custom/path/kubeconfig"),
    argo.WithContext("production"),
)
if err != nil {
    return err
}
```

## Connection Modes

### 1. Kubernetes API Mode (Default)

Uses kubeconfig file to connect to Kubernetes API server.

```go
// Use default kubeconfig location (~/.kube/config)
ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())

// Or specify custom kubeconfig path
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/path/to/kubeconfig"),
    argo.WithContext("my-context"),
)
```

### 2. In-Cluster Mode

Use when running inside a Kubernetes pod.

```go
ctx, client, err := argo.NewClient(ctx, argo.InClusterConfig())

// Or using functional options
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithInCluster(true),
)
```

### 3. Argo Server Mode

Connect via Argo Server HTTP API.

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithArgoServer("https://argo-server:2746", "Bearer token"),
)

// For development/testing with HTTP
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithArgoServer("http://argo-server:2746", ""),
    argo.WithArgoServerInsecure(true),
)
```

## Configuration

### Config Struct

```go
type Config struct {
    // KubeConfigPath specifies the path to kubeconfig file
    KubeConfigPath string

    // Context specifies the kubeconfig context to use
    Context string

    // InCluster indicates whether to use in-cluster configuration
    InCluster bool

    // ArgoServerOpts configures connection to Argo Server
    ArgoServerOpts ArgoServerOpts

    // OTelConfig enables OpenTelemetry instrumentation
    OTelConfig *otel.Config
}
```

### Pre-configured Factories

```go
// Default configuration (uses ~/.kube/config)
config := argo.DefaultConfig()

// In-cluster configuration
config := argo.InClusterConfig()

// Argo Server configuration
config := argo.ArgoServerConfig("https://argo-server:2746", "Bearer token")
```

## Functional Options

All available functional options:

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    // Kubernetes API options
    argo.WithKubeConfig("/path/to/kubeconfig"),
    argo.WithContext("production"),
    argo.WithInCluster(true),

    // Argo Server options
    argo.WithArgoServer("https://argo-server:2746", "Bearer token"),
    argo.WithArgoServerInsecure(false),
    argo.WithArgoServerHTTP1(false),

    // Observability
    argo.WithOTelConfig(otelConfig),
)
```

## OpenTelemetry Integration

Enable distributed tracing and monitoring:

```go
import (
    "github.com/jasoet/pkg/v2/argo"
    "github.com/jasoet/pkg/v2/otel"
)

// Create OTel config
otelConfig := otel.NewConfig("my-service").
    WithTracerProvider(tracerProvider).
    WithMeterProvider(meterProvider)

// Create Argo client with OTel
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/path/to/kubeconfig"),
    argo.WithOTelConfig(otelConfig),
)
```

## Working with Workflows

### List Workflows

```go
import (
    "github.com/argoproj/argo-workflows/v3/pkg/apiclient/workflow"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

wfClient := client.NewWorkflowServiceClient()

resp, err := wfClient.ListWorkflows(ctx, &workflow.WorkflowListRequest{
    Namespace: "argo",
    ListOptions: &metav1.ListOptions{
        Limit: 10,
    },
})
if err != nil {
    return err
}

for _, wf := range resp.Items {
    fmt.Printf("Workflow: %s, Status: %s\n", wf.Name, wf.Status.Phase)
}
```

### Create a Workflow

```go
import (
    "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

wf := &v1alpha1.Workflow{
    ObjectMeta: metav1.ObjectMeta{
        GenerateName: "hello-world-",
        Namespace:    "argo",
    },
    Spec: v1alpha1.WorkflowSpec{
        Entrypoint: "hello",
        Templates: []v1alpha1.Template{
            {
                Name: "hello",
                Container: &corev1.Container{
                    Image:   "alpine:latest",
                    Command: []string{"echo"},
                    Args:    []string{"Hello, Argo!"},
                },
            },
        },
    },
}

wfClient := client.NewWorkflowServiceClient()
created, err := wfClient.CreateWorkflow(ctx, &workflow.WorkflowCreateRequest{
    Namespace: "argo",
    Workflow:  wf,
})
if err != nil {
    return err
}

fmt.Printf("Created workflow: %s\n", created.Name)
```

### Watch Workflows

```go
import (
    "io"
)

wfClient := client.NewWorkflowServiceClient()

stream, err := wfClient.WatchWorkflows(ctx, &workflow.WatchWorkflowsRequest{
    Namespace: "argo",
    ListOptions: &metav1.ListOptions{
        Watch: true,
    },
})
if err != nil {
    return err
}

for {
    event, err := stream.Recv()
    if err == io.EOF {
        break
    }
    if err != nil {
        return err
    }

    fmt.Printf("Workflow %s: %s\n", event.Object.Name, event.Object.Status.Phase)
}
```

## Error Handling

The library uses proper error handling without fatal errors:

```go
ctx, client, err := argo.NewClient(ctx, config)
if err != nil {
    // Handle error gracefully
    log.Error().Err(err).Msg("Failed to create Argo client")
    return err
}

// Always check errors from operations
wfClient := client.NewWorkflowServiceClient()
resp, err := wfClient.ListWorkflows(ctx, req)
if err != nil {
    log.Error().Err(err).Msg("Failed to list workflows")
    return err
}
```

## Examples

### Example 1: Default Configuration

```go
ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())
if err != nil {
    return err
}
```

### Example 2: Custom Kubeconfig and Context

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/etc/kubernetes/admin.conf"),
    argo.WithContext("prod-cluster"),
)
```

### Example 3: In-Cluster Usage

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithInCluster(true),
)
```

### Example 4: Argo Server with Authentication

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithArgoServer("https://argo-server.example.com", "Bearer my-token"),
)
```

### Example 5: With OpenTelemetry

```go
otelConfig := otel.NewConfig("workflow-manager")
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/etc/kubernetes/admin.conf"),
    argo.WithOTelConfig(otelConfig),
)
```

## Running Examples

```bash
# Run the comprehensive example
go run -tags=example ./argo/examples

# Or build and run
go build -tags=example -o argo-example ./argo/examples
./argo-example
```

See [examples/README.md](examples/README.md) for more details.

## Comparison with Original Implementation

### Before (scp/api)

```go
// util/argo/argo.go - tightly coupled, uses fatal errors
func NewClient(ctx context.Context) (context.Context, apiclient.Client) {
    ctx, argoClient, err := apiclient.NewClientFromOpts(
        apiclient.Opts{
            ArgoServerOpts:       apiclient.ArgoServerOpts{},
            ClientConfigSupplier: kube.GetCmdConfig,
            Context:              ctx,
        })
    if err != nil {
        log.Fatal().Err(err).Msg("unable to create argo client")  // Fatal!
    }
    return ctx, argoClient
}
```

### After (pkg/v2/argo)

```go
// Flexible, reusable, proper error handling
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/path/to/kubeconfig"),
    argo.WithContext("production"),
    argo.WithOTelConfig(otelConfig),
)
if err != nil {
    return fmt.Errorf("failed to create client: %w", err)  // Graceful!
}
defer client.Close()
```

## Benefits

✅ **Reusable** - Can be used across multiple projects
✅ **Flexible** - Config struct + functional options
✅ **Library-friendly** - Returns errors instead of fatal
✅ **Testable** - Easy to mock and test
✅ **Observable** - OpenTelemetry integration ready
✅ **Well-documented** - Comprehensive docs and examples
✅ **Production-ready** - Proper error handling and logging

## Best Practices

1. **Use context for lifecycle management**
   ```go
   ctx, cancel := context.WithCancel(context.Background())
   defer cancel() // Clean up context when done

   ctx, client, err := argo.NewClient(ctx, config)
   if err != nil {
       return err
   }
   ```

2. **Use functional options for flexibility**
   ```go
   ctx, client, err := argo.NewClientWithOptions(ctx,
       argo.WithKubeConfig(kubeconfigPath),
       argo.WithContext(contextName),
   )
   ```

3. **Enable OpenTelemetry in production**
   ```go
   ctx, client, err := argo.NewClientWithOptions(ctx,
       argo.WithOTelConfig(otelConfig),
   )
   ```

4. **Handle errors gracefully**
   ```go
   if err != nil {
       log.Error().Err(err).Msg("Operation failed")
       return fmt.Errorf("operation failed: %w", err)
   }
   ```

## Testing

```bash
# Unit tests
go test ./argo

# Integration tests (requires Kubernetes cluster)
go test -tags=integration ./argo

# All tests with coverage
go test -cover ./argo
```

## Contributing

Contributions are welcome! Please see the main [CONTRIBUTING.md](../CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](../LICENSE) file for details.

## Related Documentation

- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)
- [pkg Library Documentation](../README.md)

---

**[⬆ Back to Top](#argo-workflows-client)**
