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

## Workflow Builder API

The workflow builder API provides a high-level, fluent interface for constructing Argo Workflows without needing to understand the low-level protobuf-generated structs. It includes template sources, pre-built patterns, and full OpenTelemetry instrumentation.

### Quick Start with Builder

```go
import (
    "github.com/jasoet/pkg/v2/argo/builder"
    "github.com/jasoet/pkg/v2/argo/builder/template"
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

// Submit workflow
created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
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
    builder.WithActiveDeadline(3600), // 1 hour timeout

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
import "github.com/jasoet/pkg/v2/argo/patterns"

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
    "wc -w",                          // map command
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

Higher-level operations with full OpenTelemetry instrumentation:

#### Submit Workflow

```go
import "github.com/jasoet/pkg/v2/argo"

wf, err := builder.NewWorkflowBuilder("deploy", "argo").
    Add(deployStep).
    Build()
if err != nil {
    return err
}

created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
if err != nil {
    return err
}

fmt.Printf("Workflow %s submitted\n", created.Name)
```

#### Submit and Wait

Submit a workflow and wait for completion with automatic polling:

```go
completed, err := argo.SubmitAndWait(ctx, client, wf, otelConfig, 10*time.Minute)
if err != nil {
    return err
}

if completed.Status.Phase == v1alpha1.WorkflowSucceeded {
    fmt.Println("Workflow completed successfully")
}
```

#### Get Workflow Status

```go
status, err := argo.GetWorkflowStatus(ctx, client, "argo", "my-workflow-abc123", otelConfig)
if err != nil {
    return err
}

fmt.Printf("Phase: %s\n", status.Phase)
fmt.Printf("Progress: %s\n", status.Progress)
```

#### List Workflows

```go
// List all workflows
workflows, err := argo.ListWorkflows(ctx, client, "argo", "", otelConfig)

// List with label selector
workflows, err := argo.ListWorkflows(ctx, client, "argo", "app=myapp", otelConfig)
```

#### Delete Workflow

```go
err := argo.DeleteWorkflow(ctx, client, "argo", "my-workflow-abc123", otelConfig)
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
    "time"

    "github.com/jasoet/pkg/v2/argo"
    "github.com/jasoet/pkg/v2/argo/builder"
    "github.com/jasoet/pkg/v2/argo/builder/template"
    "github.com/jasoet/pkg/v2/otel"
)

func main() {
    ctx := context.Background()

    // Create OTel config
    otelConfig := otel.NewConfig("workflow-manager")

    // Create Argo client
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

    // Submit and wait
    completed, err := argo.SubmitAndWait(ctx, client, wf, otelConfig, 10*time.Minute)
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

```bash
# Run the comprehensive example
go run -tags=example ./examples/argo

# Or build and run
go build -tags=example -o argo-example ./examples/argo
./argo-example
```

See [examples/argo/README.md](../examples/argo/README.md) for more details.

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
