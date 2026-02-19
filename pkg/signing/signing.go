// SPDX-License-Identifier: Apache-2.0
package signing

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
	"github.com/Work-Fort/Anvil/pkg/ui"
)

// KeyFormat represents the PGP key file format
type KeyFormat int

const (
	// KeyFormatArmored is ASCII-armored format (.asc) - text-based, universal compatibility
	KeyFormatArmored KeyFormat = iota
	// KeyFormatBinary is binary format (.gpg) - more compact, but less portable
	KeyFormatBinary
)

// KeyInfo represents information about a PGP key
type KeyInfo struct {
	KeyID       string
	Fingerprint string
	Name        string
	Email       string
	Created     time.Time
	Expires     time.Time
}

// GenerateKeyOptions holds options for generating a PGP key
type GenerateKeyOptions struct {
	Name       string
	Email      string
	Expiry     string    // Format: 0=never, <n>=days, <n>w=weeks, <n>m=months, <n>y=years
	Format     KeyFormat // Output format for saved keys (default: KeyFormatArmored)
	SkipBackup bool      // Skip creating initial backup (used during rotation)
	Password   string    // Password for encrypting private key (empty = no encryption)
	OutputDir  string    // Directory to write keys to; defaults to GetSigningKeyLocation() when empty
}

// ListKeys lists all PGP keys in the local keyring
// Uses public key only (no password required)
func ListKeys() ([]KeyInfo, error) {
	publicKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		return []KeyInfo{}, nil
	}

	// Read the public key (no password needed)
	keyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key: %w", err)
	}

	// Parse the key
	key, err := crypto.NewKeyFromArmored(string(keyData))
	if err != nil {
		// Try binary format
		key, err = crypto.NewKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("failed to parse key: %w", err)
		}
	}

	// Get key info from entity
	entity := key.GetEntity()
	if entity == nil || entity.PrimaryKey == nil {
		return nil, fmt.Errorf("invalid key structure")
	}

	keyInfo := KeyInfo{
		KeyID:       fmt.Sprintf("%X", entity.PrimaryKey.KeyId),
		Fingerprint: fmt.Sprintf("%X", entity.PrimaryKey.Fingerprint),
		Created:     entity.PrimaryKey.CreationTime,
	}

	// Get name, email, and expiry from first identity
	for _, identity := range entity.Identities {
		if identity.UserId != nil {
			keyInfo.Name = identity.UserId.Name
			keyInfo.Email = identity.UserId.Email
		}
		if sig, err := identity.LatestValidSelfCertification(time.Now(), nil); err == nil &&
			sig != nil && sig.KeyLifetimeSecs != nil && *sig.KeyLifetimeSecs > 0 {
			keyInfo.Expires = entity.PrimaryKey.CreationTime.Add(
				time.Duration(*sig.KeyLifetimeSecs) * time.Second)
		}
		break
	}

	return []KeyInfo{keyInfo}, nil
}

// parseExpiry converts an expiry string to a key lifetime in seconds.
// Format: "" or "0" = never, <n> or <n>d = days, <n>w = weeks, <n>m = months, <n>y = years.
func parseExpiry(s string) (uint32, error) {
	s = strings.TrimSpace(s)
	if s == "" || s == "0" {
		return 0, nil
	}

	var numStr string
	var multiplier int64

	switch {
	case strings.HasSuffix(s, "y"):
		numStr, multiplier = s[:len(s)-1], 365*24*3600
	case strings.HasSuffix(s, "m"):
		numStr, multiplier = s[:len(s)-1], 30*24*3600
	case strings.HasSuffix(s, "w"):
		numStr, multiplier = s[:len(s)-1], 7*24*3600
	case strings.HasSuffix(s, "d"):
		numStr, multiplier = s[:len(s)-1], 24*3600
	default:
		numStr, multiplier = s, 24*3600 // bare number = days
	}

	n, err := strconv.ParseInt(numStr, 10, 64)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid expiry %q: expected a positive integer with optional suffix (d/w/m/y)", s)
	}
	return uint32(n * multiplier), nil
}

// GenerateKey generates a new PGP signing key
func GenerateKey(opts GenerateKeyOptions) (*KeyInfo, error) {
	// Resolve output directory; default to global keys dir
	outputDir := opts.OutputDir
	if outputDir == "" {
		outputDir = config.GetSigningKeyLocation()
		// Only guard against overwriting the global key when writing there
		if keyExists() {
			return nil, fmt.Errorf("signing key already exists - use RotateKey() or RemoveKey() first")
		}
	}

	// Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create keys directory: %w", err)
	}

	lifetimeSecs, err := parseExpiry(opts.Expiry)
	if err != nil {
		return nil, err
	}

	// Use RFC4880 profile for RSA 4096-bit keys
	pgp := crypto.PGPWithProfile(profile.RFC4880())

	keyGen := pgp.KeyGeneration().AddUserId(opts.Name, opts.Email)
	if lifetimeSecs > 0 {
		keyGen = keyGen.Lifetime(int32(lifetimeSecs))
	}

	key, err := keyGen.New().GenerateKeyWithSecurity(constants.HighSecurity)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	// Extract public key
	publicKey, err := key.ToPublic()
	if err != nil {
		return nil, fmt.Errorf("failed to extract public key: %w", err)
	}

	// Determine output format (default to armored for compatibility)
	format := opts.Format
	if format != KeyFormatBinary {
		format = KeyFormatArmored
	}

	// Get private key bytes
	var privateKeyData []byte
	if format == KeyFormatBinary {
		privateKeyData, err = key.Serialize()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize private key: %w", err)
		}
	} else {
		armored, err := key.Armor()
		if err != nil {
			return nil, fmt.Errorf("failed to armor private key: %w", err)
		}
		privateKeyData = []byte(armored)
	}

	// Encrypt private key if password provided
	if opts.Password != "" {
		privateKeyData, err = EncryptPrivateKey(privateKeyData, opts.Password)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt private key: %w", err)
		}
	}

	// Internal key files always use .asc regardless of content format,
	// matching the convention used by all read operations in this package.
	privateKeyPath := filepath.Join(outputDir, "signing-key-private.asc")
	if err := os.WriteFile(privateKeyPath, privateKeyData, 0600); err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	publicKeyPath := filepath.Join(outputDir, "signing-key.asc")
	if err := saveKey(publicKey, publicKeyPath, format, 0644); err != nil {
		return nil, fmt.Errorf("failed to save public key: %w", err)
	}

	// Save public key to history (using configured location and format)
	timestamp := time.Now().UTC().Format("2006-01-02-150405")
	historyLocation := config.GetSigningHistoryLocation()
	// When OutputDir is set (repo mode), history lives relative to the repo root;
	// otherwise fall back to the global data dir.
	historyBaseDir := filepath.Dir(filepath.Clean(outputDir))
	historyDir := filepath.Join(historyBaseDir, historyLocation)
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	// Determine history file format from config; extension is always .asc
	historyFormat := KeyFormatArmored
	if config.GetSigningHistoryFormat() == "binary" {
		historyFormat = KeyFormatBinary
	}

	historyPath := filepath.Join(historyDir, timestamp+".asc")
	if err := saveKey(publicKey, historyPath, historyFormat, 0644); err != nil {
		return nil, fmt.Errorf("failed to save public key to history: %w", err)
	}

	// Skip the initial backup in repo mode: the key lives under a repo-relative
	// path (e.g. "keys/") and a backups/ subdirectory would clutter the tree.
	if !opts.SkipBackup && !config.IsRepoMode() {
		backupDir := filepath.Join(outputDir, "backups", "initial-"+timestamp)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}

		if err := copyFile(publicKeyPath, filepath.Join(backupDir, "signing-key.asc")); err != nil {
			return nil, fmt.Errorf("failed to backup public key: %w", err)
		}
		if err := copyFile(privateKeyPath, filepath.Join(backupDir, "signing-key-private.asc")); err != nil {
			return nil, fmt.Errorf("failed to backup private key: %w", err)
		}
	}

	// Build key info directly from the generated key (avoids assuming global keys dir)
	entity := publicKey.GetEntity()
	if entity == nil || entity.PrimaryKey == nil {
		return nil, fmt.Errorf("generated key has invalid structure")
	}

	keyInfo := KeyInfo{
		KeyID:       fmt.Sprintf("%X", entity.PrimaryKey.KeyId),
		Fingerprint: fmt.Sprintf("%X", entity.PrimaryKey.Fingerprint),
		Created:     entity.PrimaryKey.CreationTime,
	}

	for _, identity := range entity.Identities {
		if identity.UserId != nil {
			keyInfo.Name = identity.UserId.Name
			keyInfo.Email = identity.UserId.Email
		}
		break
	}

	return &keyInfo, nil
}

// SignArtifacts signs the SHA256SUMS file in the given directory
// Uses ASCII-armored format for release asset compatibility
func SignArtifacts(artifactsDir string) error {
	return SignArtifactsWithFormat(artifactsDir, KeyFormatArmored)
}

// SignArtifactsWithFormat signs the SHA256SUMS file with specified format
func SignArtifactsWithFormat(artifactsDir string, format KeyFormat) error {
	// Find SHA256SUMS file
	sha256sumsPath := filepath.Join(artifactsDir, "SHA256SUMS")
	data, err := os.ReadFile(sha256sumsPath)
	if err != nil {
		return fmt.Errorf("failed to read SHA256SUMS: %w", err)
	}

	// Load private key
	key, err := loadPrivateKey()
	if err != nil {
		return fmt.Errorf("failed to load private key: %w", err)
	}

	// Create signing context with RFC4880 profile
	pgp := crypto.PGPWithProfile(profile.RFC4880())

	// Create signer with detached signature
	signer, err := pgp.Sign().
		SigningKey(key).
		Detached().
		New()
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}
	defer signer.ClearPrivateParams()

	// Sign the data with appropriate encoding
	encoding := crypto.Armor
	if format == KeyFormatBinary {
		encoding = crypto.Bytes
	}

	signature, err := signer.Sign(data, encoding)
	if err != nil {
		return fmt.Errorf("failed to sign data: %w", err)
	}

	// Write signature to file
	signaturePath := sha256sumsPath + ".asc"
	if err := os.WriteFile(signaturePath, signature, 0644); err != nil {
		return fmt.Errorf("failed to write signature: %w", err)
	}

	// Copy public key into the artifacts directory so consumers can verify
	pubKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
	destKeyPath := filepath.Join(artifactsDir, "signing-key.asc")
	if src, err := os.ReadFile(pubKeyPath); err == nil {
		if err := os.WriteFile(destKeyPath, src, 0644); err != nil {
			return fmt.Errorf("failed to copy public key: %w", err)
		}
	}

	return nil
}

// VerifyArtifacts verifies the PGP signature on SHA256SUMS
func VerifyArtifacts(artifactsDir string) error {
	// Find SHA256SUMS and signature files
	sha256sumsPath := filepath.Join(artifactsDir, "SHA256SUMS")
	signaturePath := sha256sumsPath + ".asc"

	data, err := os.ReadFile(sha256sumsPath)
	if err != nil {
		return fmt.Errorf("SHA256SUMS file not found: %w", err)
	}

	signature, err := os.ReadFile(signaturePath)
	if err != nil {
		return fmt.Errorf("SHA256SUMS.asc signature file not found: %w", err)
	}

	// Load public key
	publicKey, err := loadPublicKey()
	if err != nil {
		return fmt.Errorf("failed to load public key: %w", err)
	}

	// Create verification context with RFC4880 profile
	pgp := crypto.PGPWithProfile(profile.RFC4880())

	// Create verifier
	verifier, err := pgp.Verify().
		VerificationKey(publicKey).
		New()
	if err != nil {
		return fmt.Errorf("failed to create verifier: %w", err)
	}

	// Try armored format first
	verifyResult, err := verifier.VerifyDetached(data, signature, crypto.Armor)
	if err != nil {
		// Try binary format
		verifyResult, err = verifier.VerifyDetached(data, signature, crypto.Bytes)
		if err != nil {
			return fmt.Errorf("signature verification failed (tried both armored and binary formats): %w", err)
		}
	}

	// Check for signature errors
	if sigErr := verifyResult.SignatureError(); sigErr != nil {
		return fmt.Errorf("signature error: %w", sigErr)
	}

	return nil
}

// ExportEncryptedBackup exports an encrypted backup of the signing key
// Uses GPG for compatibility with existing backup workflows
func ExportEncryptedBackup(email, outputPath string) error {
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
		password, err := GetSigningPassword(
			PasswordSourceAuto,
			"Enter password to unlock signing key",
		)
		if err != nil {
			return fmt.Errorf("failed to get password: %w", err)
		}

		keyData, err = DecryptPrivateKey(keyData, password)
		if err != nil {
			return fmt.Errorf("failed to decrypt key: %w", err)
		}
	}

	// Get NEW passphrase for backup encryption
	passphrase, err := ui.PasswordInputConfirm(
		"Enter passphrase for backup encryption",
		"Confirm passphrase",
	)
	if err != nil {
		return fmt.Errorf("failed to read passphrase: %w", err)
	}

	// Encrypt with new passphrase
	encryptedBackup, err := EncryptPrivateKey(keyData, passphrase)
	if err != nil {
		return fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Write to output file
	if err := os.WriteFile(outputPath, encryptedBackup, 0600); err != nil {
		return fmt.Errorf("failed to write backup file: %w", err)
	}

	return nil
}

// ImportKey imports a signing key from armored or binary data (NOT encrypted)
// This is used for CI environments where keys are stored as secrets
func ImportKey(keyData []byte) error {
	// Check if key already exists
	if keyExists() {
		return fmt.Errorf("signing key already exists - use RemoveKey() first")
	}

	// Parse the key (auto-detect armored vs binary)
	var key *crypto.Key
	var err error

	// Try armored first
	key, err = crypto.NewKeyFromArmored(string(keyData))
	if err != nil {
		// Try binary format
		key, err = crypto.NewKey(keyData)
		if err != nil {
			return fmt.Errorf("failed to parse key (tried both armored and binary formats): %w", err)
		}
	}

	// Create directories
	if err := os.MkdirAll(config.GetSigningKeyLocation(), 0755); err != nil {
		return fmt.Errorf("failed to create keys directory: %w", err)
	}

	// Save private key (preserve input format)
	privateKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")
	if err := os.WriteFile(privateKeyPath, keyData, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	// Extract and save public key (always armored for compatibility)
	publicKey, err := key.ToPublic()
	if err != nil {
		return fmt.Errorf("failed to extract public key: %w", err)
	}

	publicKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
	if err := saveKey(publicKey, publicKeyPath, KeyFormatArmored, 0644); err != nil {
		return fmt.Errorf("failed to save public key: %w", err)
	}

	return nil
}

// ImportEncryptedBackup imports a signing key from an encrypted backup
// Uses GPG for compatibility with existing backup workflows
func ImportEncryptedBackup(backupPath string) error {
	// Check if key already exists
	if keyExists() {
		return fmt.Errorf("signing key already exists - use RemoveKey() first")
	}

	// Check if backup file exists
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		return fmt.Errorf("backup file not found: %s", backupPath)
	}

	// Get passphrase using TUI
	passphrase, err := ui.PasswordInput(
		"Enter passphrase to decrypt backup",
		"Enter passphrase",
	)
	if err != nil {
		return fmt.Errorf("failed to read passphrase: %w", err)
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

	if _, err := passphraseFile.WriteString(passphrase); err != nil {
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

	// Load the decrypted key to extract public key
	key, err := loadPrivateKey()
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

// CheckExpiry checks if the signing key will expire soon
// Returns nil if key is valid for >60 days or never expires
// Returns error if key expires in ≤60 days or has already expired
func CheckExpiry() error {
	keys, err := ListKeys()
	if err != nil {
		return fmt.Errorf("failed to list keys: %w", err)
	}
	if len(keys) == 0 {
		return fmt.Errorf("no signing key found")
	}

	key := keys[0]
	if key.Expires.IsZero() {
		// Key never expires
		return nil
	}

	daysUntilExpiry := time.Until(key.Expires).Hours() / 24
	if daysUntilExpiry <= 0 {
		return fmt.Errorf("key has already expired on %s", key.Expires.Format("2006-01-02"))
	}
	if daysUntilExpiry <= 60 {
		return fmt.Errorf("key will expire in %.0f days on %s", daysUntilExpiry, key.Expires.Format("2006-01-02"))
	}

	return nil
}

// RemoveKey removes the local signing key
func RemoveKey() error {
	// Remove exported keys
	publicKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
	privateKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")

	if err := os.Remove(publicKeyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove public key: %w", err)
	}
	if err := os.Remove(privateKeyPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove private key: %w", err)
	}

	return nil
}

// RotateKey backs up the current key and generates a new one
func RotateKey(opts GenerateKeyOptions) (*KeyInfo, error) {
	// Check if current key exists
	if !keyExists() {
		return nil, fmt.Errorf("no existing key to rotate - use GenerateKey() instead")
	}

	// Back up the current key before replacing it, unless in repo mode
	// (where the key lives in a repo-relative directory and backups would
	// clutter the working tree).
	if !config.IsRepoMode() {
		timestamp := time.Now().UTC().Format("2006-01-02-150405")
		backupDir := filepath.Join(config.GetSigningKeyLocation(), "backups", timestamp)
		if err := os.MkdirAll(backupDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create backup directory: %w", err)
		}

		publicKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
		privateKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")

		if _, err := os.Stat(publicKeyPath); err == nil {
			if err := copyFile(publicKeyPath, filepath.Join(backupDir, "signing-key.asc")); err != nil {
				return nil, fmt.Errorf("failed to backup public key: %w", err)
			}
		}
		if _, err := os.Stat(privateKeyPath); err == nil {
			if err := copyFile(privateKeyPath, filepath.Join(backupDir, "signing-key-private.asc")); err != nil {
				return nil, fmt.Errorf("failed to backup private key: %w", err)
			}
		}
	}

	// Remove old key
	if err := RemoveKey(); err != nil {
		return nil, fmt.Errorf("failed to remove old key: %w", err)
	}

	// Generate new key (skip backup - we already backed up the old key)
	opts.SkipBackup = true
	return GenerateKey(opts)
}

// Helper functions

func keyExists() bool {
	publicKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
	_, err := os.Stat(publicKeyPath)
	return err == nil
}

// saveKey saves a key in the specified format
func saveKey(key *crypto.Key, path string, format KeyFormat, perm os.FileMode) error {
	var data []byte
	var err error

	if format == KeyFormatBinary {
		// Binary format
		data, err = key.Serialize()
		if err != nil {
			return fmt.Errorf("failed to serialize key: %w", err)
		}
	} else {
		// ASCII-armored format (default)
		armored, err := key.Armor()
		if err != nil {
			return fmt.Errorf("failed to armor key: %w", err)
		}
		data = []byte(armored)
	}

	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to write key: %w", err)
	}

	return nil
}

// loadKey loads a key from either ASCII-armored or binary format (auto-detects)
func loadKey(path string) (*crypto.Key, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	// Try ASCII-armored first
	key, err := crypto.NewKeyFromArmored(string(keyData))
	if err == nil {
		return key, nil
	}

	// Try binary format
	key, err = crypto.NewKey(keyData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse key (tried both armored and binary formats): %w", err)
	}

	return key, nil
}

func loadPrivateKey() (*crypto.Key, error) {
	privateKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key-private.asc")
	keyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read key: %w", err)
	}

	// Check if encrypted
	if IsKeyEncrypted(keyData) {
		// Get password using auto-detection (ENV → TUI)
		password, err := GetSigningPassword(
			PasswordSourceAuto,
			"Enter password to unlock signing key",
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get password: %w", err)
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

func loadPublicKey() (*crypto.Key, error) {
	publicKeyPath := filepath.Join(config.GetSigningKeyLocation(), "signing-key.asc")
	return loadKey(publicKeyPath)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Preserve permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, srcInfo.Mode())
}
