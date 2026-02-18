//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jasoet/pkg/v2/docker"
)

func main() {
	ctx := context.Background()

	// Example 1: Get all logs
	fmt.Println("=== Example 1: Get All Logs ===")
	getAllLogsExample(ctx)

	// Example 2: Stream logs in real-time
	fmt.Println("\n=== Example 2: Stream Logs ===")
	streamLogsExample(ctx)

	// Example 3: Follow logs to stdout
	fmt.Println("\n=== Example 3: Follow Logs ===")
	followLogsExample(ctx)

	// Example 4: Log filtering
	fmt.Println("\n=== Example 4: Log Filtering ===")
	logFilteringExample(ctx)
}

func getAllLogsExample(ctx context.Context) {
	// Create a container that generates logs
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "for i in 1 2 3 4 5; do echo 'Message '$i; sleep 1; done"),
		docker.WithName("example-logs-all"),
		docker.WithAutoRemove(true),
	)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	fmt.Println("Starting container...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer exec.Terminate(ctx)

	// Wait for container to finish
	fmt.Println("Waiting for container to finish...")
	exitCode, _ := exec.Wait(ctx)
	fmt.Printf("Container exited with code: %d\n\n", exitCode)

	// Get all logs
	fmt.Println("Container logs:")
	logs, err := exec.Logs(ctx)
	if err != nil {
		log.Fatalf("Failed to get logs: %v", err)
	}
	fmt.Println(logs)
}

func streamLogsExample(ctx context.Context) {
	// Create a long-running container
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "for i in $(seq 1 10); do echo 'Stream line '$i; sleep 1; done"),
		docker.WithName("example-logs-stream"),
		docker.WithAutoRemove(true),
	)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	fmt.Println("Starting container...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer exec.Terminate(ctx)

	// Stream logs in real-time
	fmt.Println("Streaming logs (first 5 lines):")
	logCh, errCh := exec.StreamLogs(ctx, docker.WithFollow())

	lineCount := 0
	timeout := time.After(15 * time.Second)

streamLoop:
	for {
		select {
		case log, ok := <-logCh:
			if !ok {
				break streamLoop
			}
			fmt.Printf("  %s", log.Content)
			lineCount++
			if lineCount >= 5 {
				break streamLoop
			}
		case err := <-errCh:
			if err != nil {
				fmt.Printf("Error streaming logs: %v\n", err)
			}
		case <-timeout:
			break streamLoop
		}
	}
	fmt.Println("\nStream ended")
}

func followLogsExample(ctx context.Context) {
	// Create a container
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", "echo 'Starting...'; for i in 1 2 3; do echo 'Log line '$i; sleep 1; done; echo 'Done!'"),
		docker.WithName("example-logs-follow"),
		docker.WithAutoRemove(true),
	)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	fmt.Println("Starting container...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer exec.Terminate(ctx)

	// Follow logs to stdout
	fmt.Println("Following logs to stdout:")
	fmt.Println("---")

	// Create a context with timeout
	followCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := exec.FollowLogs(followCtx, os.Stdout); err != nil {
		if err != context.DeadlineExceeded {
			log.Printf("Error following logs: %v", err)
		}
	}

	fmt.Println("---")
	fmt.Println("Follow ended")
}

func logFilteringExample(ctx context.Context) {
	// Create a container that writes to both stdout and stderr
	exec, err := docker.New(
		docker.WithImage("alpine:latest"),
		docker.WithCmd("sh", "-c", `
			echo "STDOUT: Message 1"
			echo "STDERR: Error 1" >&2
			echo "STDOUT: Message 2"
			echo "STDERR: Error 2" >&2
			echo "STDOUT: Message 3"
		`),
		docker.WithName("example-logs-filter"),
		docker.WithAutoRemove(true),
	)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	fmt.Println("Starting container...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer exec.Terminate(ctx)

	// Wait for container to finish
	exec.Wait(ctx)

	// Get all logs
	fmt.Println("\n1. All logs (stdout + stderr):")
	allLogs, _ := exec.Logs(ctx,
		docker.WithStdout(true),
		docker.WithStderr(true),
	)
	fmt.Println(allLogs)

	// Get only stdout
	fmt.Println("2. Only stdout:")
	stdout, _ := exec.GetStdout(ctx)
	fmt.Println(stdout)

	// Get only stderr
	fmt.Println("3. Only stderr:")
	stderr, _ := exec.GetStderr(ctx)
	fmt.Println(stderr)

	// Get logs with timestamps
	fmt.Println("4. Logs with timestamps:")
	timestampedLogs, _ := exec.Logs(ctx,
		docker.WithTimestamps(),
	)
	fmt.Println(timestampedLogs)

	// Get last N lines
	fmt.Println("5. Last 2 lines:")
	lastLines, _ := exec.GetLastNLines(ctx, 2)
	fmt.Println(lastLines)

	// Get logs since time
	fmt.Println("6. Logs from last 5 seconds:")
	recentLogs, _ := exec.GetLogsSince(ctx, "5s")
	fmt.Println(recentLogs)

	fmt.Println("\nLog filtering example completed! ✓")
}
