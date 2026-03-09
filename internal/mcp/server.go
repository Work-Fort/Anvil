// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// NewServer creates the Anvil MCP server with all tools registered.
func NewServer(version string) *server.MCPServer {
	s := server.NewMCPServer(
		"anvil",
		version,
		server.WithToolCapabilities(true),
		server.WithRecovery(),
	)

	bm := NewBuildManager()

	registerContextTools(s)
	registerConfigTools(s)
	registerKernelConfigTools(s)
	registerKernelMgmtTools(s)
	registerKernelBuildTools(s, bm)
	registerFirecrackerTools(s)
	registerSigningTools(s)
	registerArchiveTools(s)
	registerUtilityTools(s)

	return s
}
