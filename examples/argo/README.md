# Argo Workflows Client Examples

This directory contains comprehensive examples demonstrating various ways to use the Argo Workflows client library. The examples cover everything from basic client configuration to advanced workflow patterns and production-ready deployments.

## Prerequisites

1. **Kubernetes Cluster**: Access to a Kubernetes cluster (local or remote)
2. **Argo Workflows**: Argo Workflows installed in the cluster
3. **Kubeconfig**: Valid kubeconfig file (usually at `~/.kube/config`)

### Optional

- **Argo Server**: For Argo Server mode examples
- **OpenTelemetry**: For observability examples

## Example Files Overview

| File | Description | Topics Covered |
|------|-------------|----------------|
| `main.go` | Client configuration and basic usage | Client initialization, connection modes, kubeconfig |
| `builder_example.go` | WorkflowBuilder API usage | Building workflows, sequential steps, exit handlers |
| `operations_example.go` | Workflow operations and lifecycle | Submit, SubmitAndWait, GetStatus, List, Delete |
| `templates_example.go` | All template types | Container, Script, HTTP, Noop templates |
| `advanced_features_example.go` | Advanced workflow features | Parameters, retry, volumes, TTL, metrics |
| `patterns_example.go` | Common workflow patterns | CI/CD, ETL, microservices, ML pipelines |

## Running the Examples

### Build and Run

Each example file can be run independently:

```bash
# Run client configuration examples
go run -tags=example ./argo/examples/main.go

# Run workflow builder examples
go run -tags=example ./argo/examples/builder_example.go

# Run operations examples
go run -tags=example ./argo/examples/operations_example.go

# Run template examples
go run -tags=example ./argo/examples/templates_example.go

# Run advanced features examples
go run -tags=example ./argo/examples/advanced_features_example.go

# Run pattern examples
go run -tags=example ./argo/examples/patterns_example.go
```

Each example file has a `main()` function with commented-out example functions. Uncomment the one you want to run.

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

## Detailed Examples Guide

### 1. Client Configuration Examples (`main.go`)

Demonstrates different ways to initialize and configure the Argo Workflows client.

#### Examples Included:

- **Default Configuration**: Simplest setup using `~/.kube/config`
- **Functional Options**: Custom kubeconfig path and context
- **In-Cluster Configuration**: For running inside Kubernetes pods
- **Argo Server Mode**: Connect via Argo Server HTTP API
- **With OpenTelemetry**: Enable distributed tracing and metrics

```go
// Simple default configuration
ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())

// Custom configuration
ctx, client, err := argo.NewClientWithOptions(ctx,
    argo.WithKubeConfig("/custom/kubeconfig"),
    argo.WithContext("production"),
)
```

### 2. Workflow Builder Examples (`builder_example.go`)

Shows how to use the WorkflowBuilder API for constructing workflows programmatically.

#### Examples Included:

- **Simple Sequential Workflow**: Build → Test → Deploy pattern
- **Workflow with Exit Handlers**: Cleanup steps that always run
- **OpenTelemetry Integration**: Workflows with observability

```go
wf, err := builder.NewWorkflowBuilder("cicd", "argo",
    builder.WithServiceAccount("argo-workflow")).
    Add(buildStep).
    Add(testStep).
    Add(deployStep).
    AddExitHandler(cleanupStep).
    Build()
```

### 3. Workflow Operations Examples (`operations_example.go`)

Comprehensive examples of workflow lifecycle management and operations.

#### Examples Included (12 total):

1. **Submit Workflow**: Basic workflow submission
2. **Submit with OpenTelemetry**: Traced workflow submission
3. **Submit and Wait**: Submit and block until completion
4. **Submit and Wait with Error Handling**: Advanced error handling patterns
5. **Get Workflow Status**: Retrieve current status
6. **Monitor Workflow Status**: Polling pattern for status updates
7. **List All Workflows**: List workflows in a namespace
8. **List with Label Selectors**: Filter workflows by labels
9. **Delete Workflow**: Remove workflows
10. **Complete Workflow Lifecycle**: End-to-end management
11. **Batch Operations**: Submit multiple workflows
12. **Error Handling Patterns**: Comprehensive error scenarios

```go
// Submit and wait example
completed, err := argo.SubmitAndWait(ctx, client, wf, otelConfig, 5*time.Minute)
if err != nil {
    log.Fatalf("Workflow failed: %v", err)
}

// List with labels
workflows, err := argo.ListWorkflows(ctx, client, "argo", "app=myapp,env=prod", nil)
```

### 4. Template Types Examples (`templates_example.go`)

Demonstrates all available template types and their configuration options.

#### Examples Included (22 total):

**Container Templates (8 examples):**
- Basic containers
- Command and arguments
- Environment variables (direct and from secrets/configmaps)
- Volume mounts
- Resource limits (CPU/memory)
- Conditional execution (when, continue-on)
- Retry strategies
- Fully configured container with all options

**Script Templates (6 examples):**
- Bash scripts
- Python scripts
- Node.js scripts
- Ruby scripts
- Scripts with resources and volumes
- Custom image scripts

**HTTP Templates (5 examples):**
- GET requests
- POST requests with JSON body
- Webhook notifications
- API polling
- Complex success conditions

**Noop Templates (1 example):**
- Placeholder steps for testing

**Mixed Templates (2 examples):**
- Combining different template types
- Template type comparison guide

```go
// Container example
container := template.NewContainer("deploy", "myapp:v1").
    Command("sh", "-c").
    Args("/app/deploy.sh").
    Env("ENV", "production").
    CPU("1000m", "2000m").
    Memory("512Mi", "1Gi")

// Script example
script := template.NewScript("process", "python").
    Script(`
import pandas as pd
data = pd.read_csv("/data/input.csv")
data.to_parquet("/data/output.parquet")
`).
    CPU("2000m").
    Memory("2Gi")

// HTTP example
httpCall := template.NewHTTP("api-call").
    URL("https://api.example.com/data").
    Method("POST").
    Header("Content-Type", "application/json").
    Body(`{"key": "value"}`)
```

### 5. Advanced Features Examples (`advanced_features_example.go`)

Covers advanced workflow configuration and production-ready features.

#### Examples Included (17 total):

**Parameters (2 examples):**
- Workflow parameters
- Default and optional parameters

**Retry Strategies (2 examples):**
- Workflow-level retry with exponential backoff
- Per-step retry strategies

**Volumes (4 examples):**
- EmptyDir for temporary storage
- ConfigMap volumes
- Secret volumes
- PersistentVolumeClaim (PVC)

**Advanced Configuration (9 examples):**
- Exit handlers for cleanup
- TTL for automatic deletion
- Log archival
- Labels and annotations for organization
- Custom service accounts for RBAC
- ActiveDeadlineSeconds for timeouts
- OpenTelemetry metrics
- Complete advanced workflow (all features combined)
- Simple vs advanced comparison

```go
// Advanced workflow with multiple features
wf, err := builder.NewWorkflowBuilder("production-wf", "argo",
    builder.WithServiceAccount("custom-sa"),
    builder.WithLabels(map[string]string{"app": "myapp", "env": "prod"}),
    builder.WithAnnotations(map[string]string{"owner": "team@example.com"}),
    builder.WithRetryStrategy(&v1alpha1.RetryStrategy{
        Limit: intstr.FromInt32(3),
        RetryPolicy: v1alpha1.RetryPolicyOnFailure,
    }),
    builder.WithTTL(&ttlSeconds),
    builder.WithArchiveLogs(true),
    builder.WithActiveDeadlineSeconds(&deadline),
    builder.WithVolumes(volumes)).
    Add(step).
    AddExitHandler(cleanup).
    Build()
```

### 6. Workflow Patterns Examples (`patterns_example.go`)

Real-world workflow patterns and architectures for common use cases.

#### Examples Included (8 total):

1. **Sequential Workflow**: Steps that run one after another
2. **CI/CD Pipeline**: Complete continuous integration/deployment (11 stages)
3. **Data Pipeline (ETL)**: Extract, Transform, Load pattern
4. **Microservices Deployment**: Deploy multiple services with health checks
5. **Backup and Restore**: Database backup with validation
6. **ML Training Pipeline**: Machine learning workflow
7. **Monitoring and Alerting**: Health checks with conditional alerting
8. **Patterns Summary**: When to use each pattern

```go
// CI/CD Pipeline example
wf, err := builder.NewWorkflowBuilder("cicd-pipeline", "argo").
    Add(gitClone).          // Stage 1: Source
    Add(goBuild).           // Stage 2: Build
    Add(dockerBuild).
    Add(unitTests).         // Stage 3: Test
    Add(integrationTests).
    Add(securityScan).      // Stage 4: Security
    Add(dockerPush).        // Stage 5: Push
    Add(deployStaging).     // Stage 6: Deploy
    Add(smokeTest).         // Stage 7: Smoke test
    Add(deployProduction).  // Stage 8: Production (conditional)
    Add(notifySuccess).     // Stage 9: Notification
    Build()
```

## Quick Start Guide

### For Beginners

If you're new to Argo Workflows, start with these examples in order:

1. **Client Setup** (`main.go`): Learn how to connect to Argo
2. **Simple Workflow** (`builder_example.go`): Create your first workflow
3. **Operations** (`operations_example.go`): Submit and monitor workflows
4. **Templates** (`templates_example.go`): Understand different template types

### For Production Use

For production-ready workflows, explore:

1. **Advanced Features** (`advanced_features_example.go`): Parameters, retry, volumes, TTL
2. **Patterns** (`patterns_example.go`): Real-world CI/CD, ETL, and deployment patterns

### Total Examples Count

This example collection includes **72+ complete examples** covering:
- 5 client configuration examples
- 3 workflow builder examples
- 12 workflow operations examples
- 22 template type examples
- 17 advanced feature examples
- 8 workflow pattern examples
- Additional helper functions and troubleshooting guides

## Quick Reference

### Common Operations

#### Submit a Workflow

```go
created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
if err != nil {
    log.Fatalf("Failed to submit: %v", err)
}
fmt.Printf("Workflow submitted: %s\n", created.Name)
```

#### Wait for Completion

```go
completed, err := argo.SubmitAndWait(ctx, client, wf, nil, 5*time.Minute)
if err != nil {
    log.Fatalf("Workflow failed: %v", err)
}
fmt.Printf("Status: %s\n", completed.Status.Phase)
```

#### Get Status

```go
status, err := argo.GetWorkflowStatus(ctx, client, "argo", "workflow-name", nil)
if err != nil {
    log.Fatalf("Failed to get status: %v", err)
}
fmt.Printf("Phase: %s\n", status.Phase)
```

#### List Workflows

```go
workflows, err := argo.ListWorkflows(ctx, client, "argo", "", nil)
if err != nil {
    log.Fatalf("Failed to list: %v", err)
}
fmt.Printf("Found %d workflows\n", len(workflows))
```

#### Delete Workflow

```go
err := argo.DeleteWorkflow(ctx, client, "argo", "workflow-name", nil)
if err != nil {
    log.Fatalf("Failed to delete: %v", err)
}
```

## Building Workflows

### Basic Workflow Structure

```go
// Step 1: Create a template
step := template.NewContainer("hello", "alpine:3.19",
    template.WithCommand("echo", "Hello, World!"))

// Step 2: Build the workflow
wf, err := builder.NewWorkflowBuilder("hello-workflow", "argo",
    builder.WithServiceAccount("default")).
    Add(step).
    Build()

// Step 3: Submit the workflow
created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
```

### Using Different Template Types

```go
// Container template
container := template.NewContainer("build", "golang:1.25",
    template.WithCommand("go", "build"))

// Script template
script := template.NewScript("analyze", "python",
    template.WithScriptContent("print('Analyzing data...')"))

// HTTP template
httpCall := template.NewHTTP("webhook",
    template.WithHTTPURL("https://api.example.com/notify"),
    template.WithHTTPMethod("POST"))

// Add to workflow
wf, err := builder.NewWorkflowBuilder("mixed", "argo").
    Add(container).
    Add(script).
    Add(httpCall).
    Build()
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
