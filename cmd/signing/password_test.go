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
