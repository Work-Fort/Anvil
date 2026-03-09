// SPDX-License-Identifier: Apache-2.0
package mcp

import (
	"context"
	"os"
	"time"

	"github.com/Work-Fort/Anvil/pkg/signing"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerSigningTools(s *server.MCPServer) {
	s.AddTool(gomcp.NewTool("signing_list",
		gomcp.WithDescription("List all signing keys in the local keyring. CLI: anvil signing list"),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleSigningList)

	s.AddTool(gomcp.NewTool("signing_check_expiry",
		gomcp.WithDescription("Check if signing keys are expiring soon. CLI: anvil signing check-expiry"),
		gomcp.WithNumber("days", gomcp.Description("Warn if key expires within this many days (default: 60)")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleSigningCheckExpiry)

	s.AddTool(gomcp.NewTool("signing_remove",
		gomcp.WithDescription("Remove the signing key from the local keyring. CLI: anvil signing remove"),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleSigningRemove)

	s.AddTool(gomcp.NewTool("signing_generate",
		gomcp.WithDescription("Generate a new PGP signing key. CLI: anvil signing generate"),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Key holder name")),
		gomcp.WithString("email", gomcp.Required(), gomcp.Description("Key holder email")),
		gomcp.WithString("expiry", gomcp.Description("Key expiry duration (e.g. 1y, 6m). Default: 1y")),
	), handleSigningGenerateKey)

	s.AddTool(gomcp.NewTool("signing_rotate",
		gomcp.WithDescription("Rotate the signing key (generates new, removes old). CLI: anvil signing rotate"),
		gomcp.WithString("name", gomcp.Required(), gomcp.Description("Key holder name")),
		gomcp.WithString("email", gomcp.Required(), gomcp.Description("Key holder email")),
		gomcp.WithString("expiry", gomcp.Description("Key expiry duration (default: 1y)")),
		gomcp.WithDestructiveHintAnnotation(true),
	), handleSigningRotateKey)

	s.AddTool(gomcp.NewTool("signing_sign",
		gomcp.WithDescription("Sign artifacts in a directory. CLI: anvil signing sign"),
		gomcp.WithString("path", gomcp.Required(), gomcp.Description("Path to artifacts directory")),
	), handleSigningSign)

	s.AddTool(gomcp.NewTool("signing_verify",
		gomcp.WithDescription("Verify signatures of artifacts in a directory. CLI: anvil signing verify"),
		gomcp.WithString("path", gomcp.Required(), gomcp.Description("Path to artifacts directory")),
		gomcp.WithReadOnlyHintAnnotation(true),
	), handleSigningVerify)

	s.AddTool(gomcp.NewTool("signing_export",
		gomcp.WithDescription("Export encrypted key backup. CLI: anvil signing export"),
		gomcp.WithString("email", gomcp.Required(), gomcp.Description("Email of the key to export")),
		gomcp.WithString("output_path", gomcp.Required(), gomcp.Description("Output file path for encrypted backup")),
		gomcp.WithString("passphrase", gomcp.Required(), gomcp.Description("Passphrase for backup encryption")),
	), handleSigningExportBackup)

	s.AddTool(gomcp.NewTool("signing_import",
		gomcp.WithDescription("Import key from encrypted backup. CLI: anvil signing import"),
		gomcp.WithString("backup_path", gomcp.Required(), gomcp.Description("Path to encrypted backup file")),
		gomcp.WithString("passphrase", gomcp.Required(), gomcp.Description("Passphrase to decrypt the backup")),
	), handleSigningImportBackup)
}

func handleSigningList(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
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
		"keys":    keyList,
	})
}

func handleSigningCheckExpiry(_ context.Context, req gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	days := req.GetInt("days", 60)

	keys, err := signing.ListKeys()
	if err != nil {
		return errResult(err)
	}

	now := time.Now()
	warnBefore := now.AddDate(0, 0, days)

	var expiring []map[string]any
	var expired []map[string]any

	for _, k := range keys {
		if k.Expires.IsZero() {
			continue
		}

		entry := map[string]any{
			"key_id":      k.KeyID,
			"fingerprint": k.Fingerprint,
			"name":        k.Name,
			"email":       k.Email,
			"expires":     k.Expires.String(),
		}

		if k.Expires.Before(now) {
			expired = append(expired, entry)
		} else if k.Expires.Before(warnBefore) {
			entry["days_remaining"] = int(time.Until(k.Expires).Hours() / 24)
			expiring = append(expiring, entry)
		}
	}

	if expiring == nil {
		expiring = []map[string]any{}
	}
	if expired == nil {
		expired = []map[string]any{}
	}

	return jsonResult(map[string]any{
		"expiring":  expiring,
		"expired":   expired,
		"all_valid": len(expiring) == 0 && len(expired) == 0,
	})
}

func handleSigningRemove(_ context.Context, _ gomcp.CallToolRequest) (*gomcp.CallToolResult, error) {
	if err := signing.RemoveKey(); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"status": "removed",
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

	password := os.Getenv("ANVIL_SIGNING_PASSWORD")

	if err := signing.SignArtifacts(path, password); err != nil {
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
	passphrase, err := req.RequireString("passphrase")
	if err != nil {
		return errResult(err)
	}

	unlockPassword := os.Getenv("ANVIL_SIGNING_PASSWORD")

	if err := signing.ExportEncryptedBackup(email, outputPath, unlockPassword, passphrase); err != nil {
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
	passphrase, err := req.RequireString("passphrase")
	if err != nil {
		return errResult(err)
	}

	if err := signing.ImportEncryptedBackup(backupPath, passphrase); err != nil {
		return errResult(err)
	}

	return jsonResult(map[string]any{
		"backup_path": backupPath,
		"status":      "imported",
	})
}
