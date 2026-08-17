package docker_test

import (
	"context"
	"fmt"
	"time"

	"github.com/jasoet/pkg/v3/docker"
)

// New assembles an executor from functional options. Constructing the
// executor only validates configuration and creates a Docker client handle;
// no container is started until Start is called.
func ExampleNew() {
	exec, err := docker.New(
		docker.WithImage("nginx:latest"),
		docker.WithPorts("80:0"), // host port auto-assigned
		docker.WithEnv("ENV=production"),
		docker.WithAutoRemove(true),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = exec.Close() }()

	fmt.Println("executor created, container started:", exec.ContainerID() != "")

	// Output: executor created, container started: false
}

// NewFromRequest is sugar over New(WithRequest(req), ...): it builds an
// executor from a ContainerRequest struct and lets trailing options override
// individual struct fields.
func ExampleNewFromRequest() {
	req := docker.ContainerRequest{
		Image:        "postgres:18-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_PASSWORD": "secret",
		},
		WaitingFor: docker.WaitForLog("ready to accept connections").
			WithStartupTimeout(60 * time.Second),
	}

	// WithName overrides the (empty) struct field; options always win.
	exec, err := docker.NewFromRequest(req, docker.WithName("my-postgres"))
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = exec.Close() }()

	fmt.Println("executor created from request")

	// Output: executor created from request
}

// WaitForLog creates a strategy that blocks Start until a line matching the
// pattern appears in the container logs. The pattern is a regular expression;
// plain strings work because they are valid regexes.
//
// Output is non-deterministic; compile-checked only.
func ExampleWaitForLog() {
	strategy := docker.WaitForLog("database system is ready to accept connections").
		WithStartupTimeout(60 * time.Second)

	exec, err := docker.New(
		docker.WithImage("postgres:18-alpine"),
		docker.WithEnvMap(map[string]string{
			"POSTGRES_PASSWORD": "secret",
		}),
		docker.WithWaitStrategy(strategy),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = exec.Close() }()

	// Start blocks until the log pattern matches (requires a Docker daemon):
	//
	//	ctx := context.Background()
	//	if err := exec.Start(ctx); err != nil { ... }
	//	defer exec.Terminate(ctx)
	_ = context.Background()
}

// WaitForFunc wraps arbitrary readiness logic. The strategy receives a
// ContainerTarget exposing the container ID, a log stream, and a projected
// runtime state — no Docker client import required.
//
// Output is non-deterministic; compile-checked only.
func ExampleWaitForFunc() {
	strategy := docker.WaitForFunc(func(ctx context.Context, target docker.ContainerTarget) error {
		state, err := target.State(ctx)
		if err != nil {
			return err
		}
		if !state.Running {
			return fmt.Errorf("container %s not running", target.ID())
		}
		return nil
	}).WithStartupTimeout(30 * time.Second)

	exec, err := docker.New(
		docker.WithImage("redis:7-alpine"),
		docker.WithWaitStrategy(strategy),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer func() { _ = exec.Close() }()
}
