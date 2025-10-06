package builder

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo/builder/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWorkflowBuilder_Build(t *testing.T) {
	tests := []struct {
		name          string
		setupBuilder  func() *WorkflowBuilder
		wantErr       bool
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
