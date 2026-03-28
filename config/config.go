package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// LoadString loads configuration from a string with optional environment variable support.
// Parameters:
//   - configString: The configuration string in YAML format
//   - envPrefix: Optional environment variable prefix (default: "ENV"). Only the first value
//     is used; any additional values are ignored.
func LoadString[T any](configString string, envPrefix ...string) (*T, error) {
	// For backward compatibility
	return LoadStringWithConfig[T](configString, nil, envPrefix...)
}

// LoadStringWithConfig loads configuration from a string with optional environment variable support
// and allows custom configuration of viper.
// Parameters:
//   - configString: The configuration string in YAML format
//   - configFn: Optional function to customize viper configuration before unmarshaling
//   - envPrefix: Optional environment variable prefix (default: "ENV"). Only the first value
//     is used; any additional values are ignored.
func LoadStringWithConfig[T any](configString string, configFn func(*viper.Viper), envPrefix ...string) (*T, error) {
	viperConfig := viper.New()

	prefix := "ENV"
	if len(envPrefix) > 0 && envPrefix[0] != "" && strings.TrimSpace(envPrefix[0]) != "" {
		prefix = envPrefix[0]
	}

	viperConfig.SetEnvPrefix(prefix)
	viperConfig.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viperConfig.AutomaticEnv()

	viperConfig.SetConfigType("yaml")
	err := viperConfig.ReadConfig(strings.NewReader(configString))
	if err != nil {
		return nil, fmt.Errorf("config: failed to parse YAML: %w", err)
	}

	// Apply custom configuration if provided
	if configFn != nil {
		configFn(viperConfig)
	}

	var config T

	err = viperConfig.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("config: failed to unmarshal into %T: %w", config, err)
	}
	return &config, nil
}

// NestedEnvVars processes environment variables with a specific prefix and sets them in the viper configuration.
// This function is useful for handling nested configuration structures from environment variables.
// Parameters:
//   - prefix: The prefix for environment variables to process (e.g. "MY_APP_")
//   - keyDepth: The zero-based index into the full underscore-split key (including prefix tokens)
//     at which the entity name token is located. For example, given the env var MY_APP_USER_NAME,
//     the split parts are ["MY", "APP", "USER", "NAME"]. With keyDepth=2, "USER" (index 2) is
//     treated as the entity name and "NAME" becomes the field name.
//   - configPath: The base path in the configuration where values should be set
//   - viperConfig: The viper configuration instance to modify
//
// NOTE: This function is NOT goroutine-safe when called with a shared *viper.Viper instance.
// Concurrent calls sharing the same viperConfig must be protected by an external mutex.
func NestedEnvVars(prefix string, keyDepth int, configPath string, viperConfig *viper.Viper) {
	if keyDepth < 0 {
		return
	}

	nestedEnvVars := make(map[string]map[string]string)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envKey := parts[0]
				envValue := parts[1]

				keyParts := strings.Split(envKey, "_")
				if len(keyParts) >= keyDepth+2 { // +2 for the entity name and field
					entityName := strings.ToLower(keyParts[keyDepth])
					fieldName := strings.ToLower(strings.Join(keyParts[keyDepth+1:], "_"))

					if _, ok := nestedEnvVars[entityName]; !ok {
						nestedEnvVars[entityName] = make(map[string]string)
					}
					nestedEnvVars[entityName][fieldName] = envValue
				}
			}
		}
	}

	for entityName, fields := range nestedEnvVars {
		entityKey := configPath + "." + entityName

		for fieldName, fieldValue := range fields {
			fieldKey := entityKey + "." + fieldName
			if !viperConfig.IsSet(fieldKey) {
				viperConfig.Set(fieldKey, fieldValue)
			}
		}
	}
}
