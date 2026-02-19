// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"os"
	"testing"
)

func TestParsePasswordSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected PasswordSource
		wantErr  bool
	}{
		{
			name:     "auto",
			input:    "auto",
			expected: PasswordSourceAuto,
			wantErr:  false,
		},
		{
			name:     "empty string (defaults to auto)",
			input:    "",
			expected: PasswordSourceAuto,
			wantErr:  false,
		},
		{
			name:     "env",
			input:    "env",
			expected: PasswordSourceEnv,
			wantErr:  false,
		},
		{
			name:     "stdin",
			input:    "stdin",
			expected: PasswordSourceStdin,
			wantErr:  false,
		},
		{
			name:     "tui",
			input:    "tui",
			expected: PasswordSourceTUI,
			wantErr:  false,
		},
		{
			name:     "uppercase ENV",
			input:    "ENV",
			expected: PasswordSourceEnv,
			wantErr:  false,
		},
		{
			name:     "invalid source",
			input:    "invalid",
			expected: PasswordSourceAuto,
			wantErr:  true,
		},
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
	// Save and restore original env
	origEnv := os.Getenv(EnvSigningPassword)
	defer func() {
		if origEnv != "" {
			os.Setenv(EnvSigningPassword, origEnv)
		} else {
			os.Unsetenv(EnvSigningPassword)
		}
	}()

	tests := []struct {
		name    string
		envVal  string
		wantErr bool
	}{
		{
			name:    "valid password in env",
			envVal:  "test-password-123",
			wantErr: false,
		},
		{
			name:    "empty env variable",
			envVal:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envVal != "" {
				os.Setenv(EnvSigningPassword, tt.envVal)
			} else {
				os.Unsetenv(EnvSigningPassword)
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
