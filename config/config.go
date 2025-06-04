package config

import (
	"github.com/spf13/viper"
	"os"
	"strings"
)

// LoadString loads configuration from a string with optional environment variable support.
// Parameters:
// - configString: The configuration string in YAML format
// - envPrefix: Optional environment variable prefix (default: "ENV")
// - configFn: Optional function to customize viper configuration before unmarshaling
func LoadString[T any](configString string, envPrefix ...string) (*T, error) {
	// For backward compatibility
	return LoadStringWithConfig[T](configString, nil, envPrefix...)
}

// LoadStringWithConfig loads configuration from a string with optional environment variable support
// and allows custom configuration of viper.
// Parameters:
// - configString: The configuration string in YAML format
// - configFn: Optional function to customize viper configuration before unmarshaling
// - envPrefix: Optional environment variable prefix (default: "ENV")
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
		return nil, err
	}

	// Apply custom configuration if provided
	if configFn != nil {
		configFn(viperConfig)
	}

	var config T

	err = viperConfig.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// NestedEnvVars processes environment variables with a specific prefix and sets them in the viper configuration.
// This function is useful for handling nested configuration structures from environment variables.
// Parameters:
// - prefix: The prefix for environment variables to process
// - keyDepth: The depth at which entity names are found in the key parts
// - configPath: The base path in the configuration where values should be set
// - viperConfig: The viper configuration instance to modify
func NestedEnvVars(prefix string, keyDepth int, configPath string, viperConfig *viper.Viper) {
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
					fieldName := strings.ToLower(keyParts[keyDepth+1])

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

		if !viperConfig.IsSet(entityKey) {
			for fieldName, fieldValue := range fields {
				viperConfig.Set(entityKey+"."+fieldName, fieldValue)
			}
		}
	}
}
