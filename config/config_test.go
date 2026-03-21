package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
)

// TestConfig is a sample configuration struct for testing
type TestConfig struct {
	Name    string `yaml:"name" mapstructure:"name"`
	Version string `yaml:"version" mapstructure:"version"`
	Nested  struct {
		Value int `yaml:"value" mapstructure:"value"`
	} `yaml:"nested" mapstructure:"nested"`
}

// StringSliceConfig is a configuration struct with a string slice field
type StringSliceConfig struct {
	Name     string   `yaml:"name" mapstructure:"name"`
	Tags     []string `yaml:"tags" mapstructure:"tags"`
	Features []string `yaml:"features" mapstructure:"features"`
}

func TestLoadString(t *testing.T) {
	// Test basic configuration loading
	yamlConfig := `
name: test-app
version: 1.0.0
nested:
  value: 42
`
	config, err := LoadString[TestConfig](yamlConfig)
	assert.NoError(t, err)
	assert.Equal(t, "test-app", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, 42, config.Nested.Value)

	// Test with environment variables
	t.Setenv("ENV_NAME", "env-app")

	config, err = LoadString[TestConfig](yamlConfig)
	assert.NoError(t, err)
	assert.Equal(t, "env-app", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, 42, config.Nested.Value)

	// Test with custom environment prefix
	t.Setenv("CUSTOM_NAME", "custom-app")

	config, err = LoadString[TestConfig](yamlConfig, "CUSTOM")
	assert.NoError(t, err)
	assert.Equal(t, "custom-app", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, 42, config.Nested.Value)
}

func TestLoadStringWithConfig(t *testing.T) {
	// Test with custom configuration function
	yamlConfig := `
name: test-app
version: 1.0.0
nested:
  value: 42
`
	customConfigFn := func(v *viper.Viper) {
		v.Set("name", "custom-function-app")
		v.Set("nested.value", 100)
	}

	config, err := LoadStringWithConfig[TestConfig](yamlConfig, customConfigFn)
	assert.NoError(t, err)
	assert.Equal(t, "custom-function-app", config.Name)
	assert.Equal(t, "1.0.0", config.Version)
	assert.Equal(t, 100, config.Nested.Value)

	// Test with NestedEnvVars
	t.Setenv("TEST_GOERS_ACCOUNTS_USER_NAME", "test-user")

	nestedConfigFn := func(v *viper.Viper) {
		NestedEnvVars("TEST_GOERS_ACCOUNTS_", 3, "goers.accounts", v)
	}

	type NestedConfig struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Goers   struct {
			Accounts map[string]map[string]string `yaml:"accounts"`
		} `yaml:"goers"`
	}

	nestedConfig, err := LoadStringWithConfig[NestedConfig](yamlConfig, nestedConfigFn)
	assert.NoError(t, err)
	assert.Equal(t, "test-app", nestedConfig.Name)
	assert.Equal(t, "1.0.0", nestedConfig.Version)
	assert.Equal(t, "test-user", nestedConfig.Goers.Accounts["user"]["name"])
}

func TestNestedEnvVars(t *testing.T) {
	// Setup test environment variables
	t.Setenv("TEST_APP_USER_NAME", "john")
	t.Setenv("TEST_APP_USER_EMAIL", "john@example.com")
	t.Setenv("TEST_APP_ADMIN_NAME", "admin")
	t.Setenv("TEST_APP_ADMIN_EMAIL", "admin@example.com")

	// Create viper instance
	v := viper.New()

	// Call NestedEnvVars
	NestedEnvVars("TEST_APP_", 2, "app", v)

	// Verify the values were set correctly
	assert.Equal(t, "john", v.GetString("app.user.name"))
	assert.Equal(t, "john@example.com", v.GetString("app.user.email"))
	assert.Equal(t, "admin", v.GetString("app.admin.name"))
	assert.Equal(t, "admin@example.com", v.GetString("app.admin.email"))
}

func TestNestedEnvVars_NegativeKeyDepth(t *testing.T) {
	v := viper.New()
	t.Setenv("TEST_APP_DB_HOST", "localhost")
	assert.NotPanics(t, func() {
		NestedEnvVars("TEST_APP_", -1, "app", v)
	})
}

func TestNestedEnvVars_MultiSegmentFieldName(t *testing.T) {
	v := viper.New()
	t.Setenv("TEST_APP_DB_CONNECTION_TIMEOUT", "30")
	NestedEnvVars("TEST_APP_", 2, "app", v)
	assert.Equal(t, "30", v.GetString("app.db.connection_timeout"))
}

func TestStringSliceConfig(t *testing.T) {
	// Test basic string slice configuration loading
	yamlConfig := `
name: slice-app
tags:
  - tag1
  - tag2
  - tag3
features:
  - feature1
  - feature2
`
	config, err := LoadString[StringSliceConfig](yamlConfig)
	assert.NoError(t, err)
	assert.Equal(t, "slice-app", config.Name)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, config.Tags)
	assert.Equal(t, []string{"feature1", "feature2"}, config.Features)

	// Test with environment variables overriding string slices
	t.Setenv("ENV_TAGS", "env-tag1,env-tag2,env-tag3")

	config, err = LoadStringWithConfig[StringSliceConfig](yamlConfig, nil)
	assert.NoError(t, err)
	assert.Equal(t, "slice-app", config.Name)
	assert.Equal(t, []string{"env-tag1", "env-tag2", "env-tag3"}, config.Tags)
	assert.Equal(t, []string{"feature1", "feature2"}, config.Features)

	// Test with custom environment prefix
	t.Setenv("CUSTOM_FEATURES", "custom-feature1,custom-feature2,custom-feature3")
	t.Setenv("CUSTOM_TAGS", "custom-tag1,custom-tag2,custom-tag3")

	// Use LoadStringWithConfig directly with custom prefix
	config, err = LoadStringWithConfig[StringSliceConfig](yamlConfig, nil, "CUSTOM")
	assert.NoError(t, err)
	assert.Equal(t, "slice-app", config.Name)
	assert.Equal(t, []string{"custom-tag1", "custom-tag2", "custom-tag3"}, config.Tags)
	assert.Equal(t, []string{"custom-feature1", "custom-feature2", "custom-feature3"}, config.Features)
}

func TestLoadString_InvalidYAML(t *testing.T) {
	_, err := LoadString[TestConfig]("not: valid: yaml: [")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse YAML")
}
