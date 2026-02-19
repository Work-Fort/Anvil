// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

// MigrateUnencryptedKey migrates an unencrypted private key to encrypted format
// This is a one-time migration for existing keys
func MigrateUnencryptedKey() error {
	privateKeyPath := filepath.Join(config.GlobalPaths.KeysDir, "signing-key-private.asc")

	// Check if private key exists
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No key exists, no migration needed
			return nil
		}
		return fmt.Errorf("failed to read private key: %w", err)
	}

	// Check if already encrypted
	if IsKeyEncrypted(keyData) {
		// Already encrypted, no migration needed
		return nil
	}

	// Key is unencrypted, prompt user to encrypt it
	fmt.Println()
	fmt.Println("⚠ Your signing key is currently unencrypted.")
	fmt.Println()
	fmt.Println("For security, private keys should be encrypted at rest.")
	fmt.Println("Would you like to encrypt it now? (Y/n)")

	var response string
	fmt.Scanln(&response)

	if response != "" && response != "Y" && response != "y" {
		fmt.Println("Skipping encryption. You can encrypt the key later by rotating it.")
		return nil
	}

	// Get password for encryption
	password, err := ui.PasswordInputConfirm(
		"Enter password to encrypt signing key",
		"Confirm password",
	)
	if err != nil {
		return fmt.Errorf("failed to get password: %w", err)
	}

	// Encrypt the key
	encryptedData, err := EncryptPrivateKey(keyData, password)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Write encrypted key back to file
	if err := os.WriteFile(privateKeyPath, encryptedData, 0600); err != nil {
		return fmt.Errorf("failed to write encrypted key: %w", err)
	}

	// Update backups if they exist
	backupsDir := filepath.Join(config.GlobalPaths.KeysDir, "backups")
	if _, err := os.Stat(backupsDir); err == nil {
		fmt.Println()
		fmt.Println("Updating backups with encrypted key...")

		// Find all backup directories
		entries, err := os.ReadDir(backupsDir)
		if err != nil {
			return fmt.Errorf("failed to read backups directory: %w", err)
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

			// Check if backup has private key
			if _, err := os.Stat(backupPrivateKeyPath); err == nil {
				// Encrypt the backup
				if err := os.WriteFile(backupPrivateKeyPath, encryptedData, 0600); err != nil {
					return fmt.Errorf("failed to update backup %s: %w", entry.Name(), err)
				}
			}
		}
	}

	fmt.Println()
	fmt.Println("✓ Signing key encrypted successfully!")
	fmt.Println()

	return nil
}
