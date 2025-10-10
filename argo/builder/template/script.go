package template

import (
	"context"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// Script is a WorkflowSource that creates a script-based workflow step.
// It's useful for running inline scripts in various languages (bash, python, etc.).
//
// Example:
//
//	script := template.NewScript("backup", "bash").
//	    Script("tar -czf /backup/data.tar.gz /data").
//	    Env("BACKUP_DIR", "/backup")
type Script struct {
	name          string
	templateName  string
	image         string
	scriptContent string
	command       []string
	source        string
	env           []corev1.EnvVar
	volumeMounts  []corev1.VolumeMount
	workingDir    string
	cpuRequest    string
	cpuLimit      string
	memoryRequest string
	memoryLimit   string
	when          string
	continueOn    *v1alpha1.ContinueOn
	retryStrategy *v1alpha1.RetryStrategy
	otelConfig    *otel.Config
}

// NewScript creates a new script workflow source.
// The image should contain the interpreter for the script language.
//
// Parameters:
//   - name: Step name
//   - language: Script language ("bash", "python", "sh", etc.) - determines the default image
//   - opts: Optional configuration functions
//
// Example:
//
//	script := template.NewScript("process", "python",
//	    template.WithScriptContent("print('Processing data...')"),
//	    template.WithScriptEnv("API_KEY", "secret"))
func NewScript(name, language string, opts ...ScriptOption) *Script {
	s := &Script{
		name:         name,
		templateName: name + "-template",
		env:          make([]corev1.EnvVar, 0),
		volumeMounts: make([]corev1.VolumeMount, 0),
	}

	// Set default image based on language
	switch language {
	case "bash", "sh":
		s.image = "bash:5.2"
		s.command = []string{"bash"}
	case "python", "python3":
		s.image = "python:3.11-slim"
		s.command = []string{"python"}
	case "node", "nodejs", "javascript":
		s.image = "node:20-slim"
		s.command = []string{"node"}
	case "ruby":
		s.image = "ruby:3.2-slim"
		s.command = []string{"ruby"}
	default:
		// Default to bash
		s.image = "bash:5.2"
		s.command = []string{"bash"}
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Script sets the inline script content.
//
// Example:
//
//	script.Script("echo 'Hello, World!'")
func (s *Script) Script(content string) *Script {
	s.scriptContent = content
	return s
}

// Source sets the script source (from artifact or configmap).
//
// Example:
//
//	script.Source("{{inputs.artifacts.script}}")
func (s *Script) Source(source string) *Script {
	s.source = source
	return s
}

// Image overrides the default image for the script.
//
// Example:
//
//	script.Image("custom/python:3.11")
func (s *Script) Image(image string) *Script {
	s.image = image
	return s
}

// Command overrides the default command.
//
// Example:
//
//	script.Command("python3", "-u")
func (s *Script) Command(cmd ...string) *Script {
	s.command = cmd
	return s
}

// Env adds an environment variable.
//
// Example:
//
//	script.Env("LOG_LEVEL", "debug")
func (s *Script) Env(name, value string) *Script {
	s.env = append(s.env, corev1.EnvVar{
		Name:  name,
		Value: value,
	})
	return s
}

// VolumeMount adds a volume mount.
//
// Example:
//
//	script.VolumeMount("data", "/data", false)
func (s *Script) VolumeMount(name, mountPath string, readOnly bool) *Script {
	s.volumeMounts = append(s.volumeMounts, corev1.VolumeMount{
		Name:      name,
		MountPath: mountPath,
		ReadOnly:  readOnly,
	})
	return s
}

// WorkingDir sets the working directory.
//
// Example:
//
//	script.WorkingDir("/workspace")
func (s *Script) WorkingDir(dir string) *Script {
	s.workingDir = dir
	return s
}

// CPU sets CPU request and limit.
//
// Example:
//
//	script.CPU("500m", "1000m")
func (s *Script) CPU(request string, limit ...string) *Script {
	s.cpuRequest = request
	if len(limit) > 0 {
		s.cpuLimit = limit[0]
	} else {
		s.cpuLimit = request
	}
	return s
}

// Memory sets memory request and limit.
//
// Example:
//
//	script.Memory("256Mi", "512Mi")
func (s *Script) Memory(request string, limit ...string) *Script {
	s.memoryRequest = request
	if len(limit) > 0 {
		s.memoryLimit = limit[0]
	} else {
		s.memoryLimit = request
	}
	return s
}

// When sets a conditional expression.
//
// Example:
//
//	script.When("{{workflow.status}} == Succeeded")
func (s *Script) When(condition string) *Script {
	s.when = condition
	return s
}

// Steps implements WorkflowSource interface.
func (s *Script) Steps() ([]v1alpha1.WorkflowStep, error) {
	ctx := context.Background()

	logger := otel.NewLogHelper(ctx, s.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder/template", "Script.Steps")
	logger.Debug("Generating script steps",
		otel.F("name", s.name),
		otel.F("image", s.image))

	step := v1alpha1.WorkflowStep{
		Name:     s.name,
		Template: s.templateName,
	}

	if s.when != "" {
		step.When = s.when
	}

	if s.continueOn != nil {
		step.ContinueOn = s.continueOn
	}

	return []v1alpha1.WorkflowStep{step}, nil
}

// Templates implements WorkflowSource interface.
func (s *Script) Templates() ([]v1alpha1.Template, error) {
	ctx := context.Background()

	logger := otel.NewLogHelper(ctx, s.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder/template", "Script.Templates")
	logger.Debug("Generating script template",
		otel.F("name", s.templateName),
		otel.F("image", s.image))

	script := &v1alpha1.ScriptTemplate{
		Container: corev1.Container{
			Name:         s.name,
			Image:        s.image,
			Command:      s.command,
			Env:          s.env,
			VolumeMounts: s.volumeMounts,
			WorkingDir:   s.workingDir,
		},
		Source: s.scriptContent,
	}

	// Set resource requirements if specified
	if s.cpuRequest != "" || s.memoryRequest != "" {
		script.Container.Resources = buildResourceRequirements(
			s.cpuRequest, s.cpuLimit, s.memoryRequest, s.memoryLimit)
	}

	template := v1alpha1.Template{
		Name:   s.templateName,
		Script: script,
	}

	if s.retryStrategy != nil {
		template.RetryStrategy = s.retryStrategy
	}

	return []v1alpha1.Template{template}, nil
}

// ScriptOption is a functional option for configuring Script.
type ScriptOption func(*Script)

// WithScriptContent sets the script content.
func WithScriptContent(content string) ScriptOption {
	return func(s *Script) {
		s.scriptContent = content
	}
}

// WithScriptImage sets the container image.
func WithScriptImage(image string) ScriptOption {
	return func(s *Script) {
		s.image = image
	}
}

// WithScriptCommand sets the command.
func WithScriptCommand(cmd ...string) ScriptOption {
	return func(s *Script) {
		s.command = cmd
	}
}

// WithScriptEnv adds an environment variable.
func WithScriptEnv(name, value string) ScriptOption {
	return func(s *Script) {
		s.env = append(s.env, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}
}

// WithScriptOTelConfig enables OpenTelemetry instrumentation.
func WithScriptOTelConfig(cfg *otel.Config) ScriptOption {
	return func(s *Script) {
		s.otelConfig = cfg
	}
}

// WithScriptWorkingDir sets the working directory.
func WithScriptWorkingDir(dir string) ScriptOption {
	return func(s *Script) {
		s.workingDir = dir
	}
}

// buildResourceRequirements is a helper to build resource requirements.
func buildResourceRequirements(cpuReq, cpuLim, memReq, memLim string) corev1.ResourceRequirements {
	reqs := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}

	if cpuReq != "" {
		reqs.Requests[corev1.ResourceCPU] = mustParseQuantity(cpuReq)
	}
	if cpuLim != "" {
		reqs.Limits[corev1.ResourceCPU] = mustParseQuantity(cpuLim)
	}
	if memReq != "" {
		reqs.Requests[corev1.ResourceMemory] = mustParseQuantity(memReq)
	}
	if memLim != "" {
		reqs.Limits[corev1.ResourceMemory] = mustParseQuantity(memLim)
	}

	return reqs
}

// mustParseQuantity is a helper that parses resource quantities.
func mustParseQuantity(s string) resource.Quantity {
	return resource.MustParse(s)
}
