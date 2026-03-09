// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"

	"github.com/Work-Fort/Anvil/pkg/signing"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSigningTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("signing_key_info",
		gomcp.WithDescription("Get current signing key details"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleSigningKeyInfo)

	s.AddTool(gomcp.NewTool("signing_generate_key",
		gomcp.WithDescription("Generate a new PGP signing key"),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Key holder name")),
		gomcp.WithString("email", gomcp.Required(), gomcp.Description("Key holder email")),
		gomcp.WithString("expiry", gomcp.Description("Key expiry duration (e.g. 1y, 6m). Default: 1y")),
	), handleSigningGenerateKey)

	s.AddTool(gomcp.NewTool("signing_rotate_key",
		gomcp.WithDescription("Rotate the signing key (generates new, removes old)"),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Key holder name")),
		gomcp.WithString("email", gomcp.Required(), gomcp.Description("Key holder email")),
		gomcp.WithString("expiry", gomcp.Description("Key expiry duration (default: 1y)")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleSigningRotateKey)

	s.AddTool(gomcp.NewTool("signing_sign",
		gomcp.WithDescription("Sign artifacts in a directory"),
		gomcp.WithString("path", gomcp.Required(), gomcp.Description("Path to artifacts directory")),
	), handleSigningSign)

	s.AddTool(gomcp.NewTool("signing_verify",
		gomcp.WithDescription("Verify signatures of artifacts in a directory"),
		gomcp.WithString("path", gomcp.Required(), gomcp.Description("Path to artifacts directory")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleSigningVerify)

	s.AddTool(gomcp.NewTool("signing_export_backup",
		gomcp.WithDescription("Export encrypted key backup"),
		gomcp.WithString("email", gomcp.Required(), gomcp.Description("Email of the key to export")),
		gomcp.WithString("output_path", gomcp.Required(), gomcp.Description("Output file path for encrypted backup")),
	), handleSigningExportBackup)

	s.AddTool(gomcp.NewTool("signing_import_backup",
		gomcp.WithDescription("Import key from encrypted backup"),
		gomcp.WithString("backup_path", gomcp.Required(), gomcp.Description("Path to encrypted backup file")),
	), handleSigningImportBackup)
}

func handleSigningKeyInfo(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	keys, err := signing.ListKeys()
	if err != nil {
		return errResult(err)
	}

	if len(keys) == 0 {
		return jsonResult(map[string]any{
			"has_key": false,
			"keys":    []any{},
		})
	}

	keyList := make([]map[string]any, len(keys))
	for i, k := range keys {
		keyList[i] = map[string]any{
			"key_id":      k.KeyID,
			"fingerprint": k.Fingerprint,
			"name":        k.Name,
			"email":       k.Email,
			"created":     k.Created.String(),
			"expires":     k.Expires.String(),
		}
	}

	return jsonResult(map[string]any{
		"has_key": true,
		"keys":   keyList,
	})
}

func handleSigningGenerateKey(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return errResult(err)
	}
	email, err := req.RequireString("email")
	if err != nil {
		return errResult(err)
	}
	expiry := req.GetString("expiry", "1y")

	opts := signing.GenerateKeyOptions{
		Name:   name,
		Email:  email,
		Expiry: expiry,
	}

	info, err := signing.GenerateKey(opts)
	if err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"key_id":      info.KeyID,
		"fingerprint": info.Fingerprint,
		"name":        info.Name,
		"email":       info.Email,
		"status":      "generated",
	})
}

func handleSigningRotateKey(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return errResult(err)
	}
	email, err := req.RequireString("email")
	if err != nil {
		return errResult(err)
	}
	expiry := req.GetString("expiry", "1y")

	opts := signing.GenerateKeyOptions{
		Name:   name,
		Email:  email,
		Expiry: expiry,
	}

	info, err := signing.RotateKey(opts)
	if err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"key_id":      info.KeyID,
		"fingerprint": info.Fingerprint,
		"status":      "rotated",
	})
}

func handleSigningSign(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return errResult(err)
	}

	if err := signing.SignArtifacts(path); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"path":   path,
		"status": "signed",
	})
}

func handleSigningVerify(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	path, err := req.RequireString("path")
	if err != nil {
		return errResult(err)
	}

	if err := signing.VerifyArtifacts(path); err != nil {
		return jsonResult(map[string]any{
			"path":     path,
			"verified": false,
			"error":    err.Error(),
		})
	}

	return jsonResult(map[string]any{
		"path":     path,
		"verified": true,
	})
}

func handleSigningExportBackup(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	email, err := req.RequireString("email")
	if err != nil {
		return errResult(err)
	}
	outputPath, err := req.RequireString("output_path")
	if err != nil {
		return errResult(err)
	}

	if err := signing.ExportEncryptedBackup(email, outputPath); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"output_path": outputPath,
		"status":      "exported",
	})
}

func handleSigningImportBackup(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	backupPath, err := req.RequireString("backup_path")
	if err != nil {
		return errResult(err)
	}

	if err := signing.ImportEncryptedBackup(backupPath); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"backup_path": backupPath,
		"status":      "imported",
	})
}
