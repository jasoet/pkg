package patterns

import (
	"fmt"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
)

// FanOutFanIn creates a workflow that executes multiple tasks in parallel,
// then aggregates results in a final step.
//
// Example:
//
//	wf := patterns.FanOutFanIn(
//	    "parallel-processing", "argo",
//	    "busybox:latest",
//	    []string{"task-1", "task-2", "task-3"},
//	)
func FanOutFanIn(name, namespace, image string, tasks []string, opts ...builder.BuilderOption) (*v1alpha1.Workflow, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("at least one task is required for fan-out/fan-in pattern")
	}

	wb := builder.NewWorkflowBuilder(name, namespace, opts...)

	// Create parallel tasks (fan-out)
	parallelSteps := make([]v1alpha1.WorkflowStep, 0, len(tasks))
	for _, taskName := range tasks {
		task := template.NewContainer(taskName, image,
			template.WithCommand("sh", "-c"),
			template.WithArgs(fmt.Sprintf("echo 'Processing %s...' && sleep 2 && echo 'Result: %s-output'", taskName, taskName)))

		steps, err := task.Steps()
		if err != nil {
			return nil, fmt.Errorf("failed to generate steps for task %s: %w", taskName, err)
		}

		templates, err := task.Templates()
		if err != nil {
			return nil, fmt.Errorf("failed to generate templates for task %s: %w", taskName, err)
		}

		// Add parallel step
		parallelSteps = append(parallelSteps, steps[0])

		// Add templates to builder
		for _, tmpl := range templates {
			wb = wb.AddTemplate(tmpl)
		}
	}

	// Add fan-in step (aggregation)
	aggregate := template.NewContainer("aggregate", image,
		template.WithCommand("sh", "-c"),
		template.WithArgs("echo 'Aggregating all results...' && echo 'All parallel tasks completed'"))

	// Create entry template with parallel steps
	entryTemplate := v1alpha1.Template{
		Name: name + "-main",
		Steps: []v1alpha1.ParallelSteps{
			{Steps: parallelSteps}, // Parallel execution
		},
	}

	// Add aggregate step as sequential after parallel
	aggSteps, err := aggregate.Steps()
	if err != nil {
		return nil, fmt.Errorf("failed to generate aggregate steps: %w", err)
	}
	entryTemplate.Steps = append(entryTemplate.Steps, v1alpha1.ParallelSteps{Steps: aggSteps})

	aggTemplates, err := aggregate.Templates()
	if err != nil {
		return nil, fmt.Errorf("failed to generate aggregate templates: %w", err)
	}

	wb = wb.AddTemplate(entryTemplate).
		AddTemplate(aggTemplates[0])

	return wb.BuildWithEntrypoint(name + "-main")
}

// ParallelDataProcessing creates a workflow that processes multiple data items in parallel.
// Each item is processed independently with the same processing logic.
//
// Example:
//
//	wf := patterns.ParallelDataProcessing(
//	    "batch-processor", "argo",
//	    "myregistry/processor:v1",
//	    []string{"data-1.csv", "data-2.csv", "data-3.csv"},
//	    "process.sh",
//	)
func ParallelDataProcessing(name, namespace, image string, dataItems []string, processingCommand string, opts ...builder.BuilderOption) (*v1alpha1.Workflow, error) {
	if len(dataItems) == 0 {
		return nil, fmt.Errorf("at least one data item is required")
	}

	wb := builder.NewWorkflowBuilder(name, namespace, opts...)

	// Create parallel processing tasks
	parallelSteps := make([]v1alpha1.WorkflowStep, 0, len(dataItems))
	for i, dataItem := range dataItems {
		taskName := fmt.Sprintf("process-%d", i)
		task := template.NewContainer(taskName, image,
			template.WithCommand("sh", "-c"),
			template.WithArgs(fmt.Sprintf("%s %s", processingCommand, dataItem)),
			template.WithEnv("DATA_ITEM", dataItem),
			template.WithEnv("ITEM_INDEX", fmt.Sprintf("%d", i)))

		steps, err := task.Steps()
		if err != nil {
			return nil, fmt.Errorf("failed to generate steps for %s: %w", taskName, err)
		}

		templates, err := task.Templates()
		if err != nil {
			return nil, fmt.Errorf("failed to generate templates for %s: %w", taskName, err)
		}

		parallelSteps = append(parallelSteps, steps[0])

		for _, tmpl := range templates {
			wb = wb.AddTemplate(tmpl)
		}
	}

	// Create entry template with all parallel steps
	entryTemplate := v1alpha1.Template{
		Name: name + "-main",
		Steps: []v1alpha1.ParallelSteps{
			{Steps: parallelSteps},
		},
	}

	wb = wb.AddTemplate(entryTemplate)

	return wb.BuildWithEntrypoint(name + "-main")
}

// MapReduce creates a map-reduce style workflow where:
// 1. Map phase: Process items in parallel
// 2. Reduce phase: Aggregate results sequentially
//
// Example:
//
//	wf := patterns.MapReduce(
//	    "word-count", "argo",
//	    "alpine:latest",
//	    []string{"file1.txt", "file2.txt", "file3.txt"},
//	    "wc -w", // map command
//	    "awk '{sum+=$1} END {print sum}'", // reduce command
//	)
func MapReduce(name, namespace, image string, inputs []string, mapCmd, reduceCmd string, opts ...builder.BuilderOption) (*v1alpha1.Workflow, error) {
	if len(inputs) == 0 {
		return nil, fmt.Errorf("at least one input is required for map-reduce")
	}

	wb := builder.NewWorkflowBuilder(name, namespace, opts...)

	// Map phase: Create parallel tasks
	mapSteps := make([]v1alpha1.WorkflowStep, 0, len(inputs))
	for i, input := range inputs {
		mapTaskName := fmt.Sprintf("map-%d", i)
		mapTask := template.NewContainer(mapTaskName, image,
			template.WithCommand("sh", "-c"),
			template.WithArgs(fmt.Sprintf("echo 'Mapping %s' && %s %s", input, mapCmd, input)),
			template.WithEnv("INPUT", input))

		steps, err := mapTask.Steps()
		if err != nil {
			return nil, fmt.Errorf("failed to generate map steps: %w", err)
		}

		templates, err := mapTask.Templates()
		if err != nil {
			return nil, fmt.Errorf("failed to generate map templates: %w", err)
		}

		mapSteps = append(mapSteps, steps[0])

		for _, tmpl := range templates {
			wb = wb.AddTemplate(tmpl)
		}
	}

	// Reduce phase: Aggregate results
	reduce := template.NewContainer("reduce", image,
		template.WithCommand("sh", "-c"),
		template.WithArgs(fmt.Sprintf("echo 'Reducing results...' && %s", reduceCmd)))

	reduceSteps, err := reduce.Steps()
	if err != nil {
		return nil, fmt.Errorf("failed to generate reduce steps: %w", err)
	}

	reduceTemplates, err := reduce.Templates()
	if err != nil {
		return nil, fmt.Errorf("failed to generate reduce templates: %w", err)
	}

	// Create entry template with map then reduce
	entryTemplate := v1alpha1.Template{
		Name: name + "-main",
		Steps: []v1alpha1.ParallelSteps{
			{Steps: mapSteps},    // Parallel map phase
			{Steps: reduceSteps}, // Sequential reduce phase
		},
	}

	wb = wb.AddTemplate(entryTemplate)
	for _, tmpl := range reduceTemplates {
		wb = wb.AddTemplate(tmpl)
	}

	return wb.BuildWithEntrypoint(name + "-main")
}

// ParallelTestSuite creates a workflow that runs multiple test suites in parallel.
// This is useful for speeding up CI/CD pipelines with independent test suites.
//
// Example:
//
//	wf := patterns.ParallelTestSuite(
//	    "test-suite", "argo",
//	    "golang:1.25",
//	    map[string]string{
//	        "unit": "go test ./internal/...",
//	        "integration": "go test ./tests/integration/...",
//	        "e2e": "go test ./tests/e2e/...",
//	    },
//	)
func ParallelTestSuite(name, namespace, image string, testSuites map[string]string, opts ...builder.BuilderOption) (*v1alpha1.Workflow, error) {
	if len(testSuites) == 0 {
		return nil, fmt.Errorf("at least one test suite is required")
	}

	wb := builder.NewWorkflowBuilder(name, namespace, opts...)

	// Create parallel test steps
	parallelSteps := make([]v1alpha1.WorkflowStep, 0, len(testSuites))
	for suiteName, testCmd := range testSuites {
		testTask := template.NewContainer("test-"+suiteName, image,
			template.WithCommand("sh", "-c"),
			template.WithArgs(fmt.Sprintf("echo 'Running %s tests...' && %s", suiteName, testCmd)),
			template.WithWorkingDir("/workspace"))

		steps, err := testTask.Steps()
		if err != nil {
			return nil, fmt.Errorf("failed to generate test steps for %s: %w", suiteName, err)
		}

		templates, err := testTask.Templates()
		if err != nil {
			return nil, fmt.Errorf("failed to generate test templates for %s: %w", suiteName, err)
		}

		parallelSteps = append(parallelSteps, steps[0])

		for _, tmpl := range templates {
			wb = wb.AddTemplate(tmpl)
		}
	}

	// Create entry template
	entryTemplate := v1alpha1.Template{
		Name: name + "-main",
		Steps: []v1alpha1.ParallelSteps{
			{Steps: parallelSteps},
		},
	}

	wb = wb.AddTemplate(entryTemplate)

	return wb.BuildWithEntrypoint(name + "-main")
}

// ParallelDeployment creates a workflow that deploys to multiple environments in parallel.
// This is useful when environments are independent and can be deployed simultaneously.
//
// Example:
//
//	wf := patterns.ParallelDeployment(
//	    "multi-region-deploy", "argo",
//	    "myregistry/deployer:v1",
//	    []string{"us-west", "us-east", "eu-central"},
//	)
func ParallelDeployment(name, namespace, deployImage string, environments []string, opts ...builder.BuilderOption) (*v1alpha1.Workflow, error) {
	if len(environments) == 0 {
		return nil, fmt.Errorf("at least one environment is required")
	}

	wb := builder.NewWorkflowBuilder(name, namespace, opts...)

	// Create parallel deployment steps
	parallelSteps := make([]v1alpha1.WorkflowStep, 0, len(environments))
	for _, env := range environments {
		deployTask := template.NewContainer("deploy-"+env, deployImage,
			template.WithCommand("deploy.sh"),
			template.WithEnv("ENVIRONMENT", env),
			template.WithEnv("APP_NAME", name))

		// Add health check after deployment
		healthCheck := template.NewHTTP("health-"+env,
			template.WithHTTPURL(fmt.Sprintf("https://%s.myapp.com/health", env)),
			template.WithHTTPMethod("GET"),
			template.WithHTTPSuccessCond("response.statusCode == 200"))

		// Get steps and templates for deploy
		deploySteps, err := deployTask.Steps()
		if err != nil {
			return nil, fmt.Errorf("failed to generate deploy steps for %s: %w", env, err)
		}

		deployTemplates, err := deployTask.Templates()
		if err != nil {
			return nil, fmt.Errorf("failed to generate deploy templates for %s: %w", env, err)
		}

		// Get steps and templates for health check
		healthSteps, err := healthCheck.Steps()
		if err != nil {
			return nil, fmt.Errorf("failed to generate health check steps for %s: %w", env, err)
		}

		healthTemplates, err := healthCheck.Templates()
		if err != nil {
			return nil, fmt.Errorf("failed to generate health check templates for %s: %w", env, err)
		}

		// Create a template that combines deploy + health check for this environment
		envTemplate := v1alpha1.Template{
			Name: "deploy-and-check-" + env,
			Steps: []v1alpha1.ParallelSteps{
				{Steps: deploySteps},
				{Steps: healthSteps},
			},
		}

		wb = wb.AddTemplate(envTemplate)
		for _, tmpl := range deployTemplates {
			wb = wb.AddTemplate(tmpl)
		}
		for _, tmpl := range healthTemplates {
			wb = wb.AddTemplate(tmpl)
		}

		// Add to parallel steps
		parallelSteps = append(parallelSteps, v1alpha1.WorkflowStep{
			Name:     "env-" + env,
			Template: "deploy-and-check-" + env,
		})
	}

	// Create entry template
	entryTemplate := v1alpha1.Template{
		Name: name + "-main",
		Steps: []v1alpha1.ParallelSteps{
			{Steps: parallelSteps},
		},
	}

	wb = wb.AddTemplate(entryTemplate)

	return wb.BuildWithEntrypoint(name + "-main")
}
