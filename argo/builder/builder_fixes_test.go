package builder

import (
	"context"
	"io"
	"os"
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	"github.com/jasoet/pkg/v3/argo/builder/template"
	"github.com/jasoet/pkg/v3/otel"
)

func findTemplateByName(wf *v1alpha1.Workflow, name string) *v1alpha1.Template {
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].Name == name {
			return &wf.Spec.Templates[i]
		}
	}
	return nil
}

// TestBuild_DefaultRetryStrategy_SkipsOrchestrationTemplates verifies the default retry
// strategy is applied to leaf templates only — never to the generated "main" entrypoint or
// the "exit-handler" steps template. Retrying an orchestration template would re-run
// already-succeeded steps (deploys, payments).
func TestBuild_DefaultRetryStrategy_SkipsOrchestrationTemplates(t *testing.T) {
	limit := intstr.FromInt32(3)
	retry := &v1alpha1.RetryStrategy{Limit: &limit, RetryPolicy: "Always"}

	wf, err := NewWorkflowBuilder("test", "argo", WithRetryStrategy(retry)).
		Add(template.NewContainer("deploy", "app:v1", template.WithCommand("deploy.sh"))).
		AddExitHandler(template.NewContainer("cleanup", "app:v1", template.WithCommand("cleanup.sh"))).
		Build()
	require.NoError(t, err)

	mainTmpl := findTemplateByName(wf, "main")
	require.NotNil(t, mainTmpl)
	assert.Nil(t, mainTmpl.RetryStrategy, "main (entrypoint steps) template must NOT get the default retry strategy")

	exitTmpl := findTemplateByName(wf, "exit-handler")
	require.NotNil(t, exitTmpl)
	assert.Nil(t, exitTmpl.RetryStrategy, "exit-handler steps template must NOT get the default retry strategy")

	leaf := findTemplateByName(wf, "deploy-template")
	require.NotNil(t, leaf)
	require.NotNil(t, leaf.RetryStrategy, "leaf container template SHOULD get the default retry strategy")
	assert.Equal(t, 3, leaf.RetryStrategy.Limit.IntValue())
}

// TestBuild_DeepCopiesBuilderState verifies a built workflow does not alias builder-owned
// maps/slices/pointers, so mutating one workflow affects neither the builder nor another
// workflow produced by the same builder.
func TestBuild_DeepCopiesBuilderState(t *testing.T) {
	b := NewWorkflowBuilder("test", "argo",
		WithLabels(map[string]string{"app": "orig"}),
		WithAnnotations(map[string]string{"note": "orig"}),
		WithVolume(corev1.Volume{Name: "data", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}}),
	).Add(template.NewContainer("step", "alpine:latest", template.WithCommand("echo", "hi")))

	wf1, err := b.Build()
	require.NoError(t, err)
	wf2, err := b.Build()
	require.NoError(t, err)

	// Mutate wf1's owned state.
	wf1.Labels["app"] = "mutated"
	wf1.Annotations["note"] = "mutated"
	wf1.Spec.Volumes[0].Name = "mutated"
	if tmpl := findTemplateByName(wf1, "step-template"); tmpl != nil && tmpl.Container != nil {
		tmpl.Container.Command[0] = "mutated"
	}

	// wf2 must be unaffected.
	assert.Equal(t, "orig", wf2.Labels["app"])
	assert.Equal(t, "orig", wf2.Annotations["note"])
	assert.Equal(t, "data", wf2.Spec.Volumes[0].Name)
	tmpl2 := findTemplateByName(wf2, "step-template")
	require.NotNil(t, tmpl2)
	assert.Equal(t, "echo", tmpl2.Container.Command[0])

	// Builder-owned state must be unaffected too.
	assert.Equal(t, "orig", b.labels["app"])
	assert.Equal(t, "orig", b.annotations["note"])
	assert.Equal(t, "data", b.volumes[0].Name)
}

// TestBuild_EmptyInsertsNoop verifies an empty builder yields a valid workflow with a
// no-op step (not a zero-step entrypoint that the server rejects).
func TestBuild_EmptyInsertsNoop(t *testing.T) {
	wf, err := NewWorkflowBuilder("empty", "argo").Build()
	require.NoError(t, err)

	mainTmpl := findTemplateByName(wf, "main")
	require.NotNil(t, mainTmpl)
	require.Len(t, mainTmpl.Steps, 1)
	require.Len(t, mainTmpl.Steps[0].Steps, 1)
	assert.Equal(t, "noop", mainTmpl.Steps[0].Steps[0].Name)

	noopTmpl := findTemplateByName(wf, "noop-template")
	require.NotNil(t, noopTmpl, "noop leaf template must be inserted")
	require.NotNil(t, noopTmpl.Container)
}

// TestInsertTemplate_ConflictRecordsError verifies that adding a DIFFERENT template under
// an existing name records an ErrTemplateConflict rather than silently dropping it.
func TestInsertTemplate_ConflictRecordsError(t *testing.T) {
	tmplA := v1alpha1.Template{Name: "dup", Container: &corev1.Container{Image: "a:1"}}
	tmplB := v1alpha1.Template{Name: "dup", Container: &corev1.Container{Image: "b:2"}}

	_, err := NewWorkflowBuilder("test", "argo").
		AddTemplate(tmplA).
		AddTemplate(tmplB).
		Add(template.NewContainer("step", "alpine:latest")).
		Build()

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTemplateConflict)
	assert.Contains(t, err.Error(), "dup")
}

// TestInsertTemplate_IdenticalDuplicateIsNoError verifies identical re-adds are silently
// deduplicated (patterns legitimately add the same template many times).
func TestInsertTemplate_IdenticalDuplicateIsNoError(t *testing.T) {
	tmpl := v1alpha1.Template{Name: "same", Container: &corev1.Container{Image: "a:1"}}

	wf, err := NewWorkflowBuilder("test", "argo").
		AddTemplate(tmpl).
		AddTemplate(tmpl).
		Add(template.NewContainer("step", "alpine:latest")).
		Build()

	require.NoError(t, err)
	assert.NotNil(t, findTemplateByName(wf, "same"))
}

// TestBuild_JoinsAllErrors verifies Build aggregates every accumulated error (not just the
// first) via errors.Join.
func TestBuild_JoinsAllErrors(t *testing.T) {
	_, err := NewWorkflowBuilder("test", "argo").
		AddTemplate(v1alpha1.Template{Name: "x", Container: &corev1.Container{Image: "a:1"}}).
		AddTemplate(v1alpha1.Template{Name: "x", Container: &corev1.Container{Image: "b:2"}}).
		AddTemplate(v1alpha1.Template{Name: "y", Container: &corev1.Container{Image: "a:1"}}).
		AddTemplate(v1alpha1.Template{Name: "y", Container: &corev1.Container{Image: "c:3"}}).
		Build()

	require.Error(t, err)
	assert.ErrorIs(t, err, ErrTemplateConflict)
	// Both conflicting names should appear in the joined error.
	assert.Contains(t, err.Error(), "x")
	assert.Contains(t, err.Error(), "y")
}

// TestAddExitHandler_PreservesOrder verifies that multiple cleanup steps keep their
// insertion order (they are no longer reversed by repeated prepending) and still run
// before normal exit steps.
func TestAddExitHandler_PreservesOrder(t *testing.T) {
	wf, err := NewWorkflowBuilder("test", "argo").
		Add(template.NewContainer("main-step", "alpine:latest", template.WithCommand("echo", "hi"))).
		AddExitHandler(template.NewContainer("cleanup-a", "alpine:latest", template.WithCommand("echo", "a"))).
		AddExitHandler(template.NewContainer("cleanup-b", "alpine:latest", template.WithCommand("echo", "b"))).
		AddExitHandler(template.NewContainer("notify", "alpine:latest", template.WithCommand("echo", "n"))).
		Build()
	require.NoError(t, err)

	exit := findTemplateByName(wf, "exit-handler")
	require.NotNil(t, exit)

	var order []string
	for _, ps := range exit.Steps {
		for _, s := range ps.Steps {
			order = append(order, s.Name)
		}
	}
	// Priority cleanup steps first (in insertion order a,b), then normal steps.
	assert.Equal(t, []string{"cleanup-a", "cleanup-b", "notify"}, order)
}

// TestBuild_NoStderrLogsWithoutOTelConfig verifies that, without an OTel config, the
// builder does not emit unleveled Debug/Info logs to stderr.
func TestBuild_NoStderrLogsWithoutOTelConfig(t *testing.T) {
	old := os.Stderr
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = old })

	_, buildErr := NewWorkflowBuilder("quiet", "argo").
		Add(template.NewContainer("step", "alpine:latest", template.WithCommand("echo", "hi"))).
		Build()
	require.NoError(t, buildErr)

	require.NoError(t, w.Close())
	os.Stderr = old
	out, err := io.ReadAll(r)
	require.NoError(t, err)
	assert.Empty(t, string(out), "no Debug/Info stderr output expected when OTel config is absent")
}

// TestBuild_WithContext_RootsSpansAtParent verifies WithContext threads a parent context so
// builder spans are children of the caller's trace rather than orphan roots.
func TestBuild_WithContext_RootsSpansAtParent(t *testing.T) {
	exporter := tracetest.NewInMemoryExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))
	t.Cleanup(func() { _ = tp.Shutdown(context.Background()) })

	cfg := otel.NewConfig("test", otel.WithTracerProvider(tp))
	parentCtx, parentSpan := tp.Tracer("test").Start(context.Background(), "parent")
	parentTraceID := parentSpan.SpanContext().TraceID()

	_, err := NewWorkflowBuilder("ctx", "argo", WithContext(parentCtx), WithOTelConfig(cfg)).
		Add(template.NewContainer("step", "alpine:latest", template.WithCommand("echo", "hi"))).
		Build()
	require.NoError(t, err)
	parentSpan.End()

	spans := exporter.GetSpans()
	require.NotEmpty(t, spans)
	for _, s := range spans {
		assert.Equal(t, parentTraceID, s.SpanContext.TraceID(),
			"builder span %q must share the parent trace, not start a new root", s.Name)
	}
}
