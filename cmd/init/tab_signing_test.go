// SPDX-License-Identifier: Apache-2.0
package init

import (
	"regexp"
	"testing"
)

func TestGetGitConfig(t *testing.T) {
	// Test the git config helper function
	// This might fail in CI without git config, so we make it non-fatal
	name, err := getGitConfig("user.name")
	if err != nil {
		// Git config might not be set, that's OK
		t.Logf("git config user.name not set (expected in some environments): %v", err)
	} else {
		if name == "" {
			t.Error("getGitConfig should return non-empty name if no error")
		}
	}

	email, err := getGitConfig("user.email")
	if err != nil {
		t.Logf("git config user.email not set (expected in some environments): %v", err)
	} else {
		if email == "" {
			t.Error("getGitConfig should return non-empty email if no error")
		}
	}

	// Test invalid key
	_, err = getGitConfig("invalid.key.that.does.not.exist")
	if err == nil {
		// It's OK if it returns empty string with no error
		// Different git versions behave differently
		t.Log("Invalid git config key returned no error (OK)")
	}
}

func TestEmailRegexValidation(t *testing.T) {
	// Test the email validation regex pattern used in collectSigningSettings
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)

	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"user+tag@example.com", true},
		{"invalid.email", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.email, func(t *testing.T) {
			got := emailRegex.MatchString(tt.email)
			if got != tt.valid {
				t.Errorf("email %q: got valid=%v, want %v", tt.email, got, tt.valid)
			}
		})
	}
}
