package template

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Container is a WorkflowSource that creates a container-based workflow step.
// It provides a fluent API for configuring containers with common options like
// commands, arguments, environment variables, and resource requests.
//
// Example:
//
//	deploy := template.NewContainer("deploy", "myapp:v1").
//	    Command("deploy.sh").
//	    Args("--env", "production").
//	    Env("DATABASE_URL", "postgres://...").
//	    CPU("1000m").
//	    Memory("512Mi")
type Container struct {
	name           string
	templateName   string
	image          string
	command        []string
	args           []string
	env            []corev1.EnvVar
	volumeMounts   []corev1.VolumeMount
	workingDir     string
	imagePullPolicy corev1.PullPolicy
	cpuRequest     string
	cpuLimit       string
	memoryRequest  string
	memoryLimit    string
	when           string
	continueOn     *v1alpha1.ContinueOn
	retryStrategy  *v1alpha1.RetryStrategy
	otelConfig     *otel.Config
}

// NewContainer creates a new container workflow source.
//
// Parameters:
//   - name: Step name (used in workflow step definition)
//   - image: Container image (e.g., "alpine:latest", "myapp:v1")
//   - opts: Optional configuration functions
//
// Example:
//
//	container := template.NewContainer("deploy", "myapp:v1",
//	    template.WithCommand("deploy.sh"),
//	    template.WithArgs("--env", "production"),
//	    template.WithOTelConfig(otelConfig))
func NewContainer(name, image string, opts ...ContainerOption) *Container {
	c := &Container{
		name:          name,
		templateName:  name + "-template",
		image:         image,
		env:           make([]corev1.EnvVar, 0),
		volumeMounts:  make([]corev1.VolumeMount, 0),
		imagePullPolicy: corev1.PullIfNotPresent,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Command sets the container command (entrypoint override).
// Can be called multiple times or with multiple arguments.
//
// Example:
//
//	container.Command("python", "app.py")
//	// or
//	container.Command("python").Command("app.py")
func (c *Container) Command(cmd ...string) *Container {
	c.command = append(c.command, cmd...)
	return c
}

// Args sets the container arguments.
// Can be called multiple times or with multiple arguments.
//
// Example:
//
//	container.Args("--port", "8080", "--host", "0.0.0.0")
func (c *Container) Args(args ...string) *Container {
	c.args = append(c.args, args...)
	return c
}

// Env adds an environment variable to the container.
//
// Example:
//
//	container.Env("DATABASE_URL", "postgres://...").
//	    Env("LOG_LEVEL", "debug")
func (c *Container) Env(name, value string) *Container {
	c.env = append(c.env, corev1.EnvVar{
		Name:  name,
		Value: value,
	})
	return c
}

// EnvFrom adds an environment variable from a source (ConfigMap or Secret).
//
// Example:
//
//	container.EnvFrom("API_KEY", corev1.EnvVarSource{
//	    SecretKeyRef: &corev1.SecretKeySelector{
//	        LocalObjectReference: corev1.LocalObjectReference{Name: "api-secrets"},
//	        Key: "api-key",
//	    },
//	})
func (c *Container) EnvFrom(name string, source corev1.EnvVarSource) *Container {
	c.env = append(c.env, corev1.EnvVar{
		Name:      name,
		ValueFrom: &source,
	})
	return c
}

// VolumeMount adds a volume mount to the container.
//
// Example:
//
//	container.VolumeMount("config", "/etc/config", true)
func (c *Container) VolumeMount(name, mountPath string, readOnly bool) *Container {
	c.volumeMounts = append(c.volumeMounts, corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	})
	return c
}

// WorkingDir sets the working directory for the container.
//
// Example:
//
//	container.WorkingDir("/app")
func (c *Container) WorkingDir(dir string) *Container {
	c.workingDir = dir
	return c
}

// ImagePullPolicy sets the image pull policy.
//
// Example:
//
//	container.ImagePullPolicy(corev1.PullAlways)
func (c *Container) ImagePullPolicy(policy corev1.PullPolicy) *Container {
	c.imagePullPolicy = policy
	return c
}

// CPU sets CPU request and limit.
//
// Example:
//
//	container.CPU("500m")  // request and limit
//	container.CPU("500m", "1000m")  // request and limit separately
func (c *Container) CPU(request string, limit ...string) *Container {
	c.cpuRequest = request
	if len(limit) > 0 {
		c.cpuLimit = limit[0]
	} else {
		c.cpuLimit = request
	}
	return c
}

// Memory sets memory request and limit.
//
// Example:
//
//	container.Memory("256Mi")  // request and limit
//	container.Memory("256Mi", "512Mi")  // request and limit separately
func (c *Container) Memory(request string, limit ...string) *Container {
	c.memoryRequest = request
	if len(limit) > 0 {
		c.memoryLimit = limit[0]
	} else {
		c.memoryLimit = request
	}
	return c
}

// When sets a conditional expression for when this step should run.
//
// Example:
//
//	container.When("{{workflow.status}} == Succeeded")
func (c *Container) When(condition string) *Container {
	c.when = condition
	return c
}

// ContinueOn sets the step to continue on specific conditions.
//
// Example:
//
//	container.ContinueOn(&v1alpha1.ContinueOn{
//	    Failed: true,  // continue even if step fails
//	})
func (c *Container) ContinueOn(continueOn *v1alpha1.ContinueOn) *Container {
	c.continueOn = continueOn
	return c
}

// WithRetry sets a retry strategy for this specific step.
//
// Example:
//
//	container.WithRetry(&v1alpha1.RetryStrategy{
//	    Limit: intstr.FromInt(3),
//	    RetryPolicy: "Always",
//	})
func (c *Container) WithRetry(retry *v1alpha1.RetryStrategy) *Container {
	c.retryStrategy = retry
	return c
}

// Steps implements WorkflowSource interface.
func (c *Container) Steps() ([]v1alpha1.WorkflowStep, error) {
	ctx := context.Background()

	logger := otel.NewLogHelper(ctx, c.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder/template", "Container.Steps")
	logger.Debug("Generating container steps",
		otel.F("name", c.name),
		otel.F("image", c.image))

	step := v1alpha1.WorkflowStep{
		Name:     c.name,
		Template: c.templateName,
	}

	// Add conditional execution
	if c.when != "" {
		step.When = c.when
	}

	// Add continue-on configuration
	if c.continueOn != nil {
		step.ContinueOn = c.continueOn
	}

	return []v1alpha1.WorkflowStep{step}, nil
}

// Templates implements WorkflowSource interface.
func (c *Container) Templates() ([]v1alpha1.Template, error) {
	ctx := context.Background()

	logger := otel.NewLogHelper(ctx, c.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder/template", "Container.Templates")
	logger.Debug("Generating container template",
		otel.F("name", c.templateName),
		otel.F("image", c.image))

	container := &corev1.Container{
		Name:            c.name,
		Image:           c.image,
		Command:         c.command,
		Args:            c.args,
		Env:             c.env,
		VolumeMounts:    c.volumeMounts,
		WorkingDir:      c.workingDir,
		ImagePullPolicy: c.imagePullPolicy,
	}

	// Set resource requirements if specified
	if c.cpuRequest != "" || c.memoryRequest != "" {
		container.Resources = corev1.ResourceRequirements{
			Requests: corev1.ResourceList{},
			Limits:   corev1.ResourceList{},
		}

		if c.cpuRequest != "" {
			container.Resources.Requests[corev1.ResourceCPU] = resource.MustParse(c.cpuRequest)
		}
		if c.cpuLimit != "" {
			container.Resources.Limits[corev1.ResourceCPU] = resource.MustParse(c.cpuLimit)
		}
		if c.memoryRequest != "" {
			container.Resources.Requests[corev1.ResourceMemory] = resource.MustParse(c.memoryRequest)
		}
		if c.memoryLimit != "" {
			container.Resources.Limits[corev1.ResourceMemory] = resource.MustParse(c.memoryLimit)
		}
	}

	template := v1alpha1.Template{
		Name:      c.templateName,
		Container: container,
	}

	// Add retry strategy if specified
	if c.retryStrategy != nil {
		template.RetryStrategy = c.retryStrategy
	}

	return []v1alpha1.Template{template}, nil
}

// ContainerOption is a functional option for configuring Container.
type ContainerOption func(*Container)

// WithCommand sets the container command.
func WithCommand(cmd ...string) ContainerOption {
	return func(c *Container) {
		c.command = cmd
	}
}

// WithArgs sets the container arguments.
func WithArgs(args ...string) ContainerOption {
	return func(c *Container) {
		c.args = args
	}
}

// WithEnv adds an environment variable.
func WithEnv(name, value string) ContainerOption {
	return func(c *Container) {
		c.env = append(c.env, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}
}

// WithOTelConfig enables OpenTelemetry instrumentation.
func WithOTelConfig(cfg *otel.Config) ContainerOption {
	return func(c *Container) {
		c.otelConfig = cfg
	}
}

// WithWorkingDir sets the working directory.
func WithWorkingDir(dir string) ContainerOption {
	return func(c *Container) {
		c.workingDir = dir
	}
}

// WithImagePullPolicy sets the image pull policy.
func WithImagePullPolicy(policy corev1.PullPolicy) ContainerOption {
	return func(c *Container) {
		c.imagePullPolicy = policy
	}
}

// WithCPU sets CPU request and limit.
func WithCPU(request string, limit ...string) ContainerOption {
	return func(c *Container) {
		c.cpuRequest = request
		if len(limit) > 0 {
			c.cpuLimit = limit[0]
		} else {
			c.cpuLimit = request
		}
	}
}

// WithMemory sets memory request and limit.
func WithMemory(request string, limit ...string) ContainerOption {
	return func(c *Container) {
		c.memoryRequest = request
		if len(limit) > 0 {
			c.memoryLimit = limit[0]
		} else {
			c.memoryLimit = request
		}
	}
}

// WithWhen sets a conditional expression.
func WithWhen(condition string) ContainerOption {
	return func(c *Container) {
		c.when = condition
	}
}
