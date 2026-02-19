// SPDX-License-Identifier: Apache-2.0
package config

import (
	"strings"
	"testing"
)

func TestConfigKeyDefinition_Validation(t *testing.T) {
	def := ConfigKeyDefinition{
		Key:         "use-tui",
		Type:        "bool",
		Default:     true,
		Description: "Use TUI for interactive prompts",
	}

	// Test that definition is valid
	if def.Key == "" {
		t.Error("Key should not be empty")
	}
	if def.Type != "bool" {
		t.Errorf("Type = %v, want bool", def.Type)
	}
}

func TestConfigRegistry_ContainsUseTUI(t *testing.T) {
	def, ok := ConfigRegistry["use-tui"]
	if !ok {
		t.Fatal("ConfigRegistry should contain 'use-tui' key")
	}
	if def.Type != "bool" {
		t.Errorf("use-tui type = %v, want bool", def.Type)
	}
	if def.Default != true {
		t.Errorf("use-tui default = %v, want true", def.Default)
	}
	if def.UserConstraints != nil || def.RepoConstraints != nil {
		t.Error("use-tui should have no scope constraints")
	}
}

func TestConfigRegistry_ContainsLogLevel(t *testing.T) {
	def, ok := ConfigRegistry["log-level"]
	if !ok {
		t.Fatal("ConfigRegistry should contain 'log-level' key")
	}
	if def.Type != "enum" {
		t.Errorf("log-level type = %v, want enum", def.Type)
	}
	expectedEnums := []string{"disabled", "debug", "info", "warn", "error"}
	if len(def.EnumValues) != len(expectedEnums) {
		t.Errorf("log-level enum count = %d, want %d", len(def.EnumValues), len(expectedEnums))
	}
	if def.UserConstraints != nil || def.RepoConstraints != nil {
		t.Error("log-level should have no scope constraints")
	}
}

func TestConfigRegistry_ContainsGitHubToken(t *testing.T) {
	def, ok := ConfigRegistry["github-token"]
	if !ok {
		t.Fatal("ConfigRegistry should contain 'github-token' key")
	}
	if def.Type != "string" {
		t.Errorf("github-token type = %v, want string", def.Type)
	}
	if def.RepoConstraints == nil || !def.RepoConstraints.Forbidden {
		t.Error("github-token should be forbidden in repo scope")
	}
	if def.UserConstraints != nil && def.UserConstraints.Forbidden {
		t.Error("github-token should be allowed in user scope")
	}
}

func TestConfigRegistry_ContainsSigningKeys(t *testing.T) {
	signingKeys := []string{
		"signing.key.name",
		"signing.key.email",
		"signing.key.expiry",
		"signing.key.format",
		"signing.history.location",
		"signing.history.format",
	}

	for _, key := range signingKeys {
		t.Run(key, func(t *testing.T) {
			def, ok := ConfigRegistry[key]
			if !ok {
				t.Fatalf("ConfigRegistry should contain '%s' key", key)
			}
			if (def.UserConstraints != nil && def.UserConstraints.Forbidden) ||
				(def.RepoConstraints != nil && def.RepoConstraints.Forbidden) {
				t.Errorf("%s should not be forbidden in any scope", key)
			}
		})
	}
}

func TestConfigRegistry_SigningKeyEmail_HasPattern(t *testing.T) {
	def := ConfigRegistry["signing.key.email"]
	if def.Pattern == "" {
		t.Error("signing.key.email should have email pattern validation")
	}
}

func TestConfigRegistry_SigningKeyFormat_IsEnum(t *testing.T) {
	def := ConfigRegistry["signing.key.format"]
	if def.Type != "enum" {
		t.Errorf("signing.key.format type = %v, want enum", def.Type)
	}
	if len(def.EnumValues) != 2 {
		t.Errorf("signing.key.format enum count = %d, want 2", len(def.EnumValues))
	}
}

func TestGetKeyDefinition_ExistingKey(t *testing.T) {
	def := GetKeyDefinition("use-tui")
	if def == nil {
		t.Fatal("GetKeyDefinition should return definition for 'use-tui'")
	}
	if def.Key != "use-tui" {
		t.Errorf("def.Key = %v, want use-tui", def.Key)
	}
}

func TestGetKeyDefinition_NonExistentKey(t *testing.T) {
	def := GetKeyDefinition("nonexistent")
	if def != nil {
		t.Error("GetKeyDefinition should return nil for nonexistent key")
	}
}

func TestValidateKeyScope_UserKeyInUserScope(t *testing.T) {
	err := ValidateKeyScope("use-tui", ScopeUser)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow user key in user scope: %v", err)
	}
}

func TestValidateKeyScope_UserKeyInRepoScope(t *testing.T) {
	err := ValidateKeyScope("use-tui", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow use-tui in repo scope: %v", err)
	}
}

func TestValidateKeyScope_RepoKeyInRepoScope(t *testing.T) {
	err := ValidateKeyScope("signing.key.name", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow repo key in repo scope: %v", err)
	}
}

func TestValidateKeyScope_RepoKeyInUserScope(t *testing.T) {
	err := ValidateKeyScope("signing.key.name", ScopeUser)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow signing.key.name in user scope: %v", err)
	}
}

func TestValidateKeyScope_UnknownKey(t *testing.T) {
	err := ValidateKeyScope("unknown-key", ScopeUser)
	if err == nil {
		t.Error("ValidateKeyScope should reject unknown key")
	}
	if !strings.Contains(err.Error(), "unknown configuration key") {
		t.Errorf("Error should mention unknown key: %v", err)
	}
}

func TestValidateKeyScope_RestrictedUserKeyInRepoScope(t *testing.T) {
	err := ValidateKeyScope("github-token", ScopeRepo)
	if err == nil {
		t.Error("ValidateKeyScope should reject forbidden key in repo scope")
	}
	if !strings.Contains(err.Error(), "cannot be set in repo config") {
		t.Errorf("Error should mention repo restriction: %v", err)
	}
	if !strings.Contains(err.Error(), "sensitive") {
		t.Errorf("Error should mention sensitive nature: %v", err)
	}
}

func TestValidateKeyScope_RestrictedUserKeyInUserScope(t *testing.T) {
	err := ValidateKeyScope("github-token", ScopeUser)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow github-token in user scope: %v", err)
	}
}

func TestValidateKeyScope_FlexibleKeyInAnyScope(t *testing.T) {
	err := ValidateKeyScope("use-tui", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow use-tui in repo scope: %v", err)
	}

	err = ValidateKeyScope("use-tui", ScopeUser)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow use-tui in user scope: %v", err)
	}

	err = ValidateKeyScope("signing.key.name", ScopeUser)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow signing.key.name in user scope: %v", err)
	}

	err = ValidateKeyScope("signing.key.name", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow signing.key.name in repo scope: %v", err)
	}
}

func TestValidateValue_BooleanValid(t *testing.T) {
	err := ValidateValue("use-tui", true, ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept boolean: %v", err)
	}
}

func TestValidateValue_BooleanInvalid(t *testing.T) {
	err := ValidateValue("use-tui", "not-a-bool", ScopeUser)
	if err == nil {
		t.Error("ValidateValue should reject non-boolean for bool field")
	}
}

func TestValidateValue_StringValid(t *testing.T) {
	err := ValidateValue("signing.key.name", "My Project", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept string: %v", err)
	}
}

func TestValidateValue_StringInvalid(t *testing.T) {
	err := ValidateValue("signing.key.name", 123, ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject non-string for string field")
	}
}

func TestValidateValue_EmailPattern_Valid(t *testing.T) {
	err := ValidateValue("signing.key.email", "test@example.com", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept valid email: %v", err)
	}
}

func TestValidateValue_EmailPattern_Invalid(t *testing.T) {
	err := ValidateValue("signing.key.email", "not-an-email", ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject invalid email")
	}
}

func TestValidateValue_ExpiryPattern_Valid(t *testing.T) {
	validExpiries := []string{"0", "1d", "2w", "3m", "1y"}
	for _, expiry := range validExpiries {
		t.Run(expiry, func(t *testing.T) {
			err := ValidateValue("signing.key.expiry", expiry, ScopeRepo)
			if err != nil {
				t.Errorf("ValidateValue should accept '%s': %v", expiry, err)
			}
		})
	}
}

func TestValidateValue_ExpiryPattern_Invalid(t *testing.T) {
	err := ValidateValue("signing.key.expiry", "invalid", ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject invalid expiry format")
	}
}

func TestValidateValue_EnumValid(t *testing.T) {
	err := ValidateValue("log-level", "debug", ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept valid enum: %v", err)
	}
}

func TestValidateValue_EnumInvalid(t *testing.T) {
	err := ValidateValue("log-level", "invalid-level", ScopeUser)
	if err == nil {
		t.Error("ValidateValue should reject invalid enum value")
	}
}

func TestScopeConstraints_ForbiddenInUserScope(t *testing.T) {
	testKey := ConfigKeyDefinition{
		Key:  "test-forbidden-user",
		Type: "string",
		UserConstraints: &ScopeConstraints{
			Forbidden: true,
		},
	}

	ConfigRegistry["test-forbidden-user"] = testKey
	defer delete(ConfigRegistry, "test-forbidden-user")

	err := ValidateKeyScope("test-forbidden-user", ScopeUser)
	if err == nil {
		t.Error("ValidateKeyScope should reject forbidden key in user scope")
	}

	err = ValidateKeyScope("test-forbidden-user", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow key in repo scope: %v", err)
	}
}

func TestScopeConstraints_ForbiddenInRepoScope(t *testing.T) {
	err := ValidateKeyScope("github-token", ScopeRepo)
	if err == nil {
		t.Error("ValidateKeyScope should reject github-token in repo scope")
	}

	err = ValidateKeyScope("github-token", ScopeUser)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow github-token in user scope: %v", err)
	}
}

func TestScopeConstraints_DifferentEnumValuesPerScope(t *testing.T) {
	testKey := ConfigKeyDefinition{
		Key:        "test-enum-scope",
		Type:       "enum",
		EnumValues: []string{"a", "b", "c"},
		UserConstraints: &ScopeConstraints{
			EnumValues: []string{"a", "b"},
		},
		RepoConstraints: &ScopeConstraints{
			EnumValues: []string{"b", "c"},
		},
	}

	ConfigRegistry["test-enum-scope"] = testKey
	defer delete(ConfigRegistry, "test-enum-scope")

	err := ValidateValue("test-enum-scope", "a", ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept 'a' in user scope: %v", err)
	}

	err = ValidateValue("test-enum-scope", "c", ScopeUser)
	if err == nil {
		t.Error("ValidateValue should reject 'c' in user scope")
	}

	err = ValidateValue("test-enum-scope", "c", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept 'c' in repo scope: %v", err)
	}

	err = ValidateValue("test-enum-scope", "a", ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject 'a' in repo scope")
	}
}

func TestScopeConstraints_DifferentPatternsPerScope(t *testing.T) {
	testKey := ConfigKeyDefinition{
		Key:     "test-pattern-scope",
		Type:    "string",
		Pattern: "^[A-Z]+$",
		UserConstraints: &ScopeConstraints{
			Pattern: "^[a-z]+$",
		},
		RepoConstraints: &ScopeConstraints{
			Pattern: "^[0-9]+$",
		},
	}

	ConfigRegistry["test-pattern-scope"] = testKey
	defer delete(ConfigRegistry, "test-pattern-scope")

	err := ValidateValue("test-pattern-scope", "abc", ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept lowercase in user scope: %v", err)
	}

	err = ValidateValue("test-pattern-scope", "123", ScopeUser)
	if err == nil {
		t.Error("ValidateValue should reject numbers in user scope")
	}

	err = ValidateValue("test-pattern-scope", "123", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept numbers in repo scope: %v", err)
	}

	err = ValidateValue("test-pattern-scope", "abc", ScopeRepo)
	if err == nil {
		t.Error("ValidateValue should reject lowercase in repo scope")
	}
}

func TestScopeConstraints_NoConstraints(t *testing.T) {
	err := ValidateKeyScope("use-tui", ScopeUser)
	if err != nil {
		t.Errorf("Key without constraints should be allowed in user scope: %v", err)
	}

	err = ValidateKeyScope("use-tui", ScopeRepo)
	if err != nil {
		t.Errorf("Key without constraints should be allowed in repo scope: %v", err)
	}

	err = ValidateValue("use-tui", true, ScopeUser)
	if err != nil {
		t.Errorf("ValidateValue should accept boolean in user scope: %v", err)
	}

	err = ValidateValue("use-tui", true, ScopeRepo)
	if err != nil {
		t.Errorf("ValidateValue should accept boolean in repo scope: %v", err)
	}
}

func TestConfigRegistry_SigningKeyLocation(t *testing.T) {
	def, ok := ConfigRegistry["signing.key.location"]
	if !ok {
		t.Fatal("ConfigRegistry should contain 'signing.key.location' key")
	}
	if def.Type != "string" {
		t.Errorf("signing.key.location type = %v, want string", def.Type)
	}
	// Default is set in InitViper() using GlobalPaths.KeysDir
	if def.UserConstraints != nil && def.UserConstraints.Forbidden {
		t.Error("signing.key.location should be allowed in user config")
	}
	if def.RepoConstraints != nil && def.RepoConstraints.Forbidden {
		t.Error("signing.key.location should be allowed in repo config")
	}
}

func TestValidateValue_SigningKeyLocation_Scope(t *testing.T) {
	// Should be allowed in user scope
	err := ValidateKeyScope("signing.key.location", ScopeUser)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow signing.key.location in user scope: %v", err)
	}

	// Should be allowed in repo scope
	err = ValidateKeyScope("signing.key.location", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow signing.key.location in repo scope: %v", err)
	}
}

func TestConfigRegistry_KernelsConfigX86_64(t *testing.T) {
	def, ok := ConfigRegistry["kernels.config.x86_64"]
	if !ok {
		t.Fatal("ConfigRegistry should contain 'kernels.config.x86_64' key")
	}
	if def.Type != "string" {
		t.Errorf("kernels.config.x86_64 type = %v, want string", def.Type)
	}
	if def.Default != "" {
		t.Errorf("kernels.config.x86_64 should have no default (required)")
	}
	if def.UserConstraints == nil || !def.UserConstraints.Forbidden {
		t.Error("kernels.config.x86_64 should be forbidden in user scope")
	}
	if def.RepoConstraints != nil && def.RepoConstraints.Forbidden {
		t.Error("kernels.config.x86_64 should be allowed in repo scope")
	}
}

func TestConfigRegistry_KernelsConfigAarch64(t *testing.T) {
	def, ok := ConfigRegistry["kernels.config.aarch64"]
	if !ok {
		t.Fatal("ConfigRegistry should contain 'kernels.config.aarch64' key")
	}
	if def.Type != "string" {
		t.Errorf("kernels.config.aarch64 type = %v, want string", def.Type)
	}
	if def.Default != "" {
		t.Errorf("kernels.config.aarch64 should have no default (required)")
	}
	if def.UserConstraints == nil || !def.UserConstraints.Forbidden {
		t.Error("kernels.config.aarch64 should be forbidden in user scope")
	}
	if def.RepoConstraints != nil && def.RepoConstraints.Forbidden {
		t.Error("kernels.config.aarch64 should be allowed in repo scope")
	}
}

func TestValidateKeyScope_KernelConfigInUserScope(t *testing.T) {
	err := ValidateKeyScope("kernels.config.x86_64", ScopeUser)
	if err == nil {
		t.Error("ValidateKeyScope should reject kernels.config.x86_64 in user scope")
	}
	if !strings.Contains(err.Error(), "cannot be set in user config") {
		t.Errorf("Error should mention user config restriction: %v", err)
	}

	err = ValidateKeyScope("kernels.config.aarch64", ScopeUser)
	if err == nil {
		t.Error("ValidateKeyScope should reject kernels.config.aarch64 in user scope")
	}
}

func TestValidateKeyScope_KernelConfigInRepoScope(t *testing.T) {
	err := ValidateKeyScope("kernels.config.x86_64", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow kernels.config.x86_64 in repo scope: %v", err)
	}

	err = ValidateKeyScope("kernels.config.aarch64", ScopeRepo)
	if err != nil {
		t.Errorf("ValidateKeyScope should allow kernels.config.aarch64 in repo scope: %v", err)
	}
}

func TestGetRequiredRepoKeys(t *testing.T) {
	required := GetRequiredRepoKeys()

	// Should include kernel config keys
	hasX86 := false
	hasArm := false
	for _, key := range required {
		if key == "kernels.config.x86_64" {
			hasX86 = true
		}
		if key == "kernels.config.aarch64" {
			hasArm = true
		}
	}

	if !hasX86 {
		t.Error("GetRequiredRepoKeys should include kernels.config.x86_64")
	}
	if !hasArm {
		t.Error("GetRequiredRepoKeys should include kernels.config.aarch64")
	}

	// Should not include signing.key.location (has special handling)
	for _, key := range required {
		if key == "signing.key.location" {
			t.Error("GetRequiredRepoKeys should not include signing.key.location (has default set in InitViper)")
		}
	}

	// Should not include keys forbidden in repo scope
	for _, key := range required {
		if key == "github-token" {
			t.Error("GetRequiredRepoKeys should not include keys forbidden in repo scope")
		}
	}
}
