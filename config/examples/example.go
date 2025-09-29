//go:build example

package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/jasoet/pkg/config"
)

// AppConfig is a sample configuration struct
type AppConfig struct {
	Name        string `yaml:"name"`
	Version     string `yaml:"version"`
	Environment string `yaml:"environment"`
	Database    struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"database"`
	Features struct {
		EnableCache bool `yaml:"enableCache"`
		MaxRetries  int  `yaml:"maxRetries"`
	} `yaml:"features"`
	Services map[string]struct {
		URL     string `yaml:"url"`
		Timeout int    `yaml:"timeout"`
		Retries int    `yaml:"retries"`
		Enabled bool   `yaml:"enabled"`
	} `yaml:"services"`
}

// NestedConfig demonstrates nested configuration with environment variables
type NestedConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Goers   struct {
		Accounts map[string]map[string]string `yaml:"accounts"`
	} `yaml:"goers"`
}

func main() {
	// Example 1: Basic configuration loading
	fmt.Println("Example 1: Basic configuration loading")
	yamlConfig := `
name: my-app
version: 1.0.0
environment: development
database:
  host: localhost
  port: 5432
  username: postgres
  password: secret
features:
  enableCache: true
  maxRetries: 3
services:
  auth:
    url: http://auth-service:8080
    timeout: 5000
    retries: 3
    enabled: true
  payment:
    url: http://payment-service:8080
    timeout: 10000
    retries: 5
    enabled: true
`
	appConfig, err := config.LoadString[AppConfig](yamlConfig)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("App Name: %s\n", appConfig.Name)
	fmt.Printf("Version: %s\n", appConfig.Version)
	fmt.Printf("Database Host: %s\n", appConfig.Database.Host)
	fmt.Printf("Auth Service URL: %s\n", appConfig.Services["auth"].URL)
	fmt.Println()

	// Example 2: Using environment variables to override configuration
	fmt.Println("Example 2: Using environment variables to override configuration")
	os.Setenv("ENV_NAME", "env-app")
	os.Setenv("ENV_DATABASE_HOST", "db.example.com")
	os.Setenv("ENV_SERVICES_AUTH_URL", "https://auth.example.com")

	appConfig, err = config.LoadString[AppConfig](yamlConfig)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("App Name (from env): %s\n", appConfig.Name)
	fmt.Printf("Database Host (from env): %s\n", appConfig.Database.Host)
	fmt.Printf("Auth Service URL (from env): %s\n", appConfig.Services["auth"].URL)
	fmt.Println()

	// Example 3: Using custom environment prefix
	fmt.Println("Example 3: Using custom environment prefix")
	os.Setenv("CUSTOM_NAME", "custom-app")
	os.Setenv("CUSTOM_DATABASE_HOST", "custom-db.example.com")

	appConfig, err = config.LoadString[AppConfig](yamlConfig, "CUSTOM")
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("App Name (from custom env): %s\n", appConfig.Name)
	fmt.Printf("Database Host (from custom env): %s\n", appConfig.Database.Host)
	fmt.Println()

	// Example 4: Using custom configuration function
	fmt.Println("Example 4: Using custom configuration function")
	customConfigFn := func(v *viper.Viper) {
		v.Set("name", "custom-function-app")
		v.Set("database.host", "custom-function-db.example.com")
		v.Set("services.payment.enabled", false)
	}

	appConfig, err = config.LoadStringWithConfig[AppConfig](yamlConfig, customConfigFn)
	if err != nil {
		fmt.Printf("Error loading configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("App Name (from custom function): %s\n", appConfig.Name)
	fmt.Printf("Database Host (from custom function): %s\n", appConfig.Database.Host)
	fmt.Printf("Payment Service Enabled: %v\n", appConfig.Services["payment"].Enabled)
	fmt.Println()

	// Example 5: Using NestedEnvVars for complex environment variable handling
	fmt.Println("Example 5: Using NestedEnvVars for complex environment variable handling")
	nestedYamlConfig := `
name: nested-app
version: 1.0.0
goers:
  accounts: {}
`
	// Set up nested environment variables
	os.Setenv("APP_GOERS_ACCOUNTS_USER_NAME", "john")
	os.Setenv("APP_GOERS_ACCOUNTS_USER_EMAIL", "john@example.com")
	os.Setenv("APP_GOERS_ACCOUNTS_ADMIN_NAME", "admin")
	os.Setenv("APP_GOERS_ACCOUNTS_ADMIN_EMAIL", "admin@example.com")

	nestedConfigFn := func(v *viper.Viper) {
		// Process nested environment variables
		nestedEnvPrefix := strings.ToUpper("APP_GOERS_ACCOUNTS_")
		config.NestedEnvVars(nestedEnvPrefix, 3, "goers.accounts", v)
	}

	nestedConfig, err := config.LoadStringWithConfig[NestedConfig](nestedYamlConfig, nestedConfigFn)
	if err != nil {
		fmt.Printf("Error loading nested configuration: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Nested App Name: %s\n", nestedConfig.Name)
	fmt.Printf("User Name: %s\n", nestedConfig.Goers.Accounts["user"]["name"])
	fmt.Printf("User Email: %s\n", nestedConfig.Goers.Accounts["user"]["email"])
	fmt.Printf("Admin Name: %s\n", nestedConfig.Goers.Accounts["admin"]["name"])
	fmt.Printf("Admin Email: %s\n", nestedConfig.Goers.Accounts["admin"]["email"])

	// Clean up environment variables
	os.Unsetenv("ENV_NAME")
	os.Unsetenv("ENV_DATABASE_HOST")
	os.Unsetenv("ENV_SERVICES_AUTH_URL")
	os.Unsetenv("CUSTOM_NAME")
	os.Unsetenv("CUSTOM_DATABASE_HOST")
	os.Unsetenv("APP_GOERS_ACCOUNTS_USER_NAME")
	os.Unsetenv("APP_GOERS_ACCOUNTS_USER_EMAIL")
	os.Unsetenv("APP_GOERS_ACCOUNTS_ADMIN_NAME")
	os.Unsetenv("APP_GOERS_ACCOUNTS_ADMIN_EMAIL")
}
