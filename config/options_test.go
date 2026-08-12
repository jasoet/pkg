package config_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jasoet/pkg/v3/config"
)

type appCfg struct {
	Debug  bool                         `yaml:"debug"`
	Server struct{ Port int }           `yaml:"server"`
	Users  map[string]map[string]string `yaml:"users"`
}

func TestLoadStringWithOptions_DefaultsAndPrefix(t *testing.T) {
	cfg, err := config.LoadStringWithOptions[appCfg](`server: {port: 8080}`,
		config.WithDefaults(map[string]any{"debug": true}),
		config.WithEnvPrefix("APP"),
	)
	require.NoError(t, err)
	assert.True(t, cfg.Debug)
	assert.Equal(t, 8080, cfg.Server.Port)
}

func TestLoadStringWithOptions_NestedEnvVars(t *testing.T) {
	t.Setenv("APP_USERS_ADMIN_NAME", "alice")
	cfg, err := config.LoadStringWithOptions[appCfg](``,
		config.WithNestedEnvVars("APP", 1, "users"),
	)
	require.NoError(t, err)
	assert.Equal(t, "alice", cfg.Users["admin"]["name"])
}

func TestLoadStringWithOptions_NestedDoesNotOverrideYAML(t *testing.T) {
	// Precedence contract: nested env vars fill only keys absent from YAML.
	t.Setenv("APP_USERS_ADMIN_NAME", "alice")
	cfg, err := config.LoadStringWithOptions[appCfg](`users: {admin: {name: bob}}`,
		config.WithNestedEnvVars("APP", 1, "users"),
	)
	require.NoError(t, err)
	assert.Equal(t, "bob", cfg.Users["admin"]["name"])
}

func TestLoadStringWithOptions_NestedPrefixIsAnchored(t *testing.T) {
	// The prefix must match at an underscore boundary: "APP" must not capture
	// "APPLE_*" env vars into the config.
	t.Setenv("APP_USERS_ADMIN_NAME", "alice")
	t.Setenv("APPLE_USERS_HACKER_NAME", "mallory")

	cfg, err := config.LoadStringWithOptions[appCfg](``,
		config.WithNestedEnvVars("APP", 1, "users"),
	)
	require.NoError(t, err)
	assert.Equal(t, "alice", cfg.Users["admin"]["name"])
	_, leaked := cfg.Users["hacker"]
	assert.False(t, leaked, "APPLE_* must not be captured by prefix APP")
}

func TestLoadStringWithOptions_NestedEmptyPrefixRejected(t *testing.T) {
	// An empty prefix must capture nothing rather than the whole environment.
	t.Setenv("SOME_UNRELATED_ENV_VAR", "value")

	cfg, err := config.LoadStringWithOptions[appCfg](``,
		config.WithNestedEnvVars("", 0, "users"),
	)
	require.NoError(t, err)
	assert.Empty(t, cfg.Users, "empty prefix must capture no environment variables")
}

func TestLoadStringWithOptions_NestedEnvBeatsDefault(t *testing.T) {
	// env > default precedence: a nested env var must win over a WithDefaults
	// value for the same key (the fill guard checks YAML presence, not defaults).
	t.Setenv("APP_USERS_ADMIN_NAME", "alice")

	cfg, err := config.LoadStringWithOptions[appCfg](``,
		config.WithDefaults(map[string]any{"users.admin.name": "default-name"}),
		config.WithNestedEnvVars("APP", 1, "users"),
	)
	require.NoError(t, err)
	assert.Equal(t, "alice", cfg.Users["admin"]["name"], "nested env var must beat WithDefaults")
}
