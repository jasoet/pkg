# Argo Workflows Client

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Argo Workflows client library with flexible configuration, context-based OpenTelemetry propagation, and proper error handling.

## Package Posture

This package is an **SDK integration**: it exposes `argo-workflows` types (`apiclient.Client`, `v1alpha1.Workflow`, `workflow.*Request`) directly by design, rather than wrapping them behind an abstraction layer. It adds value on top of the raw SDK: unified client construction (kubeconfig / in-cluster / Argo Server), a fluent workflow builder, pre-built patterns, and optional OpenTelemetry instrumentation. If you need an SDK operation that is not wrapped here, use `client.NewWorkflowServiceClient()` directly.

## Features

- **Multiple Connection Modes**: Kubernetes API, In-Cluster, or Argo Server HTTP
- **Flexible Configuration**: Config structs and functional options
- **Context-Based OpenTelemetry**: OTel config propagates through `context.Context` — set it once at client creation, operations pick it up automatically
- **Production-Ready**: Proper error handling, no fatal errors
- **Well-Documented**: Comprehensive examples and documentation

## Installation

```bash
go get github.com/jasoet/pkg/v3/argo
```

## Quick Start

### Basic Usage (Default Configuration)

```go
package main

import (
    "context"

    "github.com/jasoet/pkg/v3/argo"
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
ctx, client, err := argo.NewClient(ctx,
    argo.ServerConfig("https://argo-server:2746", "Bearer token"),
)

// Or using functional options
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
    KubeConfigPath string `yaml:"kubeConfigPath" mapstructure:"kubeConfigPath"`

    // Context specifies the kubeconfig context to use
    Context string `yaml:"context" mapstructure:"context"`

    // InCluster indicates whether to use in-cluster configuration
    InCluster bool `yaml:"inCluster" mapstructure:"inCluster"`

    // ArgoServerOpts configures connection to Argo Server
    ArgoServerOpts ServerOpts `yaml:"argoServer" mapstructure:"argoServer"`

    // OTelConfig enables OpenTelemetry instrumentation
    OTelConfig *otel.Config `yaml:"-" mapstructure:"-"`
}
```

### Pre-configured Factories

```go
// Default configuration (uses ~/.kube/config)
config := argo.DefaultConfig()

// In-cluster configuration
config := argo.InClusterConfig()

// Argo Server configuration
config := argo.ServerConfig("https://argo-server:2746", "Bearer token")
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
    argo.WithArgoServerOpts(argo.ServerOpts{
        URL:       "https://argo-server:2746",
        AuthToken: "Bearer token",
    }),

    // Apply a complete pre-built config
    argo.WithConfig(myConfig),

    // Observability
    argo.WithOTelConfig(otelConfig),
)
```

## OpenTelemetry Integration

OTel configuration propagates through `context.Context`. `NewClient` / `NewClientWithOptions` inject the configured `*otel.Config` into the context they return; the package operations (`SubmitWorkflow`, `SubmitAndWait`, `GetWorkflowStatus`, `ListWorkflows`, `DeleteWorkflow`) resolve it from the context via `otel.ConfigFromContext(ctx)`. Pass the returned `ctx` to operations and instrumentation is automatic — no per-call config argument.

```go
import (
    "github.com/jasoet/pkg/v3/argo"
    "github.com/jasoet/pkg/v3/otel"
)

// Create OTel config
otelConfig := otel.NewConfig("my-service",
    otel.WithTracerProvider(tracerProvider),
    otel.WithMeterProvider(meterProvider),
)

// Create Argo client with OTel — the returned ctx carries the config
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/path/to/kubeconfig"),
    argo.WithOTelConfig(otelConfig),
)
if err != nil {
    return err
}

// Operations read the OTel config from ctx
created, err := argo.SubmitWorkflow(ctx, client, wf)
```

You can also inject a config into any context manually:

```go
ctx = otel.ContextWithConfig(ctx, otelConfig)
```

## Working with Workflows

The examples below use the raw Argo SDK client (`client.NewWorkflowServiceClient()`). For the higher-level instrumented wrappers, see [Enhanced Client Operations](#enhanced-client-operations).

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

## Workflow Builder API

The workflow builder API provides a high-level, fluent interface for constructing Argo Workflows without needing to understand the low-level protobuf-generated structs. It includes template sources, pre-built patterns, and optional OpenTelemetry instrumentation.

### Quick Start with Builder

```go
import (
    "github.com/jasoet/pkg/v3/argo"
    "github.com/jasoet/pkg/v3/argo/builder"
    "github.com/jasoet/pkg/v3/argo/builder/template"
)

// Create workflow steps
deploy := template.NewContainer("deploy", "myapp:v1",
    template.WithCommand("deploy.sh"),
    template.WithEnv("ENV", "production"))

healthCheck := template.NewHTTP("health-check",
    template.WithHTTPURL("https://myapp.com/health"),
    template.WithHTTPMethod("GET"))

// Build workflow
wf, err := builder.NewWorkflowBuilder("deployment", "argo",
    builder.WithServiceAccount("argo-workflow"),
    builder.WithLabels(map[string]string{"app": "myapp"})).
    Add(deploy).
    Add(healthCheck).
    Build()

if err != nil {
    return err
}

// Submit workflow (ctx carries the OTel config if one was set)
created, err := argo.SubmitWorkflow(ctx, client, wf)
```

### Template Sources

Template sources are composable workflow components that implement the `WorkflowSource` interface.

#### Container Template

Execute commands in containers:

```go
container := template.NewContainer("build", "golang:1.25",
    template.WithCommand("go", "build", "-o", "app"),
    template.WithWorkingDir("/workspace"),
    template.WithEnv("CGO_ENABLED", "0"),
    template.WithCPU("1000m"),
    template.WithMemory("2Gi"))
```

#### Script Template

Run inline scripts in various languages:

```go
// Bash script
bashScript := template.NewScript("backup", "bash",
    template.WithScriptContent(`
        echo "Creating backup..."
        tar -czf backup.tar.gz /data
        echo "Backup complete"
    `),
    template.WithScriptWorkingDir("/backup"))

// Python script
pythonScript := template.NewScript("process", "python",
    template.WithScriptContent(`
        import json
        print("Processing data...")
        # Your Python code here
    `))

// Custom image with specific command
customScript := template.NewScript("custom", "bash",
    template.WithScriptImage("myregistry/custom:v1"),
    template.WithScriptCommand("bash", "-x"),
    template.WithScriptContent("echo 'Custom script'"))
```

#### HTTP Template

Make HTTP requests for health checks, webhooks, or API calls:

```go
healthCheck := template.NewHTTP("api-check",
    template.WithHTTPURL("https://api.example.com/health"),
    template.WithHTTPMethod("GET"),
    template.WithHTTPSuccessCond("response.statusCode == 200"),
    template.WithHTTPTimeout(30))

webhook := template.NewHTTP("notify",
    template.WithHTTPURL("https://hooks.slack.com/services/..."),
    template.WithHTTPMethod("POST"),
    template.WithHTTPHeader("Content-Type", "application/json"),
    template.WithHTTPBody(`{"text": "Deployment complete"}`))
```

### Workflow Builder Options

Configure workflows with functional options:

```go
wf, err := builder.NewWorkflowBuilder("myworkflow", "argo",
    // Service Account
    builder.WithServiceAccount("argo-workflow"),

    // Labels and Annotations
    builder.WithLabels(map[string]string{
        "app": "myapp",
        "env": "production",
    }),
    builder.WithAnnotations(map[string]string{
        "description": "Production deployment",
    }),

    // Resource Management
    builder.WithArchiveLogs(true),
    builder.WithActiveDeadlineSeconds(3600), // 1 hour timeout

    // Retry Strategy
    builder.WithRetryStrategy(&v1alpha1.RetryStrategy{
        Limit:       intstr.FromInt(3),
        RetryPolicy: "Always",
    }),

    // Volumes
    builder.WithVolume(corev1.Volume{
        Name: "data",
        VolumeSource: corev1.VolumeSource{
            EmptyDir: &corev1.EmptyDirVolumeSource{},
        },
    }),

    // Garbage collection and TTL
    builder.WithPodGC(&v1alpha1.PodGC{Strategy: v1alpha1.PodGCOnWorkflowSuccess}),
    builder.WithTTL(&v1alpha1.TTLStrategy{SecondsAfterCompletion: &ttl}),

    // OpenTelemetry
    builder.WithOTelConfig(otelConfig),
).Build()
```

### Exit Handlers

Add cleanup steps that always run, regardless of workflow success or failure:

```go
// Main workflow steps
deploy := template.NewContainer("deploy", "myapp:v1",
    template.WithCommand("deploy.sh"))

// Cleanup step (always runs)
cleanup := template.NewScript("cleanup", "bash",
    template.WithScriptContent("echo 'Cleaning up resources...'"))

// Notification (always runs)
notify := template.NewScript("notify", "bash",
    template.WithScriptContent(`
        echo "Workflow Status: {{workflow.status}}"
        echo "Duration: {{workflow.duration}}"
    `))

wf, err := builder.NewWorkflowBuilder("deployment", "argo").
    Add(deploy).
    AddExitHandler(cleanup).
    AddExitHandler(notify).
    Build()
```

### Pre-Built Workflow Patterns

#### CI/CD Patterns

##### Build-Test-Deploy

```go
import "github.com/jasoet/pkg/v3/argo/patterns"

wf, err := patterns.BuildTestDeploy(
    "myapp", "argo",
    "golang:1.25",      // build image
    "golang:1.25",      // test image
    "deployer:v1",      // deploy image
    builder.WithServiceAccount("argo-workflow"),
)
```

##### Build-Test-Deploy with Cleanup

```go
wf, err := patterns.BuildTestDeployWithCleanup(
    "myapp", "argo",
    "golang:1.25",
    "busybox:latest",
    builder.WithArchiveLogs(true),
)
```

##### Conditional Deployment

Deploy only if tests pass, with automatic rollback on failure:

```go
wf, err := patterns.ConditionalDeploy(
    "safe-deploy", "argo",
    "golang:1.25",
)
```

##### Multi-Environment Deployment

Deploy sequentially to multiple environments:

```go
wf, err := patterns.MultiEnvironmentDeploy(
    "multi-env", "argo",
    "deployer:v1",
    []string{"staging", "production"},
)
```

#### Parallel Execution Patterns

##### Fan-Out/Fan-In

Execute multiple tasks in parallel, then aggregate results:

```go
wf, err := patterns.FanOutFanIn(
    "parallel-tasks", "argo",
    "busybox:latest",
    []string{"task-1", "task-2", "task-3"},
)
```

##### Parallel Data Processing

Process multiple data items independently:

```go
wf, err := patterns.ParallelDataProcessing(
    "batch-process", "argo",
    "processor:v1",
    []string{"data-1.csv", "data-2.csv", "data-3.csv"},
    "process.sh",
)
```

##### Map-Reduce

Classic map-reduce pattern with parallel mapping and sequential reduction:

```go
wf, err := patterns.MapReduce(
    "word-count", "argo",
    "alpine:latest",
    []string{"file1.txt", "file2.txt", "file3.txt"},
    "wc -w",                           // map command
    "awk '{sum+=$1} END {print sum}'", // reduce command
)
```

##### Parallel Test Suites

Run multiple test suites in parallel to speed up CI/CD:

```go
wf, err := patterns.ParallelTestSuite(
    "tests", "argo",
    "golang:1.25",
    map[string]string{
        "unit":        "go test ./internal/...",
        "integration": "go test ./tests/integration/...",
        "e2e":         "go test ./tests/e2e/...",
    },
)
```

##### Parallel Deployment

Deploy to multiple regions/environments simultaneously:

```go
wf, err := patterns.ParallelDeployment(
    "multi-region", "argo",
    "deployer:v1",
    []string{"us-west", "us-east", "eu-central"},
)
```

### Enhanced Client Operations

Higher-level operations with optional OpenTelemetry instrumentation. None of them take an OTel config argument — they resolve it from `ctx` via `otel.ConfigFromContext(ctx)`. When the `ctx` came from `NewClient` / `NewClientWithOptions` configured with `WithOTelConfig`, instrumentation is automatic.

#### Submit Workflow

```go
import "github.com/jasoet/pkg/v3/argo"

wf, err := builder.NewWorkflowBuilder("deploy", "argo").
    Add(deployStep).
    Build()
if err != nil {
    return err
}

created, err := argo.SubmitWorkflow(ctx, client, wf)
if err != nil {
    return err
}

fmt.Printf("Workflow %s submitted\n", created.Name)
```

#### Submit and Wait

Submit a workflow and wait for completion with automatic polling:

```go
completed, err := argo.SubmitAndWait(ctx, client, wf, 10*time.Minute)
if err != nil {
    return err
}

if completed.Status.Phase == v1alpha1.WorkflowSucceeded {
    fmt.Println("Workflow completed successfully")
}
```

#### Get Workflow Status

```go
status, err := argo.GetWorkflowStatus(ctx, client, "argo", "my-workflow-abc123")
if err != nil {
    return err
}

fmt.Printf("Phase: %s\n", status.Phase)
fmt.Printf("Progress: %s\n", status.Progress)
```

#### List Workflows

```go
// List all workflows
workflows, err := argo.ListWorkflows(ctx, client, "argo", "")

// List with label selector
workflows, err := argo.ListWorkflows(ctx, client, "argo", "app=myapp")
```

#### Delete Workflow

```go
err := argo.DeleteWorkflow(ctx, client, "argo", "my-workflow-abc123")
if err != nil {
    return err
}
```

### Advanced: Custom Templates

For advanced use cases, manually construct templates:

```go
import "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"

// Create custom template
customTemplate := v1alpha1.Template{
    Name: "custom-step",
    Steps: [][]v1alpha1.WorkflowStep{
        {
            {Name: "step-1", Template: "step-1-template"},
            {Name: "step-2", Template: "step-2-template"},
        },
    },
}

// Add to builder
wf, err := builder.NewWorkflowBuilder("advanced", "argo").
    AddTemplate(customTemplate).
    BuildWithEntrypoint("custom-step")
```

### Complete Example

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/jasoet/pkg/v3/argo"
    "github.com/jasoet/pkg/v3/argo/builder"
    "github.com/jasoet/pkg/v3/argo/builder/template"
    "github.com/jasoet/pkg/v3/otel"
)

func main() {
    ctx := context.Background()

    // Create OTel config
    otelConfig := otel.NewConfig("workflow-manager")

    // Create Argo client — returned ctx carries the OTel config
    ctx, client, err := argo.NewClientWithOptions(ctx,
        argo.WithOTelConfig(otelConfig))
    if err != nil {
        panic(err)
    }

    // Build workflow
    preCheck := template.NewContainer("pre-check", "alpine:latest",
        template.WithCommand("sh", "-c", "echo 'Pre-flight checks...'"))

    deploy := template.NewContainer("deploy", "myapp:v1",
        template.WithCommand("deploy.sh"),
        template.WithEnv("ENV", "production"),
        template.WithCPU("500m"),
        template.WithMemory("256Mi"))

    healthCheck := template.NewHTTP("health-check",
        template.WithHTTPURL("https://myapp.com/health"),
        template.WithHTTPMethod("GET"),
        template.WithHTTPSuccessCond("response.statusCode == 200"))

    notify := template.NewScript("notify", "bash",
        template.WithScriptContent(`
            echo "Deployment Status: {{workflow.status}}"
        `))

    wf, err := builder.NewWorkflowBuilder("deployment", "argo",
        builder.WithOTelConfig(otelConfig),
        builder.WithServiceAccount("argo-workflow"),
        builder.WithLabels(map[string]string{"app": "myapp"}),
        builder.WithArchiveLogs(true)).
        Add(preCheck).
        Add(deploy).
        Add(healthCheck).
        AddExitHandler(notify).
        Build()

    if err != nil {
        panic(err)
    }

    // Submit and wait — OTel config resolved from ctx
    completed, err := argo.SubmitAndWait(ctx, client, wf, 10*time.Minute)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Workflow completed: %s\n", completed.Status.Phase)
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

Runnable examples live under `examples/argo/`, one directory per topic:

```bash
# Run the basic client example
go run -tags=example ./examples/argo/basic

# Or build and run
go build -tags=example -o argo-example ./examples/argo/basic
./argo-example
```

See [examples/argo/README.md](../examples/argo/README.md) for more details.

## Migration Notes

### Migrating from v2 (positional OTel config)

The five package operations no longer take a positional `*otel.Config` argument. They resolve instrumentation from the context:

```go
// Before (v2)
created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
completed, err := argo.SubmitAndWait(ctx, client, wf, otelConfig, 10*time.Minute)
status, err := argo.GetWorkflowStatus(ctx, client, "argo", name, otelConfig)
workflows, err := argo.ListWorkflows(ctx, client, "argo", "", otelConfig)
err = argo.DeleteWorkflow(ctx, client, "argo", name, otelConfig)

// After (v3) — set the config once when creating the client
ctx, client, err := argo.NewClientWithOptions(ctx, argo.WithOTelConfig(otelConfig))

created, err := argo.SubmitWorkflow(ctx, client, wf)
completed, err := argo.SubmitAndWait(ctx, client, wf, 10*time.Minute)
status, err := argo.GetWorkflowStatus(ctx, client, "argo", name)
workflows, err := argo.ListWorkflows(ctx, client, "argo", "")
err = argo.DeleteWorkflow(ctx, client, "argo", name)
```

If you construct clients without `WithOTelConfig` but still want instrumented calls, inject the config manually:

```go
ctx = otel.ContextWithConfig(ctx, otelConfig)
```

Also note: `argo.Option` no longer returns an error (it is now `func(*Config)`). The Argo Server config factory remains `argo.ServerConfig(...)` and the builder timeout option remains `builder.WithActiveDeadlineSeconds(seconds int64)` (unchanged from v2). Finally, instrument operations by threading the ctx returned by `NewClient`/`NewClientWithOptions` — calling operations with a fresh `context.Background()` compiles fine but silently disables instrumentation.

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
