package template

import (
	"testing"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHTTP(t *testing.T) {
	http := NewHTTP("health-check")

	assert.Equal(t, "health-check", http.name)
	assert.Equal(t, "health-check-template", http.templateName)
	assert.Equal(t, "GET", http.method)
	assert.Equal(t, int32(30), http.timeoutSec)
	assert.Empty(t, http.headers)
}

func TestHTTPWithURL(t *testing.T) {
	url := "https://api.example.com/health"
	http := NewHTTP("api-check",
		WithHTTPURL(url))

	steps, err := http.Steps()
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, "api-check", steps[0].Name)

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, "api-check-template", tmpl.Name)
	assert.NotNil(t, tmpl.HTTP)
	assert.Equal(t, url, tmpl.HTTP.URL)
	assert.Equal(t, "GET", tmpl.HTTP.Method)
}

func TestHTTPWithMethod(t *testing.T) {
	http := NewHTTP("webhook",
		WithHTTPURL("https://hooks.example.com"),
		WithHTTPMethod("POST"))

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, "POST", tmpl.HTTP.Method)
}

func TestHTTPWithHeaders(t *testing.T) {
	http := NewHTTP("api-call",
		WithHTTPURL("https://api.example.com"),
		WithHTTPHeader("Content-Type", "application/json"),
		WithHTTPHeader("Authorization", "Bearer token"))

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	require.Len(t, tmpl.HTTP.Headers, 2)

	assert.Equal(t, "Content-Type", tmpl.HTTP.Headers[0].Name)
	assert.Equal(t, "application/json", tmpl.HTTP.Headers[0].Value)
	assert.Equal(t, "Authorization", tmpl.HTTP.Headers[1].Name)
	assert.Equal(t, "Bearer token", tmpl.HTTP.Headers[1].Value)
}

func TestHTTPWithBody(t *testing.T) {
	body := `{"message": "Hello, World!"}`
	http := NewHTTP("post-data",
		WithHTTPURL("https://api.example.com/data"),
		WithHTTPMethod("POST"),
		WithHTTPBody(body))

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, body, tmpl.HTTP.Body)
}

func TestHTTPWithSuccessCondition(t *testing.T) {
	successCond := "response.statusCode == 200"
	http := NewHTTP("check",
		WithHTTPURL("https://api.example.com"),
		WithHTTPSuccessCond(successCond))

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, successCond, tmpl.HTTP.SuccessCondition)
}

func TestHTTPDefaultSuccessCondition(t *testing.T) {
	http := NewHTTP("check",
		WithHTTPURL("https://api.example.com"))

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	// Should have default 2xx success condition
	assert.Equal(t, "response.statusCode >= 200 && response.statusCode < 300", tmpl.HTTP.SuccessCondition)
}

func TestHTTPWithTimeout(t *testing.T) {
	timeout := int32(60)
	http := NewHTTP("slow-api",
		WithHTTPURL("https://slow.example.com"),
		WithHTTPTimeout(timeout))

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	require.NotNil(t, tmpl.HTTP.TimeoutSeconds)
	assert.Equal(t, int64(60), *tmpl.HTTP.TimeoutSeconds)
}

func TestHTTPFluent(t *testing.T) {
	http := NewHTTP("fluent").
		URL("https://api.example.com/resource").
		Method("PUT").
		Header("Content-Type", "application/json").
		Header("X-Custom-Header", "value").
		Body(`{"data": "test"}`).
		SuccessCondition("response.statusCode == 200").
		Timeout(45).
		When("{{workflow.status}} == Running")

	assert.Equal(t, "https://api.example.com/resource", http.url)
	assert.Equal(t, "PUT", http.method)
	assert.Len(t, http.headers, 2)
	assert.Equal(t, `{"data": "test"}`, http.body)
	assert.Equal(t, "response.statusCode == 200", http.successCond)
	assert.Equal(t, int32(45), http.timeoutSec)
	assert.Equal(t, "{{workflow.status}} == Running", http.when)
}

func TestHTTPStepsWithCondition(t *testing.T) {
	condition := "{{steps.test.outputs.exitCode}} == 0"
	http := NewHTTP("conditional-webhook",
		WithHTTPURL("https://hooks.example.com")).
		When(condition)

	steps, err := http.Steps()
	require.NoError(t, err)
	require.Len(t, steps, 1)
	assert.Equal(t, condition, steps[0].When)
}

func TestHTTPRequiresURL(t *testing.T) {
	// HTTP without URL should fail
	http := NewHTTP("no-url")

	_, err := http.Steps()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP URL is required")
}

func TestHTTPComplexRequest(t *testing.T) {
	http := NewHTTP("complex",
		WithHTTPURL("https://api.example.com/v1/webhook"),
		WithHTTPMethod("POST"),
		WithHTTPHeader("Content-Type", "application/json"),
		WithHTTPHeader("Authorization", "Bearer {{workflow.parameters.token}}"),
		WithHTTPHeader("X-Request-ID", "{{workflow.uid}}"),
		WithHTTPBody(`{
			"workflow": "{{workflow.name}}",
			"status": "{{workflow.status}}",
			"timestamp": "{{workflow.creationTimestamp}}"
		}`),
		WithHTTPSuccessCond("response.statusCode >= 200 && response.statusCode < 300"),
		WithHTTPTimeout(30))

	templates, err := http.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, "POST", tmpl.HTTP.Method)
	assert.Equal(t, "https://api.example.com/v1/webhook", tmpl.HTTP.URL)
	assert.Len(t, tmpl.HTTP.Headers, 3)
	assert.Contains(t, tmpl.HTTP.Body, "workflow")
	assert.Contains(t, tmpl.HTTP.Body, "status")
	assert.Equal(t, "response.statusCode >= 200 && response.statusCode < 300", tmpl.HTTP.SuccessCondition)
	assert.Equal(t, int64(30), *tmpl.HTTP.TimeoutSeconds)
}

func TestHTTPHealthCheckPattern(t *testing.T) {
	// Common pattern: health check after deployment
	healthCheck := NewHTTP("health-check",
		WithHTTPURL("https://myapp.com/health"),
		WithHTTPMethod("GET"),
		WithHTTPHeader("Accept", "application/json"),
		WithHTTPSuccessCond("response.statusCode == 200 && response.body.status == 'healthy'"),
		WithHTTPTimeout(10))

	templates, err := healthCheck.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, "GET", tmpl.HTTP.Method)
	assert.Equal(t, "https://myapp.com/health", tmpl.HTTP.URL)
	assert.Contains(t, tmpl.HTTP.SuccessCondition, "response.statusCode == 200")
	assert.Contains(t, tmpl.HTTP.SuccessCondition, "response.body.status == 'healthy'")
}

func TestHTTPWebhookPattern(t *testing.T) {
	// Common pattern: webhook notification
	webhook := NewHTTP("slack-notify",
		WithHTTPURL("https://hooks.slack.com/services/xxx"),
		WithHTTPMethod("POST"),
		WithHTTPHeader("Content-Type", "application/json"),
		WithHTTPBody(`{
			"text": "Workflow {{workflow.name}} completed with status {{workflow.status}}"
		}`))

	templates, err := webhook.Templates()
	require.NoError(t, err)
	require.Len(t, templates, 1)

	tmpl := templates[0]
	assert.Equal(t, "POST", tmpl.HTTP.Method)
	assert.Contains(t, tmpl.HTTP.URL, "hooks.slack.com")
	assert.Contains(t, tmpl.HTTP.Body, "Workflow")
}

func TestHTTPWithContinueOn(t *testing.T) {
	http := NewHTTP("optional-check",
		WithHTTPURL("https://api.example.com"))

	http.continueOn = &v1alpha1.ContinueOn{
		Failed: true,
	}

	steps, err := http.Steps()
	require.NoError(t, err)
	require.Len(t, steps, 1)

	step := steps[0]
	require.NotNil(t, step.ContinueOn)
	assert.True(t, step.ContinueOn.Failed)
}
