// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"testing"
)

func TestEncryptDecryptPrivateKey(t *testing.T) {
	tests := []struct {
		name       string
		keyData    []byte
		passphrase string
		wantErr    bool
	}{
		{
			name:       "valid encryption and decryption",
			keyData:    []byte("test key data"),
			passphrase: "test-password-123",
			wantErr:    false,
		},
		{
			name:       "empty key data",
			keyData:    []byte{},
			passphrase: "test-password",
			wantErr:    true,
		},
		{
			name:       "empty passphrase",
			keyData:    []byte("test key data"),
			passphrase: "",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			encrypted, err := EncryptPrivateKey(tt.keyData, tt.passphrase)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncryptPrivateKey() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr {
				return
			}

			// Verify encrypted data is different from original
			if string(encrypted) == string(tt.keyData) {
				t.Error("EncryptPrivateKey() encrypted data should differ from original")
			}

			// Verify encrypted data contains PGP message marker
			if !IsKeyEncrypted(encrypted) {
				t.Error("EncryptPrivateKey() result should be detected as encrypted")
			}

			// Test decryption
			decrypted, err := DecryptPrivateKey(encrypted, tt.passphrase)
			if err != nil {
				t.Errorf("DecryptPrivateKey() error = %v", err)
				return
			}

			// Verify decrypted data matches original
			if string(decrypted) != string(tt.keyData) {
				t.Errorf("DecryptPrivateKey() = %v, want %v", string(decrypted), string(tt.keyData))
			}
		})
	}
}

func TestDecryptPrivateKeyWrongPassword(t *testing.T) {
	keyData := []byte("test key data")
	correctPassword := "correct-password"
	wrongPassword := "wrong-password"

	// Encrypt with correct password
	encrypted, err := EncryptPrivateKey(keyData, correctPassword)
	if err != nil {
		t.Fatalf("EncryptPrivateKey() error = %v", err)
	}

	// Try to decrypt with wrong password
	_, err = DecryptPrivateKey(encrypted, wrongPassword)
	if err == nil {
		t.Error("DecryptPrivateKey() should fail with wrong password")
	}
}

func TestIsKeyEncrypted(t *testing.T) {
	tests := []struct {
		name     string
		keyData  []byte
		expected bool
	}{
		{
			name:     "encrypted key (PGP message)",
			keyData:  []byte("-----BEGIN PGP MESSAGE-----\ndata\n-----END PGP MESSAGE-----"),
			expected: true,
		},
		{
			name:     "unencrypted key (PGP private key)",
			keyData:  []byte("-----BEGIN PGP PRIVATE KEY BLOCK-----\ndata\n-----END PGP PRIVATE KEY BLOCK-----"),
			expected: false,
		},
		{
			name:     "unencrypted key (PGP public key)",
			keyData:  []byte("-----BEGIN PGP PUBLIC KEY BLOCK-----\ndata\n-----END PGP PUBLIC KEY BLOCK-----"),
			expected: false,
		},
		{
			name:     "random data",
			keyData:  []byte("some random data"),
			expected: false,
		},
		{
			name:     "empty data",
			keyData:  []byte{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsKeyEncrypted(tt.keyData)
			if result != tt.expected {
				t.Errorf("IsKeyEncrypted() = %v, want %v", result, tt.expected)
			}
		})
	}
}
