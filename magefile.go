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
