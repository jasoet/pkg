package builder

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

// WorkflowSource is the core interface that workflow components implement.
// It provides a composable way to build workflows by defining the steps and templates
// required for a workflow operation.
//
// This interface allows you to create reusable workflow building blocks that can be
// combined using WorkflowBuilder.Add() to construct complex workflows.
//
// Example implementation:
//
//	type MyWorkflowSource struct {
//	    name string
//	}
//
//	func (m *MyWorkflowSource) Steps() ([]v1alpha1.WorkflowStep, error) {
//	    return []v1alpha1.WorkflowStep{{
//	        Name:     "my-step",
//	        Template: "my-template",
//	    }}, nil
//	}
//
//	func (m *MyWorkflowSource) Templates() ([]v1alpha1.Template, error) {
//	    return []v1alpha1.Template{{
//	        Name: "my-template",
//	        Container: &corev1.Container{
//	            Image: "alpine:latest",
//	            Command: []string{"echo", "hello"},
//	        },
//	    }}, nil
//	}
type WorkflowSource interface {
	// Steps returns the workflow steps to execute.
	// These steps will be added to the workflow's entrypoint in the order they are returned.
	Steps() ([]v1alpha1.WorkflowStep, error)

	// Templates returns all templates required by the steps.
	// The WorkflowBuilder will automatically deduplicate templates with the same name.
	Templates() ([]v1alpha1.Template, error)
}

// WorkflowSourceV2 is an extended interface that supports parallel step execution.
// Use this interface when you need to define steps that should run in parallel.
//
// Example:
//
//	func (m *MyParallelSource) ParallelSteps() ([]v1alpha1.ParallelSteps, error) {
//	    return []v1alpha1.ParallelSteps{
//	        {
//	            Steps: []v1alpha1.WorkflowStep{
//	                {Name: "step-1", Template: "template-1"},
//	                {Name: "step-2", Template: "template-2"}, // runs in parallel with step-1
//	            },
//	        },
//	        {
//	            Steps: []v1alpha1.WorkflowStep{
//	                {Name: "step-3", Template: "template-3"}, // runs after step-1 and step-2 complete
//	            },
//	        },
//	    }, nil
//	}
type WorkflowSourceV2 interface {
	// ParallelSteps returns workflow steps organized for parallel execution.
	// Each ParallelSteps group contains steps that run in parallel.
	// Groups execute sequentially - the next group waits for all steps in the previous group to complete.
	ParallelSteps() ([]v1alpha1.ParallelSteps, error)

	// Templates returns all templates required by the parallel steps.
	Templates() ([]v1alpha1.Template, error)
}

// WorkflowMetricsProvider is an optional interface that workflow sources can implement
// to provide custom Prometheus metrics for the workflow.
//
// Example:
//
//	func (m *MySource) Metrics() (*v1alpha1.Metrics, error) {
//	    return &v1alpha1.Metrics{
//	        Prometheus: []*v1alpha1.Prometheus{
//	            {
//	                Name: "my_workflow_count",
//	                Help: "Number of workflow executions",
//	                Counter: &v1alpha1.Counter{Value: "1"},
//	            },
//	        },
//	    }, nil
//	}
type WorkflowMetricsProvider interface {
	// Metrics returns workflow metrics configuration.
	// These metrics will be exposed by the workflow when it executes.
	Metrics() (*v1alpha1.Metrics, error)
}
