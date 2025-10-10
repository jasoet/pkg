package patterns

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
)

// BuildTestDeploy creates a simple CI/CD workflow pattern with build, test, and deploy stages.
// This is a common pattern for continuous integration and deployment pipelines.
//
// Example:
//
//	wf := patterns.BuildTestDeploy(
//	    "myapp", "argo",
//	    "golang:1.25",
//	    "myregistry/myapp:latest",
//	    "myregistry/deployer:v1",
//	)
func BuildTestDeploy(name, namespace, buildImage, testImage, deployImage string, opts ...builder.Option) (*v1alpha1.Workflow, error) {
	// Build stage
	build := template.NewContainer("build", buildImage,
		template.WithCommand("sh", "-c"),
		template.WithArgs("echo 'Building application...' && go build -o app"),
		template.WithWorkingDir("/workspace"))

	// Test stage
	test := template.NewContainer("test", testImage,
		template.WithCommand("sh", "-c"),
		template.WithArgs("echo 'Running tests...' && go test ./..."),
		template.WithWorkingDir("/workspace"))

	// Deploy stage
	deploy := template.NewContainer("deploy", deployImage,
		template.WithCommand("sh", "-c"),
		template.WithArgs("echo 'Deploying application...'"))

	// Health check after deploy
	healthCheck := template.NewHTTP("health-check",
		template.WithHTTPURL("https://myapp/health"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPSuccessCond("response.statusCode == 200"))

	// Build workflow
	return builder.NewWorkflowBuilder(name, namespace, opts...).
		Add(build).
		Add(test).
		Add(deploy).
		Add(healthCheck).
		Build()
}

// BuildTestDeployWithCleanup creates a CI/CD workflow with cleanup on exit.
// The cleanup always runs regardless of workflow success or failure.
//
// Example:
//
//	wf := patterns.BuildTestDeployWithCleanup(
//	    "myapp", "argo",
//	    "golang:1.25",
//	    "busybox:latest",
//	)
func BuildTestDeployWithCleanup(name, namespace, buildImage, cleanupImage string, opts ...builder.Option) (*v1alpha1.Workflow, error) {
	// Build stage
	build := template.NewContainer("build", buildImage,
		template.WithCommand("go", "build", "-o", "app"),
		template.WithWorkingDir("/workspace"))

	// Test stage
	test := template.NewContainer("test", buildImage,
		template.WithCommand("go", "test", "./..."),
		template.WithWorkingDir("/workspace"))

	// Deploy stage
	deploy := template.NewContainer("deploy", buildImage,
		template.WithCommand("sh", "-c", "echo 'Deploying...'"))

	// Cleanup (runs on exit)
	cleanup := template.NewContainer("cleanup", cleanupImage,
		template.WithCommand("sh", "-c", "echo 'Cleaning up temporary resources...'"))

	// Notification (runs on exit)
	notify := template.NewScript("notify", "bash",
		template.WithScriptContent(`
echo "Workflow completed"
echo "Status: {{workflow.status}}"
echo "Duration: {{workflow.duration}}"
`))

	// Build workflow with exit handlers
	return builder.NewWorkflowBuilder(name, namespace, opts...).
		Add(build).
		Add(test).
		Add(deploy).
		AddExitHandler(cleanup).
		AddExitHandler(notify).
		Build()
}

// ConditionalDeploy creates a workflow that deploys only if tests pass.
// This demonstrates conditional execution using the 'when' clause.
//
// Example:
//
//	wf := patterns.ConditionalDeploy(
//	    "conditional-deploy", "argo",
//	    "golang:1.25",
//	)
func ConditionalDeploy(name, namespace, image string, opts ...builder.Option) (*v1alpha1.Workflow, error) {
	// Test stage
	test := template.NewContainer("test", image,
		template.WithCommand("go", "test", "./..."))

	// Deploy only if tests pass
	deploy := template.NewContainer("deploy", image,
		template.WithCommand("sh", "-c", "echo 'Deploying to production...'")).
		When("{{steps.test.outputs.exitCode}} == 0")

	// Rollback if deploy fails
	rollback := template.NewContainer("rollback", image,
		template.WithCommand("sh", "-c", "echo 'Rolling back deployment...'")).
		When("{{steps.deploy.outputs.exitCode}} != 0")

	return builder.NewWorkflowBuilder(name, namespace, opts...).
		Add(test).
		Add(deploy).
		Add(rollback).
		Build()
}

// MultiEnvironmentDeploy creates a workflow that deploys to multiple environments sequentially.
//
// Example:
//
//	wf := patterns.MultiEnvironmentDeploy(
//	    "multi-env-deploy", "argo",
//	    "myregistry/deployer:v1",
//	    []string{"staging", "production"},
//	)
func MultiEnvironmentDeploy(name, namespace, deployImage string, environments []string, opts ...builder.Option) (*v1alpha1.Workflow, error) {
	wb := builder.NewWorkflowBuilder(name, namespace, opts...)

	// Add deployment step for each environment
	for _, env := range environments {
		deployStep := template.NewContainer("deploy-"+env, deployImage,
			template.WithCommand("deploy.sh"),
			template.WithEnv("ENVIRONMENT", env),
			template.WithEnv("APP_NAME", name))

		// Health check for this environment
		healthCheck := template.NewHTTP("health-check-"+env,
			template.WithHTTPURL("https://"+env+".myapp.com/health"),
			template.WithHTTPMethod("GET"))

		wb.Add(deployStep).Add(healthCheck)
	}

	return wb.Build()
}
