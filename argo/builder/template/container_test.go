//go:build !integration && !argo

package template

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func TestNewContainer(t *testing.T) {
	t.Run("creates basic container template", func(t *testing.T) {
		tmpl := NewContainer("test-container", "alpine:latest")
		require.NotNil(t, tmpl)

		steps, err := tmpl.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "test-container", steps[0].Name)

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.NotNil(t, templates[0].Container)
		assert.Equal(t, "alpine:latest", templates[0].Container.Image)
	})

	t.Run("creates container with command", func(t *testing.T) {
		tmpl := NewContainer("test-cmd", "alpine:latest",
			WithCommand("echo", "hello"))

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, []string{"echo", "hello"}, templates[0].Container.Command)
	})

	t.Run("creates container with args", func(t *testing.T) {
		tmpl := NewContainer("test-args", "alpine:latest",
			WithArgs("arg1", "arg2"))

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, []string{"arg1", "arg2"}, templates[0].Container.Args)
	})

	t.Run("creates container with environment variables", func(t *testing.T) {
		tmpl := NewContainer("test-env", "alpine:latest",
			WithEnv("KEY1", "value1"),
			WithEnv("KEY2", "value2"))

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		require.Len(t, templates[0].Container.Env, 2)
		assert.Equal(t, "KEY1", templates[0].Container.Env[0].Name)
		assert.Equal(t, "value1", templates[0].Container.Env[0].Value)
		assert.Equal(t, "KEY2", templates[0].Container.Env[1].Name)
		assert.Equal(t, "value2", templates[0].Container.Env[1].Value)
	})

	t.Run("creates container with OTel config", func(t *testing.T) {
		cfg := otel.NewConfig("test-service")
		tmpl := NewContainer("test-otel", "alpine:latest",
			WithOTelConfig(cfg))

		require.NotNil(t, tmpl)
		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
	})

	t.Run("creates container with working directory", func(t *testing.T) {
		tmpl := NewContainer("test-workdir", "alpine:latest",
			WithWorkingDir("/app"))

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "/app", templates[0].Container.WorkingDir)
	})

	t.Run("creates container with image pull policy", func(t *testing.T) {
		tmpl := NewContainer("test-pull", "alpine:latest",
			WithImagePullPolicy(corev1.PullAlways))

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, corev1.PullAlways, templates[0].Container.ImagePullPolicy)
	})

	t.Run("creates container with CPU resources", func(t *testing.T) {
		tmpl := NewContainer("test-cpu", "alpine:latest",
			WithCPU("500m"))

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		expectedCPU := resource.MustParse("500m")
		assert.Equal(t, expectedCPU, templates[0].Container.Resources.Requests[corev1.ResourceCPU])
	})

	t.Run("creates container with memory resources", func(t *testing.T) {
		tmpl := NewContainer("test-memory", "alpine:latest",
			WithMemory("256Mi"))

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		expectedMemory := resource.MustParse("256Mi")
		assert.Equal(t, expectedMemory, templates[0].Container.Resources.Requests[corev1.ResourceMemory])
	})

	t.Run("creates container with when condition", func(t *testing.T) {
		tmpl := NewContainer("test-when", "alpine:latest",
			WithWhen("{{workflow.status}} == 'Succeeded'"))

		steps, err := tmpl.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "{{workflow.status}} == 'Succeeded'", steps[0].When)
	})
}

func TestContainerCommand(t *testing.T) {
	t.Run("sets command on container", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.Command("sh", "-c")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, []string{"sh", "-c"}, templates[0].Container.Command)
	})

	t.Run("appends to existing command", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.Command("sh").Command("-c")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, []string{"sh", "-c"}, templates[0].Container.Command)
	})
}

func TestContainerArgs(t *testing.T) {
	t.Run("sets args on container", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.Args("echo", "hello")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, []string{"echo", "hello"}, templates[0].Container.Args)
	})
}

func TestContainerEnv(t *testing.T) {
	t.Run("adds environment variable", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.Env("KEY", "value")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		require.Len(t, templates[0].Container.Env, 1)
		assert.Equal(t, "KEY", templates[0].Container.Env[0].Name)
		assert.Equal(t, "value", templates[0].Container.Env[0].Value)
	})

	t.Run("adds multiple environment variables", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest").
			Env("KEY1", "value1").
			Env("KEY2", "value2")

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		require.Len(t, templates[0].Container.Env, 2)
	})
}

func TestContainerEnvFrom(t *testing.T) {
	t.Run("adds envFrom with secret reference", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.EnvFrom("API_KEY", corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "api-secrets",
				},
				Key: "api-key",
			},
		})

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		require.Len(t, templates[0].Container.Env, 1)
		assert.Equal(t, "API_KEY", templates[0].Container.Env[0].Name)
		assert.NotNil(t, templates[0].Container.Env[0].ValueFrom)
		assert.NotNil(t, templates[0].Container.Env[0].ValueFrom.SecretKeyRef)
		assert.Equal(t, "api-secrets", templates[0].Container.Env[0].ValueFrom.SecretKeyRef.Name)
	})
}

func TestContainerVolumeMount(t *testing.T) {
	t.Run("adds volume mount", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.VolumeMount("data-vol", "/data", false)

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		require.Len(t, templates[0].Container.VolumeMounts, 1)
		assert.Equal(t, "data-vol", templates[0].Container.VolumeMounts[0].Name)
		assert.Equal(t, "/data", templates[0].Container.VolumeMounts[0].MountPath)
		assert.False(t, templates[0].Container.VolumeMounts[0].ReadOnly)
	})

	t.Run("adds readonly volume mount", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.VolumeMount("config-vol", "/config", true)

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		require.Len(t, templates[0].Container.VolumeMounts, 1)
		assert.True(t, templates[0].Container.VolumeMounts[0].ReadOnly)
	})
}

func TestContainerWorkingDir(t *testing.T) {
	t.Run("sets working directory", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.WorkingDir("/workspace")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, "/workspace", templates[0].Container.WorkingDir)
	})
}

func TestContainerImagePullPolicy(t *testing.T) {
	t.Run("sets image pull policy", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.ImagePullPolicy(corev1.PullNever)

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, corev1.PullNever, templates[0].Container.ImagePullPolicy)
	})
}

func TestContainerCPU(t *testing.T) {
	t.Run("sets CPU resources with single value", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.CPU("250m")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		expectedCPU := resource.MustParse("250m")
		assert.Equal(t, expectedCPU, templates[0].Container.Resources.Requests[corev1.ResourceCPU])
		assert.Equal(t, expectedCPU, templates[0].Container.Resources.Limits[corev1.ResourceCPU])
	})

	t.Run("sets CPU resources with request and limit", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.CPU("250m", "500m")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		expectedRequest := resource.MustParse("250m")
		expectedLimit := resource.MustParse("500m")
		assert.Equal(t, expectedRequest, templates[0].Container.Resources.Requests[corev1.ResourceCPU])
		assert.Equal(t, expectedLimit, templates[0].Container.Resources.Limits[corev1.ResourceCPU])
	})
}

func TestContainerMemory(t *testing.T) {
	t.Run("sets memory resources", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.Memory("512Mi")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		expectedMemory := resource.MustParse("512Mi")
		assert.Equal(t, expectedMemory, templates[0].Container.Resources.Requests[corev1.ResourceMemory])
	})

	t.Run("sets memory resources with request and limit", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.Memory("256Mi", "512Mi")

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		expectedRequest := resource.MustParse("256Mi")
		expectedLimit := resource.MustParse("512Mi")
		assert.Equal(t, expectedRequest, templates[0].Container.Resources.Requests[corev1.ResourceMemory])
		assert.Equal(t, expectedLimit, templates[0].Container.Resources.Limits[corev1.ResourceMemory])
	})
}

func TestContainerWhen(t *testing.T) {
	t.Run("sets when condition", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.When("{{workflow.status}} == 'Running'")

		steps, err := result.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "{{workflow.status}} == 'Running'", steps[0].When)
	})
}

func TestContainerContinueOn(t *testing.T) {
	t.Run("sets continue on failed", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		result := tmpl.ContinueOn(&v1alpha1.ContinueOn{
			Failed: true,
		})

		steps, err := result.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.True(t, steps[0].ContinueOn.Failed)
	})
}

func TestContainerWithRetry(t *testing.T) {
	t.Run("sets retry strategy", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")
		retryStrategy := &v1alpha1.RetryStrategy{
			RetryPolicy: v1alpha1.RetryPolicyAlways,
		}
		result := tmpl.WithRetry(retryStrategy)

		templates, err := result.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, v1alpha1.RetryPolicyAlways, templates[0].RetryStrategy.RetryPolicy)
	})
}

func TestContainerChaining(t *testing.T) {
	t.Run("chains multiple methods", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest").
			Command("sh", "-c").
			Args("echo 'hello'").
			Env("ENV", "prod").
			WorkingDir("/app").
			CPU("100m").
			Memory("128Mi").
			When("{{workflow.status}} == 'Succeeded'")

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)

		assert.Equal(t, []string{"sh", "-c"}, templates[0].Container.Command)
		assert.Equal(t, []string{"echo 'hello'"}, templates[0].Container.Args)
		assert.Len(t, templates[0].Container.Env, 1)
		assert.Equal(t, "/app", templates[0].Container.WorkingDir)

		steps, err := tmpl.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "{{workflow.status}} == 'Succeeded'", steps[0].When)
	})
}

func TestContainerSteps(t *testing.T) {
	t.Run("generates steps correctly", func(t *testing.T) {
		tmpl := NewContainer("my-step", "alpine:latest")

		steps, err := tmpl.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)

		assert.Equal(t, "my-step", steps[0].Name)
		assert.Equal(t, "my-step-template", steps[0].Template)
	})

	t.Run("includes when condition in steps", func(t *testing.T) {
		tmpl := NewContainer("conditional-step", "alpine:latest").
			When("{{inputs.parameters.run}} == 'true'")

		steps, err := tmpl.Steps()
		require.NoError(t, err)
		require.Len(t, steps, 1)
		assert.Equal(t, "{{inputs.parameters.run}} == 'true'", steps[0].When)
	})
}

func TestContainerTemplates(t *testing.T) {
	t.Run("generates template correctly", func(t *testing.T) {
		tmpl := NewContainer("my-container", "nginx:latest")

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)

		assert.Equal(t, "my-container-template", templates[0].Name)
		assert.NotNil(t, templates[0].Container)
		assert.Equal(t, "nginx:latest", templates[0].Container.Image)
	})

	t.Run("sets default image pull policy", func(t *testing.T) {
		tmpl := NewContainer("test", "alpine:latest")

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, corev1.PullIfNotPresent, templates[0].Container.ImagePullPolicy)
	})

	t.Run("includes retry strategy in template", func(t *testing.T) {
		retry := &v1alpha1.RetryStrategy{
			RetryPolicy: v1alpha1.RetryPolicyOnFailure,
		}
		tmpl := NewContainer("retry-test", "alpine:latest").WithRetry(retry)

		templates, err := tmpl.Templates()
		require.NoError(t, err)
		require.Len(t, templates, 1)
		assert.Equal(t, v1alpha1.RetryPolicyOnFailure, templates[0].RetryStrategy.RetryPolicy)
	})
}
