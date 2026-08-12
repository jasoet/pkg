package patterns

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/argo/builder"
)

// containerArgs returns the single Args entry of the container template with the given
// name. Patterns generate one `sh -c "<script>"` arg per leaf step, so tests can assert
// on the exact shell script the workflow will run.
func containerArgs(t *testing.T, wf *v1alpha1.Workflow, templateName string) string {
	t.Helper()
	for i := range wf.Spec.Templates {
		tmpl := wf.Spec.Templates[i]
		if tmpl.Name == templateName {
			require.NotNil(t, tmpl.Container, "template %s should be a container template", templateName)
			require.Len(t, tmpl.Container.Args, 1, "template %s should have exactly one arg", templateName)
			return tmpl.Container.Args[0]
		}
	}
	t.Fatalf("template %s not found", templateName)
	return ""
}

func TestFanOutFanIn(t *testing.T) {
	tasks := []string{"task-1", "task-2", "task-3"}

	wf, err := FanOutFanIn(
		"fan-out-in", "argo",
		"busybox:latest",
		tasks,
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "fan-out-in-", wf.GenerateName)
	assert.Equal(t, "argo", wf.Namespace)
	assert.Equal(t, "fan-out-in-main", wf.Spec.Entrypoint)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "fan-out-in-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	require.NotEmpty(t, mainTemplate.Steps)

	// First step group should have all parallel tasks
	firstGroup := mainTemplate.Steps[0]
	assert.Len(t, firstGroup.Steps, len(tasks), "first step group should have all parallel tasks")

	// Should have aggregate step after parallel tasks
	assert.GreaterOrEqual(t, len(mainTemplate.Steps), 2, "should have parallel steps + aggregate step")
}

func TestFanOutFanInRequiresTasks(t *testing.T) {
	_, err := FanOutFanIn(
		"empty", "argo",
		"busybox:latest",
		[]string{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one task is required")
}

func TestParallelDataProcessing(t *testing.T) {
	dataItems := []string{"data-1.csv", "data-2.csv", "data-3.csv"}

	wf, err := ParallelDataProcessing(
		"batch-process", "argo",
		"processor:v1",
		dataItems,
		"process.sh",
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "batch-process-", wf.GenerateName)
	assert.Equal(t, "batch-process-main", wf.Spec.Entrypoint)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "batch-process-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	require.NotEmpty(t, mainTemplate.Steps)

	// Should have one parallel group with all data items
	firstGroup := mainTemplate.Steps[0]
	assert.Len(t, firstGroup.Steps, len(dataItems))
}

func TestParallelDataProcessingRequiresData(t *testing.T) {
	_, err := ParallelDataProcessing(
		"empty", "argo",
		"processor:v1",
		[]string{},
		"process.sh",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one data item is required")
}

func TestMapReduce(t *testing.T) {
	inputs := []string{"file1.txt", "file2.txt", "file3.txt"}
	mapCmd := "wc -w"
	reduceCmd := "awk '{sum+=$1} END {print sum}'"

	wf, err := MapReduce(
		"word-count", "argo",
		"alpine:latest",
		inputs,
		mapCmd,
		reduceCmd,
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "word-count-", wf.GenerateName)
	assert.Equal(t, "word-count-main", wf.Spec.Entrypoint)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "word-count-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	require.Len(t, mainTemplate.Steps, 2, "should have map phase and reduce phase")

	// First group (map phase) should have all parallel tasks
	mapPhase := mainTemplate.Steps[0]
	assert.Len(t, mapPhase.Steps, len(inputs), "map phase should have one step per input")

	// Second group (reduce phase) should have single reduce step
	reducePhase := mainTemplate.Steps[1]
	assert.Len(t, reducePhase.Steps, 1, "reduce phase should have single step")
}

// TestMapReduce_ProducesRunnableShellCommand verifies the documented word-count example
// generates a runnable `sh -c` script: the user command "wc -w" must appear as a real
// program+arg fragment (wc -w file), NOT single-quoted into a bogus executable 'wc -w'.
func TestMapReduce_ProducesRunnableShellCommand(t *testing.T) {
	wf, err := MapReduce(
		"word-count", "argo",
		"alpine:latest",
		[]string{"file1.txt"},
		"wc -w",
		"awk '{sum+=$1} END {print sum}'",
	)
	require.NoError(t, err)
	require.NotNil(t, wf)

	mapArgs := containerArgs(t, wf, "map-0-template")
	// Command must be unquoted so the shell runs `wc -w` against the file.
	assert.Contains(t, mapArgs, "wc -w 'file1.txt'")
	assert.NotContains(t, mapArgs, "'wc -w'", "the map command must not be single-quoted into one executable name")

	reduceArgs := containerArgs(t, wf, "reduce-template")
	// The awk fragment (which itself contains quotes) must be passed through verbatim.
	assert.Contains(t, reduceArgs, "awk '{sum+=$1} END {print sum}'")
	assert.NotContains(t, reduceArgs, "'awk ", "the reduce command must not be wrapped in an extra quote")
}

// TestParallelDataProcessing_ProducesRunnableShellCommand verifies the processing command
// is passed as a shell fragment (unquoted) while the data filename is quoted.
func TestParallelDataProcessing_ProducesRunnableShellCommand(t *testing.T) {
	wf, err := ParallelDataProcessing(
		"batch", "argo",
		"processor:v1",
		[]string{"data-1.csv"},
		"process.sh --verbose",
	)
	require.NoError(t, err)

	args := containerArgs(t, wf, "process-0-template")
	assert.Contains(t, args, "process.sh --verbose 'data-1.csv'")
	assert.NotContains(t, args, "'process.sh --verbose'", "the processing command must not be single-quoted")
}

// TestParallelTestSuite_ProducesRunnableShellCommand verifies each test command is a
// runnable shell fragment rather than a single-quoted bogus executable name.
func TestParallelTestSuite_ProducesRunnableShellCommand(t *testing.T) {
	wf, err := ParallelTestSuite(
		"suite", "argo",
		"golang:1.25",
		map[string]string{"unit": "go test ./internal/..."},
	)
	require.NoError(t, err)

	args := containerArgs(t, wf, "test-unit-template")
	assert.Contains(t, args, "go test ./internal/...")
	assert.NotContains(t, args, "'go test ./internal/...'", "the test command must not be single-quoted")
}

// TestParallelTestSuite_DeterministicOrder verifies the generated step order is stable
// across builds regardless of Go map iteration randomization.
func TestParallelTestSuite_DeterministicOrder(t *testing.T) {
	suites := map[string]string{
		"unit":        "go test ./internal/...",
		"integration": "go test ./tests/integration/...",
		"e2e":         "go test ./tests/e2e/...",
	}

	var firstOrder []string
	for run := 0; run < 5; run++ {
		wf, err := ParallelTestSuite("suite", "argo", "golang:1.25", suites)
		require.NoError(t, err)

		var mainTemplate *v1alpha1.Template
		for i := range wf.Spec.Templates {
			if wf.Spec.Templates[i].Name == "suite-main" {
				mainTemplate = &wf.Spec.Templates[i]
				break
			}
		}
		require.NotNil(t, mainTemplate)

		order := make([]string, 0, len(suites))
		for _, step := range mainTemplate.Steps[0].Steps {
			order = append(order, step.Name)
		}
		// Expect suites sorted by name: e2e, integration, unit.
		assert.Equal(t, []string{"test-e2e", "test-integration", "test-unit"}, order)

		if run == 0 {
			firstOrder = order
		} else {
			assert.Equal(t, firstOrder, order, "step order must be identical across builds")
		}
	}
}

func TestMapReduceRequiresInputs(t *testing.T) {
	_, err := MapReduce(
		"empty", "argo",
		"alpine:latest",
		[]string{},
		"map",
		"reduce",
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one input is required")
}

func TestParallelTestSuite(t *testing.T) {
	testSuites := map[string]string{
		"unit":        "go test ./internal/...",
		"integration": "go test ./tests/integration/...",
		"e2e":         "go test ./tests/e2e/...",
	}

	wf, err := ParallelTestSuite(
		"test-suite", "argo",
		"golang:1.25",
		testSuites,
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "test-suite-", wf.GenerateName)
	assert.Equal(t, "test-suite-main", wf.Spec.Entrypoint)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "test-suite-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	require.NotEmpty(t, mainTemplate.Steps)

	// Should have one parallel group with all test suites
	firstGroup := mainTemplate.Steps[0]
	assert.Len(t, firstGroup.Steps, len(testSuites))

	// Verify test templates exist
	templateNames := make(map[string]bool)
	for _, tmpl := range wf.Spec.Templates {
		templateNames[tmpl.Name] = true
	}

	assert.True(t, templateNames["test-unit-template"])
	assert.True(t, templateNames["test-integration-template"])
	assert.True(t, templateNames["test-e2e-template"])
}

func TestParallelTestSuiteRequiresSuites(t *testing.T) {
	_, err := ParallelTestSuite(
		"empty", "argo",
		"golang:1.25",
		map[string]string{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one test suite is required")
}

func TestParallelDeployment(t *testing.T) {
	environments := []string{"us-west", "us-east", "eu-central"}

	wf, err := ParallelDeployment(
		"multi-region", "argo",
		"deployer:v1",
		environments,
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "multi-region-", wf.GenerateName)
	assert.Equal(t, "multi-region-main", wf.Spec.Entrypoint)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "multi-region-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	require.NotEmpty(t, mainTemplate.Steps)

	// Should have parallel deployment for each environment
	firstGroup := mainTemplate.Steps[0]
	assert.Len(t, firstGroup.Steps, len(environments))

	// Each environment should have deploy + health check templates
	for _, env := range environments {
		combinedTemplateName := "deploy-and-check-" + env
		found := false
		for _, tmpl := range wf.Spec.Templates {
			if tmpl.Name == combinedTemplateName {
				found = true
				// This template should have 2 steps: deploy and health check
				assert.Len(t, tmpl.Steps, 2, "deploy-and-check template should have 2 steps")
				break
			}
		}
		assert.True(t, found, "should have combined template for %s", env)
	}
}

func TestParallelDeploymentRequiresEnvironments(t *testing.T) {
	_, err := ParallelDeployment(
		"empty", "argo",
		"deployer:v1",
		[]string{},
	)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "at least one environment is required")
}

func TestFanOutFanInWithOptions(t *testing.T) {
	wf, err := FanOutFanIn(
		"fan-out", "argo",
		"busybox:latest",
		[]string{"task-1", "task-2"},
		builder.WithServiceAccount("workflow-sa"),
		builder.WithLabels(map[string]string{"type": "parallel"}),
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "workflow-sa", wf.Spec.ServiceAccountName)
	assert.Equal(t, "parallel", wf.Labels["type"])
}

func TestMapReduceLargeDataset(t *testing.T) {
	// Test with larger dataset
	inputs := make([]string, 10)
	for i := range inputs {
		inputs[i] = "data-" + string(rune(i+'0')) + ".txt"
	}

	wf, err := MapReduce(
		"big-mapreduce", "argo",
		"alpine:latest",
		inputs,
		"wc -l",
		"awk '{sum+=$1} END {print sum}'",
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	// Find main template
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "big-mapreduce-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	require.Len(t, mainTemplate.Steps, 2)

	// Map phase should have 10 parallel tasks
	mapPhase := mainTemplate.Steps[0]
	assert.Len(t, mapPhase.Steps, 10)
}

func TestParallelTestSuiteWithSingleTest(t *testing.T) {
	// Test with just one test suite
	wf, err := ParallelTestSuite(
		"single-test", "argo",
		"golang:1.25",
		map[string]string{
			"unit": "go test ./...",
		},
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	// Should still work with single test
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "single-test-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	firstGroup := mainTemplate.Steps[0]
	assert.Len(t, firstGroup.Steps, 1)
}

func TestParallelDeploymentIntegration(t *testing.T) {
	// Full integration test
	wf, err := ParallelDeployment(
		"prod-deploy", "argo",
		"deployer:v1",
		[]string{"us-west-2", "eu-west-1"},
		builder.WithServiceAccount("deployer-sa"),
		builder.WithLabels(map[string]string{
			"env":  "production",
			"type": "deployment",
		}),
		builder.WithArchiveLogs(true),
	)

	require.NoError(t, err)
	require.NotNil(t, wf)

	assert.Equal(t, "prod-deploy-", wf.GenerateName)
	assert.Equal(t, "deployer-sa", wf.Spec.ServiceAccountName)
	assert.Equal(t, "production", wf.Labels["env"])
	assert.Equal(t, "deployment", wf.Labels["type"])
	require.NotNil(t, wf.Spec.ArchiveLogs)
	assert.True(t, *wf.Spec.ArchiveLogs)

	// Verify structure
	var mainTemplate *v1alpha1.Template
	for _, tmpl := range wf.Spec.Templates {
		if tmpl.Name == "prod-deploy-main" {
			mainTemplate = &tmpl
			break
		}
	}

	require.NotNil(t, mainTemplate)
	firstGroup := mainTemplate.Steps[0]
	assert.Len(t, firstGroup.Steps, 2, "should have 2 parallel deployments")
}
