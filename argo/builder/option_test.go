package builder

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestWithOTelConfig(t *testing.T) {
	cfg := otel.NewConfig("test-service")
	wb := NewWorkflowBuilder("test", "argo", WithOTelConfig(cfg))

	assert.NotNil(t, wb.otelConfig)
	assert.Equal(t, "test-service", wb.otelConfig.ServiceName)
	assert.NotNil(t, wb.otel, "should initialize OTel instrumentation")
}

func TestWithServiceAccount(t *testing.T) {
	sa := "custom-service-account"
	wb := NewWorkflowBuilder("test", "argo", WithServiceAccount(sa))

	wf, err := wb.Build()
	require.NoError(t, err)
	assert.Equal(t, sa, wf.Spec.ServiceAccountName)
}

func TestWithLabels(t *testing.T) {
	labels := map[string]string{
		"app":  "myapp",
		"env":  "production",
		"team": "platform",
	}

	wb := NewWorkflowBuilder("test", "argo", WithLabels(labels))

	wf, err := wb.Build()
	require.NoError(t, err)
	assert.Equal(t, labels, wf.Labels)
}

func TestWithAnnotations(t *testing.T) {
	annotations := map[string]string{
		"description": "Test workflow",
		"owner":       "platform-team",
	}

	wb := NewWorkflowBuilder("test", "argo", WithAnnotations(annotations))

	wf, err := wb.Build()
	require.NoError(t, err)
	assert.Equal(t, annotations, wf.Annotations)
}

func TestWithRetryStrategy(t *testing.T) {
	limit := intstr.FromInt(3)
	retryStrategy := &v1alpha1.RetryStrategy{
		Limit:       &limit,
		RetryPolicy: "Always",
	}

	wb := NewWorkflowBuilder("test", "argo", WithRetryStrategy(retryStrategy))

	// Retry strategy should be applied to templates
	assert.NotNil(t, wb.retryStrategy)
	assert.Equal(t, retryStrategy, wb.retryStrategy)
}

func TestWithVolume(t *testing.T) {
	volume := corev1.Volume{
		Name: "data",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	wb := NewWorkflowBuilder("test", "argo", WithVolume(volume))

	wf, err := wb.Build()
	require.NoError(t, err)
	require.Len(t, wf.Spec.Volumes, 1)
	assert.Equal(t, "data", wf.Spec.Volumes[0].Name)
}

func TestWithArchiveLogs(t *testing.T) {
	t.Run("enable archive logs", func(t *testing.T) {
		wb := NewWorkflowBuilder("test", "argo", WithArchiveLogs(true))

		wf, err := wb.Build()
		require.NoError(t, err)
		require.NotNil(t, wf.Spec.ArchiveLogs)
		assert.True(t, *wf.Spec.ArchiveLogs)
	})

	t.Run("disable archive logs", func(t *testing.T) {
		wb := NewWorkflowBuilder("test", "argo", WithArchiveLogs(false))

		wf, err := wb.Build()
		require.NoError(t, err)
		require.NotNil(t, wf.Spec.ArchiveLogs)
		assert.False(t, *wf.Spec.ArchiveLogs)
	})
}

func TestWithPodGC(t *testing.T) {
	podGC := &v1alpha1.PodGC{
		Strategy: v1alpha1.PodGCOnPodSuccess,
	}

	wb := NewWorkflowBuilder("test", "argo", WithPodGC(podGC))

	wf, err := wb.Build()
	require.NoError(t, err)
	require.NotNil(t, wf.Spec.PodGC)
	assert.Equal(t, v1alpha1.PodGCOnPodSuccess, wf.Spec.PodGC.Strategy)
}

func TestWithTTL(t *testing.T) {
	secondsAfterCompletion := int32(3600)
	ttl := &v1alpha1.TTLStrategy{
		SecondsAfterCompletion: &secondsAfterCompletion,
	}

	wb := NewWorkflowBuilder("test", "argo", WithTTL(ttl))

	wf, err := wb.Build()
	require.NoError(t, err)
	require.NotNil(t, wf.Spec.TTLStrategy)
	require.NotNil(t, wf.Spec.TTLStrategy.SecondsAfterCompletion)
	assert.Equal(t, int32(3600), *wf.Spec.TTLStrategy.SecondsAfterCompletion)
}

func TestWithActiveDeadlineSeconds(t *testing.T) {
	deadline := int64(7200)

	wb := NewWorkflowBuilder("test", "argo", WithActiveDeadlineSeconds(deadline))

	wf, err := wb.Build()
	require.NoError(t, err)
	require.NotNil(t, wf.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, deadline, *wf.Spec.ActiveDeadlineSeconds)
}

func TestWithMetrics(t *testing.T) {
	// Mock metrics provider
	provider := &mockMetricsProvider{
		metrics: &v1alpha1.Metrics{
			Prometheus: []*v1alpha1.Prometheus{
				{
					Name: "workflow_duration",
					Help: "Duration of workflow execution",
				},
			},
		},
	}

	wb := NewWorkflowBuilder("test", "argo").WithMetrics(provider)

	wf, err := wb.Build()
	require.NoError(t, err)
	require.NotNil(t, wf.Spec.Metrics)
	require.Len(t, wf.Spec.Metrics.Prometheus, 1)
	assert.Equal(t, "workflow_duration", wf.Spec.Metrics.Prometheus[0].Name)
}

// mockMetricsProvider implements WorkflowMetricsProvider for testing
type mockMetricsProvider struct {
	metrics *v1alpha1.Metrics
	err     error
}

func (m *mockMetricsProvider) Metrics() (*v1alpha1.Metrics, error) {
	return m.metrics, m.err
}

func TestMultipleOptions(t *testing.T) {
	labels := map[string]string{"app": "test"}
	annotations := map[string]string{"owner": "team"}
	volume := corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	wb := NewWorkflowBuilder("test", "argo",
		WithServiceAccount("custom-sa"),
		WithLabels(labels),
		WithAnnotations(annotations),
		WithVolume(volume),
		WithArchiveLogs(true),
	)

	wf, err := wb.Build()
	require.NoError(t, err)

	// Verify all options applied
	assert.Equal(t, "custom-sa", wf.Spec.ServiceAccountName)
	assert.Equal(t, labels, wf.Labels)
	assert.Equal(t, annotations, wf.Annotations)
	require.Len(t, wf.Spec.Volumes, 1)
	assert.Equal(t, "config", wf.Spec.Volumes[0].Name)
	require.NotNil(t, wf.Spec.ArchiveLogs)
	assert.True(t, *wf.Spec.ArchiveLogs)
}

func TestWithOTelConfig_InitializesInstrumentation(t *testing.T) {
	cfg := otel.NewConfig("test")
	wb := NewWorkflowBuilder("test", "argo", WithOTelConfig(cfg))

	// Verify OTel instrumentation is initialized
	assert.NotNil(t, wb.otel)
	assert.NotNil(t, wb.otelConfig)

	// Build should work with OTel enabled
	wf, err := wb.Build()
	require.NoError(t, err)
	assert.NotNil(t, wf)
}

func TestOptionsAppliedToWorkflow(t *testing.T) {
	// Create comprehensive workflow with all options
	limit := intstr.FromInt(2)
	deadline := int64(3600)
	secondsAfterCompletion := int32(7200)

	wb := NewWorkflowBuilder("comprehensive", "argo",
		WithServiceAccount("argo-sa"),
		WithLabels(map[string]string{
			"app": "comprehensive-test",
			"env": "test",
		}),
		WithAnnotations(map[string]string{
			"description": "Comprehensive options test",
		}),
		WithRetryStrategy(&v1alpha1.RetryStrategy{
			Limit:       &limit,
			RetryPolicy: "OnFailure",
		}),
		WithVolume(corev1.Volume{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		}),
		WithArchiveLogs(true),
		WithPodGC(&v1alpha1.PodGC{
			Strategy: v1alpha1.PodGCOnWorkflowSuccess,
		}),
		WithTTL(&v1alpha1.TTLStrategy{
			SecondsAfterCompletion: &secondsAfterCompletion,
		}),
		WithActiveDeadlineSeconds(deadline),
	)

	wf, err := wb.Build()
	require.NoError(t, err)

	// Verify all settings
	assert.Equal(t, "argo-sa", wf.Spec.ServiceAccountName)
	assert.Equal(t, "comprehensive-test", wf.Labels["app"])
	assert.Equal(t, "test", wf.Labels["env"])
	assert.Equal(t, "Comprehensive options test", wf.Annotations["description"])
	assert.Len(t, wf.Spec.Volumes, 1)
	assert.NotNil(t, wf.Spec.ArchiveLogs)
	assert.True(t, *wf.Spec.ArchiveLogs)
	assert.NotNil(t, wf.Spec.PodGC)
	assert.Equal(t, v1alpha1.PodGCOnWorkflowSuccess, wf.Spec.PodGC.Strategy)
	assert.NotNil(t, wf.Spec.TTLStrategy)
	assert.Equal(t, int32(7200), *wf.Spec.TTLStrategy.SecondsAfterCompletion)
	assert.NotNil(t, wf.Spec.ActiveDeadlineSeconds)
	assert.Equal(t, int64(3600), *wf.Spec.ActiveDeadlineSeconds)
}

func TestDefaultServiceAccount(t *testing.T) {
	// Without specifying service account, should use default
	wb := NewWorkflowBuilder("test", "argo")

	wf, err := wb.Build()
	require.NoError(t, err)
	assert.Equal(t, "argo-workflow", wf.Spec.ServiceAccountName, "should use default service account")
}

func TestEmptyLabelsAndAnnotations(t *testing.T) {
	wb := NewWorkflowBuilder("test", "argo")

	wf, err := wb.Build()
	require.NoError(t, err)
	assert.Empty(t, wf.Labels)
	assert.Empty(t, wf.Annotations)
}

func TestMultipleVolumes(t *testing.T) {
	volume1 := corev1.Volume{
		Name: "data",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	volume2 := corev1.Volume{
		Name: "config",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	wb := NewWorkflowBuilder("test", "argo",
		WithVolume(volume1),
		WithVolume(volume2),
	)

	wf, err := wb.Build()
	require.NoError(t, err)
	require.Len(t, wf.Spec.Volumes, 2)
	assert.Equal(t, "data", wf.Spec.Volumes[0].Name)
	assert.Equal(t, "config", wf.Spec.Volumes[1].Name)
}
