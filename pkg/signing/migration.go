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
