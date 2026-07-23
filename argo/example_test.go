package argo_test

import (
	"context"
	"fmt"

	"github.com/jasoet/pkg/v3/argo"
	"github.com/jasoet/pkg/v3/argo/builder"
	"github.com/jasoet/pkg/v3/argo/builder/template"
	"github.com/jasoet/pkg/v3/otel"
)

// ExampleNewClientWithOptions demonstrates creating an Argo Workflows client
// with functional options. The returned context carries the configured OTel
// config, so package operations called with that context are instrumented
// automatically.
//
// This example requires a reachable Kubernetes cluster and does not run
// during `go test` (no Output comment).
func ExampleNewClientWithOptions() {
	ctx := context.Background()

	otelConfig := otel.NewConfig("my-service")

	// The returned ctx carries the OTel config configured via WithOTelConfig.
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig("/path/to/kubeconfig"),
		argo.WithContext("production"),
		argo.WithOTelConfig(otelConfig),
	)
	if err != nil {
		fmt.Println("failed to create client:", err)
		return
	}

	fmt.Println("client created:", client != nil, "ctx:", ctx != nil)
}

// ExampleSubmitWorkflow demonstrates building a workflow with the fluent
// builder API and submitting it. SubmitWorkflow resolves its OpenTelemetry
// config from the context via otel.ConfigFromContext — when ctx came from a
// client created with argo.WithOTelConfig, the call is instrumented without
// any extra argument.
//
// This example requires a reachable Argo Workflows installation and does not
// run during `go test` (no Output comment).
func ExampleSubmitWorkflow() {
	ctx := context.Background()

	ctx, client, err := argo.NewClient(ctx, argo.DefaultConfig())
	if err != nil {
		fmt.Println("failed to create client:", err)
		return
	}

	hello := template.NewContainer("hello", "alpine:latest",
		template.WithCommand("echo", "Hello, Argo!"))

	wf, err := builder.NewWorkflowBuilder("hello-world", "argo").
		Add(hello).
		Build()
	if err != nil {
		fmt.Println("failed to build workflow:", err)
		return
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf)
	if err != nil {
		fmt.Println("failed to submit workflow:", err)
		return
	}

	fmt.Println("submitted workflow:", created.Name)
}
