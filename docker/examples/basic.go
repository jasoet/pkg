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

	// Example 1: Simple nginx container with functional options
	fmt.Println("=== Example 1: Functional Options ===")
	functionalOptionsExample(ctx)

	// Example 2: Same container with struct-based configuration
	fmt.Println("\n=== Example 2: Struct-Based Configuration ===")
	structBasedExample(ctx)

	// Example 3: Hybrid approach
	fmt.Println("\n=== Example 3: Hybrid Approach ===")
	hybridExample(ctx)
}

func functionalOptionsExample(ctx context.Context) {
	// Create executor with functional options
	exec, err := docker.New(
		docker.WithImage("nginx:alpine"),
		docker.WithPorts("80:8080"),
		docker.WithEnv("NGINX_HOST=localhost"),
		docker.WithName("example-nginx-functional"),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForLog("start worker processes").
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Start the container
	fmt.Println("Starting nginx container...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer exec.Terminate(ctx)

	// Get the endpoint
	endpoint, _ := exec.Endpoint(ctx, "80/tcp")
	fmt.Printf("Nginx is running at: http://%s\n", endpoint)

	// Test HTTP request
	resp, err := http.Get(fmt.Sprintf("http://%s", endpoint))
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return
	}
	defer resp.Body.Close()

	fmt.Printf("HTTP Status: %s\n", resp.Status)
	fmt.Println("Container is working! ✓")
}

func structBasedExample(ctx context.Context) {
	// Create executor with ContainerRequest (testcontainers-like)
	req := docker.ContainerRequest{
		Image:        "nginx:alpine",
		ExposedPorts: []string{"80/tcp"},
		Env: map[string]string{
			"NGINX_HOST": "localhost",
		},
		Name:       "example-nginx-struct",
		AutoRemove: true,
		WaitingFor: docker.WaitForLog("start worker processes").
			WithStartupTimeout(30 * time.Second),
	}

	exec, err := docker.NewFromRequest(req)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Start the container
	fmt.Println("Starting nginx container...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer exec.Terminate(ctx)

	// Get status
	status, _ := exec.Status(ctx)
	fmt.Printf("Container State: %s\n", status.State)
	fmt.Printf("Container Running: %v\n", status.Running)

	// Get endpoint
	endpoint, _ := exec.Endpoint(ctx, "80/tcp")
	fmt.Printf("Nginx is running at: http://%s\n", endpoint)

	fmt.Println("Container is working! ✓")
}

func hybridExample(ctx context.Context) {
	// Start with struct for base configuration
	req := docker.ContainerRequest{
		Image:        "nginx:alpine",
		ExposedPorts: []string{"80/tcp"},
	}

	// Add functional options for additional configuration
	exec, err := docker.New(
		docker.WithRequest(req),
		docker.WithName("example-nginx-hybrid"),
		docker.WithEnvMap(map[string]string{
			"NGINX_HOST": "localhost",
			"NGINX_PORT": "80",
		}),
		docker.WithAutoRemove(true),
		docker.WithWaitStrategy(
			docker.WaitForHTTP("80", "/", 200).
				WithStartupTimeout(30*time.Second),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create executor: %v", err)
	}

	// Start the container
	fmt.Println("Starting nginx container with HTTP wait strategy...")
	if err := exec.Start(ctx); err != nil {
		log.Fatalf("Failed to start container: %v", err)
	}
	defer exec.Terminate(ctx)

	// Get all ports
	ports, _ := exec.GetAllPorts(ctx)
	fmt.Printf("Exposed Ports: %v\n", ports)

	// Get host and mapped port separately
	host, _ := exec.Host(ctx)
	port, _ := exec.MappedPort(ctx, "80/tcp")
	fmt.Printf("Accessible at: %s:%s\n", host, port)

	fmt.Println("Container is working! ✓")
}
