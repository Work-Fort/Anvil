// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"github.com/mark3labs/mcp-go/server"
)

// serverVersion holds the version string for use by tools like get_context.
var serverVersion string

// NewServer creates the Anvil MCP server with all tools registered.
func NewServer(version string) *server.MCPServer {
	serverVersion = version
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
