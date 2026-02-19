// SPDX-License-Identifier: Apache-2.0
package config

import (
	"encoding/json"
	"strings"
)

// JSONSchema represents a JSON Schema Draft 2020-12 document
type JSONSchema struct {
	Schema               string                 `json:"$schema"`
	Title                string                 `json:"title"`
	Description          string                 `json:"description"`
	Type                 string                 `json:"type"`
	Properties           map[string]interface{} `json:"properties"`
	AdditionalProperties bool                   `json:"additionalProperties"`
}

// JSONSchemaProperty represents a property in the JSON Schema
type JSONSchemaProperty struct {
	Type        string                 `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
	Enum        []string               `json:"enum,omitempty"`
	Pattern     string                 `json:"pattern,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// GenerateJSONSchema generates a JSON Schema from the ConfigRegistry
// If scope is nil, includes all keys. Otherwise, filters by the specified scope.
func GenerateJSONSchema() ([]byte, error) {
	return GenerateJSONSchemaForScope(nil)
}

// GenerateJSONSchemaForScope generates a JSON Schema filtered by scope
// Pass nil to include all keys, or a specific scope to filter
func GenerateJSONSchemaForScope(scope *ConfigScope) ([]byte, error) {
	title := "Cracker Barrel Configuration"
	description := "Configuration schema for Cracker Barrel CLI tool"

	if scope != nil {
		if *scope == ScopeUser {
			title = "Cracker Barrel User Configuration"
			description = "User-specific configuration (personal preferences)"
		} else {
			title = "Cracker Barrel Repo Configuration"
			description = "Repository-specific configuration (project settings)"
		}
	}

	schema := JSONSchema{
		Schema:               "https://json-schema.org/draft/2020-12/schema",
		Title:                title,
		Description:          description,
		Type:                 "object",
		Properties:           make(map[string]interface{}),
		AdditionalProperties: false,
	}

	// Build nested properties from flat registry
	for _, def := range ConfigRegistry {
		// Filter by scope if specified
		// Exclude keys that are forbidden in this scope
		if scope != nil {
			var constraints *ScopeConstraints
			if *scope == ScopeUser {
				constraints = def.UserConstraints
			} else {
				constraints = def.RepoConstraints
			}

			// Skip if forbidden in this scope
			if constraints != nil && constraints.Forbidden {
				continue
			}
		}
		addProperty(&schema, def)
	}

	return json.MarshalIndent(schema, "", "  ")
}

// addProperty adds a property to the schema, handling nested keys
func addProperty(schema *JSONSchema, def ConfigKeyDefinition) {
	parts := strings.Split(def.Key, ".")

	// If it's a top-level key (no dots), add directly
	if len(parts) == 1 {
		schema.Properties[def.Key] = buildProperty(def)
		return
	}

	// Handle nested keys
	current := schema.Properties
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]

		// Get or create the nested object
		if _, exists := current[part]; !exists {
			current[part] = &JSONSchemaProperty{
				Type:       "object",
				Properties: make(map[string]interface{}),
			}
		}

		// Navigate deeper
		prop := current[part].(*JSONSchemaProperty)
		current = prop.Properties
	}

	// Add the final property
	lastKey := parts[len(parts)-1]
	current[lastKey] = buildProperty(def)
}

// buildProperty creates a JSONSchemaProperty from a ConfigKeyDefinition
func buildProperty(def ConfigKeyDefinition) *JSONSchemaProperty {
	prop := &JSONSchemaProperty{
		Description: def.Description,
		Default:     def.Default,
	}

	switch def.Type {
	case "bool":
		prop.Type = "boolean"
	case "string":
		prop.Type = "string"
		if def.Pattern != "" {
			prop.Pattern = def.Pattern
		}
	case "enum":
		prop.Type = "string"
		prop.Enum = def.EnumValues
	}

	return prop
}
