//go:build example

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jasoet/pkg/v2/docker"
)

func main() {
	ctx := context.Background()

	// Run a multi-container example: nginx + redis
	multiContainerExample(ctx)
}

func multiContainerExample(ctx context.Context) {
	fmt.Println("=== Multi-Container Example: Nginx + Redis ===\n")

	// Create nginx container
	fmt.Println("Creating nginx container...")
	nginx, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:0"), // Random port
		docker.WithName("example-multi-nginx"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("start worker processes").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create nginx executor: %v", err)
	}

	// Create redis container
	fmt.Println("Creating redis container...")
	redis, err := docker.New(
		docker.WithImage("redis:7-alpine"),
		docker.WithPorts("6379:0"), // Random port
		docker.WithName("example-multi-redis"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("Ready to accept connections").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create redis executor: %v", err)
	}

	// Start both containers
	fmt.Println("\nStarting containers...")
	fmt.Println("  - Starting nginx...")
	if err := nginx.Start(ctx); err != nil {
		log.Fatalf("Failed to start nginx: %v", err)
	}
	defer func() {
		fmt.Println("\nStopping nginx...")
		nginx.Terminate(ctx)
	}()

	fmt.Println("  - Starting redis...")
	if err := redis.Start(ctx); err != nil {
		log.Fatalf("Failed to start redis: %v", err)
	}
	defer func() {
		fmt.Println("Stopping redis...")
		redis.Terminate(ctx)
	}()

	fmt.Println("\n✓ Both containers are running!\n")

	// Display container information
	displayContainerInfo(ctx, "Nginx", nginx)
	fmt.Println()
	displayContainerInfo(ctx, "Redis", redis)

	// Test nginx
	fmt.Println("\nTesting nginx HTTP endpoint...")
	nginxEndpoint, _ := nginx.Endpoint(ctx, "80/tcp")
	resp, err := http.Get(fmt.Sprintf("http://%s", nginxEndpoint))
	if err != nil {
		log.Printf("Failed to connect to nginx: %v", err)
	} else {
		defer resp.Body.Close()
		fmt.Printf("  HTTP Status: %s ✓\n", resp.Status)
	}

	// Monitor containers
	fmt.Println("\nMonitoring containers for 5 seconds...")
	monitorContainers(ctx, nginx, redis, 5*time.Second)

	// Get final status
	fmt.Println("\nFinal Status:")
	nginxStatus, _ := nginx.Status(ctx)
	redisStatus, _ := redis.Status(ctx)

	fmt.Printf("  Nginx: %s (uptime: %s)\n",
		nginxStatus.State,
		time.Since(nginxStatus.StartedAt).Round(time.Second))

	fmt.Printf("  Redis: %s (uptime: %s)\n",
		redisStatus.State,
		time.Since(redisStatus.StartedAt).Round(time.Second))

	// Show recent logs
	fmt.Println("\nNginx logs (last 5 lines):")
	nginxLogs, _ := nginx.GetLastNLines(ctx, 5)
	fmt.Println(nginxLogs)

	fmt.Println("Redis logs (last 5 lines):")
	redisLogs, _ := redis.GetLastNLines(ctx, 5)
	fmt.Println(redisLogs)

	fmt.Println("\nExample completed successfully! ✓")
}

func displayContainerInfo(ctx context.Context, name string, exec *docker.Executor) {
	status, err := exec.Status(ctx)
	if err != nil {
		log.Printf("Failed to get %s status: %v", name, err)
		return
	}

	endpoint, err := exec.Endpoint(ctx, "")
	if err != nil {
		// Try to get first port
		ports, _ := exec.GetAllPorts(ctx)
		if len(ports) > 0 {
			for containerPort, hostPort := range ports {
				host, _ := exec.Host(ctx)
				endpoint = fmt.Sprintf("%s:%s (container port: %s)", host, hostPort, containerPort)
				break
			}
		}
	}

	networks, _ := exec.GetNetworks(ctx)
	ip, _ := exec.GetIPAddress(ctx, "")

	fmt.Printf("Container: %s\n", name)
	fmt.Printf("  ID: %s\n", status.ID[:12])
	fmt.Printf("  Name: %s\n", status.Name)
	fmt.Printf("  Image: %s\n", status.Image)
	fmt.Printf("  State: %s\n", status.State)
	fmt.Printf("  Running: %v\n", status.Running)
	if endpoint != "" {
		fmt.Printf("  Endpoint: %s\n", endpoint)
	}
	fmt.Printf("  Networks: %v\n", networks)
	if ip != "" {
		fmt.Printf("  IP Address: %s\n", ip)
	}
	fmt.Printf("  Started: %s\n", status.StartedAt.Format(time.RFC3339))
}

func monitorContainers(ctx context.Context, nginx, redis *docker.Executor, duration time.Duration) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.After(duration)
	startTime := time.Now()

	for {
		select {
		case <-ticker.C:
			elapsed := time.Since(startTime).Round(time.Second)

			nginxRunning, _ := nginx.IsRunning(ctx)
			redisRunning, _ := redis.IsRunning(ctx)

			fmt.Printf("  [%s] Nginx: %v, Redis: %v\n",
				elapsed,
				formatRunningStatus(nginxRunning),
				formatRunningStatus(redisRunning))

		case <-timeout:
			return
		}
	}
}

func formatRunningStatus(running bool) string {
	if running {
		return "✓ running"
	}
	return "✗ stopped"
}
