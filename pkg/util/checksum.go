// SPDX-License-Identifier: Apache-2.0
package util

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/log"
)

// VerifySHA256File verifies a file against a SHA256SUMS file
func VerifySHA256File(filePath, checksumsPath string) error {
	log.Debugf("Verifying SHA256 checksum for %s", filePath)

	// Calculate file hash
	fileHash, err := CalculateSHA256(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Read checksums file
	checksums, err := ParseSHA256SUMSFile(checksumsPath)
	if err != nil {
		return fmt.Errorf("failed to read checksums file: %w", err)
	}

	// Get filename to match against checksums file
	filename := filepath.Base(filePath)

	// Find expected hash
	expectedHash, found := checksums[filename]
	if !found {
		return fmt.Errorf("file %s not found in checksums", filename)
	}

	// Compare hashes (case-insensitive)
	if !strings.EqualFold(fileHash, expectedHash) {
		return fmt.Errorf("checksum mismatch for %s: expected %s, got %s", filename, expectedHash, fileHash)
	}

	log.Debugf("Checksum verified for %s", filename)
	return nil
}

// CalculateSHA256 calculates the SHA256 hash of a file
func CalculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ParseSHA256SUMSFile parses a SHA256SUMS file and returns a map of filename -> hash
func ParseSHA256SUMSFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open checksums file: %w", err)
	}
	defer file.Close()

	checksums := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// SHA256SUMS format: "hash  filename" or "hash *filename"
		// Split on whitespace
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}

		hash := parts[0]
		filename := parts[1]

		// Remove leading * if present (indicates binary mode)
		filename = strings.TrimPrefix(filename, "*")

		checksums[filename] = hash
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read checksums file: %w", err)
	}

	return checksums, nil
}
