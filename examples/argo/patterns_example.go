//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jasoet/pkg/v2/argo"
	"github.com/jasoet/pkg/v2/argo/builder"
	"github.com/jasoet/pkg/v2/argo/builder/template"
)

// Example 1: Sequential Workflow Pattern
// Demonstrates steps that run one after another
func exampleSequentialWorkflow() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Each step runs after the previous completes
	checkout := template.NewContainer("checkout", "alpine/git:latest",
		template.WithCommand("git", "clone", "https://github.com/example/repo.git", "/workspace"))

	build := template.NewContainer("build", "golang:1.25",
		template.WithCommand("sh", "-c", "cd /workspace && go build -o app"))

	test := template.NewContainer("test", "golang:1.25",
		template.WithCommand("sh", "-c", "cd /workspace && go test ./..."))

	deploy := template.NewContainer("deploy", "kubectl:latest",
		template.WithCommand("kubectl", "apply", "-f", "/workspace/k8s/"))

	wf, err := builder.NewWorkflowBuilder("sequential-pipeline", "argo",
		builder.WithServiceAccount("default")).
		Add(checkout).
		Add(build).
		Add(test).
		Add(deploy).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Sequential workflow submitted: %s\n", created.Name)
	fmt.Println("Execution order: checkout → build → test → deploy")
}

// Example 2: CI/CD Pipeline Pattern
// Demonstrates a complete continuous integration/deployment workflow
func exampleCICDPipeline() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Stage 1: Source
	gitClone := template.NewContainer("git-clone", "alpine/git:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("git clone {{workflow.parameters.repo-url}} /workspace && cd /workspace && git checkout {{workflow.parameters.branch}}"))

	// Stage 2: Build
	goBuild := template.NewContainer("go-build", "golang:1.25",
		template.WithCommand("sh", "-c", "cd /workspace && go build -o app ./cmd/server"),
		template.WithCPU("1000m", "2000m"),
		template.WithMemory("512Mi", "1Gi"))

	dockerBuild := template.NewContainer("docker-build", "docker:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("cd /workspace && docker build -t {{workflow.parameters.image-name}}:{{workflow.parameters.image-tag}} ."))

	// Stage 3: Test
	unitTests := template.NewContainer("unit-tests", "golang:1.25",
		template.WithCommand("sh", "-c", "cd /workspace && go test -v ./..."))

	integrationTests := template.NewContainer("integration-tests", "golang:1.25",
		template.WithCommand("sh", "-c", "cd /workspace && go test -v -tags=integration ./..."))

	// Stage 4: Security Scan
	securityScan := template.NewContainer("security-scan", "aquasec/trivy:latest",
		template.WithCommand("trivy", "image", "{{workflow.parameters.image-name}}:{{workflow.parameters.image-tag}}"))

	// Stage 5: Push
	dockerPush := template.NewContainer("docker-push", "docker:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("docker push {{workflow.parameters.image-name}}:{{workflow.parameters.image-tag}}"))

	// Stage 6: Deploy
	deployStaging := template.NewContainer("deploy-staging", "kubectl:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("kubectl set image deployment/myapp myapp={{workflow.parameters.image-name}}:{{workflow.parameters.image-tag}} -n staging"))

	// Stage 7: Smoke Test
	smokeTest := template.NewHTTP("smoke-test",
		template.WithHTTPURL("https://staging.example.com/health"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPSuccessCond("response.statusCode == 200"))

	// Stage 8: Production Deploy (conditional)
	deployProduction := template.NewContainer("deploy-production", "kubectl:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("kubectl set image deployment/myapp myapp={{workflow.parameters.image-name}}:{{workflow.parameters.image-tag}} -n production"),
		template.WithWhen("{{workflow.parameters.deploy-to-prod}} == true"))

	// Notification
	notifySuccess := template.NewHTTP("notify-success",
		template.WithHTTPURL("https://hooks.slack.com/services/YOUR/WEBHOOK"),
		template.WithHTTPMethod("POST"),
		template.WithHTTPBody(`{"text": "✅ CI/CD Pipeline succeeded for {{workflow.parameters.branch}}"}`))

	wf, err := builder.NewWorkflowBuilder("cicd-pipeline", "argo",
		builder.WithServiceAccount("cicd-sa"),
		builder.WithLabels(map[string]string{
			"pipeline": "cicd",
			"type":     "deployment",
		})).
		Add(gitClone).
		Add(goBuild).
		Add(dockerBuild).
		Add(unitTests).
		Add(integrationTests).
		Add(securityScan).
		Add(dockerPush).
		Add(deployStaging).
		Add(smokeTest).
		Add(deployProduction).
		Add(notifySuccess).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("CI/CD pipeline workflow submitted: %s\n", created.Name)
}

// Example 3: Data Pipeline Pattern
// Demonstrates ETL (Extract, Transform, Load) workflow
func exampleDataPipeline() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Extract: Pull data from multiple sources
	extractDB := template.NewContainer("extract-db", "postgres:15",
		template.WithCommand("sh", "-c"),
		template.WithArgs("pg_dump -h {{workflow.parameters.db-host}} -U postgres -d mydb > /data/db.sql"))

	extractAPI := template.NewHTTP("extract-api",
		template.WithHTTPURL("{{workflow.parameters.api-url}}/export"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPHeader("Authorization", "Bearer {{workflow.parameters.api-token}}"))

	extractFiles := template.NewContainer("extract-files", "alpine:3.19",
		template.WithCommand("sh", "-c"),
		template.WithArgs("wget {{workflow.parameters.file-url}} -O /data/input.csv"))

	// Transform: Process and clean data
	transformData := template.NewScript("transform-data", "python",
		template.WithScriptContent(`
import pandas as pd
import json

print("Loading data...")
# Load from multiple sources
db_data = pd.read_sql("SELECT * FROM /data/db.sql")
api_data = pd.read_json("/data/api_response.json")
file_data = pd.read_csv("/data/input.csv")

print("Transforming data...")
# Clean, merge, and transform
merged = pd.concat([db_data, api_data, file_data])
transformed = merged.dropna().drop_duplicates()

print("Saving transformed data...")
transformed.to_parquet("/data/transformed.parquet")
print(f"Transformed {len(transformed)} records")
`),
		template.WithScriptEnv("PYTHONUNBUFFERED", "1"),
		template.WithScriptImage("python:3.11"))

	// Load: Store processed data
	loadWarehouse := template.NewContainer("load-warehouse", "postgres:15",
		template.WithCommand("sh", "-c"),
		template.WithArgs("psql -h {{workflow.parameters.warehouse-host}} -U postgres -d warehouse -c 'COPY transformed FROM /data/transformed.parquet'"))

	// Validate: Check data quality
	validateData := template.NewScript("validate-data", "python",
		template.WithScriptContent(`
import pandas as pd

data = pd.read_parquet("/data/transformed.parquet")

# Data quality checks
assert len(data) > 0, "No data loaded"
assert data.isnull().sum().sum() == 0, "Null values found"
assert len(data) == len(data.drop_duplicates()), "Duplicate records found"

print(f"✓ Data quality checks passed for {len(data)} records")
`))

	// Notify completion
	notifyComplete := template.NewHTTP("notify-complete",
		template.WithHTTPURL("https://api.example.com/pipeline/complete"),
		template.WithHTTPMethod("POST"),
		template.WithHTTPBody(`{"pipeline": "etl", "status": "success", "records": "{{outputs.parameters.record-count}}"}`))

	wf, err := builder.NewWorkflowBuilder("data-pipeline", "argo",
		builder.WithServiceAccount("default"),
		builder.WithLabels(map[string]string{
			"pipeline": "etl",
			"data":     "processing",
		})).
		Add(extractDB).
		Add(extractAPI).
		Add(extractFiles).
		Add(transformData).
		Add(loadWarehouse).
		Add(validateData).
		Add(notifyComplete).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Data pipeline workflow submitted: %s\n", created.Name)
}

// Example 4: Microservices Deployment Pattern
// Demonstrates deploying multiple services with health checks
func exampleMicroservicesDeployment() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Pre-deployment checks
	checkNamespace := template.NewContainer("check-namespace", "kubectl:latest",
		template.WithCommand("kubectl", "get", "namespace", "{{workflow.parameters.namespace}}"))

	// Deploy services
	deployAuth := template.NewContainer("deploy-auth-service", "kubectl:latest",
		template.WithCommand("kubectl", "apply", "-f", "/manifests/auth-service.yaml", "-n", "{{workflow.parameters.namespace}}"))

	deployAPI := template.NewContainer("deploy-api-service", "kubectl:latest",
		template.WithCommand("kubectl", "apply", "-f", "/manifests/api-service.yaml", "-n", "{{workflow.parameters.namespace}}"))

	deployWeb := template.NewContainer("deploy-web-service", "kubectl:latest",
		template.WithCommand("kubectl", "apply", "-f", "/manifests/web-service.yaml", "-n", "{{workflow.parameters.namespace}}"))

	// Wait for rollout
	waitAuth := template.NewContainer("wait-auth", "kubectl:latest",
		template.WithCommand("kubectl", "rollout", "status", "deployment/auth-service", "-n", "{{workflow.parameters.namespace}}"))

	waitAPI := template.NewContainer("wait-api", "kubectl:latest",
		template.WithCommand("kubectl", "rollout", "status", "deployment/api-service", "-n", "{{workflow.parameters.namespace}}"))

	waitWeb := template.NewContainer("wait-web", "kubectl:latest",
		template.WithCommand("kubectl", "rollout", "status", "deployment/web-service", "-n", "{{workflow.parameters.namespace}}"))

	// Health checks
	healthAuth := template.NewHTTP("health-auth",
		template.WithHTTPURL("{{workflow.parameters.auth-url}}/health"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPSuccessCond("response.statusCode == 200"))

	healthAPI := template.NewHTTP("health-api",
		template.WithHTTPURL("{{workflow.parameters.api-url}}/health"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPSuccessCond("response.statusCode == 200"))

	healthWeb := template.NewHTTP("health-web",
		template.WithHTTPURL("{{workflow.parameters.web-url}}/health"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPSuccessCond("response.statusCode == 200"))

	wf, err := builder.NewWorkflowBuilder("microservices-deploy", "argo",
		builder.WithServiceAccount("deployment-sa")).
		Add(checkNamespace).
		Add(deployAuth).
		Add(deployAPI).
		Add(deployWeb).
		Add(waitAuth).
		Add(waitAPI).
		Add(waitWeb).
		Add(healthAuth).
		Add(healthAPI).
		Add(healthWeb).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Microservices deployment workflow submitted: %s\n", created.Name)
}

// Example 5: Backup and Restore Pattern
// Demonstrates database backup and validation workflow
func exampleBackupRestore() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Pre-backup validation
	checkDatabase := template.NewContainer("check-database", "postgres:15",
		template.WithCommand("pg_isready", "-h", "{{workflow.parameters.db-host}}"))

	// Backup
	backupDB := template.NewScript("backup-database", "bash",
		template.WithScriptContent(`
#!/bin/bash
set -e

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="/backup/db_backup_$DATE.sql.gz"

echo "Starting backup at $(date)"
pg_dump -h {{workflow.parameters.db-host}} \
        -U postgres \
        -d {{workflow.parameters.db-name}} \
        | gzip > $BACKUP_FILE

echo "Backup created: $BACKUP_FILE"
echo "Size: $(du -h $BACKUP_FILE | cut -f1)"

# Upload to S3 or other storage
# aws s3 cp $BACKUP_FILE s3://backups/postgres/

echo "Backup completed successfully"
`))

	// Verify backup
	verifyBackup := template.NewScript("verify-backup", "bash",
		template.WithScriptContent(`
#!/bin/bash
set -e

BACKUP_FILE=$(ls -t /backup/*.sql.gz | head -1)

echo "Verifying backup: $BACKUP_FILE"

# Check file exists and is not empty
if [ ! -s "$BACKUP_FILE" ]; then
    echo "Error: Backup file is empty or does not exist"
    exit 1
fi

# Test gunzip
gunzip -t "$BACKUP_FILE"

echo "✓ Backup verification successful"
`))

	// Cleanup old backups
	cleanupOld := template.NewScript("cleanup-old-backups", "bash",
		template.WithScriptContent(`
#!/bin/bash

echo "Cleaning up backups older than 30 days..."
find /backup -name "db_backup_*.sql.gz" -mtime +30 -delete

REMAINING=$(ls /backup/*.sql.gz | wc -l)
echo "Remaining backups: $REMAINING"
`))

	// Notify
	notifyBackup := template.NewHTTP("notify-backup",
		template.WithHTTPURL("https://api.example.com/notifications"),
		template.WithHTTPMethod("POST"),
		template.WithHTTPBody(`{"type": "backup", "status": "success", "database": "{{workflow.parameters.db-name}}"}`))

	wf, err := builder.NewWorkflowBuilder("database-backup", "argo",
		builder.WithServiceAccount("default")).
		Add(checkDatabase).
		Add(backupDB).
		Add(verifyBackup).
		Add(cleanupOld).
		Add(notifyBackup).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Backup workflow submitted: %s\n", created.Name)
}

// Example 6: ML Training Pipeline Pattern
// Demonstrates machine learning workflow
func exampleMLPipeline() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Data preparation
	prepareData := template.NewScript("prepare-data", "python",
		template.WithScriptContent(`
import pandas as pd
from sklearn.model_selection import train_test_split

print("Loading raw data...")
data = pd.read_csv("/data/raw/dataset.csv")

print("Preprocessing...")
# Clean and preprocess
data = data.dropna()
X = data.drop("target", axis=1)
y = data["target"]

# Split
X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.2)

print(f"Training set: {len(X_train)} samples")
print(f"Test set: {len(X_test)} samples")

# Save splits
X_train.to_csv("/data/processed/X_train.csv", index=False)
X_test.to_csv("/data/processed/X_test.csv", index=False)
y_train.to_csv("/data/processed/y_train.csv", index=False)
y_test.to_csv("/data/processed/y_test.csv", index=False)
`),
		template.WithScriptImage("python:3.11"))

	// Train model
	trainModel := template.NewScript("train-model", "python",
		template.WithScriptContent(`
import pandas as pd
from sklearn.ensemble import RandomForestClassifier
import joblib

print("Loading training data...")
X_train = pd.read_csv("/data/processed/X_train.csv")
y_train = pd.read_csv("/data/processed/y_train.csv")

print("Training model...")
model = RandomForestClassifier(n_estimators=100, random_state=42)
model.fit(X_train, y_train)

print("Saving model...")
joblib.dump(model, "/models/model.pkl")
print("Model training complete!")
`),
		template.WithScriptImage("python:3.11")).
		CPU("4000m", "8000m").
		Memory("8Gi", "16Gi")

	// Evaluate model
	evaluateModel := template.NewScript("evaluate-model", "python",
		template.WithScriptContent(`
import pandas as pd
import joblib
from sklearn.metrics import accuracy_score, classification_report

print("Loading model and test data...")
model = joblib.load("/models/model.pkl")
X_test = pd.read_csv("/data/processed/X_test.csv")
y_test = pd.read_csv("/data/processed/y_test.csv")

print("Evaluating model...")
y_pred = model.predict(X_test)
accuracy = accuracy_score(y_test, y_pred)

print(f"Accuracy: {accuracy:.4f}")
print("\nClassification Report:")
print(classification_report(y_test, y_pred))

# Save metrics
with open("/metrics/accuracy.txt", "w") as f:
    f.write(f"{accuracy:.4f}")
`))

	// Deploy model
	deployModel := template.NewContainer("deploy-model", "kubectl:latest",
		template.WithCommand("sh", "-c"),
		template.WithArgs("kubectl apply -f /manifests/model-serving.yaml"))

	wf, err := builder.NewWorkflowBuilder("ml-pipeline", "argo",
		builder.WithServiceAccount("ml-sa"),
		builder.WithLabels(map[string]string{
			"pipeline": "ml",
			"type":     "training",
		})).
		Add(prepareData).
		Add(trainModel).
		Add(evaluateModel).
		Add(deployModel).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("ML pipeline workflow submitted: %s\n", created.Name)
}

// Example 7: Monitoring and Alerting Pattern
// Demonstrates health check and alerting workflow
func exampleMonitoringWorkflow() {
	ctx := context.Background()

	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	ctx, client, err := argo.NewClientWithOptions(ctx,
		argo.WithKubeConfig(kubeconfigPath),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Check services
	checkAPI := template.NewHTTP("check-api",
		template.WithHTTPURL("{{workflow.parameters.api-url}}/health"),
		template.WithHTTPMethod("GET"),
		template.WithHTTPSuccessCond("response.statusCode == 200"),
		template.WithHTTPTimeout(10))

	checkDB := template.NewContainer("check-database", "postgres:15",
		template.WithCommand("pg_isready", "-h", "{{workflow.parameters.db-host}}"))

	checkCache := template.NewContainer("check-redis", "redis:7",
		template.WithCommand("redis-cli", "-h", "{{workflow.parameters.redis-host}}", "ping"))

	// Collect metrics
	collectMetrics := template.NewScript("collect-metrics", "python",
		template.WithScriptContent(`
import requests
import json

metrics = {
    "timestamp": "{{workflow.creationTimestamp}}",
    "api_health": "healthy",
    "db_health": "healthy",
    "cache_health": "healthy"
}

# Send to monitoring system
print(json.dumps(metrics, indent=2))
`))

	// Alert if unhealthy (conditional)
	sendAlert := template.NewHTTP("send-alert",
		template.WithHTTPURL("https://api.pagerduty.com/incidents"),
		template.WithHTTPMethod("POST"),
		template.WithHTTPHeader("Authorization", "Token {{workflow.parameters.pagerduty-token}}"),
		template.WithHTTPBody(`{"incident": {"type": "incident", "title": "Service health check failed"}}`)).
		When("{{workflow.status}} == Failed")

	wf, err := builder.NewWorkflowBuilder("monitoring-check", "argo",
		builder.WithServiceAccount("default")).
		Add(checkAPI).
		Add(checkDB).
		Add(checkCache).
		Add(collectMetrics).
		Add(sendAlert).
		Build()
	if err != nil {
		log.Fatalf("Failed to build workflow: %v", err)
	}

	created, err := argo.SubmitWorkflow(ctx, client, wf, nil)
	if err != nil {
		log.Fatalf("Failed to submit workflow: %v", err)
	}

	fmt.Printf("Monitoring workflow submitted: %s\n", created.Name)
}

// Example 8: Workflow Patterns Summary
// Explains when to use each pattern
func examplePatternsSummary() {
	fmt.Println("Common Workflow Patterns")
	fmt.Println("=========================")
	fmt.Println()

	fmt.Println("1. SEQUENTIAL WORKFLOW")
	fmt.Println("   Use when: Steps must run in specific order")
	fmt.Println("   Example: Build → Test → Deploy")
	fmt.Println()

	fmt.Println("2. CI/CD PIPELINE")
	fmt.Println("   Use when: Automating software delivery")
	fmt.Println("   Example: Code → Build → Test → Scan → Deploy")
	fmt.Println()

	fmt.Println("3. DATA PIPELINE (ETL)")
	fmt.Println("   Use when: Processing and transforming data")
	fmt.Println("   Example: Extract → Transform → Load → Validate")
	fmt.Println()

	fmt.Println("4. MICROSERVICES DEPLOYMENT")
	fmt.Println("   Use when: Deploying multiple interconnected services")
	fmt.Println("   Example: Deploy services → Wait for ready → Health check")
	fmt.Println()

	fmt.Println("5. BACKUP/RESTORE")
	fmt.Println("   Use when: Data protection and recovery")
	fmt.Println("   Example: Check → Backup → Verify → Cleanup → Notify")
	fmt.Println()

	fmt.Println("6. ML PIPELINE")
	fmt.Println("   Use when: Training and deploying ML models")
	fmt.Println("   Example: Prepare → Train → Evaluate → Deploy")
	fmt.Println()

	fmt.Println("7. MONITORING/ALERTING")
	fmt.Println("   Use when: Health checks and incident response")
	fmt.Println("   Example: Check services → Collect metrics → Alert if needed")
	fmt.Println()
}

func main() {
	fmt.Println("Argo Workflow Pattern Examples")
	fmt.Println("================================")
	fmt.Println()
	fmt.Println("Uncomment the example you want to run:")
	fmt.Println()

	// Uncomment one example at a time to run:

	// exampleSequentialWorkflow()
	// exampleCICDPipeline()
	// exampleDataPipeline()
	// exampleMicroservicesDeployment()
	// exampleBackupRestore()
	// exampleMLPipeline()
	// exampleMonitoringWorkflow()
	// examplePatternsSummary()

	fmt.Println("Please uncomment one of the example functions in main()")
}
