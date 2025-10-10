package builder

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo/builder/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestWorkflowBuilder_Build(t *testing.T) {
	tests := []struct {
		name             string
		setupBuilder     func() *WorkflowBuilder
		wantErr          bool
		validateWorkflow func(t *testing.T, wf *v1alpha1.Workflow)
	}{
		{
			name: "empty workflow should have main entrypoint",
			setupBuilder: func() *WorkflowBuilder {
				return NewWorkflowBuilder("test", "argo")
			},
			wantErr: false,
			validateWorkflow: func(t *testing.T, wf *v1alpha1.Workflow) {
				assert.Equal(t, "test-", wf.GenerateName)
				assert.Equal(t, "argo", wf.Namespace)
				assert.Equal(t, "main", wf.Spec.Entrypoint)
				assert.Equal(t, "argo-workflow", wf.Spec.ServiceAccountName)
				// Should have at least the main template
				assert.GreaterOrEqual(t, len(wf.Spec.Templates), 1)
			},
		},
		{
			name: "workflow with single container step",
			setupBuilder: func() *WorkflowBuilder {
				container := template.NewContainer("hello", "alpine:latest",
					template.WithCommand("echo", "hello"))
				return NewWorkflowBuilder("test", "argo").
					Add(container)
			},
			wantErr: false,
			validateWorkflow: func(t *testing.T, wf *v1alpha1.Workflow) {
				assert.Equal(t, "test-", wf.GenerateName)
				// Should have container template + main template
				assert.GreaterOrEqual(t, len(wf.Spec.Templates), 2)

				// Check main template exists and has steps
				var mainTemplate *v1alpha1.Template
				for i := range wf.Spec.Templates {
					if wf.Spec.Templates[i].Name == "main" {
						mainTemplate = &wf.Spec.Templates[i]
						break
					}
				}
				require.NotNil(t, mainTemplate, "main template should exist")
				assert.Len(t, mainTemplate.Steps, 1)
				assert.Len(t, mainTemplate.Steps[0].Steps, 1)
				assert.Equal(t, "hello", mainTemplate.Steps[0].Steps[0].Name)
			},
		},
		{
			name: "workflow with multiple steps",
			setupBuilder: func() *WorkflowBuilder {
				step1 := template.NewContainer("step1", "alpine:latest",
					template.WithCommand("echo", "step1"))
				step2 := template.NewContainer("step2", "alpine:latest",
					template.WithCommand("echo", "step2"))
				return NewWorkflowBuilder("test", "argo").
					Add(step1).
					Add(step2)
			},
			wantErr: false,
			validateWorkflow: func(t *testing.T, wf *v1alpha1.Workflow) {
				// Should have 2 container templates + main template
				assert.GreaterOrEqual(t, len(wf.Spec.Templates), 3)

				// Check main template has 2 sequential steps
				var mainTemplate *v1alpha1.Template
				for i := range wf.Spec.Templates {
					if wf.Spec.Templates[i].Name == "main" {
						mainTemplate = &wf.Spec.Templates[i]
						break
					}
				}
				require.NotNil(t, mainTemplate)
				assert.Len(t, mainTemplate.Steps, 2) // 2 sequential groups
			},
		},
		{
			name: "workflow with exit handler",
			setupBuilder: func() *WorkflowBuilder {
				mainStep := template.NewContainer("main", "alpine:latest",
					template.WithCommand("echo", "main"))
				cleanup := template.NewContainer("cleanup", "alpine:latest",
					template.WithCommand("echo", "cleanup"))
				return NewWorkflowBuilder("test", "argo").
					Add(mainStep).
					AddExitHandler(cleanup)
			},
			wantErr: false,
			validateWorkflow: func(t *testing.T, wf *v1alpha1.Workflow) {
				// Should have exit-handler specified
				assert.Equal(t, "exit-handler", wf.Spec.OnExit)

				// Check exit-handler template exists
				var exitTemplate *v1alpha1.Template
				for i := range wf.Spec.Templates {
					if wf.Spec.Templates[i].Name == "exit-handler" {
						exitTemplate = &wf.Spec.Templates[i]
						break
					}
				}
				require.NotNil(t, exitTemplate, "exit-handler template should exist")
				assert.Len(t, exitTemplate.Steps, 1)
			},
		},
		{
			name: "workflow with custom service account",
			setupBuilder: func() *WorkflowBuilder {
				step := template.NewContainer("step", "alpine:latest")
				return NewWorkflowBuilder("test", "argo",
					WithServiceAccount("custom-sa")).
					Add(step)
			},
			wantErr: false,
			validateWorkflow: func(t *testing.T, wf *v1alpha1.Workflow) {
				assert.Equal(t, "custom-sa", wf.Spec.ServiceAccountName)
			},
		},
		{
			name: "workflow with labels and annotations",
			setupBuilder: func() *WorkflowBuilder {
				step := template.NewContainer("step", "alpine:latest")
				return NewWorkflowBuilder("test", "argo",
					WithLabels(map[string]string{
						"app": "myapp",
						"env": "prod",
					}),
					WithAnnotations(map[string]string{
						"description": "test workflow",
					})).
					Add(step)
			},
			wantErr: false,
			validateWorkflow: func(t *testing.T, wf *v1alpha1.Workflow) {
				assert.Equal(t, "myapp", wf.Labels["app"])
				assert.Equal(t, "prod", wf.Labels["env"])
				assert.Equal(t, "test workflow", wf.Annotations["description"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.setupBuilder()
			wf, err := builder.Build()

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, wf)
			} else {
				assert.NoError(t, err)
				require.NotNil(t, wf)
				if tt.validateWorkflow != nil {
					tt.validateWorkflow(t, wf)
				}
			}
		})
	}
}

func TestWorkflowBuilder_TemplateDeduplication(t *testing.T) {
	// Create two containers that use the same image/command
	// The templates should be deduplicated
	step1 := template.NewContainer("step1", "alpine:latest",
		template.WithCommand("echo", "hello"))
	step2 := template.NewContainer("step2", "alpine:latest",
		template.WithCommand("echo", "world"))

	builder := NewWorkflowBuilder("test", "argo").
		Add(step1).
		Add(step2)

	wf, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, wf)

	// Check that we have the correct number of templates
	// Should have: step1-template, step2-template, main = 3 templates
	assert.Len(t, wf.Spec.Templates, 3)
}

func TestContainer_FluentAPI(t *testing.T) {
	container := template.NewContainer("test", "alpine:latest").
		Command("sh", "-c").
		Args("echo hello").
		Env("KEY", "value").
		WorkingDir("/app").
		CPU("100m").
		Memory("128Mi")

	steps, err := container.Steps()
	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "test", steps[0].Name)

	templates, err := container.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	require.NotNil(t, tmpl.Container)
	assert.Equal(t, "alpine:latest", tmpl.Container.Image)
	assert.Equal(t, []string{"sh", "-c"}, tmpl.Container.Command)
	assert.Equal(t, []string{"echo hello"}, tmpl.Container.Args)
	assert.Len(t, tmpl.Container.Env, 1)
	assert.Equal(t, "KEY", tmpl.Container.Env[0].Name)
	assert.Equal(t, "value", tmpl.Container.Env[0].Value)
	assert.Equal(t, "/app", tmpl.Container.WorkingDir)
}

func TestNoop_Implementation(t *testing.T) {
	noop := template.NewNoop()

	steps, err := noop.Steps()
	require.NoError(t, err)
	assert.Len(t, steps, 1)
	assert.Equal(t, "noop", steps[0].Name)

	templates, err := noop.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	require.NotNil(t, tmpl.Container)
	assert.Equal(t, "alpine:3.19", tmpl.Container.Image)
}

func TestWorkflowBuilder_AddTemplate(t *testing.T) {
	// Create a raw Argo template
	rawTemplate := v1alpha1.Template{
		Name: "custom-template",
		Container: &corev1.Container{
			Image:   "busybox:latest",
			Command: []string{"echo"},
			Args:    []string{"custom"},
		},
	}

	builder := NewWorkflowBuilder("test", "argo").
		AddTemplate(rawTemplate).
		Add(template.NewContainer("main-step", "alpine:latest"))

	wf, err := builder.Build()
	require.NoError(t, err)
	require.NotNil(t, wf)

	// Find the custom template - it should exist
	var customTemplate *v1alpha1.Template
	for i := range wf.Spec.Templates {
		if wf.Spec.Templates[i].Name == "custom-template" {
			customTemplate = &wf.Spec.Templates[i]
			break
		}
	}
	require.NotNil(t, customTemplate, "custom template should exist")
	assert.Equal(t, "busybox:latest", customTemplate.Container.Image)
}

func TestWorkflowBuilder_WithMetrics_Error(t *testing.T) {
	// Test error handling in WithMetrics
	provider := &mockMetricsProvider{
		metrics: nil,
		err:     assert.AnError,
	}

	builder := NewWorkflowBuilder("test", "argo").
		WithMetrics(provider).
		Add(template.NewContainer("step", "alpine:latest"))

	wf, err := builder.Build()
	require.Error(t, err)
	assert.Nil(t, wf)
	assert.Contains(t, err.Error(), "failed to get metrics")
}

func TestWorkflowBuilder_AddParallel(t *testing.T) {
	t.Run("adds parallel workflow source", func(t *testing.T) {
		parallelSource := &mockParallelSource{
			parallelSteps: []v1alpha1.ParallelSteps{
				{
					Steps: []v1alpha1.WorkflowStep{
						{Name: "parallel-1", Template: "template-1"},
						{Name: "parallel-2", Template: "template-2"},
					},
				},
			},
			templates: []v1alpha1.Template{
				{
					Name: "template-1",
					Container: &corev1.Container{
						Image: "alpine:latest",
					},
				},
				{
					Name: "template-2",
					Container: &corev1.Container{
						Image: "alpine:latest",
					},
				},
			},
		}

		builder := NewWorkflowBuilder("test", "argo").
			AddParallel(parallelSource)

		wf, err := builder.Build()
		require.NoError(t, err)
		require.NotNil(t, wf)

		// Find main template
		var mainTemplate *v1alpha1.Template
		for i := range wf.Spec.Templates {
			if wf.Spec.Templates[i].Name == "main" {
				mainTemplate = &wf.Spec.Templates[i]
				break
			}
		}
		require.NotNil(t, mainTemplate)

		// Verify parallel steps are added
		require.Len(t, mainTemplate.Steps, 1)
		assert.Len(t, mainTemplate.Steps[0].Steps, 2) // Two parallel steps
		assert.Equal(t, "parallel-1", mainTemplate.Steps[0].Steps[0].Name)
		assert.Equal(t, "parallel-2", mainTemplate.Steps[0].Steps[1].Name)
	})

	t.Run("handles template errors", func(t *testing.T) {
		parallelSource := &mockParallelSource{
			templatesErr: assert.AnError,
		}

		builder := NewWorkflowBuilder("test", "argo").
			AddParallel(parallelSource)

		wf, err := builder.Build()
		require.Error(t, err)
		assert.Nil(t, wf)
		assert.Contains(t, err.Error(), "failed to get templates")
	})

	t.Run("handles parallel steps errors", func(t *testing.T) {
		parallelSource := &mockParallelSource{
			templates: []v1alpha1.Template{
				{Name: "test", Container: &corev1.Container{Image: "alpine"}},
			},
			parallelStepsErr: assert.AnError,
		}

		builder := NewWorkflowBuilder("test", "argo").
			AddParallel(parallelSource)

		wf, err := builder.Build()
		require.Error(t, err)
		assert.Nil(t, wf)
		assert.Contains(t, err.Error(), "failed to get parallel steps")
	})
}

func TestWorkflowBuilder_BuildWithEntrypoint(t *testing.T) {
	t.Run("builds workflow with custom entrypoint", func(t *testing.T) {
		customTemplate := v1alpha1.Template{
			Name: "custom-main",
			Steps: []v1alpha1.ParallelSteps{
				{
					Steps: []v1alpha1.WorkflowStep{
						{Name: "step-1", Template: "template-1"},
					},
				},
			},
		}

		stepTemplate := v1alpha1.Template{
			Name: "template-1",
			Container: &corev1.Container{
				Image:   "alpine:latest",
				Command: []string{"echo", "hello"},
			},
		}

		builder := NewWorkflowBuilder("test", "argo").
			AddTemplate(customTemplate).
			AddTemplate(stepTemplate)

		wf, err := builder.BuildWithEntrypoint("custom-main")
		require.NoError(t, err)
		require.NotNil(t, wf)

		assert.Equal(t, "custom-main", wf.Spec.Entrypoint)

		// Verify custom template exists
		var customMain *v1alpha1.Template
		for i := range wf.Spec.Templates {
			if wf.Spec.Templates[i].Name == "custom-main" {
				customMain = &wf.Spec.Templates[i]
				break
			}
		}
		require.NotNil(t, customMain, "custom entrypoint should exist")
	})

	t.Run("returns error when entrypoint not found", func(t *testing.T) {
		builder := NewWorkflowBuilder("test", "argo").
			AddTemplate(v1alpha1.Template{
				Name:      "some-template",
				Container: &corev1.Container{Image: "alpine"},
			})

		wf, err := builder.BuildWithEntrypoint("nonexistent")
		require.Error(t, err)
		assert.Nil(t, wf)
		assert.Contains(t, err.Error(), "entrypoint template 'nonexistent' not found")
	})

	t.Run("builds with exit handler", func(t *testing.T) {
		entryTemplate := v1alpha1.Template{
			Name: "my-entry",
			Steps: []v1alpha1.ParallelSteps{
				{Steps: []v1alpha1.WorkflowStep{{Name: "main", Template: "main-tmpl"}}},
			},
		}

		cleanup := template.NewContainer("cleanup", "alpine:latest",
			template.WithCommand("echo", "cleanup"))

		builder := NewWorkflowBuilder("test", "argo").
			AddTemplate(entryTemplate).
			AddExitHandler(cleanup)

		wf, err := builder.BuildWithEntrypoint("my-entry")
		require.NoError(t, err)
		require.NotNil(t, wf)

		assert.Equal(t, "my-entry", wf.Spec.Entrypoint)
		assert.Equal(t, "exit-handler", wf.Spec.OnExit)
	})
}

// mockParallelSource implements WorkflowSourceV2 for testing
type mockParallelSource struct {
	parallelSteps    []v1alpha1.ParallelSteps
	templates        []v1alpha1.Template
	parallelStepsErr error
	templatesErr     error
}

func (m *mockParallelSource) ParallelSteps() ([]v1alpha1.ParallelSteps, error) {
	return m.parallelSteps, m.parallelStepsErr
}

func (m *mockParallelSource) Templates() ([]v1alpha1.Template, error) {
	return m.templates, m.templatesErr
}
