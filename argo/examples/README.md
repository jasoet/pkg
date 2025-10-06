# Argo Workflows Client Examples

This directory contains comprehensive examples demonstrating various ways to use the Argo Workflows client library.

## Prerequisites

1. **Kubernetes Cluster**: Access to a Kubernetes cluster (local or remote)
2. **Argo Workflows**: Argo Workflows installed in the cluster
3. **Kubeconfig**: Valid kubeconfig file (usually at `~/.kube/config`)

### Optional

- **Argo Server**: For Argo Server mode examples
- **OpenTelemetry**: For observability examples

## Running the Examples

### Build and Run

```bash
# From the pkg root directory
go run -tags=example ./argo/examples

# Or build first
go build -tags=example -o argo-example ./argo/examples
./argo-example
```

### Environment Variables

Some examples use environment variables for configuration:

```bash
# Kubeconfig path (default: ~/.kube/config)
export KUBECONFIG=/path/to/kubeconfig

# Argo Server URL (for Argo Server mode)
export ARGO_SERVER_URL=https://argo-server:2746

# Argo authentication token (for Argo Server mode)
export ARGO_AUTH_TOKEN="Bearer your-token-here"
```

## Examples Overview

### Example 1: Default Configuration

Demonstrates the simplest way to create an Argo client using default settings.

```go
ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())
if err != nil {
    return err
}
```

**Features:**
- Uses default kubeconfig location (`~/.kube/config`)
- Uses current context from kubeconfig
- Connects via Kubernetes API

### Example 2: Functional Options

Shows how to use functional options for more flexible configuration.

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/custom/path/kubeconfig"),
    argo.WithContext("production"),
)
```

**Features:**
- Custom kubeconfig path
- Custom context selection
- Demonstrates functional options pattern

### Example 3: In-Cluster Configuration

Demonstrates how to use the client when running inside a Kubernetes pod.

```go
ctx, client, err := argo.NewClient(ctx, argo.InClusterConfig())
if err != nil {
    return err
}
```

**Features:**
- Uses in-cluster service account
- No kubeconfig required
- Ideal for containerized applications

**Note:** This example is commented out in the code as it requires running inside a Kubernetes pod.

### Example 4: Argo Server Mode

Shows how to connect via Argo Server HTTP API instead of Kubernetes API.

```go
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithArgoServer("https://argo-server:2746", "Bearer token"),
)
```

**Features:**
- HTTP/HTTPS connection to Argo Server
- Authentication with bearer token
- Alternative to direct Kubernetes API access

**Note:** This example is commented out in the code as it requires a running Argo Server.

### Example 5: With OpenTelemetry

Demonstrates how to enable OpenTelemetry instrumentation for tracing and monitoring.

```go
otelConfig := otel.NewConfig("my-service")
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithOTelConfig(otelConfig),
)
```

**Features:**
- Distributed tracing
- Metrics collection
- Observability integration

### Example 6: Create and Submit Workflow

Shows how to programmatically create and submit a workflow.

```go
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
```

**Features:**
- Workflow definition
- Workflow submission
- Error handling

## Helper Functions

### listWorkflows

A reusable helper function that demonstrates how to list workflows:

```go
func listWorkflows(ctx context.Context, client apiclient.Client) error {
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

    return nil
}
```

## Common Operations

### List Workflows

```go
wfClient := client.NewWorkflowServiceClient()
resp, err := wfClient.ListWorkflows(ctx, &workflow.WorkflowListRequest{
    Namespace: "argo",
})
```

### Get Workflow

```go
wfClient := client.NewWorkflowServiceClient()
wf, err := wfClient.GetWorkflow(ctx, &workflow.WorkflowGetRequest{
    Namespace: "argo",
    Name:      "workflow-name",
})
```

### Watch Workflows

```go
wfClient := client.NewWorkflowServiceClient()
stream, err := wfClient.WatchWorkflows(ctx, &workflow.WatchWorkflowsRequest{
    Namespace: "argo",
})

for {
    event, err := stream.Recv()
    if err == io.EOF {
        break
    }
    // Handle event
}
```

### Delete Workflow

```go
wfClient := client.NewWorkflowServiceClient()
_, err := wfClient.DeleteWorkflow(ctx, &workflow.WorkflowDeleteRequest{
    Namespace: "argo",
    Name:      "workflow-name",
})
```

## Expected Output

When you run the examples, you should see output similar to:

```
{"level":"info","time":"2025-10-06T10:00:00Z","message":"Starting Argo Workflows client example"}
{"level":"info","time":"2025-10-06T10:00:00Z","message":"=== Example 1: Default Configuration ==="}
{"level":"debug","message":"Creating Argo Workflows client"}
{"level":"info","count":5,"message":"Listed workflows"}
{"level":"info","index":1,"name":"hello-world-abc123","namespace":"argo","status":"Succeeded","message":"Workflow"}
{"level":"info","time":"2025-10-06T10:00:01Z","message":"Example 1 completed"}
...
{"level":"info","time":"2025-10-06T10:00:05Z","message":"Examples completed successfully"}
```

## Troubleshooting

### "Failed to create client"

**Cause:** Kubeconfig not found or invalid

**Solution:**
```bash
# Check kubeconfig exists
ls -la ~/.kube/config

# Test kubectl works
kubectl cluster-info

# Set KUBECONFIG if needed
export KUBECONFIG=/path/to/kubeconfig
```

### "Failed to list workflows"

**Cause:** Namespace doesn't exist or no permissions

**Solution:**
```bash
# Create argo namespace
kubectl create namespace argo

# Check permissions
kubectl auth can-i list workflows -n argo
```

### "Connection refused"

**Cause:** Kubernetes cluster not accessible

**Solution:**
```bash
# Test cluster connectivity
kubectl get nodes

# Check cluster is running
kubectl cluster-info
```

## Customizing Examples

### Change Namespace

Edit the namespace in the code:

```go
resp, err := wfClient.ListWorkflows(ctx, &workflow.WorkflowListRequest{
    Namespace: "my-namespace",  // Change this
})
```

### Add More Workflow Operations

Refer to the [Argo Workflows API documentation](https://argo-workflows.readthedocs.io/en/latest/swagger/) for more operations:

- `CreateWorkflow`
- `GetWorkflow`
- `DeleteWorkflow`
- `RetryWorkflow`
- `ResubmitWorkflow`
- `SuspendWorkflow`
- `ResumeWorkflow`
- `TerminateWorkflow`

## Additional Resources

- [Argo Workflows Documentation](https://argo-workflows.readthedocs.io/)
- [Argo Workflows Examples](https://github.com/argoproj/argo-workflows/tree/master/examples)
- [Kubernetes Client-Go](https://github.com/kubernetes/client-go)
- [pkg Library Documentation](../../README.md)

## Contributing

Found a bug or want to add more examples? Contributions are welcome!

1. Fork the repository
2. Create a feature branch
3. Add your example
4. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](../../LICENSE) file for details.

---

**[⬆ Back to Top](#argo-workflows-client-examples)**
