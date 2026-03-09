# Hexagonal Architecture: Signing Domain Refactor

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task. Commit to master. Do not push.

**Goal:** Remove interactive prompts from `pkg/signing` domain functions so the MCP server never deadlocks on password acquisition.

**Architecture:** Domain functions accept passwords as parameters — they never acquire them.
CLI adapters (`cmd/signing/`) use TUI prompts + env var + stdin. MCP adapters (`internal/mcp/`)
read from env var only. `GenerateKey` and `RotateKey` already follow this pattern; we bring
`SignArtifacts`, `ExportEncryptedBackup`, and `ImportEncryptedBackup` into compliance.

**Tech Stack:** Go, gopenpgp/v3, cobra, Bubble Tea (TUI prompts), mcp-go

---

## Background

`pkg/signing` has three functions with embedded interactive prompts that deadlock the MCP server:

```
SignArtifacts(dir)
  └─ loadPrivateKey()
       └─ GetSigningPassword(PasswordSourceAuto)
            └─ tries stdin → ENV → TUI prompt   ← reads MCP's JSON-RPC pipe, hangs forever

ExportEncryptedBackup(email, output)
  ├─ GetSigningPassword(PasswordSourceAuto)      ← prompt #1
  └─ ui.PasswordInputConfirm(...)                ← prompt #2

ImportEncryptedBackup(backupPath)
  └─ ui.PasswordInput(...)                       ← prompt
```

`GenerateKey` and `RotateKey` already do it right — they take `opts.Password` as a parameter.

**Principle:** Domain functions accept data, never acquire it.

---

### Task 1: Add password parameter to `loadPrivateKey`

**Files:**
- Modify: `pkg/signing/signing.go:723-764`

**Step 1: Change the `loadPrivateKey` function signature and body**

Replace the function at line 723:

```go
func loadPrivateKey(password string) (*crypto.Key, error) {
	privateKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	// Check if encrypted
	if IsKeyEncrypted(keyData) {
		if password == "" {
			return nil, fmt.Errorf("signing key is encrypted but no password provided — set ANVIL_SIGNING_PASSWORD or pass password explicitly")
		}

		// Decrypt
		keyData, err = DecryptPrivateKey(keyData, password)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt key: %w", err)
		}
	}

	// Parse decrypted (or unencrypted) key
	key, err := crypto.NewKeyFromArmored(string(keyData))
	if err == nil {
		return key, nil
	}

	// Try binary format
	key, err = crypto.NewKey(keyData)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse key (tried both armored and binary formats): %w",
			err,
		)
	}

	return key, nil
}
```

**Step 2: Verify it compiles (expect errors — callers not updated yet)**

Run: `mise ci 2>&1 | head -20`
Expected: Compile errors from `SignArtifactsWithFormat` (line 294) and `ImportEncryptedBackup` (line 555) — they still call `loadPrivateKey()` with no args.

---

### Task 2: Add password parameter to `SignArtifacts` and `SignArtifactsWithFormat`

**Files:**
- Modify: `pkg/signing/signing.go:278-339`

**Step 1: Update `SignArtifacts` signature**

Change line 280:
```go
// Before:
func SignArtifacts(artifactsDir string) error {
	return SignArtifactsWithFormat(artifactsDir, KeyFormatArmored)
}

// After:
func SignArtifacts(artifactsDir, password string) error {
	return SignArtifactsWithFormat(artifactsDir, KeyFormatArmored, password)
}
```

**Step 2: Update `SignArtifactsWithFormat` signature**

Change line 285:
```go
// Before:
func SignArtifactsWithFormat(artifactsDir string, format KeyFormat) error {

// After:
func SignArtifactsWithFormat(artifactsDir string, format KeyFormat, password string) error {
```

**Step 3: Update the `loadPrivateKey` call inside `SignArtifactsWithFormat`**

Change line 294:
```go
// Before:
key, err := loadPrivateKey()

// After:
key, err := loadPrivateKey(password)
```

**Step 4: Verify it compiles (expect errors from callers)**

Run: `mise ci 2>&1 | head -30`
Expected: Compile errors from `cmd/signing/sign.go:38` and `internal/mcp/tools_signing.go:224` — they still call `SignArtifacts` with old signature.

---

### Task 3: Add password parameters to `ExportEncryptedBackup`

**Files:**
- Modify: `pkg/signing/signing.go:392-449`

**Step 1: Update function signature and remove interactive calls**

Replace the function:
```go
func ExportEncryptedBackup(email, outputPath, unlockPassword, backupPassphrase string) error {
	// Check if output file already exists - MUST fail if it does
	if _, err := os.Stat(outputPath); err == nil {
		return fmt.Errorf(
			"output file already exists: %s (will not overwrite)",
			outputPath,
		)
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("failed to check output file: %w", err)
	}

	privateKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")

	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return fmt.Errorf("no signing key found: %w", err)
	}

	// If key is encrypted, decrypt it first
	if IsKeyEncrypted(keyData) {
		if unlockPassword == "" {
			return fmt.Errorf("signing key is encrypted but no unlock password provided — set ANVIL_SIGNING_PASSWORD or pass password explicitly")
		}

		keyData, err = DecryptPrivateKey(keyData, unlockPassword)
		if err != nil {
			return fmt.Errorf("failed to decrypt key: %w", err)
		}
	}

	if backupPassphrase == "" {
		return fmt.Errorf("backup passphrase is required")
	}

	// Encrypt with backup passphrase
	encryptedBackup, err := EncryptPrivateKey(keyData, backupPassphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputPath, encryptedBackup, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}
```

**Step 2: Verify it compiles (expect errors from callers)**

Run: `mise ci 2>&1 | head -30`
Expected: Compile errors from `cmd/signing/export.go:47` and `internal/mcp/tools_signing.go:264`.

---

### Task 4: Add passphrase parameter to `ImportEncryptedBackup`

**Files:**
- Modify: `pkg/signing/signing.go:498-577`

**Step 1: Update function signature and remove interactive call**

Replace the function:
```go
func ImportEncryptedBackup(backupPath, backupPassphrase string) error {
	// Check if key already exists
	if keyExists() {
		return fmt.Errorf("signing key already exists - use RemoveKey() first")
	}

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	if backupPassphrase == "" {
		return fmt.Errorf("backup passphrase is required")
	}

	// Create directories
	if err := os.MkdirAll(config.GetSigningKeyLocation(), 0755); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Create temporary file for passphrase
	passphraseFile, err := os.CreateTemp("", "gpg-passphrase-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(passphraseFile.Name())

	if _, err := passphraseFile.WriteString(backupPassphrase); err != nil {
		passphraseFile.Close()
		return fmt.Errorf("failed to write passphrase: %w", err)
	}
	passphraseFile.Close()

	// Decrypt using GPG
	privateKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")
	cmd := exec.Command("gpg",
		"--batch",
		"--yes",
		"--passphrase-file", passphraseFile.Name(),
		"--decrypt",
		"--output", privateKeyPath,
		backupPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GPG decryption failed (wrong passphrase?): %w\nOutput: %s", err, output)
	}

	// Load the decrypted key to extract public key.
	// The file GPG wrote is already decrypted, so pass empty password.
	key, err := loadPrivateKey("")
	if err != nil {
		return fmt.Errorf("failed to load decrypted key: %w", err)
	}

	// Export public key
	publicKey, err := key.ToPublic()
	if err != nil {
		return fmt.Errorf("failed to extract public key: %w", err)
	}

	armoredPublicKey, err := publicKey.Armor()
	if err != nil {
		return fmt.Errorf("failed to armor public key: %w", err)
	}

	publicKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
	if err := os.WriteFile(publicKeyPath, []byte(armoredPublicKey), 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}
```

Note: The `loadPrivateKey("")` call at the end is correct — GPG has already decrypted the
file to disk, so the key on disk is unencrypted at this point. An empty password tells
`loadPrivateKey` to skip decryption (the `IsKeyEncrypted` check will return false for
an unencrypted PGP private key block).

---

### Task 5: Remove `ui` import from `pkg/signing/signing.go`

**Files:**
- Modify: `pkg/signing/signing.go:1-18`

**Step 1: Remove the `ui` import**

After Tasks 1-4, the `ui` package import is no longer used. Remove it from the import block:

```go
import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ProtonMail/gopenpgp/v3/constants"
	"github.com/ProtonMail/gopenpgp/v3/crypto"
	"github.com/ProtonMail/gopenpgp/v3/profile"
	"github.com/Work-Fort/Anvil/pkg/config"
)
```

The `GetSigningPassword` function is still defined in `password.go` (same package), but
`signing.go` no longer calls it. The `ui` import was only needed for `ui.PasswordInput`
and `ui.PasswordInputConfirm` which we removed.

**Step 2: Verify the domain package compiles**

Run: `mise ci 2>&1 | head -30`
Expected: `pkg/signing` compiles cleanly. Errors remain from callers: `cmd/signing/sign.go`,
`cmd/signing/export.go`, `cmd/signing/import.go`, `internal/mcp/tools_signing.go`.

**Step 3: Commit the domain refactor**

```bash
git add pkg/signing/signing.go
git commit -m "refactor: remove interactive prompts from signing domain functions

Domain functions now accept passwords as parameters instead of acquiring
them via stdin/ENV/TUI. This prevents MCP server deadlocks when
PasswordSourceAuto falls through to stdin (the JSON-RPC pipe).

Functions changed:
- loadPrivateKey(password string)
- SignArtifacts(dir, password string)
- SignArtifactsWithFormat(dir, format, password string)
- ExportEncryptedBackup(email, output, unlockPassword, backupPassphrase string)
- ImportEncryptedBackup(backupPath, backupPassphrase string)"
```

Note: This commit intentionally breaks callers. The next tasks fix them.

---

### Task 6: Update CLI `cmd/signing/sign.go`

**Files:**
- Modify: `cmd/signing/sign.go`

**Step 1: Add password acquisition and update the domain call**

Replace the entire file:
```go
// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
)

func newSignCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "sign [artifacts-dir]",
		Short: "Sign release artifacts",
		Long: `Sign the SHA256SUMS file in the artifacts directory using the current signing key.

If the signing key is encrypted, you will be prompted to enter the password.
The password can be provided via:
  - Interactive prompt (default)
  - Environment variable: ANVIL_SIGNING_PASSWORD
  - Stdin (for scripts)`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			artifactsDir := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Signing artifacts..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Directory:"), valueStyle.Render(artifactsDir))
			fmt.Println()

			// Acquire password at the CLI layer (interface concern)
			password, err := signing.GetSigningPassword(
				signing.PasswordSourceAuto,
				"Enter password to unlock signing key",
			)
			if err != nil {
				return fmt.Errorf("failed to get password: %w", err)
			}

			if err := signing.SignArtifacts(artifactsDir, password); err != nil {
				return fmt.Errorf("failed to sign artifacts: %w", err)
			}

			fmt.Printf("%s Artifacts signed successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("Signature:"), valueStyle.Render(artifactsDir+"/SHA256SUMS.asc"))
			fmt.Println()

			return nil
		},
	}
}
```

---

### Task 7: Update CLI `cmd/signing/export.go`

**Files:**
- Modify: `cmd/signing/export.go`

**Step 1: Add two-password acquisition and update the domain call**

Replace the entire file:
```go
// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "export [output-path]",
		Short: "Export encrypted backup of signing key",
		Long: `Export an encrypted backup of the signing key to a file.

If the signing key is encrypted at rest, you will first be prompted for
the storage password to decrypt it, then prompted for a new passphrase
to encrypt the backup.

The backup file will NOT overwrite existing files.`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			outputPath := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Exporting encrypted backup..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Output:"), valueStyle.Render(outputPath))
			fmt.Println()

			// Get email from key info
			keys, err := signing.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}
			if len(keys) == 0 {
				return fmt.Errorf("no signing key found")
			}

			// Acquire unlock password at the CLI layer (interface concern)
			unlockPassword, err := signing.GetSigningPassword(
				signing.PasswordSourceAuto,
				"Enter password to unlock signing key",
			)
			if err != nil {
				return fmt.Errorf("failed to get unlock password: %w", err)
			}

			// Acquire backup passphrase via TUI confirmation (interface concern)
			backupPassphrase, err := ui.PasswordInputConfirm(
				"Enter passphrase for backup encryption",
				"Confirm passphrase",
			)
			if err != nil {
				return fmt.Errorf("failed to get backup passphrase: %w", err)
			}

			if err := signing.ExportEncryptedBackup(keys[0].Email, outputPath, unlockPassword, backupPassphrase); err != nil {
				return fmt.Errorf("failed to export backup: %w", err)
			}

			fmt.Printf("%s Encrypted backup exported successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("File:"), valueStyle.Render(outputPath))
			fmt.Println()

			return nil
		},
	}
}
```

---

### Task 8: Update CLI `cmd/signing/import.go`

**Files:**
- Modify: `cmd/signing/import.go`

**Step 1: Add passphrase acquisition and update the domain call**

Replace the entire file:
```go
// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"github.com/spf13/cobra"
)

func newImportCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "import [backup-file]",
		Short: "Import a signing key from encrypted backup",
		Long:  `Restore a signing key from an encrypted backup file.`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			backupPath := args[0]

			theme := config.CurrentTheme
			subtleStyle := theme.SubtleStyle()
			successStyle := theme.SuccessStyle()
			labelStyle := theme.SubtleStyle()
			valueStyle := theme.InfoStyle()

			fmt.Println()
			fmt.Println(subtleStyle.Render("Importing signing key from encrypted backup..."))
			fmt.Printf("  %s %s\n", labelStyle.Render("Backup:"), valueStyle.Render(backupPath))
			fmt.Println()

			// Acquire passphrase at the CLI layer (interface concern)
			passphrase, err := ui.PasswordInput(
				"Enter passphrase to decrypt backup",
				"Enter passphrase",
			)
			if err != nil {
				return fmt.Errorf("failed to get passphrase: %w", err)
			}

			if err := signing.ImportEncryptedBackup(backupPath, passphrase); err != nil {
				return fmt.Errorf("failed to import from backup: %w", err)
			}

			// Get imported key info
			keys, err := signing.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}
			if len(keys) == 0 {
				return fmt.Errorf("key imported but not found")
			}

			fmt.Printf("%s Signing key imported successfully!\n", successStyle.Render("✓"))
			fmt.Println()
			fmt.Printf("  %s %s\n", labelStyle.Render("Key ID:"), valueStyle.Render(keys[0].KeyID))
			fmt.Printf("  %s %s\n", labelStyle.Render("Fingerprint:"), valueStyle.Render(keys[0].Fingerprint))
			fmt.Println()

			return nil
		},
	}
}
```

**Step 2: Commit CLI adapter updates**

```bash
git add cmd/signing/sign.go cmd/signing/export.go cmd/signing/import.go
git commit -m "refactor: move password acquisition from domain to CLI adapter

CLI signing commands now acquire passwords via GetSigningPassword/TUI
before calling domain functions. This is the adapter-side complement
to the domain signature changes."
```

---

### Task 9: Update MCP adapter `internal/mcp/tools_signing.go`

**Files:**
- Modify: `internal/mcp/tools_signing.go`

**Step 1: Add `os` import**

Add `"os"` to the import block:
```go
import (
	"context"
	"os"
	"time"

	"github.com/Work-Fort/Anvil/pkg/signing"
	gomcp "github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)
```

**Step 2: Update `handleSigningSign` to read password from env**

Replace lines 218-232:
```go
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
```

If `ANVIL_SIGNING_PASSWORD` is empty and the key is encrypted, the domain returns
a clear error: "signing key is encrypted but no password provided". No hang.

**Step 3: Add `passphrase` param to `signing_export` tool schema and update handler**

In `registerSigningTools`, update the `signing_export` tool registration (lines 56-60):
```go
	s.AddTool(gomcp.NewTool("signing_export",
		gomcp.WithDescription("Export encrypted key backup. CLI: anvil signing export"),
		gomcp.WithString("email", gomcp.Required(), gomcp.Description("Email of the key to export")),
		gomcp.WithString("output_path", gomcp.Required(), gomcp.Description("Output file path for encrypted backup")),
		gomcp.WithString("passphrase", gomcp.Required(), gomcp.Description("Passphrase for backup encryption")),
	), handleSigningExportBackup)
```

Replace `handleSigningExportBackup` (lines 254-272):
```go
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
```

**Step 4: Add `passphrase` param to `signing_import` tool schema and update handler**

In `registerSigningTools`, update the `signing_import` tool registration (lines 62-65):
```go
	s.AddTool(gomcp.NewTool("signing_import",
		gomcp.WithDescription("Import key from encrypted backup. CLI: anvil signing import"),
		gomcp.WithString("backup_path", gomcp.Required(), gomcp.Description("Path to encrypted backup file")),
		gomcp.WithString("passphrase", gomcp.Required(), gomcp.Description("Passphrase to decrypt the backup")),
	), handleSigningImportBackup)
```

Replace `handleSigningImportBackup` (lines 274-288):
```go
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
```

**Step 5: Commit MCP adapter updates**

```bash
git add internal/mcp/tools_signing.go
git commit -m "refactor: update MCP signing handlers to pass passwords as data

MCP handlers now read ANVIL_SIGNING_PASSWORD from env and pass it to
domain functions. Export/import tools accept passphrase as a required
parameter. No more stdin reads — no more deadlocks."
```

---

### Task 10: Move `password.go` from domain to CLI layer

**Files:**
- Delete: `pkg/signing/password.go`
- Delete: `pkg/signing/password_test.go`
- Create: `cmd/signing/password.go`
- Create: `cmd/signing/password_test.go`

**Step 1: Create `cmd/signing/password.go`**

This is a copy of `pkg/signing/password.go` with the package changed from `signing` to the
`cmd/signing` package. However, `cmd/signing` is *also* `package signing` — so the function
names and types stay the same, but this file now lives in the CLI adapter layer.

**Important detail:** `cmd/signing/` already imports `pkg/signing` as `signing`. The
`password.go` functions reference `EnvSigningPassword` which is defined in `pkg/signing/password.go`.
After moving, we need `EnvSigningPassword` to still be accessible.

**Decision:** Keep `EnvSigningPassword` constant in `pkg/signing` as a new tiny file,
move everything else (functions, types) to `cmd/signing/`.

Create `pkg/signing/constants.go`:
```go
// SPDX-License-Identifier: Apache-2.0
package signing

const (
	// EnvSigningPassword is the environment variable name for signing password
	EnvSigningPassword = "ANVIL_SIGNING_PASSWORD"
)
```

Create `cmd/signing/password.go`:
```go
// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	signingpkg "github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

// PasswordSource indicates how to retrieve the signing password
type PasswordSource int

const (
	// PasswordSourceAuto tries ENV first, then falls back to TUI
	PasswordSourceAuto PasswordSource = iota
	// PasswordSourceEnv reads password from environment variable only
	PasswordSourceEnv
	// PasswordSourceStdin reads password from stdin
	PasswordSourceStdin
	// PasswordSourceTUI uses interactive TUI prompt
	PasswordSourceTUI
)

// GetSigningPassword retrieves the password using the specified source
func GetSigningPassword(source PasswordSource, prompt string) (string, error) {
	switch source {
	case PasswordSourceEnv:
		return getPasswordFromEnv()
	case PasswordSourceStdin:
		return getPasswordFromStdin()
	case PasswordSourceTUI:
		return getPasswordFromTUI(prompt)
	case PasswordSourceAuto:
		// Try stdin first (if data available)
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Stdin is a pipe, not a terminal
			password, err := getPasswordFromStdin()
			if err == nil {
				return password, nil
			}
		}

		// Try ENV second
		password, err := getPasswordFromEnv()
		if err == nil {
			return password, nil
		}

		// Fall back to TUI
		return getPasswordFromTUI(prompt)
	default:
		return "", fmt.Errorf("invalid password source: %d", source)
	}
}

// getPasswordFromEnv retrieves password from environment variable
func getPasswordFromEnv() (string, error) {
	password := os.Getenv(signingpkg.EnvSigningPassword)
	if password == "" {
		return "", fmt.Errorf("environment variable %s not set", signingpkg.EnvSigningPassword)
	}
	return password, nil
}

// getPasswordFromStdin reads password from stdin (single line, trim whitespace)
func getPasswordFromStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return "", fmt.Errorf("no input from stdin")
	}

	password := strings.TrimSpace(scanner.Text())
	if password == "" {
		return "", fmt.Errorf("empty password from stdin")
	}

	return password, nil
}

// getPasswordFromTUI uses interactive TUI prompt
func getPasswordFromTUI(prompt string) (string, error) {
	password, err := ui.PasswordInput(prompt, "Enter password")
	if err != nil {
		return "", fmt.Errorf("failed to get password from TUI: %w", err)
	}

	if password == "" {
		return "", fmt.Errorf("empty password")
	}

	return password, nil
}

// ParsePasswordSource parses a string into a PasswordSource
func ParsePasswordSource(s string) (PasswordSource, error) {
	switch strings.ToLower(s) {
	case "", "auto":
		return PasswordSourceAuto, nil
	case "env":
		return PasswordSourceEnv, nil
	case "stdin":
		return PasswordSourceStdin, nil
	case "tui":
		return PasswordSourceTUI, nil
	default:
		return PasswordSourceAuto, fmt.Errorf("invalid password source: %s (valid: auto, env, stdin, tui)", s)
	}
}
```

**Step 2: Create `cmd/signing/password_test.go`**

```go
// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"os"
	"testing"

	signingpkg "github.com/Work-Fort/Anvil/pkg/signing"
)

func TestParsePasswordSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected PasswordSource
		wantErr  bool
	}{
		{"auto", "auto", PasswordSourceAuto, false},
		{"empty string (defaults to auto)", "", PasswordSourceAuto, false},
		{"env", "env", PasswordSourceEnv, false},
		{"stdin", "stdin", PasswordSourceStdin, false},
		{"tui", "tui", PasswordSourceTUI, false},
		{"uppercase ENV", "ENV", PasswordSourceEnv, false},
		{"invalid source", "invalid", PasswordSourceAuto, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePasswordSource(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParsePasswordSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if result != tt.expected {
				t.Errorf("ParsePasswordSource() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestGetPasswordFromEnv(t *testing.T) {
	origEnv := os.Getenv(signingpkg.EnvSigningPassword)
	defer func() {
		if origEnv != "" {
			os.Setenv(signingpkg.EnvSigningPassword, origEnv)
		} else {
			os.Unsetenv(signingpkg.EnvSigningPassword)
		}
	}()

	tests := []struct {
		name    string
		envVal  string
		wantErr bool
	}{
		{"valid password in env", "test-password-123", false},
		{"empty env variable", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv(signingpkg.EnvSigningPassword, tt.envVal)
			} else {
				os.Unsetenv(signingpkg.EnvSigningPassword)
			}

			password, err := getPasswordFromEnv()
			if (err != nil) != tt.wantErr {
				t.Errorf("getPasswordFromEnv() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && password != tt.envVal {
				t.Errorf("getPasswordFromEnv() = %v, want %v", password, tt.envVal)
			}
		})
	}
}
```

**Step 3: Update references in `cmd/signing/sign.go`**

Since `GetSigningPassword` and `PasswordSourceAuto` are now in the same package (`cmd/signing`),
the calls in `sign.go` change from:

```go
signing.GetSigningPassword(signing.PasswordSourceAuto, ...)
```

To simply:
```go
GetSigningPassword(PasswordSourceAuto, ...)
```

And remove the `pkg/signing` import if no longer used (it's still used for `signing.SignArtifacts`).

Updated call in `cmd/signing/sign.go`:
```go
			// Acquire password at the CLI layer (interface concern)
			password, err := GetSigningPassword(
				PasswordSourceAuto,
				"Enter password to unlock signing key",
			)
```

**Step 4: Update references in `cmd/signing/export.go`**

Same pattern — `GetSigningPassword` and `PasswordSourceAuto` are now local:
```go
			// Acquire unlock password at the CLI layer (interface concern)
			unlockPassword, err := GetSigningPassword(
				PasswordSourceAuto,
				"Enter password to unlock signing key",
			)
```

The `signing` import stays for `signing.ListKeys` and `signing.ExportEncryptedBackup`.

**Step 5: Delete old files from domain package**

Delete `pkg/signing/password.go` and `pkg/signing/password_test.go`.

**Step 6: Run tests**

Run: `mise ci`
Expected: All tests pass. The `cmd/signing` package tests run the password tests.
The `pkg/signing` encryption tests still pass.

**Step 7: Commit**

```bash
git add pkg/signing/constants.go cmd/signing/password.go cmd/signing/password_test.go \
       cmd/signing/sign.go cmd/signing/export.go
git rm pkg/signing/password.go pkg/signing/password_test.go
git commit -m "refactor: move password acquisition to CLI adapter layer

GetSigningPassword and PasswordSource types moved from pkg/signing (domain)
to cmd/signing (CLI adapter). EnvSigningPassword constant stays in
pkg/signing/constants.go as pure data.

Password acquisition is an interface concern — the CLI uses TUI/stdin/env,
the MCP adapter uses env only. The domain doesn't know or care."
```

---

### Task 11: Update `cmd/init/init.go` non-interactive path

**Files:**
- Modify: `cmd/init/init.go:163-189`

**Step 1: Replace `signing.GetSigningPassword` with direct env var read**

The non-interactive init path (line 171) calls `signing.GetSigningPassword(signing.PasswordSourceAuto, ...)`.
After moving `GetSigningPassword` out of `pkg/signing`, this reference breaks.

Since `runNonInteractiveWithFlags` is explicitly non-interactive, we don't want TUI fallback.
Replace lines 171-178 with a direct env var read:

```go
	// Non-interactive: read password from env var only (no TUI)
	password := os.Getenv(signing.EnvSigningPassword)
	if password == "" {
		return fmt.Errorf("key password required in non-interactive mode: set %s env var",
			signing.EnvSigningPassword)
	}
```

This is cleaner than the original — non-interactive mode explicitly only accepts env var,
with a clear error message.

**Step 2: Run tests**

Run: `mise ci`
Expected: All tests pass.

**Step 3: Commit**

```bash
git add cmd/init/init.go
git commit -m "refactor: simplify non-interactive init password handling

Non-interactive mode now reads password directly from ANVIL_SIGNING_PASSWORD
env var instead of going through GetSigningPassword auto-detection."
```

---

### Task 12: Clean up dead `MigrateUnencryptedKey` code

**Files:**
- Modify: `pkg/signing/migration.go`

**Step 1: Verify it's dead code**

`MigrateUnencryptedKey` is defined in `pkg/signing/migration.go` but never called from any
Go source file. Grep confirms this — only the definition and docs reference it.

**Step 2: Refactor to accept parameters (preserve for future use)**

Even though it's currently uncalled, it's a useful utility. Refactor it to follow the same
pattern: accept an interactive callback instead of prompting directly.

Replace the entire file:
```go
// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Work-Fort/Anvil/pkg/config"
)

// MigrateUnencryptedKey migrates an unencrypted private key to encrypted format.
// The caller must provide the password; this function never prompts.
// Returns (migrated bool, err error) — migrated is false if no migration was needed.
func MigrateUnencryptedKey(password string) (bool, error) {
	privateKeyPath := filepath.Join(config.GlobalPaths.KeysDir, "signing-key-private.asc")

	// Check if private key exists
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to read private key: %w", err)
	}

	// Check if already encrypted
	if IsKeyEncrypted(keyData) {
		return false, nil
	}

	if password == "" {
		return false, fmt.Errorf("password required to encrypt signing key")
	}

	// Encrypt the key
	encryptedData, err := EncryptPrivateKey(keyData, password)
	if err != nil {
		return false, fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Write encrypted key back to file
	if err := os.WriteFile(privateKeyPath, encryptedData, 0600); err != nil {
		return false, fmt.Errorf("failed to write encrypted key: %w", err)
	}

	// Update backups if they exist
	backupsDir := filepath.Join(config.GlobalPaths.KeysDir, "backups")
	if _, err := os.Stat(backupsDir); err == nil {
		entries, err := os.ReadDir(backupsDir)
		if err != nil {
			return false, fmt.Errorf("failed to read backups directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			backupPrivateKeyPath := filepath.Join(
				backupsDir,
				entry.Name(),
				"signing-key-private.asc",
			)

			if _, err := os.Stat(backupPrivateKeyPath); err == nil {
				if err := os.WriteFile(backupPrivateKeyPath, encryptedData, 0600); err != nil {
					return false, fmt.Errorf("failed to update backup %s: %w", entry.Name(), err)
				}
			}
		}
	}

	return true, nil
}
```

Key changes:
- Accepts `password string` parameter instead of prompting
- Returns `(bool, error)` — the bool indicates whether migration happened
- Removed all `fmt.Println`, `fmt.Scanln`, and `ui.PasswordInputConfirm` calls
- Removed `ui` import

**Step 3: Run tests**

Run: `mise ci`
Expected: All tests pass. No callers to update (dead code).

**Step 4: Commit**

```bash
git add pkg/signing/migration.go
git commit -m "refactor: remove interactive prompts from MigrateUnencryptedKey

Function now accepts password as parameter and returns (migrated, error).
All UI output removed — callers handle display. Consistent with the
domain-never-prompts principle."
```

---

### Task 13: Final verification

**Step 1: Run full CI**

Run: `mise ci`
Expected: All checks pass — gofmt, go vet, staticcheck, tests.

**Step 2: Build and install**

Run: `mise run build`
Expected: Clean build to `build/anvil`.

**Step 3: Verify MCP signing_sign doesn't hang**

With `ANVIL_SIGNING_PASSWORD` set: MCP `signing_sign` should work and return a result.
Without `ANVIL_SIGNING_PASSWORD`: MCP `signing_sign` should return a clear error immediately
("signing key is encrypted but no password provided"), NOT hang.

**Step 4: Verify CLI signing still works interactively**

Run: `build/anvil signing sign /tmp/test-artifacts`
Expected: Prompts for password interactively (or uses env var if set).

---

## Scope Boundary

This plan refactors **only the signing domain** (`pkg/signing`). It does NOT:

- Create `internal/domain/`, `internal/app/`, `internal/infra/` directories
- Define port interfaces
- Refactor `pkg/kernel/` or `pkg/firecracker/`
- Change the MCP server architecture

These are future phases. The kernel and firecracker packages already use callback injection
(Writer, ProgressCallback, etc.) and don't have the deadlock problem. Signing is the urgent fix.

## File Change Summary

| File | Change |
|------|--------|
| `pkg/signing/signing.go` | Add password params to `loadPrivateKey`, `SignArtifacts`, `SignArtifactsWithFormat`, `ExportEncryptedBackup`, `ImportEncryptedBackup`; remove `ui` import |
| `pkg/signing/constants.go` | New — holds `EnvSigningPassword` constant |
| `pkg/signing/migration.go` | Rewrite `MigrateUnencryptedKey` to accept password param |
| `pkg/signing/password.go` | **Deleted** — moved to CLI layer |
| `pkg/signing/password_test.go` | **Deleted** — moved to CLI layer |
| `cmd/signing/password.go` | New — `GetSigningPassword`, `PasswordSource` types (from domain) |
| `cmd/signing/password_test.go` | New — tests for password functions (from domain) |
| `cmd/signing/sign.go` | Acquire password before calling domain |
| `cmd/signing/export.go` | Acquire two passwords before calling domain |
| `cmd/signing/import.go` | Acquire passphrase before calling domain |
| `cmd/init/init.go` | Replace `GetSigningPassword` with direct env var read |
| `internal/mcp/tools_signing.go` | Add `os` import, read env var for password, add passphrase params to export/import schemas |
