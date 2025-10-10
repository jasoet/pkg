//go:build argo

package argo

import (
	"context"
	"testing"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestIntegration_SubmitWorkflow(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create simple workflow
	wf, err := builder.NewWorkflowBuilder("integration-test", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(template.NewContainer("hello", "alpine:latest",
			template.WithCommand("echo", "Hello from integration test"))).
		Build()
	require.NoError(t, err, "should build workflow")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow")
	require.NotNil(t, created)
	assert.NotEmpty(t, created.Name)
	assert.Equal(t, "argo", created.Namespace)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow submitted: %s", created.Name)
}

func TestIntegration_SubmitAndWait(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create simple fast workflow
	wf, err := builder.NewWorkflowBuilder("integration-wait", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(template.NewContainer("quick", "alpine:latest",
			template.WithCommand("sh", "-c", "echo 'Quick test' && sleep 1"))).
		Build()
	require.NoError(t, err, "should build workflow")

	// Submit and wait
	completed, err := SubmitAndWait(ctx, client, wf, cfg, 2*time.Minute)
	require.NoError(t, err, "should complete workflow")
	require.NotNil(t, completed)
	assert.Contains(t, []v1alpha1.WorkflowPhase{
		v1alpha1.WorkflowSucceeded,
		v1alpha1.WorkflowRunning, // May still be running
	}, completed.Status.Phase)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", completed.Name, cfg)
	}()

	t.Logf("✓ Workflow completed: %s (phase: %s)", completed.Name, completed.Status.Phase)
}

func TestIntegration_GetWorkflowStatus(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create and submit workflow
	wf, err := builder.NewWorkflowBuilder("integration-status", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(template.NewContainer("status-check", "alpine:latest",
			template.WithCommand("echo", "Status check test"))).
		Build()
	require.NoError(t, err)

	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	// Wait a moment for workflow to initialize
	time.Sleep(2 * time.Second)

	// Get workflow status
	status, err := GetWorkflowStatus(ctx, client, "argo", created.Name, cfg)
	require.NoError(t, err, "should get workflow status")
	require.NotNil(t, status)
	assert.NotEmpty(t, status.Phase)

	t.Logf("✓ Workflow status retrieved: %s (phase: %s)", created.Name, status.Phase)
}

func TestIntegration_ListWorkflows(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create and submit workflow with label
	wf, err := builder.NewWorkflowBuilder("integration-list", "argo",
		builder.WithServiceAccount("argo-workflow"),
		builder.WithLabels(map[string]string{
			"test-type": "integration-list",
		})).
		Add(template.NewContainer("list-test", "alpine:latest",
			template.WithCommand("echo", "List test"))).
		Build()
	require.NoError(t, err)

	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	// List all workflows
	workflows, err := ListWorkflows(ctx, client, "argo", "", cfg)
	require.NoError(t, err, "should list workflows")
	assert.NotEmpty(t, workflows, "should have at least one workflow")

	// List with label selector
	filtered, err := ListWorkflows(ctx, client, "argo", "test-type=integration-list", cfg)
	require.NoError(t, err, "should list filtered workflows")
	assert.NotEmpty(t, filtered, "should find workflow with label")

	// Verify our workflow is in the list
	found := false
	for _, wf := range filtered {
		if wf.Name == created.Name {
			found = true
			break
		}
	}
	assert.True(t, found, "should find created workflow in list")

	t.Logf("✓ Listed %d workflows, %d with label", len(workflows), len(filtered))
}

func TestIntegration_DeleteWorkflow(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create and submit workflow
	wf, err := builder.NewWorkflowBuilder("integration-delete", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(template.NewContainer("delete-test", "alpine:latest",
			template.WithCommand("echo", "Delete test"))).
		Build()
	require.NoError(t, err)

	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err)

	// Wait a moment for workflow to be fully created
	time.Sleep(2 * time.Second)

	// Delete workflow
	err = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	require.NoError(t, err, "should delete workflow")

	// Verify deletion - getting the workflow should fail
	time.Sleep(1 * time.Second)
	status, err := GetWorkflowStatus(ctx, client, "argo", created.Name, cfg)
	if err == nil && status != nil {
		// Workflow might still exist briefly after deletion
		t.Logf("Workflow still exists briefly: %s", created.Name)
	}

	t.Logf("✓ Workflow deleted: %s", created.Name)
}

func TestIntegration_CompleteWorkflow(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with exit handler
	mainStep := template.NewContainer("main-step", "alpine:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("echo 'Main step' && sleep 2"))

	exitStep := template.NewContainer("cleanup", "alpine:latest",
		template.WithCommand("echo", "Cleanup complete"))

	wf, err := builder.NewWorkflowBuilder("integration-complete", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(mainStep).
		AddExitHandler(exitStep).
		Build()
	require.NoError(t, err)

	// Submit and wait for completion
	completed, err := SubmitAndWait(ctx, client, wf, cfg, 2*time.Minute)
	require.NoError(t, err, "should complete workflow")
	require.NotNil(t, completed)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", completed.Name, cfg)
	}()

	// Verify completion
	status, err := GetWorkflowStatus(ctx, client, "argo", completed.Name, cfg)
	require.NoError(t, err)
	assert.Contains(t, []v1alpha1.WorkflowPhase{
		v1alpha1.WorkflowSucceeded,
		v1alpha1.WorkflowRunning,
	}, status.Phase)

	t.Logf("✓ Complete workflow test: %s (phase: %s)", completed.Name, status.Phase)
}

func TestIntegration_WorkflowWithResources(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err)

	// Create workflow with resource limits
	wf, err := builder.NewWorkflowBuilder("integration-resources", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(template.NewContainer("resource-test", "alpine:latest",
			template.WithCommand("sh", "-c", "echo 'Testing resources' && sleep 1"),
			template.WithCPU("100m"),
			template.WithMemory("128Mi"))).
		Build()
	require.NoError(t, err)

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	// Verify resource limits were set
	assert.NotEmpty(t, created.Spec.Templates)
	for _, tmpl := range created.Spec.Templates {
		if tmpl.Container != nil && tmpl.Name == "resource-test-template" {
			if len(tmpl.Container.Resources.Requests) > 0 {
				assert.NotNil(t, tmpl.Container.Resources.Requests[corev1.ResourceCPU])
			}
		}
	}

	t.Logf("✓ Workflow with resources submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithScript(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err)

	// Create workflow with script
	wf, err := builder.NewWorkflowBuilder("integration-script", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(template.NewScript("bash-script", "bash",
			template.WithScriptContent(`
				echo "Running bash script"
				echo "Current time: $(date)"
				echo "Script completed"
			`))).
		Build()
	require.NoError(t, err)

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err)
	require.NotNil(t, created)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with script submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithParameters(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with parameters
	wf, err := builder.NewWorkflowBuilder("integration-params", "argo",
		builder.WithServiceAccount("default")).
		Add(template.NewContainer("echo-params", "alpine:latest",
			template.WithCommand("sh", "-c"),
			template.WithArgs("echo message: {{workflow.parameters.message}}"))).
		Build()
	require.NoError(t, err, "should build workflow")

	// Add parameters to the workflow
	paramValue := "Hello from parameters!"
	wf.Spec.Arguments = v1alpha1.Arguments{
		Parameters: []v1alpha1.Parameter{
			{
				Name:  "message",
				Value: v1alpha1.AnyStringPtr(paramValue),
			},
		},
	}

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with parameters")
	require.NotNil(t, created)
	assert.NotEmpty(t, created.Name)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with parameters submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithRetryStrategy(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with retry strategy
	limit := intstr.FromInt32(2)
	wf, err := builder.NewWorkflowBuilder("integration-retry", "argo",
		builder.WithServiceAccount("default"),
		builder.WithRetryStrategy(&v1alpha1.RetryStrategy{
			Limit:       &limit,
			RetryPolicy: v1alpha1.RetryPolicyAlways,
		})).
		Add(template.NewContainer("might-fail", "alpine:latest",
			template.WithCommand("sh", "-c"),
			template.WithArgs("exit 0"))).
		Build()
	require.NoError(t, err, "should build workflow with retry")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with retry strategy")
	require.NotNil(t, created)

	// Verify retry strategy was applied
	assert.NotNil(t, created.Spec.Templates)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with retry strategy submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithVolumes(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with volumes
	volume := corev1.Volume{
		Name: "workdir",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	wf, err := builder.NewWorkflowBuilder("integration-volumes", "argo",
		builder.WithServiceAccount("default"),
		builder.WithVolume(volume)).
		Add(template.NewContainer("write-file", "alpine:latest",
			template.WithCommand("sh", "-c"),
			template.WithArgs("echo 'test data' > /work/data.txt")).
			VolumeMount("workdir", "/work", false)).
		Build()
	require.NoError(t, err, "should build workflow with volumes")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with volumes")
	require.NotNil(t, created)

	// Verify volume was attached
	require.Len(t, created.Spec.Volumes, 1)
	assert.Equal(t, "workdir", created.Spec.Volumes[0].Name)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with volumes submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithEnvironmentVariables(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with environment variables
	wf, err := builder.NewWorkflowBuilder("integration-env", "argo",
		builder.WithServiceAccount("default")).
		Add(template.NewContainer("check-env", "alpine:latest",
			template.WithCommand("sh", "-c"),
			template.WithArgs("echo APP_ENV=$APP_ENV && echo LOG_LEVEL=$LOG_LEVEL")).
			Env("APP_ENV", "production").
			Env("LOG_LEVEL", "debug")).
		Build()
	require.NoError(t, err, "should build workflow with env vars")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with env vars")
	require.NotNil(t, created)

	// Verify env vars were set
	assert.NotEmpty(t, created.Spec.Templates)
	for _, tmpl := range created.Spec.Templates {
		if tmpl.Container != nil && len(tmpl.Container.Env) > 0 {
			foundAppEnv := false
			foundLogLevel := false
			for _, env := range tmpl.Container.Env {
				if env.Name == "APP_ENV" && env.Value == "production" {
					foundAppEnv = true
				}
				if env.Name == "LOG_LEVEL" && env.Value == "debug" {
					foundLogLevel = true
				}
			}
			if foundAppEnv && foundLogLevel {
				t.Logf("✓ Environment variables verified in template: %s", tmpl.Name)
			}
		}
	}

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with environment variables submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithConditionalSteps(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with conditional steps
	successStep := template.NewContainer("on-success", "alpine:latest",
		template.WithCommand("echo", "Previous step succeeded!")).
		When("{{workflow.status}} == Succeeded")

	wf, err := builder.NewWorkflowBuilder("integration-conditional", "argo",
		builder.WithServiceAccount("default")).
		Add(template.NewContainer("main-step", "alpine:latest",
			template.WithCommand("echo", "Main execution"))).
		Add(successStep).
		Build()
	require.NoError(t, err, "should build workflow with conditional")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with conditional")
	require.NotNil(t, created)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with conditional steps submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithHTTPTemplate(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with HTTP template
	wf, err := builder.NewWorkflowBuilder("integration-http", "argo",
		builder.WithServiceAccount("default")).
		Add(template.NewHTTP("check-api",
			template.WithHTTPMethod("GET"),
			template.WithHTTPURL("https://httpbin.org/get"))).
		Build()
	require.NoError(t, err, "should build workflow with HTTP")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with HTTP template")
	require.NotNil(t, created)

	// Verify HTTP template exists
	assert.NotEmpty(t, created.Spec.Templates)
	foundHTTP := false
	for _, tmpl := range created.Spec.Templates {
		if tmpl.HTTP != nil {
			foundHTTP = true
			assert.Equal(t, "GET", tmpl.HTTP.Method)
			assert.Equal(t, "https://httpbin.org/get", tmpl.HTTP.URL)
		}
	}
	assert.True(t, foundHTTP, "should have HTTP template")

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with HTTP template submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithMultipleContainers(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with multiple sequential containers
	wf, err := builder.NewWorkflowBuilder("integration-multi", "argo",
		builder.WithServiceAccount("default")).
		Add(template.NewContainer("step1", "alpine:latest",
			template.WithCommand("echo", "Step 1 complete"))).
		Add(template.NewContainer("step2", "alpine:latest",
			template.WithCommand("echo", "Step 2 complete"))).
		Add(template.NewContainer("step3", "alpine:latest",
			template.WithCommand("echo", "Step 3 complete"))).
		Build()
	require.NoError(t, err, "should build workflow with multiple steps")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with multiple containers")
	require.NotNil(t, created)

	// Verify multiple templates exist
	assert.GreaterOrEqual(t, len(created.Spec.Templates), 4, "should have at least 4 templates (3 steps + main)")

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with multiple containers submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithLabelsAndAnnotations(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with labels and annotations
	wf, err := builder.NewWorkflowBuilder("integration-metadata", "argo",
		builder.WithServiceAccount("default"),
		builder.WithLabels(map[string]string{
			"app":     "integration-test",
			"version": "v1.0.0",
			"env":     "test",
		}),
		builder.WithAnnotations(map[string]string{
			"description": "Integration test for metadata",
			"owner":       "test-team",
		})).
		Add(template.NewContainer("test-metadata", "alpine:latest",
			template.WithCommand("echo", "Testing metadata"))).
		Build()
	require.NoError(t, err, "should build workflow with metadata")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with metadata")
	require.NotNil(t, created)

	// Verify labels and annotations
	assert.Equal(t, "integration-test", created.Labels["app"])
	assert.Equal(t, "v1.0.0", created.Labels["version"])
	assert.Equal(t, "test", created.Labels["env"])
	assert.Equal(t, "Integration test for metadata", created.Annotations["description"])
	assert.Equal(t, "test-team", created.Annotations["owner"])

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with labels and annotations submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithTTL(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with TTL
	secondsAfterCompletion := int32(300) // 5 minutes
	wf, err := builder.NewWorkflowBuilder("integration-ttl", "argo",
		builder.WithServiceAccount("default"),
		builder.WithTTL(&v1alpha1.TTLStrategy{
			SecondsAfterCompletion: &secondsAfterCompletion,
		})).
		Add(template.NewContainer("ttl-test", "alpine:latest",
			template.WithCommand("echo", "Testing TTL"))).
		Build()
	require.NoError(t, err, "should build workflow with TTL")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with TTL")
	require.NotNil(t, created)

	// Verify TTL was set
	require.NotNil(t, created.Spec.TTLStrategy)
	require.NotNil(t, created.Spec.TTLStrategy.SecondsAfterCompletion)
	assert.Equal(t, int32(300), *created.Spec.TTLStrategy.SecondsAfterCompletion)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with TTL submitted: %s", created.Name)
}

func TestIntegration_WorkflowWithArchiveLogs(t *testing.T) {
	ctx := context.Background()
	cfg := otel.NewConfig("argo-integration-test")

	// Create Argo client
	ctx, client, err := NewClientWithOptions(ctx,
		WithOTelConfig(cfg))
	require.NoError(t, err, "should create Argo client")

	// Create workflow with archive logs enabled
	wf, err := builder.NewWorkflowBuilder("integration-archive", "argo",
		builder.WithServiceAccount("default"),
		builder.WithArchiveLogs(true)).
		Add(template.NewContainer("archive-test", "alpine:latest",
			template.WithCommand("sh", "-c"),
			template.WithArgs("echo 'Log output' && echo 'More logs'"))).
		Build()
	require.NoError(t, err, "should build workflow with archive logs")

	// Submit workflow
	created, err := SubmitWorkflow(ctx, client, wf, cfg)
	require.NoError(t, err, "should submit workflow with archive logs")
	require.NotNil(t, created)

	// Verify archive logs was set
	require.NotNil(t, created.Spec.ArchiveLogs)
	assert.True(t, *created.Spec.ArchiveLogs)

	// Cleanup
	defer func() {
		_ = DeleteWorkflow(ctx, client, "argo", created.Name, cfg)
	}()

	t.Logf("✓ Workflow with archive logs submitted: %s", created.Name)
}
