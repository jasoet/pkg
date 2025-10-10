package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jasoet/pkg/v2/temporal"
	"go.temporal.io/api/enums/v1"
)

func main() {
	// Configure Temporal connection
	config := &temporal.Config{
		HostPort:             getEnv("TEMPORAL_HOST", "localhost:7233"),
		Namespace:            getEnv("TEMPORAL_NAMESPACE", "default"),
		MetricsListenAddress: "0.0.0.0:0",
	}

	// Create WorkflowManager
	wm, err := temporal.NewWorkflowManager(config)
	if err != nil {
		log.Fatalf("Failed to create WorkflowManager: %v", err)
	}
	defer wm.Close()

	fmt.Println("=== Temporal Workflow Dashboard ===")
	fmt.Printf("Connected to: %s\n", config.HostPort)
	fmt.Printf("Namespace: %s\n\n", config.Namespace)

	// Create HTTP server for dashboard API
	mux := http.NewServeMux()

	// Dashboard statistics endpoint
	mux.HandleFunc("/api/stats", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		stats, err := wm.GetDashboardStats(ctx)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get stats: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(stats)
	})

	// List all workflows endpoint
	mux.HandleFunc("/api/workflows", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		pageSize := 100

		workflows, err := wm.ListWorkflows(ctx, pageSize, "")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list workflows: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workflows)
	})

	// List running workflows endpoint
	mux.HandleFunc("/api/workflows/running", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		workflows, err := wm.ListRunningWorkflows(ctx, 100)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list running workflows: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workflows)
	})

	// List failed workflows endpoint
	mux.HandleFunc("/api/workflows/failed", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		workflows, err := wm.ListFailedWorkflows(ctx, 100)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to list failed workflows: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workflows)
	})

	// Get specific workflow details endpoint
	mux.HandleFunc("/api/workflows/", func(w http.ResponseWriter, r *http.Request) {
		workflowID := r.URL.Path[len("/api/workflows/"):]
		if workflowID == "" {
			http.Error(w, "Workflow ID is required", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		details, err := wm.DescribeWorkflow(ctx, workflowID, "")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to describe workflow: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(details)
	})

	// Cancel workflow endpoint
	mux.HandleFunc("/api/workflows/cancel/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		workflowID := r.URL.Path[len("/api/workflows/cancel/"):]
		if workflowID == "" {
			http.Error(w, "Workflow ID is required", http.StatusBadRequest)
			return
		}

		ctx := context.Background()
		err := wm.CancelWorkflow(ctx, workflowID, "")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to cancel workflow: %v", err), http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "canceled", "workflowID": workflowID})
	})

	// Recent workflows endpoint
	mux.HandleFunc("/api/workflows/recent", func(w http.ResponseWriter, r *http.Request) {
		ctx := context.Background()
		workflows, err := wm.GetRecentWorkflows(ctx, 50)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get recent workflows: %v", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(workflows)
	})

	// Simple HTML dashboard
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Temporal Workflow Dashboard</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 20px; background: #f5f5f5; }
        h1 { color: #333; }
        .stats { display: flex; gap: 20px; margin: 20px 0; }
        .stat-card { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); flex: 1; }
        .stat-card h3 { margin: 0 0 10px 0; color: #666; font-size: 14px; }
        .stat-card .value { font-size: 32px; font-weight: bold; color: #333; }
        .workflows { background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 4px rgba(0,0,0,0.1); }
        .workflow-item { padding: 10px; border-bottom: 1px solid #eee; display: flex; justify-content: space-between; }
        .workflow-item:last-child { border-bottom: none; }
        .status { padding: 4px 8px; border-radius: 4px; font-size: 12px; font-weight: bold; }
        .status-running { background: #4CAF50; color: white; }
        .status-completed { background: #2196F3; color: white; }
        .status-failed { background: #f44336; color: white; }
        .btn { padding: 8px 16px; margin: 5px; cursor: pointer; border: none; border-radius: 4px; }
        .btn-refresh { background: #2196F3; color: white; }
        .loading { text-align: center; padding: 20px; }
    </style>
</head>
<body>
    <h1>⚡ Temporal Workflow Dashboard</h1>

    <button class="btn btn-refresh" onclick="loadDashboard()">🔄 Refresh</button>

    <div class="stats" id="stats">
        <div class="loading">Loading statistics...</div>
    </div>

    <div class="workflows">
        <h2>Recent Workflows</h2>
        <div id="workflows">
            <div class="loading">Loading workflows...</div>
        </div>
    </div>

    <script>
        function loadStats() {
            fetch('/api/stats')
                .then(r => r.json())
                .then(data => {
                    document.getElementById('stats').innerHTML = ` + "`" + `
                        <div class="stat-card">
                            <h3>Running</h3>
                            <div class="value" style="color: #4CAF50;">${data.TotalRunning}</div>
                        </div>
                        <div class="stat-card">
                            <h3>Completed</h3>
                            <div class="value" style="color: #2196F3;">${data.TotalCompleted}</div>
                        </div>
                        <div class="stat-card">
                            <h3>Failed</h3>
                            <div class="value" style="color: #f44336;">${data.TotalFailed}</div>
                        </div>
                        <div class="stat-card">
                            <h3>Avg Duration</h3>
                            <div class="value" style="color: #FF9800; font-size: 24px;">${formatDuration(data.AverageDuration)}</div>
                        </div>
                    ` + "`" + `;
                })
                .catch(err => console.error('Failed to load stats:', err));
        }

        function loadWorkflows() {
            fetch('/api/workflows/recent')
                .then(r => r.json())
                .then(data => {
                    const html = data.map(wf => ` + "`" + `
                        <div class="workflow-item">
                            <div>
                                <strong>${wf.WorkflowID}</strong><br>
                                <small>${wf.WorkflowType} • Started: ${new Date(wf.StartTime).toLocaleString()}</small>
                            </div>
                            <div>
                                <span class="status status-${wf.Status.toLowerCase()}">${wf.Status}</span>
                            </div>
                        </div>
                    ` + "`" + `).join('');
                    document.getElementById('workflows').innerHTML = html || '<div class="loading">No workflows found</div>';
                })
                .catch(err => console.error('Failed to load workflows:', err));
        }

        function formatDuration(ns) {
            if (!ns) return '0s';
            const seconds = ns / 1000000000;
            if (seconds < 60) return seconds.toFixed(1) + 's';
            const minutes = seconds / 60;
            if (minutes < 60) return minutes.toFixed(1) + 'm';
            const hours = minutes / 60;
            return hours.toFixed(1) + 'h';
        }

        function loadDashboard() {
            loadStats();
            loadWorkflows();
        }

        // Load dashboard on page load
        loadDashboard();

        // Auto-refresh every 30 seconds
        setInterval(loadDashboard, 30000);
    </script>
</body>
</html>
`
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(html))
	})

	// Start HTTP server
	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)

	fmt.Printf("\n🚀 Dashboard server starting on http://localhost%s\n", addr)
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("  GET  /                          - Dashboard UI")
	fmt.Println("  GET  /api/stats                 - Workflow statistics")
	fmt.Println("  GET  /api/workflows             - List all workflows")
	fmt.Println("  GET  /api/workflows/running     - List running workflows")
	fmt.Println("  GET  /api/workflows/failed      - List failed workflows")
	fmt.Println("  GET  /api/workflows/recent      - List recent workflows")
	fmt.Println("  GET  /api/workflows/{id}        - Get workflow details")
	fmt.Println("  POST /api/workflows/cancel/{id} - Cancel a workflow")
	fmt.Println()

	// Run CLI demo before starting server (optional)
	if len(os.Args) > 1 && os.Args[1] == "--cli-demo" {
		runCLIDemo(wm)
		return
	}

	// Start server
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

func runCLIDemo(wm *temporal.WorkflowManager) {
	ctx := context.Background()

	fmt.Println("=== CLI Demo Mode ===")

	// Get dashboard stats
	fmt.Println("📊 Dashboard Statistics:")
	stats, err := wm.GetDashboardStats(ctx)
	if err != nil {
		log.Printf("Failed to get dashboard stats: %v", err)
	} else {
		fmt.Printf("  Running:    %d\n", stats.TotalRunning)
		fmt.Printf("  Completed:  %d\n", stats.TotalCompleted)
		fmt.Printf("  Failed:     %d\n", stats.TotalFailed)
		fmt.Printf("  Canceled:   %d\n", stats.TotalCanceled)
		fmt.Printf("  Terminated: %d\n", stats.TotalTerminated)
		fmt.Printf("  Avg Duration: %v\n\n", stats.AverageDuration)
	}

	// List running workflows
	fmt.Println("🏃 Running Workflows:")
	runningWorkflows, err := wm.ListRunningWorkflows(ctx, 10)
	if err != nil {
		log.Printf("Failed to list running workflows: %v", err)
	} else {
		if len(runningWorkflows) == 0 {
			fmt.Println("  No running workflows")
		} else {
			for _, wf := range runningWorkflows {
				fmt.Printf("  • %s (%s) - Started: %s\n",
					wf.WorkflowID,
					wf.WorkflowType,
					wf.StartTime.Format(time.RFC3339))
			}
		}
	}
	fmt.Println()

	// List recent workflows
	fmt.Println("📋 Recent Workflows (last 10):")
	recentWorkflows, err := wm.GetRecentWorkflows(ctx, 10)
	if err != nil {
		log.Printf("Failed to get recent workflows: %v", err)
	} else {
		if len(recentWorkflows) == 0 {
			fmt.Println("  No workflows found")
		} else {
			for _, wf := range recentWorkflows {
				statusIcon := getStatusIcon(wf.Status)
				fmt.Printf("  %s %s (%s)\n",
					statusIcon,
					wf.WorkflowID,
					wf.WorkflowType)
				fmt.Printf("     Status: %s | Duration: %v\n",
					wf.Status,
					wf.ExecutionTime)
			}
		}
	}
	fmt.Println()

	// List failed workflows
	fmt.Println("❌ Failed Workflows:")
	failedWorkflows, err := wm.ListFailedWorkflows(ctx, 5)
	if err != nil {
		log.Printf("Failed to list failed workflows: %v", err)
	} else {
		if len(failedWorkflows) == 0 {
			fmt.Println("  No failed workflows")
		} else {
			for _, wf := range failedWorkflows {
				fmt.Printf("  • %s (%s) - Failed at: %s\n",
					wf.WorkflowID,
					wf.WorkflowType,
					wf.CloseTime.Format(time.RFC3339))
			}
		}
	}
}

func getStatusIcon(status enums.WorkflowExecutionStatus) string {
	switch status {
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
		return "🏃"
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return "✅"
	case enums.WORKFLOW_EXECUTION_STATUS_FAILED:
		return "❌"
	case enums.WORKFLOW_EXECUTION_STATUS_CANCELED:
		return "🚫"
	case enums.WORKFLOW_EXECUTION_STATUS_TERMINATED:
		return "⚠️"
	default:
		return "❓"
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
