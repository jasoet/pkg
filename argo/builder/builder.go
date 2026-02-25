package builder

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// WorkflowBuilder provides a fluent API for constructing Argo Workflows.
// It allows composing workflows from reusable WorkflowSource components,
// with full OpenTelemetry instrumentation for observability.
//
// Example usage:
//
//	// Create sources
//	deploy := template.NewContainer("deploy", "myapp:v1",
//	    template.WithCommand("deploy.sh"))
//
//	healthcheck := template.NewHTTP("healthcheck",
//	    template.WithURL("https://myapp/health"))
//
//	cleanup := template.NewScript("cleanup", "bash",
//	    template.WithScript("echo 'Cleaning up...'"))
//
//	// Build workflow
//	wf, err := NewWorkflowBuilder("deployment", "argo",
//	    WithOTelConfig(otelConfig),
//	    WithServiceAccount("argo-workflow"),
//	).
//	    Add(deploy).
//	    Add(healthcheck).
//	    AddExitHandler(cleanup).
//	    Build()
type WorkflowBuilder struct {
	// Workflow configuration
	namePrefix            string
	namespace             string
	serviceAccount        string
	archiveLogs           *bool
	retryStrategy         *v1alpha1.RetryStrategy
	podGC                 *v1alpha1.PodGC
	ttl                   *v1alpha1.TTLStrategy
	volumes               []corev1.Volume
	labels                map[string]string
	annotations           map[string]string
	activeDeadlineSeconds *int64

	// Workflow structure
	entryPoint      []v1alpha1.ParallelSteps
	templates       []v1alpha1.Template
	exitHandlers    []v1alpha1.ParallelSteps
	metrics         *v1alpha1.Metrics
	uniqueTemplates map[string]struct{}
	errors          []error

	// OpenTelemetry
	otelConfig *otel.Config
	otel       *otelInstrumentation
}

// NewWorkflowBuilder creates a new workflow builder with the specified name and namespace.
// Additional configuration can be provided through functional options.
//
// Parameters:
//   - name: Base name for the workflow (will be used as GenerateName with a trailing dash)
//   - namespace: Kubernetes namespace where the workflow will be created
//   - opts: Optional configuration options (WithOTelConfig, WithServiceAccount, etc.)
//
// Example:
//
//	builder := NewWorkflowBuilder("hello-world", "argo",
//	    WithOTelConfig(otelConfig),
//	    WithServiceAccount("argo-workflow"),
//	    WithArchiveLogs(true))
func NewWorkflowBuilder(name, namespace string, opts ...Option) *WorkflowBuilder {
	b := &WorkflowBuilder{
		namePrefix:      name + "-",
		namespace:       namespace,
		serviceAccount:  "argo-workflow", // default service account
		uniqueTemplates: make(map[string]struct{}),
		labels:          make(map[string]string),
		annotations:     make(map[string]string),
	}

	// Apply options
	for _, opt := range opts {
		opt(b)
	}

	// Initialize OTel instrumentation if configured
	if b.otelConfig != nil {
		b.otel = newOTelInstrumentation(b.otelConfig)
	}

	return b
}

// Add adds a WorkflowSource to the workflow.
// The source's steps will be added sequentially to the workflow's entrypoint.
// Templates will be deduplicated by name.
//
// Example:
//
//	deploy := template.NewContainer("deploy", "myapp:v1")
//	builder.Add(deploy)
func (b *WorkflowBuilder) Add(source WorkflowSource) *WorkflowBuilder {
	ctx := context.Background()

	// Start tracing
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.Add")
		defer span.End()
	}

	logger := otel.NewLogHelper(ctx, b.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder", "WorkflowBuilder.Add")
	logger.Debug("Adding workflow source")

	// Get templates from source
	templates, err := source.Templates()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to get templates: %w", err))
		logger.Error(err, "Failed to get templates from source")
		return b
	}

	// Add templates (deduplicated)
	for _, t := range templates {
		b.insertTemplate(t)
	}

	// Get steps from source
	steps, err := source.Steps()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to get steps: %w", err))
		logger.Error(err, "Failed to get steps from source")
		return b
	}

	// Convert steps to ParallelSteps (each step runs sequentially)
	for _, step := range steps {
		b.entryPoint = append(b.entryPoint, v1alpha1.ParallelSteps{
			Steps: []v1alpha1.WorkflowStep{step},
		})
	}

	// Record metrics
	if b.otel != nil {
		b.otel.incrementCounter(ctx, "sources_added", 1)
		b.otel.incrementCounter(ctx, "templates_added", int64(len(templates)))
	}

	logger.Debug("Workflow source added successfully",
		otel.F("templates_count", len(templates)),
		otel.F("steps_count", len(steps)))

	return b
}

// AddParallel adds a WorkflowSourceV2 that supports parallel step execution.
// Use this when you need steps to run in parallel rather than sequentially.
//
// Example:
//
//	parallelSource := &MyParallelSource{}
//	builder.AddParallel(parallelSource)
func (b *WorkflowBuilder) AddParallel(source WorkflowSourceV2) *WorkflowBuilder {
	ctx := context.Background()

	// Start tracing
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.AddParallel")
		defer span.End()
	}

	logger := otel.NewLogHelper(ctx, b.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder", "WorkflowBuilder.AddParallel")
	logger.Debug("Adding parallel workflow source")

	// Get templates from source
	templates, err := source.Templates()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to get templates: %w", err))
		logger.Error(err, "Failed to get templates from source")
		return b
	}

	// Add templates (deduplicated)
	for _, t := range templates {
		b.insertTemplate(t)
	}

	// Get parallel steps from source
	parallelSteps, err := source.ParallelSteps()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to get parallel steps: %w", err))
		logger.Error(err, "Failed to get parallel steps from source")
		return b
	}

	// Add parallel steps to entrypoint
	b.entryPoint = append(b.entryPoint, parallelSteps...)

	// Record metrics
	if b.otel != nil {
		b.otel.incrementCounter(ctx, "sources_added", 1)
		b.otel.incrementCounter(ctx, "templates_added", int64(len(templates)))
	}

	logger.Debug("Parallel workflow source added successfully",
		otel.F("templates_count", len(templates)),
		otel.F("parallel_groups_count", len(parallelSteps)))

	return b
}

// AddExitHandler adds a WorkflowSource as an exit handler.
// Exit handlers always run when the workflow completes, regardless of success or failure.
// They are useful for cleanup operations and callbacks.
//
// Example:
//
//	cleanup := template.NewScript("cleanup", "bash",
//	    template.WithScript("echo 'Cleaning up resources...'"))
//	builder.AddExitHandler(cleanup)
func (b *WorkflowBuilder) AddExitHandler(source WorkflowSource) *WorkflowBuilder {
	ctx := context.Background()

	// Start tracing
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.AddExitHandler")
		defer span.End()
	}

	logger := otel.NewLogHelper(ctx, b.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder", "WorkflowBuilder.AddExitHandler")
	logger.Debug("Adding exit handler")

	// Get templates from source
	templates, err := source.Templates()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to get exit handler templates: %w", err))
		logger.Error(err, "Failed to get templates from exit handler")
		return b
	}

	// Add templates (deduplicated)
	for _, t := range templates {
		b.insertTemplate(t)
	}

	// Get steps from source
	steps, err := source.Steps()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to get exit handler steps: %w", err))
		logger.Error(err, "Failed to get steps from exit handler")
		return b
	}

	// Add exit handler steps
	for _, step := range steps {
		// Check if this is a cleanup/destroy step and prioritize it
		if strings.Contains(step.Name, "destroy") || strings.Contains(step.Name, "cleanup") {
			// Insert at the beginning
			b.exitHandlers = append([]v1alpha1.ParallelSteps{
				{Steps: []v1alpha1.WorkflowStep{step}},
			}, b.exitHandlers...)
		} else {
			// Append normally
			b.exitHandlers = append(b.exitHandlers, v1alpha1.ParallelSteps{
				Steps: []v1alpha1.WorkflowStep{step},
			})
		}
	}

	logger.Debug("Exit handler added successfully",
		otel.F("templates_count", len(templates)),
		otel.F("steps_count", len(steps)))

	return b
}

// WithMetrics sets custom Prometheus metrics for the workflow.
// These metrics will be exposed when the workflow executes.
//
// Example:
//
//	metricsProvider := &MyMetricsProvider{}
//	builder.WithMetrics(metricsProvider)
func (b *WorkflowBuilder) WithMetrics(provider WorkflowMetricsProvider) *WorkflowBuilder {
	metrics, err := provider.Metrics()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("failed to get metrics: %w", err))
		return b
	}
	b.metrics = metrics
	return b
}

// Build constructs the final Workflow object.
// Returns an error if any errors occurred during workflow construction.
//
// The build process:
// 1. Validates that at least one step exists (adds a no-op if empty)
// 2. Creates the entrypoint template from collected steps
// 3. Creates exit handler template if any exit handlers were added
// 4. Assembles the complete workflow specification
//
// Example:
//
//	wf, err := builder.
//	    Add(deploy).
//	    Add(healthcheck).
//	    AddExitHandler(cleanup).
//	    Build()
//	if err != nil {
//	    log.Fatal(err)
//	}
func (b *WorkflowBuilder) Build() (*v1alpha1.Workflow, error) {
	ctx := context.Background()

	// Start tracing and timing
	startTime := time.Now()
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.Build")
		defer span.End()

		// Record duration when done
		defer func() {
			durationMs := float64(time.Since(startTime).Milliseconds())
			b.otel.recordDuration(ctx, "build_duration", durationMs)
		}()
	}

	logger := otel.NewLogHelper(ctx, b.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder", "WorkflowBuilder.Build")
	logger.Debug("Building workflow",
		otel.F("name", b.namePrefix),
		otel.F("namespace", b.namespace),
		otel.F("steps_count", len(b.entryPoint)),
		otel.F("templates_count", len(b.templates)),
		otel.F("exit_handlers_count", len(b.exitHandlers)))

	// Check for errors
	if len(b.errors) > 0 {
		if b.otel != nil {
			b.otel.recordError(ctx, "build_validation_error", b.errors[0])
		}
		logger.Error(b.errors[0], "Failed to build workflow")
		return nil, b.errors[0]
	}

	// Ensure we have at least one step
	if len(b.entryPoint) == 0 {
		logger.Warn("No steps provided, workflow will be empty")
	}

	// Build a fresh templates slice so Build() is safe to call multiple times.
	const entrypointName = "main"
	entrypoint := v1alpha1.Template{
		Name:  entrypointName,
		Steps: b.entryPoint,
	}
	templates := make([]v1alpha1.Template, len(b.templates), len(b.templates)+2)
	copy(templates, b.templates)
	templates = append(templates, entrypoint)

	// Create exit handler template if needed
	const exitHandlerName = "exit-handler"
	var onExit string
	if len(b.exitHandlers) > 0 {
		exitHandler := v1alpha1.Template{
			Name:  exitHandlerName,
			Steps: b.exitHandlers,
		}
		templates = append(templates, exitHandler)
		onExit = exitHandlerName
	}

	// Build workflow
	wf := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: b.namePrefix,
			Namespace:    b.namespace,
			Labels:       b.labels,
			Annotations:  b.annotations,
		},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint:            entrypointName,
			ServiceAccountName:    b.serviceAccount,
			Templates:             templates,
			Volumes:               b.volumes,
			Metrics:               b.metrics,
			ArchiveLogs:           b.archiveLogs,
			PodGC:                 b.podGC,
			TTLStrategy:           b.ttl,
			ActiveDeadlineSeconds: b.activeDeadlineSeconds,
			OnExit:                onExit,
		},
	}

	// Apply default retry strategy if set
	if b.retryStrategy != nil {
		for i := range wf.Spec.Templates {
			if wf.Spec.Templates[i].RetryStrategy == nil {
				wf.Spec.Templates[i].RetryStrategy = b.retryStrategy
			}
		}
	}

	// Record success metrics
	if b.otel != nil {
		b.otel.incrementCounter(ctx, "workflows_built", 1)
		b.otel.addSpanAttributes(ctx,
			attribute.String("workflow.name", b.namePrefix),
			attribute.String("workflow.namespace", b.namespace),
			attribute.Int("workflow.templates_count", len(templates)),
			attribute.Int("workflow.steps_count", len(b.entryPoint)),
			attribute.Bool("workflow.has_exit_handler", len(b.exitHandlers) > 0),
		)
	}

	logger.Info("Workflow built successfully",
		otel.F("workflow_name", wf.GenerateName),
		otel.F("templates_count", len(wf.Spec.Templates)),
		otel.F("has_exit_handler", onExit != ""),
		otel.F("build_duration_ms", time.Since(startTime).Milliseconds()))

	return wf, nil
}

// BuildWithEntrypoint builds the workflow with a custom entrypoint template name.
// This is useful when you need to manually construct templates and specify which one
// should be the entry point.
//
// Example:
//
//	// Manually create templates
//	entryTemplate := v1alpha1.Template{
//	    Name: "custom-main",
//	    Steps: [][]v1alpha1.WorkflowStep{...},
//	}
//	builder.AddTemplate(entryTemplate)
//	wf, err := builder.BuildWithEntrypoint("custom-main")
func (b *WorkflowBuilder) BuildWithEntrypoint(entrypointName string) (*v1alpha1.Workflow, error) {
	ctx := context.Background()

	// Start tracing and timing
	startTime := time.Now()
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.BuildWithEntrypoint")
		defer span.End()

		// Record duration when done
		defer func() {
			durationMs := float64(time.Since(startTime).Milliseconds())
			b.otel.recordDuration(ctx, "build_duration", durationMs)
		}()
	}

	logger := otel.NewLogHelper(ctx, b.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder", "WorkflowBuilder.BuildWithEntrypoint")
	logger.Debug("Building workflow with custom entrypoint",
		otel.F("name", b.namePrefix),
		otel.F("namespace", b.namespace),
		otel.F("entrypoint", entrypointName),
		otel.F("templates_count", len(b.templates)))

	// Check for errors
	if len(b.errors) > 0 {
		if b.otel != nil {
			b.otel.recordError(ctx, "build_validation_error", b.errors[0])
		}
		logger.Error(b.errors[0], "Failed to build workflow")
		return nil, b.errors[0]
	}

	// Verify entrypoint template exists
	found := false
	for _, t := range b.templates {
		if t.Name == entrypointName {
			found = true
			break
		}
	}
	if !found {
		err := fmt.Errorf("entrypoint template '%s' not found in templates", entrypointName)
		if b.otel != nil {
			b.otel.recordError(ctx, "build_validation_error", err)
		}
		logger.Error(err, "Entrypoint template not found")
		return nil, err
	}

	// Create exit handler template if needed
	const exitHandlerName = "exit-handler"
	var onExit string
	if len(b.exitHandlers) > 0 {
		exitHandler := v1alpha1.Template{
			Name:  exitHandlerName,
			Steps: b.exitHandlers,
		}
		b.templates = append(b.templates, exitHandler)
		onExit = exitHandlerName
	}

	// Build workflow
	wf := &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: b.namePrefix,
			Namespace:    b.namespace,
			Labels:       b.labels,
			Annotations:  b.annotations,
		},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint:            entrypointName,
			ServiceAccountName:    b.serviceAccount,
			Templates:             b.templates,
			Volumes:               b.volumes,
			Metrics:               b.metrics,
			ArchiveLogs:           b.archiveLogs,
			PodGC:                 b.podGC,
			TTLStrategy:           b.ttl,
			ActiveDeadlineSeconds: b.activeDeadlineSeconds,
			OnExit:                onExit,
		},
	}

	// Apply default retry strategy if set
	if b.retryStrategy != nil {
		for i := range wf.Spec.Templates {
			if wf.Spec.Templates[i].RetryStrategy == nil {
				wf.Spec.Templates[i].RetryStrategy = b.retryStrategy
			}
		}
	}

	// Record success metrics
	if b.otel != nil {
		b.otel.incrementCounter(ctx, "workflows_built", 1)
		b.otel.addSpanAttributes(ctx,
			attribute.String("workflow.name", b.namePrefix),
			attribute.String("workflow.namespace", b.namespace),
			attribute.String("workflow.entrypoint", entrypointName),
			attribute.Int("workflow.templates_count", len(b.templates)),
			attribute.Bool("workflow.has_exit_handler", len(b.exitHandlers) > 0),
		)
	}

	logger.Info("Workflow built successfully with custom entrypoint",
		otel.F("workflow_name", wf.GenerateName),
		otel.F("entrypoint", entrypointName),
		otel.F("templates_count", len(wf.Spec.Templates)),
		otel.F("has_exit_handler", onExit != ""),
		otel.F("build_duration_ms", time.Since(startTime).Milliseconds()))

	return wf, nil
}

// AddTemplate adds a template directly to the workflow builder.
// This is useful for advanced use cases where you need to manually construct templates.
// Templates are automatically deduplicated by name.
//
// Example:
//
//	template := v1alpha1.Template{
//	    Name: "custom-step",
//	    Container: &corev1.Container{...},
//	}
//	builder.AddTemplate(template)
func (b *WorkflowBuilder) AddTemplate(template v1alpha1.Template) *WorkflowBuilder {
	b.insertTemplate(template)
	return b
}

// insertTemplate adds a template to the workflow, deduplicating by name.
func (b *WorkflowBuilder) insertTemplate(t v1alpha1.Template) {
	if _, exists := b.uniqueTemplates[t.Name]; !exists {
		b.templates = append(b.templates, t)
		b.uniqueTemplates[t.Name] = struct{}{}
	}
}
