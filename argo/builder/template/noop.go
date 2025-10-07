package template

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

// Noop is a no-operation workflow source that produces a single step that does nothing.
// This is useful as a placeholder or for testing workflow structures.
//
// Example:
//
//	noop := template.NewNoop()
//	builder.Add(noop)
type Noop struct {
	name string
}

// NewNoop creates a new no-op workflow source.
// The step will print "noop" to demonstrate execution.
//
// Example:
//
//	noop := template.NewNoop()
//	wf, err := builder.NewWorkflowBuilder("test", "argo").
//	    Add(noop).
//	    Build()
func NewNoop() *Noop {
	return &Noop{
		name: "noop",
	}
}

// NewNoopWithName creates a no-op workflow source with a custom name.
// Useful when you need multiple no-op steps with different names.
//
// Example:
//
//	placeholder1 := template.NewNoopWithName("placeholder-1")
//	placeholder2 := template.NewNoopWithName("placeholder-2")
func NewNoopWithName(name string) *Noop {
	return &Noop{
		name: name,
	}
}

// Steps implements WorkflowSource interface.
// Returns a single step that references the no-op template.
func (n *Noop) Steps() ([]v1alpha1.WorkflowStep, error) {
	return []v1alpha1.WorkflowStep{
		{
			Name:     n.name,
			Template: n.name + "-template",
		},
	}, nil
}

// Templates implements WorkflowSource interface.
// Returns a simple container template that prints "noop".
func (n *Noop) Templates() ([]v1alpha1.Template, error) {
	return []v1alpha1.Template{
		{
			Name: n.name + "-template",
			Container: &corev1.Container{
				Image:   "alpine:3.19",
				Command: []string{"sh", "-c"},
				Args:    []string{"echo noop"},
			},
		},
	}, nil
}
