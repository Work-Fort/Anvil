// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/spf13/viper"
)

// ConfigScope indicates whether to operate on repo or user config
type ConfigScope int

const (
	ScopeRepo ConfigScope = iota // Repo config (./anvil.yaml) - committed to git
	ScopeUser                    // User config (~/.config/anvil/config.yaml) - personal preferences
)

// ConfigValue represents a configuration key-value pair with its source
type ConfigValue struct {
	Key    string
	Value  interface{}
	Source string
}

// getConfigPath returns the config file path based on scope
func getConfigPath(scope ConfigScope) string {
	if scope == ScopeUser {
		return filepath.Join(GlobalPaths.ConfigDir, ConfigFileName+DefaultConfigExt)
	}
	return filepath.Join(".", LocalConfigFile+DefaultConfigExt)
}

// getScopeName returns a human-readable scope name
func getScopeName(scope ConfigScope) string {
	if scope == ScopeUser {
		return "user"
	}
	return "repo"
}

// SetConfigValue sets a configuration value in the specified scope
func SetConfigValue(key, valueStr string, scope ConfigScope) error {
	// Validate key scope
	if err := ValidateKeyScope(key, scope); err != nil {
		return err
	}

	configPath := getConfigPath(scope)

	// Create a new Viper instance for isolated config file operations
	v := viper.New()
	v.SetConfigType(ConfigType)
	v.SetConfigFile(configPath)

	// Read existing config if it exists
	_ = v.ReadInConfig() // Ignore error if file doesn't exist

	// Parse value with smart type detection
	value := parseValue(valueStr)

	// Validate value with scope-specific constraints
	if err := ValidateValue(key, value, scope); err != nil {
		return err
	}

	// Set the value
	v.Set(key, value)

	// Write config using the safe pattern - try SafeWriteConfigAs first
	if err := v.SafeWriteConfigAs(configPath); err != nil {
		// If file already exists, overwrite it
		if _, ok := err.(viper.ConfigFileAlreadyExistsError); ok {
			if err := v.WriteConfigAs(configPath); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}
		} else {
			return fmt.Errorf("failed to create config: %w", err)
		}
	}

	return nil
}

// GetConfigValue retrieves a configuration value and its source
func GetConfigValue(key string) (*ConfigValue, error) {
	// Check if key exists
	if !viper.IsSet(key) {
		return nil, fmt.Errorf("configuration key not found: %s", key)
	}

	// Get value
	value := viper.Get(key)

	// Determine source
	source := getConfigSource(key)

	return &ConfigValue{
		Key:    key,
		Value:  value,
		Source: source,
	}, nil
}

// UnsetConfigValue removes a configuration key from the specified scope
func UnsetConfigValue(key string, scope ConfigScope) error {
	configPath := getConfigPath(scope)
	scopeName := getScopeName(scope)

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("%s config file does not exist: %s", scopeName, configPath)
	}

	// Create a new Viper instance for isolated operations
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType(ConfigType)

	// Read existing config
	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	// Check if key exists in this config file
	if !v.IsSet(key) {
		return fmt.Errorf("key '%s' not found in %s config", key, scopeName)
	}

	// Get all settings and remove the key
	settings := v.AllSettings()
	if err := deleteNestedKey(settings, key); err != nil {
		return err
	}

	// Create a fresh Viper instance to write back
	newV := viper.New()
	newV.SetConfigFile(configPath)
	newV.SetConfigType(ConfigType)

	// Set all settings except the removed key
	for k, val := range settings {
		newV.Set(k, val)
	}

	// Write the updated config
	if err := newV.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// ListConfigValues returns all configuration values with their sources
func ListConfigValues() ([]ConfigValue, error) {
	// Get all settings from Viper
	settings := viper.AllSettings()

	if len(settings) == 0 {
		return []ConfigValue{}, nil
	}

	// Flatten nested settings and collect keys
	keys := flattenKeys(settings, "")

	// Sort keys alphabetically for consistent output
	sort.Strings(keys)

	// Build result slice
	values := make([]ConfigValue, 0, len(keys))
	for _, key := range keys {
		value := viper.Get(key)
		source := getConfigSource(key)
		values = append(values, ConfigValue{
			Key:    key,
			Value:  value,
			Source: source,
		})
	}

	return values, nil
}

// parseValue attempts to parse a string value into its appropriate type
func parseValue(valueStr string) interface{} {
	// Try boolean aliases
	switch strings.ToLower(valueStr) {
	case "true", "yes", "on", "enable", "enabled":
		return true
	case "false", "no", "off", "disable", "disabled":
		return false
	}

	// Try integer
	if i, err := strconv.Atoi(valueStr); err == nil {
		return i
	}

	// Try float
	if f, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return f
	}

	// Default to string
	return valueStr
}

// keyToEnvVar converts a config key to its environment variable name
func keyToEnvVar(key string) string {
	envKey := strings.ToUpper(EnvPrefix + "_" + strings.ReplaceAll(key, "-", "_"))
	envKey = strings.ReplaceAll(envKey, ".", "_")
	return envKey
}

// getConfigSource determines where a config value comes from
func getConfigSource(key string) string {
	// Check if from environment variable
	envKey := keyToEnvVar(key)
	if os.Getenv(envKey) != "" {
		return fmt.Sprintf("from ENV: %s", envKey)
	}

	// Check if from config file
	configFile := viper.ConfigFileUsed()
	if configFile != "" {
		// Determine if local or user config
		if strings.Contains(configFile, LocalConfigFile) {
			return fmt.Sprintf("from ./%s%s", LocalConfigFile, DefaultConfigExt)
		}
		if strings.Contains(configFile, GlobalPaths.ConfigDir) {
			return fmt.Sprintf("from ~/.config/anvil/%s%s", ConfigFileName, DefaultConfigExt)
		}
		return fmt.Sprintf("from %s", configFile)
	}

	// Must be a default value
	return "default"
}

// splitKey splits a dot-notation key into parts
func splitKey(key string) []string {
	result := []string{}
	current := ""
	for _, char := range key {
		if char == '.' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

// deleteNestedKey removes a key from a nested map using dot notation
func deleteNestedKey(m map[string]interface{}, key string) error {
	keys := splitKey(key)

	// Navigate to the parent of the target key
	current := m
	for i := 0; i < len(keys)-1; i++ {
		next, ok := current[keys[i]]
		if !ok {
			return fmt.Errorf("key not found: %s", key)
		}
		nextMap, ok := next.(map[string]interface{})
		if !ok {
			return fmt.Errorf("cannot traverse through non-map value at %s", keys[i])
		}
		current = nextMap
	}

	// Delete the final key
	lastKey := keys[len(keys)-1]
	if _, exists := current[lastKey]; !exists {
		return fmt.Errorf("key not found: %s", key)
	}
	delete(current, lastKey)

	return nil
}

// flattenKeys recursively flattens nested map keys with dot notation
func flattenKeys(m map[string]interface{}, prefix string) []string {
	var keys []string

	for k, v := range m {
		fullKey := k
		if prefix != "" {
			fullKey = prefix + "." + k
		}

		// If value is a nested map, recurse
		if nestedMap, ok := v.(map[string]interface{}); ok {
			keys = append(keys, flattenKeys(nestedMap, fullKey)...)
		} else {
			keys = append(keys, fullKey)
		}
	}

	return keys
}
