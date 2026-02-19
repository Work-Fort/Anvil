// SPDX-License-Identifier: Apache-2.0
package init

import (
	"os"
	"testing"

	"github.com/Work-Fort/Anvil/pkg/signing"
)

// TestShouldUseTUI_ReturnsFalse verifies shouldUseTUI() returns false when use-tui is not set
// (since in a test environment stdin is not a terminal, this should be false regardless)
func TestShouldUseTUI_ReturnsFalse(t *testing.T) {
	// In test environment stdin is not a TTY, so shouldUseTUI should be false
	// regardless of the use-tui config value
	result := shouldUseTUI()
	if result {
		t.Error("shouldUseTUI() should return false when stdin is not a terminal (test environment)")
	}
}

// TestNonInteractiveMode_AllFlags verifies non-interactive mode runs successfully with all required flags.
// Password is supplied via environment variable (never via flag).
func TestNonInteractiveMode_AllFlags(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	t.Setenv(signing.EnvSigningPassword, "test-password-123")

	flags := InitFlags{
		KeyName:       "Test Kernels",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	err = runNonInteractiveWithFlags(flags)
	if err != nil {
		t.Errorf("runNonInteractiveWithFlags() returned unexpected error: %v", err)
	}
}

// TestNonInteractiveMode_ErrorWithoutKeyName verifies error is returned when --key-name is missing
func TestNonInteractiveMode_ErrorWithoutKeyName(t *testing.T) {
	t.Setenv(signing.EnvSigningPassword, "test-password-123")

	flags := InitFlags{
		KeyName:       "",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	err := runNonInteractiveWithFlags(flags)
	if err == nil {
		t.Error("expected error when --key-name is empty, got nil")
	}
}

// TestNonInteractiveMode_ErrorWithoutKeyEmail verifies error is returned when --key-email is missing
func TestNonInteractiveMode_ErrorWithoutKeyEmail(t *testing.T) {
	t.Setenv(signing.EnvSigningPassword, "test-password-123")

	flags := InitFlags{
		KeyName:       "Test Kernels",
		KeyEmail:      "",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	err := runNonInteractiveWithFlags(flags)
	if err == nil {
		t.Error("expected error when --key-email is empty, got nil")
	}
}

// TestNonInteractiveMode_PasswordViaEnv verifies the ENV var is the authoritative password source
func TestNonInteractiveMode_PasswordViaEnv(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	t.Setenv(signing.EnvSigningPassword, "env-supplied-password")

	flags := InitFlags{
		KeyName:       "Test Kernels",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	if err := runNonInteractiveWithFlags(flags); err != nil {
		t.Errorf("runNonInteractiveWithFlags() with ENV password returned unexpected error: %v", err)
	}
}

// TestFlagDefaults verifies that GetInitCmd() sets correct default values for flags
func TestFlagDefaults(t *testing.T) {
	cmd := GetInitCmd()

	expiryFlag := cmd.Flags().Lookup("key-expiry")
	if expiryFlag == nil {
		t.Fatal("--key-expiry flag not found")
	}
	if expiryFlag.DefValue != "1y" {
		t.Errorf("--key-expiry default = %q, want %q", expiryFlag.DefValue, "1y")
	}

	formatFlag := cmd.Flags().Lookup("key-format")
	if formatFlag == nil {
		t.Fatal("--key-format flag not found")
	}
	if formatFlag.DefValue != "armored" {
		t.Errorf("--key-format default = %q, want %q", formatFlag.DefValue, "armored")
	}

	historyFormatFlag := cmd.Flags().Lookup("history-format")
	if historyFormatFlag == nil {
		t.Fatal("--history-format flag not found")
	}
	if historyFormatFlag.DefValue != "armored" {
		t.Errorf("--history-format default = %q, want %q", historyFormatFlag.DefValue, "armored")
	}

	// --key-password must not exist; password comes from stdin or ENV only
	if cmd.Flags().Lookup("key-password") != nil {
		t.Error("--key-password flag must not exist; password is read from stdin or ANVIL_SIGNING_PASSWORD")
	}
}

// TestValidatePreFlight verifies validatePreFlight() checks initialization state
func TestValidatePreFlight(t *testing.T) {
	tmpDir := t.TempDir()
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(origDir); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	}()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}

	// Should pass when not initialized
	if err := validatePreFlight(); err != nil {
		t.Errorf("validatePreFlight() returned unexpected error: %v", err)
	}

	// Create config to simulate initialized state
	if err := os.WriteFile("anvil.yaml", []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should fail when already initialized
	if err := validatePreFlight(); err == nil {
		t.Error("validatePreFlight() should return error when anvil.yaml exists")
	}
}
