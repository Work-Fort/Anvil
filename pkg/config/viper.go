// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// InitViper initializes Viper configuration with defaults and search paths
// Precedence order: ENV > dir-conf > user-conf > defaults
func InitViper() {
	// Set config type
	viper.SetConfigType(ConfigType)

	// Set defaults (lowest precedence)
	viper.SetDefault("use-tui", true)
	viper.SetDefault("log-level", "debug")
	viper.SetDefault("github-token", "") // No default for sensitive keys
	viper.SetDefault("signing.key.name", "ACME Kernels")
	viper.SetDefault("signing.key.email", "fake@example.com")
	viper.SetDefault("signing.key.expiry", "1y")
	viper.SetDefault("signing.key.format", "armored")
	viper.SetDefault("signing.key.location", GlobalPaths.KeysDir) // XDG: ~/.local/share/anvil/keys
	viper.SetDefault("signing.history.location", "keys/history")
	viper.SetDefault("signing.history.format", "armored")
	viper.SetDefault("signing.encrypted-keys", true) // Encrypt private keys at rest by default

	// Enable environment variable support (highest precedence)
	viper.SetEnvPrefix(EnvPrefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.AutomaticEnv()
}

// LoadConfig reads config files in precedence order
// Precedence: ENV > ./anvil.yaml > ~/.config/anvil/config.yaml > defaults
func LoadConfig() error {
	// First, try to read user config from XDG config directory
	viper.SetConfigName(ConfigFileName)
	viper.AddConfigPath(GlobalPaths.ConfigDir)

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read user config file: %w", err)
		}
		// Config file not found is OK
	} else {
		// Warn about misplaced keys in user config
		warnMisplacedKeys(GlobalPaths.ConfigDir, "user")
	}

	// Then, try to merge in local directory config (overrides user config)
	viper.SetConfigName(LocalConfigFile)
	viper.AddConfigPath(".")

	if err := viper.MergeInConfig(); err != nil {
		// Ignore if local config doesn't exist
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return fmt.Errorf("failed to read local config file: %w", err)
		}
	} else {
		// Validate repo config doesn't contain forbidden keys
		if err := validateConfigFile(".", ScopeRepo); err != nil {
			return err
		}
		// Warn about misplaced keys in repo config
		warnMisplacedKeys(".", "repo")
	}

	return nil
}

// GetUseTUI returns the use-tui configuration value
func GetUseTUI() bool {
	return viper.GetBool("use-tui")
}

// GetLogLevel returns the log-level configuration value
func GetLogLevel() string {
	return viper.GetString("log-level")
}

// GetSigningKeyName returns the signing.key.name configuration value
func GetSigningKeyName() string {
	return viper.GetString("signing.key.name")
}

// GetSigningKeyEmail returns the signing.key.email configuration value
func GetSigningKeyEmail() string {
	return viper.GetString("signing.key.email")
}

// GetSigningKeyExpiry returns the signing.key.expiry configuration value
func GetSigningKeyExpiry() string {
	return viper.GetString("signing.key.expiry")
}

// GetSigningKeyFormat returns the signing.key.format configuration value
func GetSigningKeyFormat() string {
	return viper.GetString("signing.key.format")
}

// GetSigningKeyLocation returns the signing.key.location configuration value
// In a repo context (anvil.yaml exists), ENV variables are ignored
// Precedence in repo context: repo config > user config > default
// Precedence outside repo: ENV > user config > default
func GetSigningKeyLocation() string {
	// Check if we're in a repo context (repo config file exists)
	repoConfigPath := filepath.Join(".", LocalConfigFile+DefaultConfigExt)
	if _, err := os.Stat(repoConfigPath); err == nil {
		// In repo context: ignore ENV, use repo > user > default
		// Create a Viper instance without ENV binding for this key
		v := viper.New()
		v.SetConfigType(ConfigType)

		// Set default
		v.SetDefault("signing.key.location", GlobalPaths.KeysDir)

		// Read user config
		v.SetConfigName(ConfigFileName)
		v.AddConfigPath(GlobalPaths.ConfigDir)
		_ = v.ReadInConfig() // Ignore error if not found

		// Merge repo config (overrides user)
		v.SetConfigName(LocalConfigFile)
		v.AddConfigPath(".")
		_ = v.MergeInConfig() // Ignore error if not found

		return v.GetString("signing.key.location")
	}

	// Not in repo context: use normal precedence (includes ENV)
	return viper.GetString("signing.key.location")
}

// GetSigningHistoryLocation returns the signing.history.location configuration value
func GetSigningHistoryLocation() string {
	return viper.GetString("signing.history.location")
}

// GetSigningHistoryFormat returns the signing.history.format configuration value
func GetSigningHistoryFormat() string {
	return viper.GetString("signing.history.format")
}

// GetSigningEncryptedKeys returns whether to encrypt signing keys at rest
// In a repo context (anvil.yaml exists), always returns true regardless of user config
func GetSigningEncryptedKeys() bool {
	// Check if we're in a repo context (repo config file exists)
	repoConfigPath := filepath.Join(".", LocalConfigFile+DefaultConfigExt)
	if _, err := os.Stat(repoConfigPath); err == nil {
		// In repo context: always enforce encryption
		return true
	}

	// Not in repo: use normal config precedence
	return viper.GetBool("signing.encrypted-keys")
}

// GetKernelsConfigX86_64 returns the kernels.config.x86_64 configuration value
func GetKernelsConfigX86_64() string {
	return viper.GetString("kernels.config.x86_64")
}

// GetKernelsConfigAarch64 returns the kernels.config.aarch64 configuration value
func GetKernelsConfigAarch64() string {
	return viper.GetString("kernels.config.aarch64")
}

// GetKernelsArchiveLocation returns the kernels.archive.location configuration value.
// Returns an empty string when not configured (no archiving).
func GetKernelsArchiveLocation() string {
	return viper.GetString("kernels.archive.location")
}

// validateConfigFile validates that a config file doesn't contain forbidden keys for the given scope
// For repo scope, also validates that all required keys are present
func validateConfigFile(configDir string, scope ConfigScope) error {
	var configPath string
	if scope == ScopeUser {
		configPath = filepath.Join(configDir, ConfigFileName+DefaultConfigExt)
	} else {
		configPath = filepath.Join(".", LocalConfigFile+DefaultConfigExt)
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// For repo scope, missing config file means missing required keys
		if scope == ScopeRepo {
			requiredKeys := GetRequiredRepoKeys()
			if len(requiredKeys) > 0 {
				return fmt.Errorf(
					"repo config file not found: %s\n\n"+
						"Required keys for repo mode:\n"+
						"  - %s\n\n"+
						"Create a anvil.yaml file in your repository root.",
					configPath,
					strings.Join(requiredKeys, "\n  - "),
				)
			}
		}
		return nil // No config file, nothing to validate
	}

	// Create a temporary Viper instance to read just this config file
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType(ConfigType)

	if err := v.ReadInConfig(); err != nil {
		return fmt.Errorf("failed to read config file for validation: %w", err)
	}

	// Get all settings from this config file
	settings := v.AllSettings()
	if len(settings) == 0 && scope == ScopeRepo {
		// Empty repo config but we have required keys
		requiredKeys := GetRequiredRepoKeys()
		if len(requiredKeys) > 0 {
			return fmt.Errorf(
				"repo config file is empty: %s\n\n"+
					"Required keys for repo mode:\n"+
					"  - %s",
				configPath,
				strings.Join(requiredKeys, "\n  - "),
			)
		}
	}

	// Flatten keys and validate each one
	keys := flattenKeys(settings, "")
	for _, key := range keys {
		// Validate scope
		if err := ValidateKeyScope(key, scope); err != nil {
			return fmt.Errorf("invalid key in config file %s: %w", configPath, err)
		}

		// Validate value
		value := v.Get(key)
		if err := ValidateValue(key, value, scope); err != nil {
			return fmt.Errorf("invalid value in config file %s: %w", configPath, err)
		}
	}

	// For repo scope, validate that all required keys are present
	if scope == ScopeRepo {
		requiredKeys := GetRequiredRepoKeys()
		keySet := make(map[string]bool)
		for _, key := range keys {
			keySet[key] = true
		}

		var missingKeys []string
		for _, requiredKey := range requiredKeys {
			if !keySet[requiredKey] {
				missingKeys = append(missingKeys, requiredKey)
			}
		}

		if len(missingKeys) > 0 {
			return fmt.Errorf(
				"missing required keys in repo config %s:\n"+
					"  - %s\n\n"+
					"Example minimal config:\n"+
					"kernels:\n"+
					"  config:\n"+
					"    x86_64: configs/kernel-x86_64.config\n"+
					"    aarch64: configs/kernel-aarch64.config",
				configPath,
				strings.Join(missingKeys, "\n  - "),
			)
		}
	}

	return nil
}

// warnMisplacedKeys provides informational messages about unconventional key placement
// Note: All keys can be set in any scope (precedence handles conflicts), but some
// placements are unconventional. This logs at debug level to inform without blocking.
func warnMisplacedKeys(configDir, scopeName string) {
	// Determine config file path based on scope
	var configPath string
	var recommendedScope ConfigScope
	if scopeName == "user" {
		configPath = filepath.Join(configDir, ConfigFileName+DefaultConfigExt)
		recommendedScope = ScopeUser
	} else {
		configPath = filepath.Join(".", LocalConfigFile+DefaultConfigExt)
		recommendedScope = ScopeRepo
	}

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return // No config file, nothing to check
	}

	// Create a temporary Viper instance to read just this config file
	v := viper.New()
	v.SetConfigFile(configPath)
	v.SetConfigType(ConfigType)

	if err := v.ReadInConfig(); err != nil {
		return // Can't read config, skip informational messages
	}

	// Get all settings from this config file
	settings := v.AllSettings()
	if len(settings) == 0 {
		return
	}

	// Flatten keys and check each one
	keys := flattenKeys(settings, "")
	for _, key := range keys {
		def := GetKeyDefinition(key)
		if def == nil {
			continue // Unknown key, skip
		}

		// Determine recommended scope based on constraints
		// If forbidden in one scope, the other is recommended
		var hasRecommendedScope bool
		var recommendedScopeForKey ConfigScope

		if def.RepoConstraints != nil && def.RepoConstraints.Forbidden {
			hasRecommendedScope = true
			recommendedScopeForKey = ScopeUser
		} else if def.UserConstraints != nil && def.UserConstraints.Forbidden {
			hasRecommendedScope = true
			recommendedScopeForKey = ScopeRepo
		}

		// Provide info if key is in unconventional location and has a recommendation
		if hasRecommendedScope && recommendedScopeForKey != recommendedScope {
			var typicalLocation string
			if recommendedScopeForKey == ScopeUser {
				typicalLocation = "~/.config/anvil/" + ConfigFileName + DefaultConfigExt
			} else {
				typicalLocation = "./" + LocalConfigFile + DefaultConfigExt
			}

			log.Debugf("Key '%s' in %s config (typically in %s config: %s)",
				key, scopeName, getScopeName(recommendedScopeForKey), typicalLocation)
		}
	}
}

// BindFlags binds all relevant cobra flags to Viper
func BindFlags(flags *pflag.FlagSet) error {
	flagsToBind := []string{
		"use-tui",
		"log-level",
	}

	for _, flagName := range flagsToBind {
		if err := viper.BindPFlag(flagName, flags.Lookup(flagName)); err != nil {
			return fmt.Errorf("failed to bind flag %s: %w", flagName, err)
		}
	}

	return nil
}
