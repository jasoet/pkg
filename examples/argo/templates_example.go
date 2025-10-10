//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/jasoet/pkg/v2/argo"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// Example 1: Basic Container Template
// Demonstrates the simplest container template
func exampleBasicContainer() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Simple container that runs a single command
	step := template.NewContainer("hello", "alpine:3.19",
		template.WithCommand("echo", "Hello, World!"))

	wf, err := builder.NewWorkflowBuilder("basic-container", "argo",
		builder.WithServiceAccount("default")).
		Add(step).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Basic container workflow submitted: %s\n", created.Name)
}

// Example 2: Container with Command and Arguments
// Demonstrates separation of command and args
func exampleContainerCommandArgs() {
	// Using functional options
	deploy := template.NewContainer("deploy-app", "ubuntu:22.04",
		template.WithCommand("bash", "-c"),
		template.WithArgs("apt-get update && apt-get install -y curl && curl https://example.com"))

	// Using fluent API
	backup := template.NewContainer("backup-db", "postgres:15").
		Command("pg_dump").
		Args("-U", "postgres", "-d", "mydb", "-f", "/backup/db.sql")

	fmt.Printf("Deploy step: %+v\n", deploy)
	fmt.Printf("Backup step: %+v\n", backup)
}

// Example 3: Container with Environment Variables
// Demonstrates various ways to set environment variables
func exampleContainerEnvironment() {
	// Using functional options
	app1 := template.NewContainer("app1", "myapp:v1",
		template.WithEnv("LOG_LEVEL", "debug"),
		template.WithEnv("DATABASE_HOST", "postgres.default.svc.cluster.local"))

	// Using fluent API
	app2 := template.NewContainer("app2", "myapp:v1").
		Env("API_KEY", "secret-key").
		Env("PORT", "8080").
		Env("ENVIRONMENT", "production")

	// Using EnvFrom for secrets/configmaps
	app3 := template.NewContainer("app3", "myapp:v1").
		EnvFrom("API_TOKEN", corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "api-secrets"},
				Key:                  "token",
			},
		}).
		EnvFrom("CONFIG_FILE", corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "app-config"},
				Key:                  "config.json",
			},
		})

	fmt.Printf("App1: %+v\n", app1)
	fmt.Printf("App2: %+v\n", app2)
	fmt.Printf("App3: %+v\n", app3)
}

// Example 4: Container with Volume Mounts
// Demonstrates mounting volumes in containers
func exampleContainerVolumes() {
	process := template.NewContainer("process-data", "python:3.11").
		Command("python", "process.py").
		VolumeMount("data", "/data", false).     // Read-write data volume
		VolumeMount("config", "/config", true).  // Read-only config
		VolumeMount("output", "/output", false). // Write output
		WorkingDir("/app")

	fmt.Printf("Process step with volumes: %+v\n", process)
}

// Example 5: Container with Resource Limits
// Demonstrates CPU and memory resource management
func exampleContainerResources() {
	// Set same request and limit
	compute1 := template.NewContainer("compute1", "myapp:v1",
		template.WithCPU("1000m"),    // 1 CPU core
		template.WithMemory("512Mi")) // 512 MB

	// Set different request and limit
	compute2 := template.NewContainer("compute2", "myapp:v1").
		CPU("500m", "2000m").  // Request 0.5, limit 2 cores
		Memory("256Mi", "1Gi") // Request 256MB, limit 1GB

	// Using fluent API for more readable configuration
	compute3 := template.NewContainer("compute3", "ml-trainer:latest").
		Command("python", "train.py").
		CPU("2000m", "4000m"). // Request 2, limit 4 cores
		Memory("4Gi", "8Gi").  // Request 4GB, limit 8GB
		Env("BATCH_SIZE", "128")

	fmt.Printf("Compute1: %+v\n", compute1)
	fmt.Printf("Compute2: %+v\n", compute2)
	fmt.Printf("Compute3: %+v\n", compute3)
}

// Example 6: Container with Conditional Execution
// Demonstrates using "when" conditions and continue-on
func exampleContainerConditional() {
	// Step runs only when previous step succeeded
	notifySuccess := template.NewContainer("notify-success", "curlimages/curl:latest",
		template.WithCommand("curl", "-X", "POST", "https://api.example.com/success"),
		template.WithWhen("{{workflow.status}} == Succeeded"))

	// Step continues even if it fails
	cleanup := template.NewContainer("cleanup", "alpine:3.19").
		Command("sh", "-c", "rm -rf /tmp/data || true").
		ContinueOn(&v1alpha1.ContinueOn{
			Failed: true,
			Error:  true,
		})

	// Step runs only on failure
	notifyFailure := template.NewContainer("notify-failure", "curlimages/curl:latest").
		Command("curl", "-X", "POST", "https://api.example.com/failure").
		When("{{workflow.status}} == Failed")

	fmt.Printf("Conditional steps: %+v, %+v, %+v\n", notifySuccess, cleanup, notifyFailure)
}

// Example 7: Container with Retry Strategy
// Demonstrates retry policies for flaky operations
func exampleContainerRetry() {
	limit := intstr.FromInt32(3)
	backoffFactor := intstr.FromInt32(2)

	// Retry up to 3 times
	apiCall := template.NewContainer("api-call", "curlimages/curl:latest").
		Command("curl", "-f", "https://api.example.com/data").
		WithRetry(&v1alpha1.RetryStrategy{
			Limit:       &limit,
			RetryPolicy: v1alpha1.RetryPolicyAlways,
			Backoff: &v1alpha1.Backoff{
				Duration:    "1m",
				Factor:      &backoffFactor,
				MaxDuration: "10m",
			},
		})

	fmt.Printf("API call with retry: %+v\n", apiCall)
}

// Example 8: Container with All Options
// Demonstrates a fully configured container
func exampleContainerAllOptions() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	limit := intstr.FromInt32(2)

	// Fully configured container with all features
	fullFeatured := template.NewContainer("full-featured", "myapp:v1").
		Command("sh", "-c").
		Args("echo 'Starting...'; /app/run.sh; echo 'Done!'").
		Env("LOG_LEVEL", "info").
		Env("WORKERS", "4").
		EnvFrom("DB_PASSWORD", corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{Name: "db-secrets"},
				Key:                  "password",
			},
		}).
		VolumeMount("config", "/etc/config", true).
		VolumeMount("data", "/data", false).
		WorkingDir("/app").
		ImagePullPolicy(corev1.PullAlways).
		CPU("1000m", "2000m").
		Memory("512Mi", "1Gi").
		When("{{workflow.status}} == Running").
		ContinueOn(&v1alpha1.ContinueOn{Failed: false}).
		WithRetry(&v1alpha1.RetryStrategy{
			Limit:       &limit,
			RetryPolicy: v1alpha1.RetryPolicyOnFailure,
		})

	wf, err := builder.NewWorkflowBuilder("full-featured-container", "argo",
		builder.WithServiceAccount("default")).
		Add(fullFeatured).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Full-featured container workflow submitted: %s\n", created.Name)
}

// Example 9: Basic Script Template - Bash
// Demonstrates bash script execution
func exampleScriptBash() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	bashScript := template.NewScript("backup", "bash",
		template.WithScriptContent(`
#!/bin/bash
set -e

echo "Starting backup process..."
DATE=$(date +%Y%m%d)
BACKUP_FILE="/backup/data-$DATE.tar.gz"

tar -czf $BACKUP_FILE /data
echo "Backup created: $BACKUP_FILE"

# Upload to S3 (example)
# aws s3 cp $BACKUP_FILE s3://my-bucket/backups/

echo "Backup completed successfully!"
`))

	wf, err := builder.NewWorkflowBuilder("bash-script", "argo",
		builder.WithServiceAccount("default")).
		Add(bashScript).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Bash script workflow submitted: %s\n", created.Name)
}

// Example 10: Python Script Template
// Demonstrates Python script execution with data processing
func exampleScriptPython() {
	pythonScript := template.NewScript("process-data", "python",
		template.WithScriptContent(`
import json
import sys
from datetime import datetime

print(f"Processing data at {datetime.now()}")

# Read input data
data = {
    "timestamp": str(datetime.now()),
    "records_processed": 1000,
    "status": "success"
}

# Process and output
print(json.dumps(data, indent=2))
print("Processing complete!")
`),
		template.WithScriptEnv("PYTHONUNBUFFERED", "1"),
		template.WithScriptEnv("LOG_LEVEL", "INFO"))

	fmt.Printf("Python script: %+v\n", pythonScript)
}

// Example 11: Node.js Script Template
// Demonstrates JavaScript/Node.js script execution
func exampleScriptNode() {
	nodeScript := template.NewScript("api-integration", "node",
		template.WithScriptContent(`
const https = require('https');

console.log('Fetching data from API...');

https.get('https://api.github.com/users/github', (res) => {
  let data = '';

  res.on('data', (chunk) => {
    data += chunk;
  });

  res.on('end', () => {
    const user = JSON.parse(data);
    console.log('User:', user.login);
    console.log('Repos:', user.public_repos);
  });
}).on('error', (err) => {
  console.error('Error:', err.message);
  process.exit(1);
});
`),
		template.WithScriptImage("node:20-alpine"))

	fmt.Printf("Node.js script: %+v\n", nodeScript)
}

// Example 12: Ruby Script Template
// Demonstrates Ruby script execution
func exampleScriptRuby() {
	rubyScript := template.NewScript("data-transform", "ruby",
		template.WithScriptContent(`
require 'json'

puts "Starting data transformation..."

data = {
  items: (1..10).map { |i| { id: i, value: i * 2 } },
  timestamp: Time.now.to_s
}

puts JSON.pretty_generate(data)
puts "Transformation complete!"
`))

	fmt.Printf("Ruby script: %+v\n", rubyScript)
}

// Example 13: Script with Resources and Volumes
// Demonstrates script with resource limits and volume mounts
func exampleScriptWithResources() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	dataProcessor := template.NewScript("etl-process", "python").
		Script(`
import os
import time

print("ETL Process starting...")
print(f"Input dir: {os.getenv('INPUT_DIR')}")
print(f"Output dir: {os.getenv('OUTPUT_DIR')}")

# Simulate data processing
time.sleep(5)

print("ETL Process completed!")
`).
		Env("INPUT_DIR", "/data/input").
		Env("OUTPUT_DIR", "/data/output").
		VolumeMount("data", "/data", false).
		WorkingDir("/workspace").
		CPU("2000m", "4000m").
		Memory("2Gi", "4Gi")

	wf, err := builder.NewWorkflowBuilder("etl-workflow", "argo",
		builder.WithServiceAccount("default")).
		Add(dataProcessor).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("ETL script workflow submitted: %s\n", created.Name)
}

// Example 14: Script with Custom Image
// Demonstrates using a custom image with specific tools
func exampleScriptCustomImage() {
	customScript := template.NewScript("ml-training", "python").
		Image("tensorflow/tensorflow:latest-gpu").
		Command("python3").
		Script(`
import tensorflow as tf

print(f"TensorFlow version: {tf.__version__}")
print(f"GPU available: {tf.config.list_physical_devices('GPU')}")

# Training logic here...
print("Model training complete!")
`).
		CPU("4000m", "8000m").
		Memory("8Gi", "16Gi").
		Env("CUDA_VISIBLE_DEVICES", "0")

	fmt.Printf("ML training script: %+v\n", customScript)
}

// Example 15: HTTP Template - GET Request
// Demonstrates simple HTTP GET request
func exampleHTTPGet() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	healthCheck := template.NewHTTP("health-check",
		template.WithHTTPURL("https://httpbin.org/get"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPSuccessCond("response.statusCode == 200"))

	wf, err := builder.NewWorkflowBuilder("http-get", "argo",
		builder.WithServiceAccount("default")).
		Add(healthCheck).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("HTTP GET workflow submitted: %s\n", created.Name)
}

// Example 16: HTTP Template - POST Request with Body
// Demonstrates POST request with JSON body
func exampleHTTPPost() {
	apiCall := template.NewHTTP("create-resource",
		template.WithHTTPURL("https://api.example.com/v1/resources"),
		template.WithHTTPMethod("POST"),
		template.WithHTTPHeader("Content-Type", "application/json"),
		template.WithHTTPHeader("Authorization", "Bearer {{workflow.parameters.api-token}}"),
		template.WithHTTPBody(`{
			"name": "new-resource",
			"description": "Created by Argo Workflow",
			"metadata": {
				"workflow": "{{workflow.name}}",
				"namespace": "{{workflow.namespace}}"
			}
		}`),
		template.WithHTTPSuccessCond("response.statusCode >= 200 && response.statusCode < 300"),
		template.WithHTTPTimeout(60))

	fmt.Printf("HTTP POST: %+v\n", apiCall)
}

// Example 17: HTTP Template - Webhook Notification
// Demonstrates using HTTP for webhook notifications
func exampleHTTPWebhook() {
	slackNotification := template.NewHTTP("notify-slack").
		URL("https://hooks.slack.com/services/YOUR/WEBHOOK/URL").
		Method("POST").
		Header("Content-Type", "application/json").
		Body(`{
			"text": "Workflow {{workflow.name}} completed with status: {{workflow.status}}",
			"username": "Argo Workflows",
			"icon_emoji": ":rocket:"
		}`).
		SuccessCondition("response.statusCode == 200").
		Timeout(30)

	fmt.Printf("Slack webhook: %+v\n", slackNotification)
}

// Example 18: HTTP Template - API Polling
// Demonstrates polling an API endpoint
func exampleHTTPPolling() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	checkStatus := template.NewHTTP("check-job-status",
		template.WithHTTPURL("https://api.example.com/jobs/{{workflow.parameters.job-id}}"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPHeader("Authorization", "Bearer {{workflow.parameters.token}}"),
		template.WithHTTPSuccessCond("response.body.status == 'completed'"),
		template.WithHTTPTimeout(10))

	wf, err := builder.NewWorkflowBuilder("polling-workflow", "argo",
		builder.WithServiceAccount("default")).
		Add(checkStatus).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Polling workflow submitted: %s\n", created.Name)
}

// Example 19: HTTP Template with Complex Success Condition
// Demonstrates advanced success condition evaluation
func exampleHTTPComplexCondition() {
	advancedAPI := template.NewHTTP("advanced-check").
		URL("https://api.example.com/status").
		Method("GET").
		Header("Accept", "application/json").
		SuccessCondition(`
			response.statusCode == 200 &&
			response.body.status == "healthy" &&
			response.body.metrics.cpu < 80 &&
			response.body.metrics.memory < 90
		`).
		Timeout(30)

	fmt.Printf("Advanced API check: %+v\n", advancedAPI)
}

// Example 20: Noop Template
// Demonstrates placeholder steps
func exampleNoop() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Simple noop
	noop1 := template.NewNoop()

	// Named noops for clarity
	placeholder1 := template.NewNoopWithName("placeholder-1")
	placeholder2 := template.NewNoopWithName("placeholder-2")

	wf, err := builder.NewWorkflowBuilder("noop-workflow", "argo",
		builder.WithServiceAccount("default")).
		Add(noop1).
		Add(placeholder1).
		Add(placeholder2).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Noop workflow submitted: %s\n", created.Name)
}

// Example 21: Mixed Template Types
// Demonstrates combining different template types in one workflow
func exampleMixedTemplates() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Step 1: Check API availability
	healthCheck := template.NewHTTP("check-api",
		template.WithHTTPURL("https://api.example.com/health"),
		template.WithHTTPMethod("GET"))

	// Step 2: Run container-based setup
	setup := template.NewContainer("setup", "alpine:3.19",
		template.WithCommand("sh", "-c", "echo 'Setting up environment...'"))

	// Step 3: Execute Python data processing
	process := template.NewScript("process", "python",
		template.WithScriptContent(`
print("Processing data...")
# Data processing logic
print("Data processing complete!")
`))

	// Step 4: Run container-based deployment
	deploy := template.NewContainer("deploy", "kubectl:latest",
		template.WithCommand("kubectl", "apply", "-f", "/manifests/"))

	// Step 5: Send webhook notification
	notify := template.NewHTTP("notify",
		template.WithHTTPURL("https://hooks.slack.com/services/YOUR/WEBHOOK"),
		template.WithHTTPMethod("POST"),
		template.WithHTTPBody(`{"text": "Deployment complete!"}`))

	wf, err := builder.NewWorkflowBuilder("mixed-templates", "argo",
		builder.WithServiceAccount("default")).
		Add(healthCheck).
		Add(setup).
		Add(process).
		Add(deploy).
		Add(notify).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Mixed templates workflow submitted: %s\n", created.Name)
}

// Example 22: Template Comparison
// Shows when to use each template type
func exampleTemplateComparison() {
	fmt.Println("Template Type Comparison:")
	fmt.Println("=========================")
	fmt.Println()

	fmt.Println("1. CONTAINER - Best for:")
	fmt.Println("   - Running existing Docker images")
	fmt.Println("   - Complex applications with dependencies")
	fmt.Println("   - When you need volume mounts")
	fmt.Println("   - Resource-intensive workloads")
	fmt.Println()

	fmt.Println("2. SCRIPT - Best for:")
	fmt.Println("   - Quick inline scripts")
	fmt.Println("   - Data transformations")
	fmt.Println("   - Simple automation tasks")
	fmt.Println("   - When you want to keep script with workflow definition")
	fmt.Println()

	fmt.Println("3. HTTP - Best for:")
	fmt.Println("   - API calls and integrations")
	fmt.Println("   - Webhook notifications")
	fmt.Println("   - Health checks")
	fmt.Println("   - Polling external services")
	fmt.Println()

	fmt.Println("4. NOOP - Best for:")
	fmt.Println("   - Workflow structure testing")
	fmt.Println("   - Placeholders during development")
	fmt.Println("   - Conditional branching points")
	fmt.Println()
}

func main() {
	fmt.Println("Argo Workflow Template Examples")
	fmt.Println("=================================")
	fmt.Println()
	fmt.Println("Uncomment the example you want to run:")
	fmt.Println()

	// Uncomment one example at a time to run:

	// Container Templates
	// exampleBasicContainer()
	// exampleContainerCommandArgs()
	// exampleContainerEnvironment()
	// exampleContainerVolumes()
	// exampleContainerResources()
	// exampleContainerConditional()
	// exampleContainerRetry()
	// exampleContainerAllOptions()

	// Script Templates
	// exampleScriptBash()
	// exampleScriptPython()
	// exampleScriptNode()
	// exampleScriptRuby()
	// exampleScriptWithResources()
	// exampleScriptCustomImage()

	// HTTP Templates
	// exampleHTTPGet()
	// exampleHTTPPost()
	// exampleHTTPWebhook()
	// exampleHTTPPolling()
	// exampleHTTPComplexCondition()

	// Noop Templates
	// exampleNoop()

	// Mixed
	// exampleMixedTemplates()
	// exampleTemplateComparison()

	fmt.Println("Please uncomment one of the example functions in main()")
}
