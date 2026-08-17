package config_test

import (
	"fmt"
	"os"

	"github.com/jasoet/pkg/v3/config"
)

// Basic loading from a YAML string, plus the default ENV_ override mechanism.
// Examples have no *testing.T, so environment variables are managed with
// os.Setenv/os.Unsetenv directly.
func ExampleLoadString() {
	type AppConfig struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Server  struct {
			Host string `yaml:"host"`
			Port int    `yaml:"port"`
		} `yaml:"server"`
	}

	yamlConfig := `
name: my-app
version: 1.0.0
server:
  host: localhost
  port: 8080
`

	// Dots become underscores: server.port is overridden by ENV_SERVER_PORT.
	os.Setenv("ENV_SERVER_PORT", "9090")
	defer os.Unsetenv("ENV_SERVER_PORT")

	cfg, err := config.LoadString[AppConfig](yamlConfig)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("%s v%s on %s:%d\n", cfg.Name, cfg.Version, cfg.Server.Host, cfg.Server.Port)

	// Output:
	// my-app v1.0.0 on localhost:9090
}

// Loading with functional options: defaults for missing keys and a custom
// environment variable prefix.
func ExampleLoadStringWithOptions() {
	type AppConfig struct {
		Debug  bool `yaml:"debug"`
		Server struct {
			Port int `yaml:"port"`
		} `yaml:"server"`
	}

	os.Setenv("APP_SERVER_PORT", "9090")
	defer os.Unsetenv("APP_SERVER_PORT")

	cfg, err := config.LoadStringWithOptions[AppConfig](`server: {port: 8080}`,
		config.WithDefaults(map[string]any{"debug": true}),
		config.WithEnvPrefix("APP"),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Printf("debug=%v port=%d\n", cfg.Debug, cfg.Server.Port)

	// Output:
	// debug=true port=9090
}

// WithNestedEnvVars maps prefixed environment variables onto a map-typed
// config section. The prefix is stripped first, then keyDepth indexes the
// remaining underscore-split tokens: with prefix "APP" and keyDepth 1,
// APP_USERS_ADMIN_NAME yields entity "admin" with field "name" under the
// "users" config path.
//
// Precedence contract: nested env vars fill only keys absent from the YAML.
// Here users.admin.name comes from YAML, so APP_USERS_ADMIN_NAME is ignored,
// while users.admin.email is YAML-absent and filled from the environment.
func ExampleWithNestedEnvVars() {
	type Config struct {
		Users map[string]map[string]string `yaml:"users"`
	}

	os.Setenv("APP_USERS_ADMIN_NAME", "alice")
	os.Setenv("APP_USERS_ADMIN_EMAIL", "alice@example.com")
	defer os.Unsetenv("APP_USERS_ADMIN_NAME")
	defer os.Unsetenv("APP_USERS_ADMIN_EMAIL")

	cfg, err := config.LoadStringWithOptions[Config](`users: {admin: {name: bob}}`,
		config.WithNestedEnvVars("APP", 1, "users"),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println(cfg.Users["admin"]["name"])
	fmt.Println(cfg.Users["admin"]["email"])

	// Output:
	// bob
	// alice@example.com
}
