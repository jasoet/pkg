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
