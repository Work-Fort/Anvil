// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateJSONSchema(t *testing.T) {
	schema, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("GenerateJSONSchema failed: %v", err)
	}

	if len(schema) == 0 {
		t.Error("GenerateJSONSchema returned empty schema")
	}

	// Verify it's valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(schema, &result); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}

	// Verify $schema field
	schemaVersion, ok := result["$schema"].(string)
	if !ok {
		t.Error("$schema field missing or not a string")
	}
	if schemaVersion != "https://json-schema.org/draft/2020-12/schema" {
		t.Errorf("$schema = %s, want Draft 2020-12", schemaVersion)
	}

	// Verify title
	title, ok := result["title"].(string)
	if !ok || title == "" {
		t.Error("title field missing or empty")
	}

	// Verify properties exist
	properties, ok := result["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("properties field missing or not an object")
	}

	// Verify some top-level keys exist
	topLevelKeys := []string{"use-tui", "log-level", "signing"}
	for _, key := range topLevelKeys {
		if _, exists := properties[key]; !exists {
			t.Errorf("Expected property '%s' not found in schema", key)
		}
	}
}

func TestGenerateJSONSchema_NestedProperties(t *testing.T) {
	schema, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("GenerateJSONSchema failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(schema, &result); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}

	properties := result["properties"].(map[string]interface{})

	// Verify signing is an object
	signing, ok := properties["signing"].(map[string]interface{})
	if !ok {
		t.Fatal("signing should be an object")
	}

	// Verify signing has properties
	signingProps, ok := signing["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("signing should have properties")
	}

	// Verify signing.key exists
	key, ok := signingProps["key"].(map[string]interface{})
	if !ok {
		t.Fatal("signing.key should exist and be an object")
	}

	// Verify signing.key has properties
	keyProps, ok := key["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("signing.key should have properties")
	}

	// Verify signing.key.name exists
	if _, exists := keyProps["name"]; !exists {
		t.Error("signing.key.name should exist")
	}
}

func TestGenerateJSONSchema_BooleanType(t *testing.T) {
	schema, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("GenerateJSONSchema failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(schema, &result)

	properties := result["properties"].(map[string]interface{})
	useTUI := properties["use-tui"].(map[string]interface{})

	// Check type
	if useTUI["type"] != "boolean" {
		t.Errorf("use-tui type = %v, want boolean", useTUI["type"])
	}

	// Check default
	if useTUI["default"] != true {
		t.Errorf("use-tui default = %v, want true", useTUI["default"])
	}
}

func TestGenerateJSONSchema_EnumType(t *testing.T) {
	schema, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("GenerateJSONSchema failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(schema, &result)

	properties := result["properties"].(map[string]interface{})
	logLevel := properties["log-level"].(map[string]interface{})

	// Check type
	if logLevel["type"] != "string" {
		t.Errorf("log-level type = %v, want string", logLevel["type"])
	}

	// Check enum values exist
	enumValues, ok := logLevel["enum"].([]interface{})
	if !ok {
		t.Fatal("log-level should have enum values")
	}

	if len(enumValues) != 5 {
		t.Errorf("log-level enum has %d values, want 5", len(enumValues))
	}
}

func TestGenerateJSONSchema_PatternValidation(t *testing.T) {
	schema, err := GenerateJSONSchema()
	if err != nil {
		t.Fatalf("GenerateJSONSchema failed: %v", err)
	}

	var result map[string]interface{}
	json.Unmarshal(schema, &result)

	// Navigate to signing.key.email
	properties := result["properties"].(map[string]interface{})
	signing := properties["signing"].(map[string]interface{})
	signingProps := signing["properties"].(map[string]interface{})
	key := signingProps["key"].(map[string]interface{})
	keyProps := key["properties"].(map[string]interface{})
	email := keyProps["email"].(map[string]interface{})

	// Check pattern exists
	pattern, ok := email["pattern"].(string)
	if !ok || pattern == "" {
		t.Error("signing.key.email should have a pattern")
	}
}

func TestGenerateJSONSchemaForScope_UserOnly(t *testing.T) {
	scope := ScopeUser
	schema, err := GenerateJSONSchemaForScope(&scope)
	if err != nil {
		t.Fatalf("GenerateJSONSchemaForScope failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(schema, &result); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}

	// Verify title mentions user
	title, ok := result["title"].(string)
	if !ok || !strings.Contains(title, "User") {
		t.Errorf("Expected 'User' in title, got: %s", title)
	}

	properties := result["properties"].(map[string]interface{})

	// Should have user-scope keys
	if _, exists := properties["use-tui"]; !exists {
		t.Error("use-tui (user-scope) should be in user schema")
	}
	if _, exists := properties["log-level"]; !exists {
		t.Error("log-level (user-scope) should be in user schema")
	}
	if _, exists := properties["github-token"]; !exists {
		t.Error("github-token (user-only restricted) should be in user schema")
	}

	// Should have flexible repo-scope keys
	if _, exists := properties["signing"]; !exists {
		t.Error("signing (flexible repo-scope) should be in user schema")
	}
}

func TestGenerateJSONSchemaForScope_RepoOnly(t *testing.T) {
	scope := ScopeRepo
	schema, err := GenerateJSONSchemaForScope(&scope)
	if err != nil {
		t.Fatalf("GenerateJSONSchemaForScope failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(schema, &result); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}

	// Verify title mentions repo
	title, ok := result["title"].(string)
	if !ok || !strings.Contains(title, "Repo") {
		t.Errorf("Expected 'Repo' in title, got: %s", title)
	}

	properties := result["properties"].(map[string]interface{})

	// Should have repo-scope keys
	if _, exists := properties["signing"]; !exists {
		t.Error("signing (repo-scope) should be in repo schema")
	}

	// Should have flexible user-scope keys
	if _, exists := properties["use-tui"]; !exists {
		t.Error("use-tui (flexible user-scope) should be in repo schema")
	}
	if _, exists := properties["log-level"]; !exists {
		t.Error("log-level (flexible user-scope) should be in repo schema")
	}

	// Should NOT have restricted user-only keys
	if _, exists := properties["github-token"]; exists {
		t.Error("github-token (restricted user-only) should NOT be in repo schema")
	}
}

func TestGenerateJSONSchemaForScope_AllKeys(t *testing.T) {
	// Passing nil should include all keys
	schema, err := GenerateJSONSchemaForScope(nil)
	if err != nil {
		t.Fatalf("GenerateJSONSchemaForScope failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(schema, &result); err != nil {
		t.Fatalf("Schema is not valid JSON: %v", err)
	}

	properties := result["properties"].(map[string]interface{})

	// Should have both user and repo keys
	if _, exists := properties["use-tui"]; !exists {
		t.Error("use-tui (user-scope) should be in full schema")
	}
	if _, exists := properties["signing"]; !exists {
		t.Error("signing (repo-scope) should be in full schema")
	}
}
