package patterns

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTestDeploy(t *testing.T) {
	wf, err := BuildTestDeploy(
		"myapp", "argo",
		"golang:1.25",
		"golang:1.25",
		"deployer:v1",
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "myapp-", wf.GenerateName)
	assert.Equal(t, "argo", wf.Namespace)
	assert.Equal(t, "main", wf.Spec.Entrypoint)

	// Should have main template + build + test + deploy + health-check templates
	assert.GreaterOrEqual(t, len(wf.Spec.Templates), 5)

	// Find and verify templates exist
	templateNames := make(map[string]bool)
	for _, tmpl := range wf.Spec.Templates {
		templateNames[tmpl.Name] = true
	}

	assert.True(t, templateNames["build-template"])
	assert.True(t, templateNames["test-template"])
	assert.True(t, templateNames["deploy-template"])
	assert.True(t, templateNames["health-check-template"])
}

func TestBuildTestDeployWithOptions(t *testing.T) {
	wf, err := BuildTestDeploy(
		"myapp", "argo",
		"golang:1.25",
		"golang:1.25",
		"deployer:v1",
		builder.WithServiceAccount("custom-sa"),
		builder.WithLabels(map[string]string{"app": "test"}),
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "custom-sa", wf.Spec.ServiceAccountName)
	assert.Equal(t, "test", wf.Labels["app"])
}

func TestBuildTestDeployWithCleanup(t *testing.T) {
	wf, err := BuildTestDeployWithCleanup(
		"myapp", "argo",
		"golang:1.25",
		"busybox:latest",
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "myapp-", wf.GenerateName)
	assert.Equal(t, "argo", wf.Namespace)

	// Should have exit handler configured
	assert.NotEmpty(t, wf.Spec.OnExit)
	assert.Equal(t, "exit-handler", wf.Spec.OnExit)

	// Find exit handler template
	var exitHandlerTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "exit-handler" {
			exitHandlerTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, exitHandlerTemplate, "exit-handler template should exist")
	require.NotEmpty(t, exitHandlerTemplate.Steps)

	// Should have cleanup and notify steps in exit handler
	hasCleanup := false
	hasNotify := false
	for _, parallelSteps := range exitHandlerTemplate.Steps {
		for _, step := range parallelSteps.Steps {
			if step.Name == "cleanup" {
				hasCleanup = true
			}
			if step.Name == "notify" {
				hasNotify = true
			}
		}
	}

	assert.True(t, hasCleanup, "exit handler should have cleanup step")
	assert.True(t, hasNotify, "exit handler should have notify step")
}

func TestConditionalDeploy(t *testing.T) {
	wf, err := ConditionalDeploy(
		"conditional-deploy", "argo",
		"golang:1.25",
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "conditional-deploy-", wf.GenerateName)
	assert.Equal(t, "argo", wf.Namespace)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	require.NotEmpty(t, mainTemplate.Steps)

	// Check for conditional deploy and rollback steps
	hasConditionalDeploy := false
	hasRollback := false

	for _, parallelSteps := range mainTemplate.Steps {
		for _, step := range parallelSteps.Steps {
			if step.Name == "deploy" && step.When != "" {
				hasConditionalDeploy = true
			}
			if step.Name == "rollback" && step.When != "" {
				hasRollback = true
			}
		}
	}

	assert.True(t, hasConditionalDeploy, "should have conditional deploy step")
	assert.True(t, hasRollback, "should have conditional rollback step")
}

func TestMultiEnvironmentDeploy(t *testing.T) {
	environments := []string{"staging", "production"}

	wf, err := MultiEnvironmentDeploy(
		"multi-env", "argo",
		"deployer:v1",
		environments,
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "multi-env-", wf.GenerateName)
	assert.Equal(t, "argo", wf.Namespace)

	// Should have templates for each environment
	templateNames := make(map[string]bool)
	for _, tmpl := range wf.Spec.Templates {
		templateNames[tmpl.Name] = true
	}

	// Check for deploy and health check templates for each environment
	for _, env := range environments {
		deployTemplate := "deploy-" + env + "-template"
		healthCheckTemplate := "health-check-" + env + "-template"

		assert.True(t, templateNames[deployTemplate],
			"should have deploy template for %s", env)
		assert.True(t, templateNames[healthCheckTemplate],
			"should have health check template for %s", env)
	}
}

func TestMultiEnvironmentDeploySequential(t *testing.T) {
	environments := []string{"dev", "staging", "production"}

	wf, err := MultiEnvironmentDeploy(
		"sequential", "argo",
		"deployer:v1",
		environments,
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)

	// Steps should be sequential (one per parallel group for each environment)
	// Each environment has deploy + health check, so we should have at least
	// len(environments) * 2 steps
	totalSteps := 0
	for _, parallelSteps := range mainTemplate.Steps {
		totalSteps += len(parallelSteps.Steps)
	}

	assert.GreaterOrEqual(t, totalSteps, len(environments)*2,
		"should have at least 2 steps per environment (deploy + health check)")
}

func TestMultiEnvironmentDeployWithOptions(t *testing.T) {
	wf, err := MultiEnvironmentDeploy(
		"multi-env", "argo",
		"deployer:v1",
		[]string{"staging", "production"},
		builder.WithServiceAccount("deployer-sa"),
		builder.WithLabels(map[string]string{
			"type": "deployment",
			"env":  "multi",
		}),
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "deployer-sa", wf.Spec.ServiceAccountName)
	assert.Equal(t, "deployment", wf.Labels["type"])
	assert.Equal(t, "multi", wf.Labels["env"])
}

func TestBuildTestDeployIntegration(t *testing.T) {
	// Test that all components work together
	wf, err := BuildTestDeploy(
		"integration-test", "argo",
		"golang:1.25",
		"golang:1.25",
		"deployer:v1",
		builder.WithServiceAccount("argo-workflow"),
		builder.WithLabels(map[string]string{
			"app": "myapp",
			"ci":  "true",
		}),
		builder.WithArchiveLogs(true),
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	// Verify all settings
	assert.Equal(t, "integration-test-", wf.GenerateName)
	assert.Equal(t, "argo", wf.Namespace)
	assert.Equal(t, "argo-workflow", wf.Spec.ServiceAccountName)
	assert.Equal(t, "myapp", wf.Labels["app"])
	assert.Equal(t, "true", wf.Labels["ci"])
	require.NotNil(t, wf.Spec.ArchiveLogs)
	assert.True(t, *wf.Spec.ArchiveLogs)

	// Verify workflow structure
	assert.Equal(t, "main", wf.Spec.Entrypoint)
	assert.NotEmpty(t, wf.Spec.Templates)

	// Verify main template has steps
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	assert.NotEmpty(t, mainTemplate.Steps, "main template should have steps")
}
