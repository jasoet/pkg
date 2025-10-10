//go:build example

package main

import (
	"context"
	"fmt"

	"github.com/jasoet/pkg/v2/argo"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
	"github.com/jasoet/pkg/v2/logging"
	"github.com/jasoet/pkg/v2/otel"
	"github.com/rs/zerolog/log"
)

// This example demonstrates using the WorkflowBuilder API to construct
// Argo Workflows with full OpenTelemetry instrumentation.
func main() {
	// Initialize logging
	logging.Initialize("argo-builder-example", false)
	log.Info().Msg("Starting Argo Workflow Builder example")

	ctx := context.Background()

	// Example 1: Simple workflow with sequential steps
	if err := example1SimpleWorkflow(ctx); err != nil {
		log.Fatal().Err(err).Msg("Example 1 failed")
	}

	// Example 2: Workflow with exit handler
	if err := example2WithExitHandler(ctx); err != nil {
		log.Fatal().Err(err).Msg("Example 2 failed")
	}

	// Example 3: Workflow with OpenTelemetry
	if err := example3WithOTel(ctx); err != nil {
		log.Fatal().Err(err).Msg("Example 3 failed")
	}

	log.Info().Msg("All examples completed successfully")
}

// Example 1: Simple workflow with sequential container steps
func example1SimpleWorkflow(ctx context.Context) error {
	log.Info().Msg("=== Example 1: Simple Sequential Workflow ===")

	// Create workflow steps
	build := template.NewContainer("build", "golang:1.25",
		template.WithCommand("go", "build", "-o", "app"),
		template.WithWorkingDir("/workspace"))

	test := template.NewContainer("test", "golang:1.25",
		template.WithCommand("go", "test", "./..."),
		template.WithWorkingDir("/workspace"))

	deploy := template.NewContainer("deploy", "myregistry/deployer:v1",
		template.WithCommand("deploy.sh"),
		template.WithEnv("ENV", "production"))

	// Build workflow
	wf, err := builder.NewWorkflowBuilder("cicd", "argo",
		builder.WithServiceAccount("argo-workflow"),
		builder.WithLabels(map[string]string{
			"app":  "myapp",
			"type": "ci-cd",
		})).
		Add(build).
		Add(test).
		Add(deploy).
		Build()

	if err != nil {
		return fmt.Errorf("failed to build workflow: %w", err)
	}

	log.Info().
		Str("workflow", wf.GenerateName).
		Int("templates", len(wf.Spec.Templates)).
		Msg("Workflow built successfully")

	return nil
}

// Example 2: Workflow with cleanup exit handler
func example2WithExitHandler(ctx context.Context) error {
	log.Info().Msg("=== Example 2: Workflow with Exit Handler ===")

	// Main workflow steps
	process := template.NewContainer("process", "busybox:latest",
		template.WithCommand("sh", "-c", "echo 'Processing data...'"))

	// Exit handler for cleanup (always runs)
	cleanup := template.NewContainer("cleanup", "busybox:latest",
		template.WithCommand("sh", "-c", "echo 'Cleaning up resources...'"))

	// Build workflow with exit handler
	wf, err := builder.NewWorkflowBuilder("data-pipeline", "argo",
		builder.WithServiceAccount("argo-workflow")).
		Add(process).
		AddExitHandler(cleanup).
		Build()

	if err != nil {
		return fmt.Errorf("failed to build workflow: %w", err)
	}

	log.Info().
		Str("workflow", wf.GenerateName).
		Bool("has_exit_handler", wf.Spec.OnExit != "").
		Msg("Workflow with exit handler built successfully")

	return nil
}

// Example 3: Workflow with full OpenTelemetry instrumentation
func example3WithOTel(ctx context.Context) error {
	log.Info().Msg("=== Example 3: Workflow with OpenTelemetry ===")

	// Create OTel config
	otelConfig := otel.NewConfig("workflow-builder-example")
	// In production, you would add TracerProvider and MeterProvider here:
	// otelConfig.WithTracerProvider(tp).WithMeterProvider(mp)

	// Create Argo client with OTel
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithOTelConfig(otelConfig))
	if err != nil {
		log.Warn().Err(err).Msg("Failed to create Argo client (expected if no kubeconfig)")
		// Continue with example even if client creation fails
	} else {
		_ = client // client would be used to submit workflows
	}

	// Create workflow steps with OTel
	preCheck := template.NewContainer("pre-check", "alpine:latest",
		template.WithCommand("sh", "-c", "echo 'Running pre-checks...'"),
		template.WithOTelConfig(otelConfig))

	deploy := template.NewContainer("deploy", "alpine:latest",
		template.WithCommand("sh", "-c", "echo 'Deploying application...'"),
		template.WithCPU("500m"),
		template.WithMemory("256Mi"),
		template.WithOTelConfig(otelConfig))

	healthCheck := template.NewContainer("health-check", "curlimages/curl:latest",
		template.WithCommand("curl", "-f", "http://myapp/health"),
		template.WithOTelConfig(otelConfig))

	// Exit handler with OTel
	notify := template.NewContainer("notify", "alpine:latest",
		template.WithCommand("sh", "-c", "echo 'Sending notification...'"),
		template.WithOTelConfig(otelConfig))

	// Build workflow with OTel instrumentation
	wf, err := builder.NewWorkflowBuilder("deployment", "argo",
		builder.WithOTelConfig(otelConfig),
		builder.WithServiceAccount("argo-workflow"),
		builder.WithLabels(map[string]string{
			"observability": "enabled",
			"otel":          "true",
		}),
		builder.WithArchiveLogs(true)).
		Add(preCheck).
		Add(deploy).
		Add(healthCheck).
		AddExitHandler(notify).
		Build()

	if err != nil {
		return fmt.Errorf("failed to build workflow with OTel: %w", err)
	}

	log.Info().
		Str("workflow", wf.GenerateName).
		Int("templates", len(wf.Spec.Templates)).
		Bool("otel_enabled", true).
		Msg("Workflow with OTel instrumentation built successfully")

	// In production, you would submit the workflow:
	// created, err := argo.SubmitWorkflow(ctx, client, wf, otelConfig)

	return nil
}
