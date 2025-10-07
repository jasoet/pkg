//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
	"github.com/jasoet/pkg/v2/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Example 1: Workflow with Parameters
// Demonstrates passing parameters to workflows
func exampleWorkflowParameters() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Use parameters in container
	step := template.NewContainer("greet", "alpine:3.19",
		template.WithCommand("sh", "-c"),
		template.WithArgs("echo 'Hello, {{workflow.parameters.name}}! Environment: {{workflow.parameters.environment}}'"))

	wf, err := builder.NewWorkflowBuilder("parameterized-workflow", "argo",
		builder.WithServiceAccount("default")).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	// Add parameters before submission
	wf.Spec.Arguments = v1alpha1.Arguments{
		Parameters: []v1alpha1.Parameter{
			{
				Name:  "name",
				Value: v1alpha1.AnyStringPtr("World"),
			},
			{
				Name:  "environment",
				Value: v1alpha1.AnyStringPtr("production"),
			},
		},
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Parameterized workflow submitted: %s\n", created.Name)
	fmt.Println("Parameters:")
	for _, param := range created.Spec.Arguments.Parameters {
		fmt.Printf("  %s = %s\n", param.Name, param.Value.String())
	}
}

// Example 2: Workflow with Default and Optional Parameters
// Demonstrates parameter defaults and validation
func exampleParameterDefaults() {
	step := template.NewContainer("process", "alpine:3.19").
		Command("sh", "-c").
		Args(`
			echo "Region: {{workflow.parameters.region}}"
			echo "Replicas: {{workflow.parameters.replicas}}"
			echo "Debug: {{workflow.parameters.debug}}"
		`)

	fmt.Printf("Process step with parameters: %+v\n", step)
	fmt.Println("\nTo use with defaults, submit workflow with:")
	fmt.Println(`
	wf.Spec.Arguments = v1alpha1.Arguments{
		Parameters: []v1alpha1.Parameter{
			{Name: "region", Value: v1alpha1.AnyStringPtr("us-west-2")},
			{Name: "replicas", Value: v1alpha1.AnyStringPtr("3")},
			{Name: "debug", Value: v1alpha1.AnyStringPtr("false")},
		},
	}
	`)
}

// Example 3: Workflow with Retry Strategy
// Demonstrates different retry policies and configurations
func exampleRetryStrategy() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Flaky API call that might fail
	flakyCall := template.NewContainer("flaky-api", "curlimages/curl:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("curl -f https://httpbin.org/status/500 || exit 1"))

	limit := intstr.FromInt32(3)
	backoffFactor := intstr.FromInt32(2)

	wf, err := builder.NewWorkflowBuilder("retry-workflow", "argo",
		builder.WithServiceAccount("default"),
		builder.WithRetryStrategy(&v1alpha1.RetryStrategy{
			Limit:       &limit,
			RetryPolicy: v1alpha1.RetryPolicyAlways,
			Backoff: &v1alpha1.Backoff{
				Duration:    "1m",                // Start with 1 minute
				Factor:      &backoffFactor,      // Double each retry
				MaxDuration: "10m",               // Cap at 10 minutes
			},
		})).
		Add(flakyCall).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with retry strategy submitted: %s\n", created.Name)
	fmt.Println("Retry configuration:")
	fmt.Printf("  Limit: %s\n", created.Spec.RetryStrategy.Limit.String())
	fmt.Printf("  Policy: %s\n", created.Spec.RetryStrategy.RetryPolicy)
	fmt.Printf("  Backoff duration: %s\n", created.Spec.RetryStrategy.Backoff.Duration)
	fmt.Printf("  Backoff factor: %s\n", created.Spec.RetryStrategy.Backoff.Factor.String())
}

// Example 4: Per-Step Retry Strategy
// Demonstrates different retry strategies for different steps
func examplePerStepRetry() {
	limit3 := intstr.FromInt32(3)
	limit5 := intstr.FromInt32(5)
	backoffFactor := intstr.FromInt32(2)

	// Critical API call - retry more times
	criticalAPI := template.NewContainer("critical-api", "curlimages/curl:latest",
		template.WithCommand("curl", "-f", "https://api.example.com/data")).
		WithRetry(&v1alpha1.RetryStrategy{
			Limit:       &limit5,
			RetryPolicy: v1alpha1.RetryPolicyAlways,
			Backoff: &v1alpha1.Backoff{
				Duration:    "2m",
				Factor:      &backoffFactor,
				MaxDuration: "20m",
			},
		})

	// Less critical API call - fewer retries
	optionalAPI := template.NewContainer("optional-api", "curlimages/curl:latest",
		template.WithCommand("curl", "-f", "https://api.optional.com/data")).
		WithRetry(&v1alpha1.RetryStrategy{
			Limit:       &limit3,
			RetryPolicy: v1alpha1.RetryPolicyOnFailure,
		})

	fmt.Printf("Critical API (5 retries): %+v\n", criticalAPI)
	fmt.Printf("Optional API (3 retries): %+v\n", optionalAPI)
}

// Example 5: Workflow with Volumes - EmptyDir
// Demonstrates using EmptyDir for temporary storage shared between steps
func exampleVolumesEmptyDir() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Step 1: Write data
	writer := template.NewContainer("write-data", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Shared data' > /data/output.txt")).
		VolumeMount("shared-data", "/data", false)

	// Step 2: Read data
	reader := template.NewContainer("read-data", "alpine:3.19",
		template.WithCommand("sh", "-c", "cat /data/output.txt")).
		VolumeMount("shared-data", "/data", true)

	wf, err := builder.NewWorkflowBuilder("emptydir-volume", "argo",
		builder.WithServiceAccount("default"),
		builder.WithVolume(corev1.Volume{
			Name: "shared-data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		})).
		Add(writer).
		Add(reader).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with EmptyDir volume submitted: %s\n", created.Name)
}

// Example 6: Workflow with ConfigMap Volume
// Demonstrates mounting ConfigMap as volume
func exampleVolumesConfigMap() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Use config from ConfigMap
	app := template.NewContainer("app", "myapp:v1",
		template.WithCommand("sh", "-c", "cat /config/app.conf && /app/run")).
		VolumeMount("app-config", "/config", true)

	wf, err := builder.NewWorkflowBuilder("configmap-volume", "argo",
		builder.WithServiceAccount("default"),
		builder.WithVolume(corev1.Volume{
			Name: "app-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "app-config",
					},
				},
			},
		})).
		Add(app).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with ConfigMap volume submitted: %s\n", created.Name)
}

// Example 7: Workflow with Secret Volume
// Demonstrates mounting secrets as volumes
func exampleVolumesSecret() {
	step := template.NewContainer("secure-app", "myapp:v1").
		Command("/app/run").
		VolumeMount("credentials", "/secrets", true)

	fmt.Printf("Secure app with secret volume: %+v\n", step)
	fmt.Println("\nVolume definition:")
	fmt.Println(`
	builder.WithVolumes([]corev1.Volume{
		{
			Name: "credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "app-credentials",
				},
			},
		},
	})
	`)
}

// Example 8: Workflow with PersistentVolumeClaim
// Demonstrates using PVC for persistent storage
func exampleVolumesPVC() {
	backup := template.NewContainer("backup", "postgres:15").
		Command("pg_dump", "-U", "postgres", "-f", "/backup/db.sql").
		VolumeMount("backup-storage", "/backup", false)

	fmt.Printf("Backup with PVC: %+v\n", backup)
	fmt.Println("\nPVC Volume definition:")
	fmt.Println(`
	builder.WithVolumes([]corev1.Volume{
		{
			Name: "backup-storage",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: "backup-pvc",
				},
			},
		},
	})
	`)
}

// Example 9: Workflow with Exit Handler
// Demonstrates cleanup steps that always run
func exampleExitHandler() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Main workflow steps
	mainStep := template.NewContainer("process", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Processing...'; sleep 10"))

	// Cleanup that always runs
	cleanup := template.NewContainer("cleanup", "alpine:3.19",
		template.WithCommand("sh", "-c", `
			echo "Cleaning up resources..."
			echo "Workflow status: {{workflow.status}}"
			if [ "{{workflow.status}}" = "Succeeded" ]; then
				echo "Workflow succeeded - performing success cleanup"
			else
				echo "Workflow failed - performing failure cleanup"
			fi
		`))

	wf, err := builder.NewWorkflowBuilder("exit-handler", "argo",
		builder.WithServiceAccount("default")).
		Add(mainStep).
		AddExitHandler(cleanup).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with exit handler submitted: %s\n", created.Name)
	fmt.Printf("Exit handler: %s\n", created.Spec.OnExit)
}

// Example 10: Workflow with TTL
// Demonstrates automatic workflow cleanup
func exampleWorkflowTTL() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	step := template.NewContainer("short-lived", "alpine:3.19",
		template.WithCommand("echo", "This workflow will be deleted after 1 hour"))

	ttlSeconds := int32(3600) // 1 hour

	wf, err := builder.NewWorkflowBuilder("ttl-workflow", "argo",
		builder.WithServiceAccount("default"),
		builder.WithTTL(&v1alpha1.TTLStrategy{
			SecondsAfterCompletion: &ttlSeconds,
		})).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with TTL submitted: %s\n", created.Name)
	fmt.Printf("TTL: %d seconds (%.1f hours)\n", *created.Spec.TTLStrategy.SecondsAfterCompletion, float64(*created.Spec.TTLStrategy.SecondsAfterCompletion)/3600)
}

// Example 11: Workflow with Archive Logs
// Demonstrates log archival configuration
func exampleArchiveLogs() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	step := template.NewContainer("logged-process", "alpine:3.19",
		template.WithCommand("sh", "-c", "for i in 1 2 3 4 5; do echo \"Log line $i\"; sleep 1; done"))

	wf, err := builder.NewWorkflowBuilder("archive-logs", "argo",
		builder.WithServiceAccount("default"),
		builder.WithArchiveLogs(true)).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with log archival submitted: %s\n", created.Name)
	fmt.Printf("Archive logs: %t\n", created.Spec.ArchiveLogs != nil && *created.Spec.ArchiveLogs)
}

// Example 12: Workflow with Labels and Annotations
// Demonstrates metadata for organization and tooling
func exampleLabelsAnnotations() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	step := template.NewContainer("labeled-step", "alpine:3.19",
		template.WithCommand("echo", "Workflow with metadata"))

	wf, err := builder.NewWorkflowBuilder("metadata-workflow", "argo",
		builder.WithServiceAccount("default"),
		builder.WithLabels(map[string]string{
			"app":         "myapp",
			"env":         "production",
			"team":        "platform",
			"cost-center": "engineering",
		}),
		builder.WithAnnotations(map[string]string{
			"description":      "Production deployment workflow",
			"owner":            "platform-team@example.com",
			"runbook":          "https://wiki.example.com/runbooks/deployment",
			"alert-channel":    "#platform-alerts",
			"compliance-level": "high",
		})).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with labels and annotations submitted: %s\n", created.Name)
	fmt.Println("\nLabels:")
	for k, v := range created.Labels {
		fmt.Printf("  %s: %s\n", k, v)
	}
	fmt.Println("\nAnnotations:")
	for k, v := range created.Annotations {
		fmt.Printf("  %s: %s\n", k, v)
	}
}

// Example 13: Workflow with Service Account
// Demonstrates custom RBAC configuration
func exampleServiceAccount() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Workflow needs special permissions
	privilegedStep := template.NewContainer("deploy", "kubectl:latest",
		template.WithCommand("kubectl", "apply", "-f", "/manifests/"))

	wf, err := builder.NewWorkflowBuilder("privileged-workflow", "argo",
		builder.WithServiceAccount("deployment-sa")).
		Add(privilegedStep).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with custom service account submitted: %s\n", created.Name)
	fmt.Printf("Service account: %s\n", created.Spec.ServiceAccountName)
}

// Example 14: Workflow with ActiveDeadlineSeconds
// Demonstrates workflow timeout configuration
func exampleActiveDeadline() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	longRunning := template.NewContainer("long-process", "alpine:3.19",
		template.WithCommand("sh", "-c", "sleep 300"))

	deadline := int64(120) // 2 minutes

	wf, err := builder.NewWorkflowBuilder("deadline-workflow", "argo",
		builder.WithServiceAccount("default"),
		builder.WithActiveDeadlineSeconds(deadline)).
		Add(longRunning).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with deadline submitted: %s\n", created.Name)
	fmt.Printf("Active deadline: %d seconds (%.1f minutes)\n", *created.Spec.ActiveDeadlineSeconds, float64(*created.Spec.ActiveDeadlineSeconds)/60)
}

// Example 15: Workflow with OpenTelemetry Metrics
// Demonstrates observability with custom metrics
func exampleMetrics() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	otelConfig := otel.NewConfig("advanced-workflow-metrics")
	defer otelConfig.Shutdown(ctx)

	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
		argo.WithOTelConfig(otelConfig),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	step := template.NewContainer("measured-step", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Running with metrics'; sleep 5"),
		template.WithOTelConfig(otelConfig))

	// Custom metrics provider
	metricsProvider := &customMetricsProvider{}

	wf, err := builder.NewWorkflowBuilder("metrics-workflow", "argo",
		builder.WithServiceAccount("default"),
		builder.WithLabels(map[string]string{
			"metrics-enabled": "true",
		})).
		WithMetrics(metricsProvider).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Workflow with metrics submitted: %s\n", created.Name)
	fmt.Println("Metrics will be exported to OpenTelemetry collector")
}

// customMetricsProvider implements WorkflowMetricsProvider
type customMetricsProvider struct{}

func (m *customMetricsProvider) Metrics() (*v1alpha1.Metrics, error) {
	return &v1alpha1.Metrics{
		Prometheus: []*v1alpha1.Prometheus{
			{
				Name:   "workflow_duration_seconds",
				Help:   "Duration of workflow execution in seconds",
				Gauge: &v1alpha1.Gauge{
					Value: "{{workflow.duration}}",
				},
				Labels: []*v1alpha1.MetricLabel{
					{Key: "workflow_name", Value: "{{workflow.name}}"},
					{Key: "namespace", Value: "{{workflow.namespace}}"},
					{Key: "status", Value: "{{workflow.status}}"},
				},
			},
			{
				Name:   "workflow_step_duration_seconds",
				Help:   "Duration of each workflow step",
				Gauge: &v1alpha1.Gauge{
					Value: "{{workflow.steps.*.duration}}",
				},
				Labels: []*v1alpha1.MetricLabel{
					{Key: "step_name", Value: "{{workflow.steps.*.name}}"},
					{Key: "workflow_name", Value: "{{workflow.name}}"},
				},
			},
		},
	}, nil
}

// Example 16: Complete Advanced Workflow
// Demonstrates combining multiple advanced features
func exampleCompleteAdvancedWorkflow() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	otelConfig := otel.NewConfig("complete-advanced-workflow")
	defer otelConfig.Shutdown(ctx)

	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
		argo.WithOTelConfig(otelConfig),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Main processing step
	process := template.NewContainer("process", "myapp:v1",
		template.WithCommand("sh", "-c"),
		template.WithArgs("/app/process --input /data/input --output /data/output"),
		template.WithEnv("LOG_LEVEL", "{{workflow.parameters.log-level}}"),
		template.WithCPU("2000m", "4000m"),
		template.WithMemory("2Gi", "4Gi")).
		VolumeMount("data", "/data", false).
		VolumeMount("config", "/config", true)

	// Cleanup step
	cleanup := template.NewContainer("cleanup", "alpine:3.19",
		template.WithCommand("sh", "-c", "rm -rf /data/temp/*")).
		VolumeMount("data", "/data", false)

	// Configuration
	ttl := int32(86400)     // 24 hours
	deadline := int64(3600) // 1 hour
	limit := intstr.FromInt32(2)

	wf, err := builder.NewWorkflowBuilder("complete-advanced", "argo",
		builder.WithServiceAccount("advanced-workflow-sa"),
		builder.WithLabels(map[string]string{
			"app":     "data-pipeline",
			"env":     "production",
			"version": "v1.2.0",
		}),
		builder.WithAnnotations(map[string]string{
			"description": "Advanced production workflow with all features",
			"owner":       "data-team@example.com",
		}),
		builder.WithVolume(corev1.Volume{
			Name: "data",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}),
		builder.WithVolume(corev1.Volume{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
				},
			},
		}),
		builder.WithRetryStrategy(&v1alpha1.RetryStrategy{
			Limit:       &limit,
			RetryPolicy: v1alpha1.RetryPolicyOnFailure,
		}),
		builder.WithTTL(&v1alpha1.TTLStrategy{
			SecondsAfterCompletion: &ttl,
		}),
		builder.WithArchiveLogs(true),
		builder.WithActiveDeadlineSeconds(deadline)).
		Add(process).
		AddExitHandler(cleanup).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	// Add parameters
	wf.Spec.Arguments = v1alpha1.Arguments{
		Parameters: []v1alpha1.Parameter{
			{Name: "log-level", Value: v1alpha1.AnyStringPtr("info")},
		},
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Complete advanced workflow submitted: %s\n", created.Name)
	fmt.Println("\nConfiguration:")
	fmt.Printf("  Service Account: %s\n", created.Spec.ServiceAccountName)
	fmt.Printf("  TTL: %d seconds\n", *created.Spec.TTLStrategy.SecondsAfterCompletion)
	fmt.Printf("  Deadline: %d seconds\n", *created.Spec.ActiveDeadlineSeconds)
	fmt.Printf("  Archive Logs: %t\n", created.Spec.ArchiveLogs != nil && *created.Spec.ArchiveLogs)
	fmt.Printf("  Exit Handler: %s\n", created.Spec.OnExit)
	fmt.Printf("  Labels: %v\n", created.Labels)
}

// Example 17: Workflow Comparison - Simple vs Advanced
// Shows the progression from simple to advanced workflows
func exampleWorkflowComparison() {
	fmt.Println("Workflow Configuration Comparison")
	fmt.Println("===================================")
	fmt.Println()

	fmt.Println("SIMPLE WORKFLOW:")
	fmt.Println("- Basic container execution")
	fmt.Println("- Default service account")
	fmt.Println("- No retry strategy")
	fmt.Println("- No resource limits")
	fmt.Println("- Manual cleanup")
	fmt.Println()

	fmt.Println("ADVANCED WORKFLOW:")
	fmt.Println("- Parameterized execution")
	fmt.Println("- Custom service account with RBAC")
	fmt.Println("- Retry strategy with exponential backoff")
	fmt.Println("- Resource requests and limits")
	fmt.Println("- Automatic cleanup with exit handlers")
	fmt.Println("- TTL for automatic deletion")
	fmt.Println("- Log archival")
	fmt.Println("- Workflow timeout (ActiveDeadlineSeconds)")
	fmt.Println("- Volumes for data sharing")
	fmt.Println("- Labels and annotations for organization")
	fmt.Println("- OpenTelemetry observability")
	fmt.Println("- Custom metrics")
}

func main() {
	fmt.Println("Argo Workflow Advanced Features Examples")
	fmt.Println("==========================================")
	fmt.Println()
	fmt.Println("Uncomment the example you want to run:")
	fmt.Println()

	// Uncomment one example at a time to run:

	// Parameters
	// exampleWorkflowParameters()
	// exampleParameterDefaults()

	// Retry Strategies
	// exampleRetryStrategy()
	// examplePerStepRetry()

	// Volumes
	// exampleVolumesEmptyDir()
	// exampleVolumesConfigMap()
	// exampleVolumesSecret()
	// exampleVolumesPVC()

	// Advanced Features
	// exampleExitHandler()
	// exampleWorkflowTTL()
	// exampleArchiveLogs()
	// exampleLabelsAnnotations()
	// exampleServiceAccount()
	// exampleActiveDeadline()
	// exampleMetrics()

	// Complete Examples
	// exampleCompleteAdvancedWorkflow()
	// exampleWorkflowComparison()

	fmt.Println("Please uncomment one of the example functions in main()")
}
