// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"encoding/json"
	"fmt"

	gomcp "github.com/mark3labs/mcp-go/mcp"
)

// jsonResult marshals any value to a JSON text result.
func jsonResult(v any) (*gomcp.CallToolResult, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}
	return gomcp.NewToolResultText(string(b)), nil
}

// errResult returns an MCP error result (tool-level, not protocol-level).
func errResult(err error) (*gomcp.CallToolResult, error) {
	return gomcp.NewToolResultError(err.Error()), nil
}
