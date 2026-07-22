package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Option customizes the viper configuration used during loading.
// Consumers use the provided With* constructors and never need to name viper.
type Option func(*viper.Viper)

// WithEnvPrefix sets the environment variable prefix used for lookups.
func WithEnvPrefix(prefix string) Option {
	return func(v *viper.Viper) {
		v.SetEnvPrefix(prefix)
	}
}

// WithDefaults sets default values for the given keys.
func WithDefaults(defaults map[string]any) Option {
	return func(v *viper.Viper) {
		for key, value := range defaults {
			v.SetDefault(key, value)
		}
	}
}

// WithNestedEnvVars processes environment variables with the given prefix and sets
// them under configPath, filling only keys absent from the loaded configuration.
// Parameters:
//   - prefix: The prefix of environment variables to process (e.g. "MY_APP_").
//   - keyDepth: The zero-based index into the underscore-split key after the prefix
//     has been removed, at which the entity name token is located. For example, given
//     the env var MY_APP_USER_NAME with prefix "MY_APP_", the remaining parts are
//     ["USER", "NAME"]. With keyDepth=0, "USER" is treated as the entity name and
//     "NAME" becomes the field name.
//   - configPath: The base path in the configuration where values should be set.
func WithNestedEnvVars(prefix string, keyDepth int, configPath string) Option {
	return func(v *viper.Viper) {
		nestedEnvVars(prefix, keyDepth, configPath, v)
	}
}

// LoadString loads configuration from a string with optional environment variable support.
// Parameters:
//   - configString: The configuration string in YAML format
//   - envPrefix: Optional environment variable prefix (default: "ENV"). Only the first value
//     is used; any additional values are ignored.
func LoadString[T any](configString string, envPrefix ...string) (*T, error) {
	return loadString[T](configString, nil, envPrefix...)
}

// LoadStringWithOptions loads configuration from a string with optional environment variable
// support and applies the given options before unmarshaling.
// Options run after the YAML has been parsed; options that must precede parsing are not supported.
// Parameters:
//   - configString: The configuration string in YAML format
//   - opts: Options to customize the viper configuration before unmarshaling
func LoadStringWithOptions[T any](configString string, opts ...Option) (*T, error) {
	return loadString[T](configString, opts)
}

func loadString[T any](configString string, opts []Option, envPrefix ...string) (*T, error) {
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

	// Apply options after parsing, before unmarshaling
	for _, opt := range opts {
		opt(viperConfig)
	}

	var config T

	err = viperConfig.Unmarshal(&config)
	if err != nil {
		return nil, fmt.Errorf("config: failed to unmarshal into %T: %w", config, err)
	}
	return &config, nil
}

// nestedEnvVars processes environment variables with a specific prefix and sets them in the viper configuration.
// This function is useful for handling nested configuration structures from environment variables.
// Parameters:
//   - prefix: The prefix for environment variables to process (e.g. "MY_APP_")
//   - keyDepth: The zero-based index into the underscore-split key after the prefix has been
//     removed, at which the entity name token is located. For example, given the env var
//     MY_APP_USER_NAME with prefix "MY_APP_", the remaining parts are ["USER", "NAME"].
//     With keyDepth=0, "USER" is treated as the entity name and "NAME" becomes the field name.
//   - configPath: The base path in the configuration where values should be set
//   - viperConfig: The viper configuration instance to modify
//
// NOTE: This function is NOT goroutine-safe when called with a shared *viper.Viper instance.
// Concurrent calls sharing the same viperConfig must be protected by an external mutex.
func nestedEnvVars(prefix string, keyDepth int, configPath string, viperConfig *viper.Viper) {
	if keyDepth < 0 {
		return
	}

	collected := make(map[string]map[string]string)

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				envKey := parts[0]
				envValue := parts[1]

				envKey = strings.TrimPrefix(envKey, prefix)
				envKey = strings.TrimPrefix(envKey, "_")

				keyParts := strings.Split(envKey, "_")
				if len(keyParts) >= keyDepth+2 { // +2 for the entity name and field
					entityName := strings.ToLower(keyParts[keyDepth])
					fieldName := strings.ToLower(strings.Join(keyParts[keyDepth+1:], "_"))

					if _, ok := collected[entityName]; !ok {
						collected[entityName] = make(map[string]string)
					}
					collected[entityName][fieldName] = envValue
				}
			}
		}
	}

	for entityName, fields := range collected {
		entityKey := configPath + "." + entityName

		for fieldName, fieldValue := range fields {
			fieldKey := entityKey + "." + fieldName
			if !viperConfig.IsSet(fieldKey) {
				viperConfig.Set(fieldKey, fieldValue)
			}
		}
	}
}
