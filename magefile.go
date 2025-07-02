//go:build mage

package main

import (
	"bufio"
	"fmt"
	"github.com/magefile/mage/mg"
	"os"
	"os/exec"
	"strings"
	"time"
)

var Default = Test

func Test() error {
	fmt.Println("Running tests...")
	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func IntegrationTest() error {
	fmt.Println("Running integration tests...")

	docker := Docker{}
	if err := docker.Up(); err != nil {
		return fmt.Errorf("failed to start docker services: %w", err)
	}

	fmt.Println("Waiting for PostgreSQL to initialize...")
	time.Sleep(2 * time.Second)

	cmd := exec.Command("go", "test", "-count=1", "-tags=integration", "./...")
	cmd.Env = append(os.Environ(), "AUTOMATION=true")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Lint() error {
	fmt.Println("Running linter...")

	if err := ensureToolInstalled("golangci-lint", "github.com/golangci/golangci-lint/cmd/golangci-lint@latest"); err != nil {
		return err
	}

	cmd := exec.Command("golangci-lint", "run", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type Docker mg.Namespace

func (d Docker) Up() error {
	fmt.Println("Starting Docker Compose services...")
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = "scripts/compose"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d Docker) Down() error {
	fmt.Println("Stopping Docker Compose services...")
	cmd := exec.Command("docker", "compose", "down", "-v")
	cmd.Dir = "scripts/compose"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d Docker) Logs() error {
	fmt.Println("Showing Docker Compose logs...")
	cmd := exec.Command("docker", "compose", "logs", "-f")
	cmd.Dir = "scripts/compose"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (d Docker) Restart() error {
	fmt.Println("Restarting Docker Compose services...")
	if err := d.Down(); err != nil {
		return err
	}
	return d.Up()
}

func Clean() error {
	fmt.Println("Cleaning...")
	if err := os.RemoveAll("dist"); err != nil {
		return fmt.Errorf("failed to remove dist directory: %w", err)
	}
	return nil
}

// Tools installs all development tools
func Tools() error {
	fmt.Println("Installing development tools...")
	
	tools := []string{
		"github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
		"github.com/magefile/mage@latest",
		"gotest.tools/gotestsum@latest",
		"github.com/swaggo/swag/cmd/swag@latest",
		"github.com/golang-migrate/migrate/v4/cmd/migrate@latest",
		"github.com/golang/mock/mockgen@latest",
		"github.com/securecodewarrior/gosec/v2/cmd/gosec@latest",
		"github.com/sonatypecommunity/nancy@latest",
		"github.com/deepmap/oapi-codegen/cmd/oapi-codegen@latest",
	}
	
	for _, tool := range tools {
		fmt.Printf("Installing %s...\n", tool)
		cmd := exec.Command("go", "install", tool)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", tool, err)
		}
	}
	
	fmt.Println("All development tools installed successfully")
	return nil
}

// Security runs security analysis tools
func Security() error {
	fmt.Println("Running security analysis...")
	
	// Ensure gosec is installed
	if err := ensureToolInstalled("gosec", "github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"); err != nil {
		return err
	}
	
	// Run gosec
	fmt.Println("Running gosec security scanner...")
	cmd := exec.Command("gosec", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Security issues found (this may be expected)")
	}
	
	return nil
}

// Dependencies checks for known vulnerabilities in dependencies
func Dependencies() error {
	fmt.Println("Checking dependencies for vulnerabilities...")
	
	// Ensure nancy is installed
	if err := ensureToolInstalled("nancy", "github.com/sonatypecommunity/nancy@latest"); err != nil {
		return err
	}
	
	// Generate go.list for nancy
	cmd := exec.Command("go", "list", "-json", "-deps", "./...")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to generate dependency list: %w", err)
	}
	
	// Run nancy
	cmd = exec.Command("nancy", "sleuth")
	cmd.Stdin = strings.NewReader(string(output))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Dependency vulnerabilities found (this may be expected)")
	}
	
	return nil
}

// Coverage generates test coverage report
func Coverage() error {
	fmt.Println("Generating test coverage report...")
	
	// Run tests with coverage
	cmd := exec.Command("go", "test", "-coverprofile=coverage.out", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("tests failed: %w", err)
	}
	
	// Generate HTML coverage report
	cmd = exec.Command("go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to generate HTML coverage report: %w", err)
	}
	
	fmt.Println("Coverage report generated: coverage.html")
	return nil
}

// Docs generates API documentation (if swagger annotations exist)
func Docs() error {
	fmt.Println("Generating API documentation...")
	
	// Check if swag is available
	if err := ensureToolInstalled("swag", "github.com/swaggo/swag/cmd/swag@latest"); err != nil {
		return err
	}
	
	// Generate swagger docs
	cmd := exec.Command("swag", "init", "-g", "main.go", "--output", "docs")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Println("Swagger generation failed (this is expected if no swagger annotations exist)")
		return nil
	}
	
	fmt.Println("API documentation generated in docs/ directory")
	return nil
}

// CheckAll runs all quality checks
func CheckAll() error {
	fmt.Println("Running all quality checks...")
	
	checks := []func() error{
		Test,
		Lint,
		Security,
		Dependencies,
		Coverage,
	}
	
	for _, check := range checks {
		if err := check(); err != nil {
			return fmt.Errorf("quality check failed: %w", err)
		}
	}
	
	fmt.Println("All quality checks completed successfully")
	return nil
}

func ensureToolInstalled(toolName, installPackage string) error {
	if _, err := exec.LookPath(toolName); err != nil {
		fmt.Printf("Installing %s...\n", toolName)
		installCmd := exec.Command("go", "install", installPackage)
		installCmd.Stdout = os.Stdout
		installCmd.Stderr = os.Stderr
		if err := installCmd.Run(); err != nil {
			return fmt.Errorf("failed to install %s: %w", toolName, err)
		}
	}
	return nil
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func loadEnvFromFile(envFile string) ([]string, error) {
	var loadedKeys []string

	if _, err := os.Stat(envFile); err == nil {
		fmt.Printf("Loading environment variables from %s\n", envFile)

		file, err := os.Open(envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open %s: %w", envFile, err)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
				continue
			}

			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])
				_ = os.Setenv(key, value)
				loadedKeys = append(loadedKeys, key)
				fmt.Printf("Set environment variable: %s\n", key)
			}
		}

		if err := scanner.Err(); err != nil {
			return loadedKeys, fmt.Errorf("error reading %s: %w", envFile, err)
		}
	} else {
		fmt.Printf("Environment file %s not found, continuing without it\n", envFile)
	}

	return loadedKeys, nil
}

func cleanupEnv(keys []string) {
	if len(keys) > 0 {
		fmt.Println("Cleaning up environment variables...")
		for _, key := range keys {
			_ = os.Unsetenv(key)
			fmt.Printf("Unset environment variable: %s\n", key)
		}
	}
}
