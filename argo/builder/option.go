package builder

import (
	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	corev1 "k8s.io/api/core/v1"
)

// BuilderOption is a functional option for configuring WorkflowBuilder.
type BuilderOption func(*WorkflowBuilder)

// WithOTelConfig enables OpenTelemetry instrumentation for the workflow builder.
// This adds distributed tracing, metrics collection, and structured logging to workflow build operations.
//
// Example:
//
//	otelConfig := otel.NewConfig("workflow-service").
//	    WithTracerProvider(tp).
//	    WithMeterProvider(mp)
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithOTelConfig(otelConfig))
func WithOTelConfig(cfg *otel.Config) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.otelConfig = cfg
	}
}

// WithServiceAccount sets the Kubernetes service account name for the workflow.
// The service account determines what permissions the workflow has in the cluster.
//
// Example:
//
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithServiceAccount("argo-workflow"))
func WithServiceAccount(sa string) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.serviceAccount = sa
	}
}

// WithRetryStrategy sets a default retry strategy for all workflow steps.
// Individual steps can override this with their own retry configuration.
//
// Example:
//
//	retryStrategy := &v1alpha1.RetryStrategy{
//	    Limit: intstr.FromInt(3),
//	    RetryPolicy: "Always",
//	    Backoff: &v1alpha1.Backoff{
//	        Duration: "1m",
//	        Factor:   intstr.FromInt(2),
//	        MaxDuration: "10m",
//	    },
//	}
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithRetryStrategy(retryStrategy))
func WithRetryStrategy(retry *v1alpha1.RetryStrategy) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.retryStrategy = retry
	}
}

// WithVolume adds a volume to the workflow.
// Volumes can be mounted in workflow steps for persistent storage or configuration.
//
// Example:
//
//	volume := corev1.Volume{
//	    Name: "config",
//	    VolumeSource: corev1.VolumeSource{
//	        ConfigMap: &corev1.ConfigMapVolumeSource{
//	            LocalObjectReference: corev1.LocalObjectReference{
//	                Name: "my-config",
//	            },
//	        },
//	    },
//	}
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithVolume(volume))
func WithVolume(volume corev1.Volume) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.volumes = append(b.volumes, volume)
	}
}

// WithArchiveLogs enables or disables log archiving for the workflow.
// When enabled, workflow logs are persisted after the workflow completes.
//
// Example:
//
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithArchiveLogs(true))
func WithArchiveLogs(archive bool) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.archiveLogs = &archive
	}
}

// WithPodGC sets the pod garbage collection strategy for the workflow.
// This determines when workflow pods are cleaned up after completion.
//
// Example:
//
//	podGC := &v1alpha1.PodGC{
//	    Strategy: v1alpha1.PodGCOnWorkflowSuccess,
//	}
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithPodGC(podGC))
func WithPodGC(podGC *v1alpha1.PodGC) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.podGC = podGC
	}
}

// WithTTL sets the time-to-live for the workflow after completion.
// After this duration, the workflow will be automatically deleted.
//
// Example:
//
//	ttl := &v1alpha1.TTLStrategy{
//	    SecondsAfterCompletion: int32Ptr(3600), // 1 hour
//	}
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithTTL(ttl))
func WithTTL(ttl *v1alpha1.TTLStrategy) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.ttl = ttl
	}
}

// WithActiveDeadlineSeconds sets the maximum duration for the workflow.
// If the workflow runs longer than this, it will be terminated.
//
// Example:
//
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithActiveDeadlineSeconds(3600)) // 1 hour max
func WithActiveDeadlineSeconds(seconds int64) BuilderOption {
	return func(b *WorkflowBuilder) {
		b.activeDeadlineSeconds = &seconds
	}
}

// WithLabels adds labels to the workflow metadata.
// Labels can be used for filtering, organizing, and identifying workflows.
//
// Example:
//
//	labels := map[string]string{
//	    "app": "myapp",
//	    "env": "production",
//	}
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithLabels(labels))
func WithLabels(labels map[string]string) BuilderOption {
	return func(b *WorkflowBuilder) {
		if b.labels == nil {
			b.labels = make(map[string]string)
		}
		for k, v := range labels {
			b.labels[k] = v
		}
	}
}

// WithAnnotations adds annotations to the workflow metadata.
// Annotations can store arbitrary non-identifying metadata.
//
// Example:
//
//	annotations := map[string]string{
//	    "description": "Daily backup workflow",
//	    "owner": "platform-team",
//	}
//	builder := NewWorkflowBuilder("my-workflow", "argo",
//	    WithAnnotations(annotations))
func WithAnnotations(annotations map[string]string) BuilderOption {
	return func(b *WorkflowBuilder) {
		if b.annotations == nil {
			b.annotations = make(map[string]string)
		}
		for k, v := range annotations {
			b.annotations[k] = v
		}
	}
}
