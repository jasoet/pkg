package template

import (
	"context"
	"fmt"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/otel"
)

// HTTP is a WorkflowSource that creates an HTTP request workflow step.
// It's useful for making API calls, health checks, or webhooks.
//
// Example:
//
//	http := template.NewHTTP("health-check").
//	    URL("https://myapp/health").
//	    Method("GET").
//	    SuccessCond("response.statusCode >= 200 && response.statusCode < 300")
type HTTP struct {
	name         string
	templateName string
	url          string
	method       string
	headers      []v1alpha1.HTTPHeader
	body         string
	successCond  string
	timeoutSec   int32
	when         string
	continueOn   *v1alpha1.ContinueOn
	otelConfig   *otel.Config
}

// NewHTTP creates a new HTTP workflow source.
//
// Parameters:
//   - name: Step name
//   - opts: Optional configuration functions
//
// Example:
//
//	http := template.NewHTTP("api-call",
//	    template.WithHTTPURL("https://api.example.com/v1/resource"),
//	    template.WithHTTPMethod("POST"),
//	    template.WithHTTPBody(`{"key": "value"}`))
func NewHTTP(name string, opts ...HTTPOption) *HTTP {
	h := &HTTP{
		name:         name,
		templateName: name + "-template",
		method:       "GET",
		headers:      make([]v1alpha1.HTTPHeader, 0),
		timeoutSec:   30,
	}

	for _, opt := range opts {
		opt(h)
	}

	return h
}

// URL sets the HTTP URL.
//
// Example:
//
//	http.URL("https://api.example.com/health")
func (h *HTTP) URL(url string) *HTTP {
	h.url = url
	return h
}

// Method sets the HTTP method.
//
// Example:
//
//	http.Method("POST")
func (h *HTTP) Method(method string) *HTTP {
	h.method = method
	return h
}

// Header adds an HTTP header.
//
// Example:
//
//	http.Header("Content-Type", "application/json").
//	    Header("Authorization", "Bearer {{workflow.parameters.token}}")
func (h *HTTP) Header(name, value string) *HTTP {
	h.headers = append(h.headers, v1alpha1.HTTPHeader{
		Name:  name,
		Value: value,
	})
	return h
}

// Body sets the HTTP request body.
//
// Example:
//
//	http.Body(`{"status": "complete"}`)
func (h *HTTP) Body(body string) *HTTP {
	h.body = body
	return h
}

// SuccessCondition sets the condition for considering the request successful.
//
// Example:
//
//	http.SuccessCondition("response.statusCode == 200")
func (h *HTTP) SuccessCondition(cond string) *HTTP {
	h.successCond = cond
	return h
}

// Timeout sets the request timeout in seconds.
//
// Example:
//
//	http.Timeout(60) // 60 seconds
func (h *HTTP) Timeout(seconds int32) *HTTP {
	h.timeoutSec = seconds
	return h
}

// When sets a conditional expression.
//
// Example:
//
//	http.When("{{workflow.status}} == Succeeded")
func (h *HTTP) When(condition string) *HTTP {
	h.when = condition
	return h
}

// Steps implements WorkflowSource interface.
func (h *HTTP) Steps() ([]v1alpha1.WorkflowStep, error) {
	ctx := context.Background()

	logger := otel.NewLogHelper(ctx, h.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder/template", "HTTP.Steps")
	logger.Debug("Generating HTTP steps",
		otel.F("name", h.name),
		otel.F("url", h.url),
		otel.F("method", h.method))

	if h.url == "" {
		err := fmt.Errorf("HTTP URL is required for step %s", h.name)
		logger.Error(err, "HTTP URL not set")
		return nil, err
	}

	step := v1alpha1.WorkflowStep{
		Name:     h.name,
		Template: h.templateName,
	}

	if h.when != "" {
		step.When = h.when
	}

	if h.continueOn != nil {
		step.ContinueOn = h.continueOn
	}

	return []v1alpha1.WorkflowStep{step}, nil
}

// Templates implements WorkflowSource interface.
func (h *HTTP) Templates() ([]v1alpha1.Template, error) {
	ctx := context.Background()

	logger := otel.NewLogHelper(ctx, h.otelConfig,
		"github.com/jasoet/pkg/v2/argo/builder/template", "HTTP.Templates")
	logger.Debug("Generating HTTP template",
		otel.F("name", h.templateName),
		otel.F("url", h.url))

	timeout := int64(h.timeoutSec)
	httpTemplate := &v1alpha1.HTTP{
		Method:         h.method,
		URL:            h.url,
		Headers:        h.headers,
		Body:           h.body,
		TimeoutSeconds: &timeout,
	}

	// Set success condition if provided
	if h.successCond != "" {
		httpTemplate.SuccessCondition = h.successCond
	} else {
		// Default success condition: 2xx status codes
		httpTemplate.SuccessCondition = "response.statusCode >= 200 && response.statusCode < 300"
	}

	template := v1alpha1.Template{
		Name: h.templateName,
		HTTP: httpTemplate,
	}

	return []v1alpha1.Template{template}, nil
}

// HTTPOption is a functional option for configuring HTTP.
type HTTPOption func(*HTTP)

// WithHTTPURL sets the URL.
func WithHTTPURL(url string) HTTPOption {
	return func(h *HTTP) {
		h.url = url
	}
}

// WithHTTPMethod sets the HTTP method.
func WithHTTPMethod(method string) HTTPOption {
	return func(h *HTTP) {
		h.method = method
	}
}

// WithHTTPHeader adds a header.
func WithHTTPHeader(name, value string) HTTPOption {
	return func(h *HTTP) {
		h.headers = append(h.headers, v1alpha1.HTTPHeader{
			Name:  name,
			Value: value,
		})
	}
}

// WithHTTPBody sets the request body.
func WithHTTPBody(body string) HTTPOption {
	return func(h *HTTP) {
		h.body = body
	}
}

// WithHTTPSuccessCond sets the success condition.
func WithHTTPSuccessCond(cond string) HTTPOption {
	return func(h *HTTP) {
		h.successCond = cond
	}
}

// WithHTTPTimeout sets the timeout.
func WithHTTPTimeout(seconds int32) HTTPOption {
	return func(h *HTTP) {
		h.timeoutSec = seconds
	}
}

// WithHTTPOTelConfig enables OpenTelemetry instrumentation.
func WithHTTPOTelConfig(cfg *otel.Config) HTTPOption {
	return func(h *HTTP) {
		h.otelConfig = cfg
	}
}
