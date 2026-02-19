// SPDX-License-Identifier: Apache-2.0
package signing

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/Work-Fort/Anvil/pkg/ui"
)

// PasswordSource indicates how to retrieve the signing password
type PasswordSource int

const (
	// PasswordSourceAuto tries ENV first, then falls back to TUI
	PasswordSourceAuto PasswordSource = iota
	// PasswordSourceEnv reads password from environment variable only
	PasswordSourceEnv
	// PasswordSourceStdin reads password from stdin
	PasswordSourceStdin
	// PasswordSourceTUI uses interactive TUI prompt
	PasswordSourceTUI
)

const (
	// EnvSigningPassword is the environment variable name for signing password
	EnvSigningPassword = "ANVIL_SIGNING_PASSWORD"
)

// GetSigningPassword retrieves the password using the specified source
func GetSigningPassword(source PasswordSource, prompt string) (string, error) {
	switch source {
	case PasswordSourceEnv:
		return getPasswordFromEnv()
	case PasswordSourceStdin:
		return getPasswordFromStdin()
	case PasswordSourceTUI:
		return getPasswordFromTUI(prompt)
	case PasswordSourceAuto:
		// Try stdin first (if data available)
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			// Stdin is a pipe, not a terminal
			password, err := getPasswordFromStdin()
			if err == nil {
				return password, nil
			}
		}

		// Try ENV second
		password, err := getPasswordFromEnv()
		if err == nil {
			return password, nil
		}

		// Fall back to TUI
		return getPasswordFromTUI(prompt)
	default:
		return "", fmt.Errorf("invalid password source: %d", source)
	}
}

// getPasswordFromEnv retrieves password from environment variable
func getPasswordFromEnv() (string, error) {
	password := os.Getenv(EnvSigningPassword)
	if password == "" {
		return "", fmt.Errorf("environment variable %s not set", EnvSigningPassword)
	}
	return password, nil
}

// getPasswordFromStdin reads password from stdin (single line, trim whitespace)
func getPasswordFromStdin() (string, error) {
	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return "", fmt.Errorf("no input from stdin")
	}

	password := strings.TrimSpace(scanner.Text())
	if password == "" {
		return "", fmt.Errorf("empty password from stdin")
	}

	return password, nil
}

// getPasswordFromTUI uses interactive TUI prompt
func getPasswordFromTUI(prompt string) (string, error) {
	password, err := ui.PasswordInput(prompt, "Enter password")
	if err != nil {
		return "", fmt.Errorf("failed to get password from TUI: %w", err)
	}

	if password == "" {
		return "", fmt.Errorf("empty password")
	}

	return password, nil
}

// ParsePasswordSource parses a string into a PasswordSource
func ParsePasswordSource(s string) (PasswordSource, error) {
	switch strings.ToLower(s) {
	case "", "auto":
		return PasswordSourceAuto, nil
	case "env":
		return PasswordSourceEnv, nil
	case "stdin":
		return PasswordSourceStdin, nil
	case "tui":
		return PasswordSourceTUI, nil
	default:
		return PasswordSourceAuto, fmt.Errorf("invalid password source: %s (valid: auto, env, stdin, tui)", s)
	}
}
