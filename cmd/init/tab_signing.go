// SPDX-License-Identifier: Apache-2.0
package init

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"

	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/charmbracelet/huh"
)

// collectSigningSettings runs a standalone huh form to gather signing key configuration.
// The huh library uses bubbletea v1 internally, so it runs its own TUI program
// and must be called before starting the bubbletea v2 wizard.
func collectSigningSettings() (*initpkg.InitSettings, error) {
	var (
		keyName            string
		keyEmail           string
		keyExpiry          string
		keyFormat          string
		histFormat         string
		keyPassword        string
		keyPasswordConfirm string
	)

	// Detect git config defaults
	if name, err := getGitConfig("user.name"); err == nil {
		keyName = name
	}
	if email, err := getGitConfig("user.email"); err == nil {
		keyEmail = email
	}

	// Set defaults
	if keyExpiry == "" {
		keyExpiry = "1y"
	}
	if keyFormat == "" {
		keyFormat = "armored"
	}
	if histFormat == "" {
		histFormat = "armored"
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Key Name").
				Description("Full name for the signing key").
				Value(&keyName).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("key name is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Key Email").
				Description("Email address for the signing key").
				Value(&keyEmail).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("key email is required")
					}
					emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
					if !emailRegex.MatchString(s) {
						return errors.New("invalid email format")
					}
					return nil
				}),

			huh.NewSelect[string]().
				Title("Key Expiry").
				Description("How long until the key expires").
				Options(
					huh.NewOption("1 year", "1y"),
					huh.NewOption("2 years", "2y"),
					huh.NewOption("5 years", "5y"),
					huh.NewOption("Never", "0"),
				).
				Value(&keyExpiry),

			huh.NewSelect[string]().
				Title("Private Key Format").
				Description("Storage format for the private key").
				Options(
					huh.NewOption("Armored (ASCII)", "armored"),
					huh.NewOption("Binary", "binary"),
				).
				Value(&keyFormat),

			huh.NewSelect[string]().
				Title("Public Key History Format").
				Description("Storage format for public key history").
				Options(
					huh.NewOption("Armored (ASCII)", "armored"),
					huh.NewOption("Binary", "binary"),
				).
				Value(&histFormat),

			huh.NewInput().
				Title("Key Password").
				Placeholder("Enter password to encrypt private key").
				EchoMode(huh.EchoModePassword).
				Value(&keyPassword).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("password is required for key encryption")
					}
					return nil
				}),

			huh.NewInput().
				Title("Confirm Password").
				Placeholder("Confirm password").
				EchoMode(huh.EchoModePassword).
				Value(&keyPasswordConfirm).
				Validate(func(s string) error {
					if s != keyPassword {
						return errors.New("passwords do not match")
					}
					return nil
				}),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return &initpkg.InitSettings{
		KeyName:       keyName,
		KeyEmail:      keyEmail,
		KeyExpiry:     keyExpiry,
		KeyFormat:     keyFormat,
		HistoryFormat: histFormat,
		KeyPassword:   keyPassword,
	}, nil
}

// getGitConfig retrieves a git config value
func getGitConfig(key string) (string, error) {
	cmd := exec.Command("git", "config", key)
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}
