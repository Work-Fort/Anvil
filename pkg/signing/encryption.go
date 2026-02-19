// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

// EncryptPrivateKey encrypts a private key with a passphrase using GPG
// Uses AES256 cipher for encryption, returns ASCII-armored output
func EncryptPrivateKey(keyData []byte, passphrase string) ([]byte, error) {
	if len(keyData) == 0 {
		return nil, fmt.Errorf("empty key data")
	}
	if passphrase == "" {
		return nil, fmt.Errorf("empty passphrase")
	}

	// Create temporary file for passphrase (secure, auto-cleanup)
	passphraseFile, err := os.CreateTemp("", "gpg-passphrase-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(passphraseFile.Name())

	if _, err := passphraseFile.WriteString(passphrase); err != nil {
		passphraseFile.Close()
		return nil, fmt.Errorf("failed to write passphrase: %w", err)
	}
	passphraseFile.Close()

	// Run GPG encryption
	cmd := exec.Command("gpg",
		"--batch",
		"--yes",
		"--passphrase-file", passphraseFile.Name(),
		"--armor",
		"--symmetric",
		"--cipher-algo", "AES256",
	)

	cmd.Stdin = bytes.NewReader(keyData)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("GPG encryption failed: %w\nStderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// DecryptPrivateKey decrypts a private key with a passphrase using GPG
func DecryptPrivateKey(encryptedData []byte, passphrase string) ([]byte, error) {
	if len(encryptedData) == 0 {
		return nil, fmt.Errorf("empty encrypted data")
	}
	if passphrase == "" {
		return nil, fmt.Errorf("empty passphrase")
	}

	// Create temporary file for passphrase (secure, auto-cleanup)
	passphraseFile, err := os.CreateTemp("", "gpg-passphrase-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(passphraseFile.Name())

	if _, err := passphraseFile.WriteString(passphrase); err != nil {
		passphraseFile.Close()
		return nil, fmt.Errorf("failed to write passphrase: %w", err)
	}
	passphraseFile.Close()

	// Run GPG decryption
	cmd := exec.Command("gpg",
		"--batch",
		"--yes",
		"--passphrase-file", passphraseFile.Name(),
		"--decrypt",
	)

	cmd.Stdin = bytes.NewReader(encryptedData)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("GPG decryption failed (wrong passphrase?): %w\nStderr: %s", err, stderr.String())
	}

	return stdout.Bytes(), nil
}

// IsKeyEncrypted checks if key data is encrypted (PGP message format)
// Looks for "BEGIN PGP MESSAGE" marker which indicates symmetric encryption
func IsKeyEncrypted(keyData []byte) bool {
	// Check for PGP message header (symmetric encryption)
	return bytes.Contains(keyData, []byte("BEGIN PGP MESSAGE"))
}
