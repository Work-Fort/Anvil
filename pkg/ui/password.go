// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"golang.org/x/term"
)

// isTerminal checks if stdin is a terminal (interactive) or a pipe
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// PasswordInput prompts for a single password with masked input
// If stdin is piped, reads from stdin. Otherwise shows TUI.
func PasswordInput(title, placeholder string) (string, error) {
	// Check if stdin is piped
	if !isTerminal() {
		// Read from stdin (piped input)
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return strings.TrimSpace(scanner.Text()), nil
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return "", fmt.Errorf("no input provided via stdin")
	}

	// Interactive TUI
	var password string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder(placeholder).
				EchoMode(huh.EchoModePassword).
				Value(&password),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	return password, nil
}

// PasswordInputConfirm prompts for a password twice and validates they match
// Returns the password if both entries match, error otherwise
// If stdin is piped, only reads one line (no confirmation needed for programmatic input)
func PasswordInputConfirm(title, confirmTitle string) (string, error) {
	// Check if stdin is piped
	if !isTerminal() {
		// Read single line from stdin - no confirmation needed for piped input
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			return strings.TrimSpace(scanner.Text()), nil
		}
		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("failed to read from stdin: %w", err)
		}
		return "", fmt.Errorf("no password provided via stdin")
	}

	// Interactive TUI
	var password string
	var confirm string

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title(title).
				Placeholder("Enter passphrase").
				EchoMode(huh.EchoModePassword).
				Value(&password),
		),
		huh.NewGroup(
			huh.NewInput().
				Title(confirmTitle).
				Placeholder("Re-enter passphrase").
				EchoMode(huh.EchoModePassword).
				Value(&confirm).
				Validate(func(s string) error {
					if s != password {
						return fmt.Errorf("passphrases do not match")
					}
					return nil
				}),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}

	if password != confirm {
		return "", fmt.Errorf("passphrases do not match")
	}

	return password, nil
}
