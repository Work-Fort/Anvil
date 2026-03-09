// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	mcpserver "github.com/Work-Fort/Anvil/internal/mcp"
)

// NewMCPServerCmd creates the mcp-server command.
func NewMCPServerCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:    "mcp-server",
		Short:  "Start MCP stdio server for AI agent integration",
		Long:   "Start a Model Context Protocol server on stdin/stdout. Used by Claude Code and other MCP clients to manage kernels, configs, signing, and builds.",
		Hidden: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			s := mcpserver.NewServer(version)
			return server.ServeStdio(s)
		},
	}

	return cmd
}
