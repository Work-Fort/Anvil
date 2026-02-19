// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ScopeConstraints defines per-scope validation rules for a configuration key
type ScopeConstraints struct {
	Forbidden  bool     // If true, this key cannot be set in this scope
	EnumValues []string // Valid enum values for this scope (overrides global EnumValues if set)
	Pattern    string   // Regex pattern for this scope (overrides global Pattern if set)
}

// ConfigKeyDefinition defines metadata for a configuration key
type ConfigKeyDefinition struct {
	Key         string      // Configuration key (dot notation)
	Type        string      // "string", "bool", "enum", "int"
	Default     interface{} // Default value
	Description string      // Help text

	// Global constraints (apply unless overridden by scope-specific constraints)
	EnumValues []string // Valid values for enum type (if Type="enum")
	Pattern    string   // Regex pattern for validation (if Type="string")

	// Per-scope constraints (optional - if nil, key is allowed in scope with global constraints)
	UserConstraints *ScopeConstraints // Constraints when setting in user config
	RepoConstraints *ScopeConstraints // Constraints when setting in repo config
}

// ConfigRegistry holds all known configuration keys with per-scope constraints.
//
// Constraint System:
//   - No constraints: Key can be set in any scope with same validation rules
//   - Forbidden constraint: Key cannot be set in the specified scope
//   - Scope-specific EnumValues: Different allowed values per scope
//   - Scope-specific Pattern: Different regex validation per scope
var ConfigRegistry = map[string]ConfigKeyDefinition{
	"use-tui": {
		Key:         "use-tui",
		Type:        "bool",
		Default:     true,
		Description: "Use TUI for interactive prompts",
	},

	"log-level": {
		Key:         "log-level",
		Type:        "enum",
		Default:     "debug",
		Description: "Log verbosity level",
		EnumValues:  []string{"disabled", "debug", "info", "warn", "error"},
	},

	"github-token": {
		Key:         "github-token",
		Type:        "string",
		Default:     "",
		Description: "GitHub personal access token for API access",
		RepoConstraints: &ScopeConstraints{
			Forbidden: true,
		},
	},

	"signing.key.name": {
		Key:         "signing.key.name",
		Type:        "string",
		Default:     "ACME Kernels",
		Description: "Default key owner name for project releases",
	},

	"signing.key.email": {
		Key:         "signing.key.email",
		Type:        "string",
		Default:     "fake@example.com",
		Description: "Default key owner email for project releases",
		Pattern:     "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
	},

	"signing.key.expiry": {
		Key:         "signing.key.expiry",
		Type:        "string",
		Default:     "1y",
		Description: "Default key expiration (0=never, <n>d/w/m/y)",
		Pattern:     "^(0|[0-9]+[dwmy])$",
	},

	"signing.key.format": {
		Key:         "signing.key.format",
		Type:        "enum",
		Default:     "armored",
		Description: "Key format: armored (ASCII) or binary (OpenPGP native)",
		EnumValues:  []string{"armored", "binary"},
	},

	"signing.history.location": {
		Key:         "signing.history.location",
		Type:        "string",
		Default:     "keys/history",
		Description: "Directory for public key history (relative to data dir)",
	},

	"signing.history.format": {
		Key:         "signing.history.format",
		Type:        "enum",
		Default:     "armored",
		Description: "History file format: armored (ASCII) or binary (OpenPGP native)",
		EnumValues:  []string{"armored", "binary"},
	},

	"signing.encrypted-keys": {
		Key:         "signing.encrypted-keys",
		Type:        "bool",
		Default:     true,
		Description: "Encrypt private keys at rest",
		RepoConstraints: &ScopeConstraints{
			Forbidden: true, // Cannot disable encryption in repo config
		},
	},

	"signing.key.location": {
		Key:         "signing.key.location",
		Type:        "string",
		Default:     "", // Set in InitViper() using GlobalPaths.KeysDir
		Description: "Directory for current signing key (absolute for user config, relative to repo root for repo config)",
	},

	"kernels.config.x86_64": {
		Key:         "kernels.config.x86_64",
		Type:        "string",
		Default:     "", // Required - no default
		Description: "Kernel config file for x86_64 architecture (relative path to file in repo)",
		UserConstraints: &ScopeConstraints{
			Forbidden: true, // Kernel configs are repo-specific
		},
	},

	"kernels.config.aarch64": {
		Key:         "kernels.config.aarch64",
		Type:        "string",
		Default:     "", // Required - no default
		Description: "Kernel config file for aarch64 architecture (relative path to file in repo)",
		UserConstraints: &ScopeConstraints{
			Forbidden: true, // Kernel configs are repo-specific
		},
	},

	"kernels.archive.location": {
		Key:         "kernels.archive.location",
		Type:        "string",
		Default:     "", // Optional - no archiving if not set
		Description: "Local directory for archiving built kernel artifacts (relative path inside the repo)",
		UserConstraints: &ScopeConstraints{
			Forbidden: true, // Archive location is repo-specific
		},
	},
}

// GetKeyDefinition returns the definition for a key, or nil if not found
func GetKeyDefinition(key string) *ConfigKeyDefinition {
	if def, ok := ConfigRegistry[key]; ok {
		return &def
	}
	return nil
}

// GetRequiredRepoKeys returns all configuration keys that are required in repo scope
// A key is required if it has no default value and is allowed in repo scope
func GetRequiredRepoKeys() []string {
	var required []string
	for key, def := range ConfigRegistry {
		// Skip if forbidden in repo scope
		if def.RepoConstraints != nil && def.RepoConstraints.Forbidden {
			continue
		}

		// Check if it's required (no default for string types)
		if def.Type == "string" && def.Default == "" {
			// Exception: signing.key.location has empty default but gets set in InitViper
			if key == "signing.key.location" {
				continue
			}
			required = append(required, key)
		}
	}
	return required
}

// ValidateKeyScope checks if a key can be set in the given scope
// Returns an error if the key is forbidden in the specified scope
func ValidateKeyScope(key string, scope ConfigScope) error {
	def := GetKeyDefinition(key)
	if def == nil {
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Get constraints for the target scope
	var constraints *ScopeConstraints
	switch scope {
	case ScopeUser:
		constraints = def.UserConstraints
	case ScopeRepo:
		constraints = def.RepoConstraints
	}

	// Check if key is forbidden in this scope
	if constraints != nil && constraints.Forbidden {
		switch scope {
		case ScopeUser:
			return fmt.Errorf(
				"key '%s' cannot be set in user config\n\n"+
					"Hint: Remove --global flag:\n"+
					"  anvil config set %s <value>\n\n"+
					"This key must be set in repo config: ./anvil.yaml",
				key,
				key,
			)
		case ScopeRepo:
			return fmt.Errorf(
				"key '%s' cannot be set in repo config (sensitive setting)\n\n"+
					"Hint: Use --global flag:\n"+
					"  anvil config set --global %s <value>\n\n"+
					"User config: ~/.config/anvil/config.yaml\n"+
					"This setting must NOT be committed to version control.",
				key,
				key,
			)
		}
	}

	return nil
}

// ValidateValue checks if a value is valid for the given key in the specified scope
// Applies per-scope constraints if defined, otherwise uses global constraints
func ValidateValue(key string, value interface{}, scope ConfigScope) error {
	def := GetKeyDefinition(key)
	if def == nil {
		return fmt.Errorf("unknown configuration key: %s", key)
	}

	// Get scope-specific constraints
	var constraints *ScopeConstraints
	switch scope {
	case ScopeUser:
		constraints = def.UserConstraints
	case ScopeRepo:
		constraints = def.RepoConstraints
	}

	// Type validation
	switch def.Type {
	case "bool":
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("key '%s' must be a boolean", key)
		}

	case "int":
		if _, ok := value.(int); !ok {
			return fmt.Errorf("key '%s' must be an integer", key)
		}

	case "string":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("key '%s' must be a string", key)
		}

		// Pattern validation - use scope-specific pattern if available
		pattern := def.Pattern
		if constraints != nil && constraints.Pattern != "" {
			pattern = constraints.Pattern
		}

		if pattern != "" {
			matched, err := regexp.MatchString(pattern, str)
			if err != nil {
				return fmt.Errorf("pattern validation error: %w", err)
			}
			if !matched {
				scopeName := getScopeName(scope)
				return fmt.Errorf(
					"key '%s' value '%s' does not match required format for %s scope",
					key,
					str,
					scopeName,
				)
			}
		}

	case "enum":
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("key '%s' must be a string", key)
		}

		// Enum validation - use scope-specific enum if available
		enumValues := def.EnumValues
		if constraints != nil && constraints.EnumValues != nil {
			enumValues = constraints.EnumValues
		}

		// Validate against enum
		valid := false
		for _, enumVal := range enumValues {
			if str == enumVal {
				valid = true
				break
			}
		}
		if !valid {
			scopeName := getScopeName(scope)
			return fmt.Errorf(
				"key '%s' must be one of %v in %s scope (got '%s')",
				key,
				enumValues,
				scopeName,
				str,
			)
		}
	}

	// Custom validation for specific keys
	if key == "signing.key.location" {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("key '%s' must be a string", key)
		}
		// Validate based on scope
		if scope == ScopeRepo {
			// Repo config: must be relative path
			if err := validateRepoPath(str); err != nil {
				return fmt.Errorf("key '%s': %w", key, err)
			}
		} else {
			// User config: allow absolute paths (typically XDG paths)
			if err := validateKeyLocationPath(str); err != nil {
				return fmt.Errorf("key '%s': %w", key, err)
			}
		}
	}

	// Kernel config file validation (must point to existing file in repo)
	if key == "kernels.config.x86_64" || key == "kernels.config.aarch64" {
		str, ok := value.(string)
		if !ok {
			return fmt.Errorf("key '%s' must be a string", key)
		}
		// These are repo-only keys, always validate as repo paths
		if err := validateRepoFilePath(str); err != nil {
			return fmt.Errorf("key '%s': %w", key, err)
		}
	}

	return nil
}

// validateKeyLocationPath validates a key location path for user config
// - Can be absolute or relative
// - Must be existing directory OR non-existent (will be created)
// - Must NOT point to an existing file
func validateKeyLocationPath(path string) error {
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist - this is OK, it will be created
			return nil
		}
		// Some other error (permission denied, etc.)
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Path exists - must be a directory, not a file
	if !info.IsDir() {
		return fmt.Errorf("path points to an existing file; must be a directory or non-existent path")
	}

	return nil
}

// validateRepoPath validates that a path is safe for use in repo config
// - Must not traverse outside repo (no ../)
// - Must be existing directory OR non-existent (will be created)
// - Must NOT point to an existing file
func validateRepoPath(path string) error {
	// Clean the path (resolves . and .. components)
	cleaned := filepath.Clean(path)

	// Check for path traversal outside repo
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, string(filepath.Separator)+"..") {
		return fmt.Errorf("path must not traverse outside repository (no '../' allowed)")
	}

	// Check if path is absolute (must be relative)
	if filepath.IsAbs(cleaned) {
		return fmt.Errorf("path must be relative to repository root")
	}

	// Check if path exists
	info, err := os.Stat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist - this is OK, it will be created
			return nil
		}
		// Some other error (permission denied, etc.)
		return fmt.Errorf("cannot access path: %w", err)
	}

	// Path exists - must be a directory, not a file
	if !info.IsDir() {
		return fmt.Errorf("path points to an existing file; must be a directory or non-existent path")
	}

	return nil
}

// validateRepoFilePath validates that a path points to an existing file in the repo
// - Must be relative path (no absolute paths)
// - Must not traverse outside repo (no ../)
// - Must point to an existing file (not directory)
// - File must exist (required for kernel configs)
func validateRepoFilePath(path string) error {
	// Clean the path (resolves . and .. components)
	cleaned := filepath.Clean(path)

	// Check for path traversal outside repo
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, string(filepath.Separator)+"..") {
		return fmt.Errorf("path must not traverse outside repository (no '../' allowed)")
	}

	// Check if path is absolute (must be relative)
	if filepath.IsAbs(cleaned) {
		return fmt.Errorf("path must be relative to repository root")
	}

	// Check if file exists
	info, err := os.Stat(cleaned)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("file does not exist (kernel config files must exist in repo)")
		}
		// Some other error (permission denied, etc.)
		return fmt.Errorf("cannot access file: %w", err)
	}

	// Path exists - must be a file, not a directory
	if info.IsDir() {
		return fmt.Errorf("path points to a directory; must be a file")
	}

	return nil
}
