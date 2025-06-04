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

// Test runs all unit tests in the project
// Uses -count=1 flag to disable test caching
func Test() error {
	fmt.Println("Running tests...")
	cmd := exec.Command("go", "test", "-count=1", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// IntegrationTest runs integration tests with the integration tag
// Starts Docker services before running tests and sets AUTOMATION=true environment variable
// Waits for PostgreSQL to initialize before running tests
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

// Lint runs golangci-lint on the project
// Installs golangci-lint if it's not already installed
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

// Docker namespace for Docker-related commands
type Docker mg.Namespace

// Up starts all Docker Compose services in detached mode
func (d Docker) Up() error {
	fmt.Println("Starting Docker Compose services...")
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = "compose"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Down stops all Docker Compose services and removes volumes
func (d Docker) Down() error {
	fmt.Println("Stopping Docker Compose services...")
	cmd := exec.Command("docker", "compose", "down", "-v")
	cmd.Dir = "compose"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Logs shows Docker Compose logs in follow mode
func (d Docker) Logs() error {
	fmt.Println("Showing Docker Compose logs...")
	cmd := exec.Command("docker", "compose", "logs", "-f")
	cmd.Dir = "compose"
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Restart stops and then starts all Docker Compose services
// This is equivalent to running Down() followed by Up()
func (d Docker) Restart() error {
	fmt.Println("Restarting Docker Compose services...")
	if err := d.Down(); err != nil {
		return err
	}
	return d.Up()
}

// Clean removes the dist directory and all build artifacts
func Clean() error {
	fmt.Println("Cleaning...")
	if err := os.RemoveAll("dist"); err != nil {
		return fmt.Errorf("failed to remove dist directory: %w", err)
	}
	return nil
}

// ensureToolInstalled checks if a tool is installed and installs it if not found
// toolName is the command to look for in PATH
// installPackage is the Go package to install if the tool is not found
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

// getEnvOrDefault retrieves an environment variable value or returns a default if not set or empty
// key is the name of the environment variable to retrieve
// defaultValue is the value to return if the environment variable is not set or empty
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// loadEnvFromFile loads environment variables from a file and returns the keys that were set
// envFile is the path to the file containing environment variables in KEY=VALUE format
// Returns a slice of keys that were set and any error encountered
// If the file doesn't exist, it continues without error and returns an empty slice
// Comments (lines starting with #) and empty lines are ignored
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

// cleanupEnv unsets the environment variables with the given keys
// keys is a slice of environment variable names to unset
// Does nothing if the keys slice is empty
// This function is typically used with defer to clean up environment variables set by loadEnvFromFile
func cleanupEnv(keys []string) {
	if len(keys) > 0 {
		fmt.Println("Cleaning up environment variables...")
		for _, key := range keys {
			_ = os.Unsetenv(key)
			fmt.Printf("Unset environment variable: %s\n", key)
		}
	}
}
