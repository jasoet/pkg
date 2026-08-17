package builder

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jasoet/pkg/v3/otel"
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
	entryPoint   []v1alpha1.ParallelSteps
	templates    []v1alpha1.Template
	exitHandlers []v1alpha1.ParallelSteps
	// exitHandlersPriority holds cleanup/destroy steps that must run before other exit
	// handlers. Kept separate so insertion order is preserved within each group instead of
	// being reversed by repeated prepending.
	exitHandlersPriority []v1alpha1.ParallelSteps
	metrics              *v1alpha1.Metrics
	uniqueTemplates      map[string]struct{}
	errors               []error

	// baseCtx is the parent context used to root builder trace spans. It defaults to
	// context.Background() and can be overridden with WithContext so spans are children of
	// the caller's trace instead of orphan roots.
	baseCtx context.Context

	// OpenTelemetry
	otelConfig *otel.Config
	otel       *otelInstrumentation
}

// builderLogger wraps otel.LogHelper so that low-severity (Debug/Info) messages are only
// emitted when an OTel config is present. Without OTel, LogHelper falls back to an
// unleveled zerolog writer on stderr; suppressing Debug/Info there keeps the library quiet
// by default while still surfacing Warn/Error.
type builderLogger struct {
	h       *otel.LogHelper
	verbose bool
}

func (l *builderLogger) Debug(msg string, fields ...otel.Field) {
	if l.verbose {
		l.h.Debug(msg, fields...)
	}
}

func (l *builderLogger) Info(msg string, fields ...otel.Field) {
	if l.verbose {
		l.h.Info(msg, fields...)
	}
}

func (l *builderLogger) Warn(msg string, fields ...otel.Field) { l.h.Warn(msg, fields...) }

func (l *builderLogger) Error(err error, msg string, fields ...otel.Field) {
	l.h.Error(err, msg, fields...)
}

// newLogger builds a leveled logger for the given function scope.
func (b *WorkflowBuilder) newLogger(ctx context.Context, function string) *builderLogger {
	return &builderLogger{
		h:       otel.NewLogHelper(ctx, b.otelConfig, "github.com/jasoet/pkg/v3/argo/builder", function),
		verbose: b.otelConfig != nil,
	}
}

// context returns the builder's base context, defaulting to context.Background().
func (b *WorkflowBuilder) context() context.Context {
	if b.baseCtx != nil {
		return b.baseCtx
	}
	return context.Background()
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
		baseCtx:         context.Background(),
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
	ctx := b.context()

	// Start tracing
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.Add")
		defer span.End()
	}

	logger := b.newLogger(ctx, "WorkflowBuilder.Add")
	logger.Debug("Adding workflow source")

	// Get templates from source
	templates, err := source.Templates()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("%w: failed to get templates: %w", ErrTemplateSource, err))
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
		b.errors = append(b.errors, fmt.Errorf("%w: failed to get steps: %w", ErrTemplateSource, err))
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
	ctx := b.context()

	// Start tracing
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.AddParallel")
		defer span.End()
	}

	logger := b.newLogger(ctx, "WorkflowBuilder.AddParallel")
	logger.Debug("Adding parallel workflow source")

	// Get templates from source
	templates, err := source.Templates()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("%w: failed to get templates: %w", ErrTemplateSource, err))
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
		b.errors = append(b.errors, fmt.Errorf("%w: failed to get parallel steps: %w", ErrTemplateSource, err))
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
//
// AddExitHandler adds an exit handler from a WorkflowSource. Exit handlers run
// after the main workflow completes (regardless of success or failure).
//
// Note: Steps with names containing "destroy" or "cleanup" are automatically
// prioritized (run first) in the exit handler sequence, ensuring resource
// cleanup runs before other exit steps. Insertion order is preserved within the
// priority group and within the normal group.
func (b *WorkflowBuilder) AddExitHandler(source WorkflowSource) *WorkflowBuilder {
	ctx := b.context()

	// Start tracing
	if b.otel != nil {
		var span trace.Span
		ctx, span = b.otel.startSpan(ctx, "WorkflowBuilder.AddExitHandler")
		defer span.End()
	}

	logger := b.newLogger(ctx, "WorkflowBuilder.AddExitHandler")
	logger.Debug("Adding exit handler")

	// Get templates from source
	templates, err := source.Templates()
	if err != nil {
		b.errors = append(b.errors, fmt.Errorf("%w: failed to get exit handler templates: %w", ErrTemplateSource, err))
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
		b.errors = append(b.errors, fmt.Errorf("%w: failed to get exit handler steps: %w", ErrTemplateSource, err))
		logger.Error(err, "Failed to get steps from exit handler")
		return b
	}

	// Add exit handler steps, appending to the priority or normal group. Appending (rather
	// than prepending) preserves the relative insertion order within each group.
	for _, step := range steps {
		ps := v1alpha1.ParallelSteps{Steps: []v1alpha1.WorkflowStep{step}}
		if isPriorityExitStep(step.Name) {
			b.exitHandlersPriority = append(b.exitHandlersPriority, ps)
		} else {
			b.exitHandlers = append(b.exitHandlers, ps)
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
	ctx := b.context()

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

	logger := b.newLogger(ctx, "WorkflowBuilder.Build")
	logger.Debug("Building workflow",
		otel.F("name", b.namePrefix),
		otel.F("namespace", b.namespace),
		otel.F("steps_count", len(b.entryPoint)),
		otel.F("templates_count", len(b.templates)),
		otel.F("exit_handlers_count", len(b.exitHandlers)+len(b.exitHandlersPriority)))

	// Check for errors
	if err := b.joinedError(); err != nil {
		if b.otel != nil {
			b.otel.recordError(ctx, "build_validation_error", err)
		}
		logger.Error(err, "Failed to build workflow")
		return nil, err
	}

	// Build the entrypoint steps. If no steps were provided, insert a no-op step so the
	// generated workflow is valid — Argo rejects a Steps template with zero steps.
	const entrypointName = "main"
	entrySteps := b.entryPoint
	if len(entrySteps) == 0 {
		logger.Debug("No steps provided, inserting a no-op step")
		entrySteps = []v1alpha1.ParallelSteps{{Steps: []v1alpha1.WorkflowStep{b.noopStep()}}}
	}
	entrypoint := v1alpha1.Template{
		Name:  entrypointName,
		Steps: entrySteps,
	}

	// Build a fresh templates slice so Build() is safe to call multiple times.
	templates := make([]v1alpha1.Template, len(b.templates), len(b.templates)+2)
	copy(templates, b.templates)
	templates = append(templates, entrypoint)

	// Create exit handler template if needed.
	const exitHandlerName = "exit-handler"
	var onExit string
	exitSteps := b.orderedExitHandlers()
	if len(exitSteps) > 0 {
		templates = append(templates, v1alpha1.Template{
			Name:  exitHandlerName,
			Steps: exitSteps,
		})
		onExit = exitHandlerName
	}

	wf := b.assembleWorkflow(entrypointName, onExit, templates)

	// Record success metrics
	if b.otel != nil {
		b.otel.incrementCounter(ctx, "workflows_built", 1)
		b.otel.addSpanAttributes(ctx,
			attribute.String("workflow.name", b.namePrefix),
			attribute.String("workflow.namespace", b.namespace),
			attribute.Int("workflow.templates_count", len(wf.Spec.Templates)),
			attribute.Int("workflow.steps_count", len(entrySteps)),
			attribute.Bool("workflow.has_exit_handler", onExit != ""),
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
	ctx := b.context()

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

	logger := b.newLogger(ctx, "WorkflowBuilder.BuildWithEntrypoint")
	logger.Debug("Building workflow with custom entrypoint",
		otel.F("name", b.namePrefix),
		otel.F("namespace", b.namespace),
		otel.F("entrypoint", entrypointName),
		otel.F("templates_count", len(b.templates)))

	// Check for errors
	if err := b.joinedError(); err != nil {
		if b.otel != nil {
			b.otel.recordError(ctx, "build_validation_error", err)
		}
		logger.Error(err, "Failed to build workflow")
		return nil, err
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
		err := fmt.Errorf("%w: %q", ErrEntrypointNotFound, entrypointName)
		if b.otel != nil {
			b.otel.recordError(ctx, "build_validation_error", err)
		}
		logger.Error(err, "Entrypoint template not found")
		return nil, err
	}

	// Build a fresh templates slice so BuildWithEntrypoint() is safe to call multiple times.
	templates := make([]v1alpha1.Template, len(b.templates), len(b.templates)+1)
	copy(templates, b.templates)

	// Create exit handler template if needed
	const exitHandlerName = "exit-handler"
	var onExit string
	exitSteps := b.orderedExitHandlers()
	if len(exitSteps) > 0 {
		templates = append(templates, v1alpha1.Template{
			Name:  exitHandlerName,
			Steps: exitSteps,
		})
		onExit = exitHandlerName
	}

	wf := b.assembleWorkflow(entrypointName, onExit, templates)

	// Record success metrics
	if b.otel != nil {
		b.otel.incrementCounter(ctx, "workflows_built", 1)
		b.otel.addSpanAttributes(ctx,
			attribute.String("workflow.name", b.namePrefix),
			attribute.String("workflow.namespace", b.namespace),
			attribute.String("workflow.entrypoint", entrypointName),
			attribute.Int("workflow.templates_count", len(wf.Spec.Templates)),
			attribute.Bool("workflow.has_exit_handler", onExit != ""),
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

// insertTemplate adds a template to the workflow, deduplicating by name. Adding the same
// template (identical content) more than once is a no-op. Adding a DIFFERENT template under
// a name that is already taken records an ErrTemplateConflict error — silently dropping the
// second definition would otherwise make a step run the wrong image or command.
func (b *WorkflowBuilder) insertTemplate(t v1alpha1.Template) {
	if _, exists := b.uniqueTemplates[t.Name]; exists {
		if existing := b.findTemplate(t.Name); existing != nil && !reflect.DeepEqual(*existing, t) {
			b.errors = append(b.errors, fmt.Errorf("%w: %q", ErrTemplateConflict, t.Name))
		}
		return
	}
	b.templates = append(b.templates, t)
	b.uniqueTemplates[t.Name] = struct{}{}
}

// findTemplate returns a pointer to the already-registered template with the given name,
// or nil if none exists.
func (b *WorkflowBuilder) findTemplate(name string) *v1alpha1.Template {
	for i := range b.templates {
		if b.templates[i].Name == name {
			return &b.templates[i]
		}
	}
	return nil
}

// joinedError aggregates every accumulated error into a single error via errors.Join, so
// callers see all build problems (not just the first) and can match any of them with
// errors.Is. Returns nil when no errors were recorded.
func (b *WorkflowBuilder) joinedError() error {
	if len(b.errors) == 0 {
		return nil
	}
	return errors.Join(b.errors...)
}

// isPriorityExitStep reports whether an exit-handler step should run before other exit
// steps (cleanup/teardown ordering).
func isPriorityExitStep(name string) bool {
	return strings.Contains(name, "destroy") || strings.Contains(name, "cleanup")
}

// orderedExitHandlers returns priority exit steps followed by normal exit steps, each in
// insertion order.
func (b *WorkflowBuilder) orderedExitHandlers() []v1alpha1.ParallelSteps {
	if len(b.exitHandlersPriority) == 0 {
		return b.exitHandlers
	}
	out := make([]v1alpha1.ParallelSteps, 0, len(b.exitHandlersPriority)+len(b.exitHandlers))
	out = append(out, b.exitHandlersPriority...)
	out = append(out, b.exitHandlers...)
	return out
}

// noopStep returns a step referencing an inserted no-op template. The template is added to
// the builder (deduplicated) as a side effect so the generated workflow references a real
// template.
func (b *WorkflowBuilder) noopStep() v1alpha1.WorkflowStep {
	const noopTemplateName = "noop-template"
	b.insertTemplate(v1alpha1.Template{
		Name: noopTemplateName,
		Container: &corev1.Container{
			Image:   "alpine:3.19",
			Command: []string{"sh", "-c"},
			Args:    []string{"echo noop"},
		},
	})
	return v1alpha1.WorkflowStep{Name: "noop", Template: noopTemplateName}
}

// assembleWorkflow constructs the final Workflow. It deep-copies builder-owned state
// (labels, annotations, volumes, and every template) so a returned workflow can be mutated
// freely without affecting the builder or any other workflow produced by it. The default
// retry strategy, when set, is applied only to leaf templates (those without their own
// Steps) so it never wraps the generated entrypoint or exit-handler orchestration
// templates — which would otherwise re-run already-succeeded steps.
func (b *WorkflowBuilder) assembleWorkflow(entrypointName, onExit string, templates []v1alpha1.Template) *v1alpha1.Workflow {
	// Deep-copy templates so their internal pointers (Container, Script, RetryStrategy, ...)
	// are not aliased with builder-owned state.
	copiedTemplates := make([]v1alpha1.Template, len(templates))
	for i := range templates {
		copiedTemplates[i] = *templates[i].DeepCopy()
	}

	// Apply the default retry strategy to leaf templates only.
	if b.retryStrategy != nil {
		for i := range copiedTemplates {
			if copiedTemplates[i].RetryStrategy == nil && len(copiedTemplates[i].Steps) == 0 && copiedTemplates[i].DAG == nil {
				copiedTemplates[i].RetryStrategy = b.retryStrategy.DeepCopy()
			}
		}
	}

	return &v1alpha1.Workflow{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: b.namePrefix,
			Namespace:    b.namespace,
			Labels:       copyStringMap(b.labels),
			Annotations:  copyStringMap(b.annotations),
		},
		Spec: v1alpha1.WorkflowSpec{
			Entrypoint:            entrypointName,
			ServiceAccountName:    b.serviceAccount,
			Templates:             copiedTemplates,
			Volumes:               copyVolumes(b.volumes),
			Metrics:               b.metrics,
			ArchiveLogs:           b.archiveLogs,
			PodGC:                 b.podGC,
			TTLStrategy:           b.ttl,
			ActiveDeadlineSeconds: b.activeDeadlineSeconds,
			OnExit:                onExit,
		},
	}
}

// copyStringMap returns a shallow copy of a string map, or nil if the input is nil.
func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// copyVolumes returns a deep copy of the volumes slice, or nil if the input is empty.
func copyVolumes(vols []corev1.Volume) []corev1.Volume {
	if len(vols) == 0 {
		return nil
	}
	out := make([]corev1.Volume, len(vols))
	for i := range vols {
		out[i] = *vols[i].DeepCopy()
	}
	return out
}
