package template

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestNewScript(t *testing.T) {
	tests := []struct {
		name          string
		scriptName    string
		language      string
		opts          []ScriptOption
		expectedImage string
		expectedCmd   []string
	}{
		{
			name:          "bash script with defaults",
			scriptName:    "backup",
			language:      "bash",
			opts:          []ScriptOption{},
			expectedImage: "bash:5.2",
			expectedCmd:   []string{"bash"},
		},
		{
			name:          "python script with defaults",
			scriptName:    "process",
			language:      "python",
			opts:          []ScriptOption{},
			expectedImage: "python:3.11-slim",
			expectedCmd:   []string{"python"},
		},
		{
			name:          "node script with defaults",
			scriptName:    "build",
			language:      "node",
			opts:          []ScriptOption{},
			expectedImage: "node:20-slim",
			expectedCmd:   []string{"node"},
		},
		{
			name:          "ruby script with defaults",
			scriptName:    "deploy",
			language:      "ruby",
			opts:          []ScriptOption{},
			expectedImage: "ruby:3.2-slim",
			expectedCmd:   []string{"ruby"},
		},
		{
			name:          "custom image override",
			scriptName:    "custom",
			language:      "bash",
			opts:          []ScriptOption{WithScriptImage("custom:v1")},
			expectedImage: "custom:v1",
			expectedCmd:   []string{"bash"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := NewScript(tt.scriptName, tt.language, tt.opts...)

			assert.Equal(t, tt.scriptName, script.name)
			assert.Equal(t, tt.expectedImage, script.image)
			assert.Equal(t, tt.expectedCmd, script.command)
		})
	}
}

func TestScriptWithContent(t *testing.T) {
	scriptContent := "echo 'Hello, World!'"
	script := NewScript("test", "bash",
		WithScriptContent(scriptContent))

	steps, err := script.Steps()
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "test", steps[0].Name)

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, "test-template", tmpl.Name)
	assert.NotNil(t, tmpl.Script)
	assert.Equal(t, scriptContent, tmpl.Script.Source)
	assert.Equal(t, "bash:5.2", tmpl.Script.Container.Image)
}

func TestScriptWithEnvironment(t *testing.T) {
	script := NewScript("env-test", "bash",
		WithScriptContent("printenv"),
		WithScriptEnv("FOO", "bar"),
		WithScriptEnv("BAZ", "qux"))

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	require.Len(t, tmpl.Script.Container.Env, 2)
	assert.Equal(t, "FOO", tmpl.Script.Container.Env[0].Name)
	assert.Equal(t, "bar", tmpl.Script.Container.Env[0].Value)
	assert.Equal(t, "BAZ", tmpl.Script.Container.Env[1].Name)
	assert.Equal(t, "qux", tmpl.Script.Container.Env[1].Value)
}

func TestScriptWithWorkingDir(t *testing.T) {
	workingDir := "/workspace"
	script := NewScript("workdir-test", "bash",
		WithScriptContent("pwd"),
		WithScriptWorkingDir(workingDir))

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, workingDir, tmpl.Script.Container.WorkingDir)
}

func TestScriptWithCustomCommand(t *testing.T) {
	script := NewScript("cmd-test", "bash",
		WithScriptContent("echo test"),
		WithScriptCommand("bash", "-x", "-e"))

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, []string{"bash", "-x", "-e"}, tmpl.Script.Container.Command)
}

func TestScriptFluent(t *testing.T) {
	script := NewScript("fluent", "bash").
		Script("echo 'test'").
		Image("custom/bash:latest").
		Command("bash", "-x").
		Env("DEBUG", "true").
		WorkingDir("/app").
		CPU("100m", "200m").
		Memory("64Mi", "128Mi").
		When("{{workflow.status}} == Succeeded")

	assert.Equal(t, "echo 'test'", script.scriptContent)
	assert.Equal(t, "custom/bash:latest", script.image)
	assert.Equal(t, []string{"bash", "-x"}, script.command)
	assert.Equal(t, "/app", script.workingDir)
	assert.Equal(t, "100m", script.cpuRequest)
	assert.Equal(t, "200m", script.cpuLimit)
	assert.Equal(t, "64Mi", script.memoryRequest)
	assert.Equal(t, "128Mi", script.memoryLimit)
	assert.Equal(t, "{{workflow.status}} == Succeeded", script.when)
}

func TestScriptStepsWithCondition(t *testing.T) {
	condition := "{{workflow.status}} == Succeeded"
	script := NewScript("conditional", "bash",
		WithScriptContent("echo 'conditional'")).
		When(condition)

	steps, err := script.Steps()
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, condition, steps[0].When)
}

func TestScriptResourceRequirements(t *testing.T) {
	script := NewScript("resources", "bash",
		WithScriptContent("echo 'test'")).
		CPU("500m", "1000m").
		Memory("256Mi", "512Mi")

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	resources := tmpl.Script.Container.Resources

	cpuReq := resources.Requests.Cpu()
	cpuLim := resources.Limits.Cpu()
	memReq := resources.Requests.Memory()
	memLim := resources.Limits.Memory()

	assert.Equal(t, "500m", cpuReq.String())
	// 1000m is normalized to "1" by Kubernetes
	assert.Equal(t, "1", cpuLim.String())
	assert.Equal(t, "256Mi", memReq.String())
	assert.Equal(t, "512Mi", memLim.String())
}

func TestScriptVolumeMounts(t *testing.T) {
	script := NewScript("volumes", "bash",
		WithScriptContent("ls /data")).
		VolumeMount("data", "/data", false).
		VolumeMount("config", "/config", true)

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	mounts := tmpl.Script.Container.VolumeMounts
	require.Len(t, mounts, 2)

	assert.Equal(t, "data", mounts[0].Name)
	assert.Equal(t, "/data", mounts[0].MountPath)
	assert.False(t, mounts[0].ReadOnly)

	assert.Equal(t, "config", mounts[1].Name)
	assert.Equal(t, "/config", mounts[1].MountPath)
	assert.True(t, mounts[1].ReadOnly)
}

func TestScriptPythonMultiline(t *testing.T) {
	pythonCode := `
import json
import sys

def main():
    data = {"status": "success"}
    print(json.dumps(data))

if __name__ == "__main__":
    main()
`

	script := NewScript("python-script", "python",
		WithScriptContent(pythonCode),
		WithScriptEnv("PYTHONUNBUFFERED", "1"))

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, pythonCode, tmpl.Script.Source)
	assert.Equal(t, "python:3.11-slim", tmpl.Script.Container.Image)
	assert.Equal(t, []string{"python"}, tmpl.Script.Container.Command)
}

func TestScriptWithRetryStrategy(t *testing.T) {
	script := NewScript("retry", "bash",
		WithScriptContent("exit 1"))

	retryLimit := intstr.FromInt(3)
	script.retryStrategy = &v1alpha1.RetryStrategy{
		Limit: &retryLimit,
	}

	templates, err := script.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.NotNil(t, tmpl.RetryStrategy)
	assert.Equal(t, 3, tmpl.RetryStrategy.Limit.IntValue())
}

func TestScriptSource(t *testing.T) {
	t.Run("sets script source from artifact", func(t *testing.T) {
		script := NewScript("artifact-test", "bash").
			Source("{{inputs.artifacts.script}}")

		assert.Equal(t, "{{inputs.artifacts.script}}", script.source)
	})

	t.Run("sets script source from configmap", func(t *testing.T) {
		script := NewScript("configmap-test", "python").
			Source("{{inputs.parameters.script-content}}")

		assert.Equal(t, "{{inputs.parameters.script-content}}", script.source)
	})
}
