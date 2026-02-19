// SPDX-License-Identifier: Apache-2.0
package init

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/Work-Fort/Anvil/pkg/config"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// InitFlags holds the CLI flags for non-interactive mode
type InitFlags struct {
	KeyName         string
	KeyEmail        string
	KeyExpiry       string
	KeyFormat       string
	HistoryFormat   string
	ArchiveLocation string
}

// package-level flag variables bound to cobra flags
var (
	flagKeyName         string
	flagKeyEmail        string
	flagKeyExpiry       string
	flagKeyFormat       string
	flagHistoryFormat   string
	flagArchiveLocation string
)

// GetInitCmd returns the cobra command for the init subcommand.
// This is the exported entry point used by cmd/root.go.
func GetInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a kernel release repository",
		Long: `Sets up the current directory as a kernel release/signing repository.

Creates:
  - anvil.yaml (repo configuration)
  - configs/ directory with minimal kernel config templates
  - Encrypted signing keys in the keys/ directory
  - .gitignore for build artifacts

Interactive mode (default when stdin is a terminal and use-tui is true):
  Launches a step-by-step wizard to collect settings and generate files.

Non-interactive mode (--key-name and --key-email required):
  Creates the repository using the provided flags without any prompts.
  The key encryption password is read from the ANVIL_SIGNING_PASSWORD
  environment variable or from stdin (piped input).`,
		Example: `  # Interactive wizard (when stdin is a TTY)
  anvil init

  # Non-interactive (password via environment variable)
  ANVIL_SIGNING_PASSWORD="secret" anvil init \
    --key-name "ACME Kernels" \
    --key-email "releases@acme.com"

  # Non-interactive (password via stdin)
  echo "secret" | anvil init \
    --key-name "ACME Kernels" \
    --key-email "releases@acme.com"`,
		RunE: runInit,
	}

	cmd.Flags().StringVar(&flagKeyName, "key-name", "", "Signing key name (required in non-interactive mode)")
	cmd.Flags().StringVar(&flagKeyEmail, "key-email", "", "Signing key email (required in non-interactive mode)")
	cmd.Flags().StringVar(&flagKeyExpiry, "key-expiry", "1y", "Key expiry duration (0=never, 1y, 2y, 5y)")
	cmd.Flags().StringVar(&flagKeyFormat, "key-format", "armored", "Private key format (armored, binary)")
	cmd.Flags().StringVar(&flagHistoryFormat, "history-format", "armored", "Public key history format (armored, binary)")
	cmd.Flags().StringVar(&flagArchiveLocation, "archive-location", "archive", "Local archive directory (must be a relative path inside the repo)")

	return cmd
}

// runInit is the cobra RunE handler
func runInit(cmd *cobra.Command, args []string) error {
	if err := validatePreFlight(); err != nil {
		return err
	}

	if err := validateArchiveLocation(flagArchiveLocation); err != nil {
		return err
	}

	if shouldUseTUI() {
		return runInteractive()
	}

	flags := InitFlags{
		KeyName:         flagKeyName,
		KeyEmail:        flagKeyEmail,
		KeyExpiry:       flagKeyExpiry,
		KeyFormat:       flagKeyFormat,
		HistoryFormat:   flagHistoryFormat,
		ArchiveLocation: flagArchiveLocation,
	}
	return runNonInteractiveWithFlags(flags)
}

// isInteractive reports whether stdin is a terminal (TTY)
func isInteractive() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// shouldUseTUI returns true when stdin is a TTY AND use-tui is enabled in config
func shouldUseTUI() bool {
	return isInteractive() && config.GetUseTUI()
}

// validateArchiveLocation ensures the archive location is a relative path that
// does not already exist as a regular file.
func validateArchiveLocation(location string) error {
	if filepath.IsAbs(location) {
		return fmt.Errorf("--archive-location must be a relative path inside the repo, got %q", location)
	}
	if info, err := os.Stat(location); err == nil && !info.IsDir() {
		return fmt.Errorf("--archive-location %q already exists as a file", location)
	}
	return nil
}

// validatePreFlight checks whether the current directory can be initialized.
// It returns an error if anvil.yaml already exists.
func validatePreFlight() error {
	if _, err := os.Stat("anvil.yaml"); err == nil {
		return fmt.Errorf("already initialized: anvil.yaml already exists in the current directory")
	}

	// Warn (non-fatal) if the directory is not a git repository
	if _, err := os.Stat(".git"); os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "warning: not a git repository - consider running 'git init' first")
	}

	return nil
}

// runInteractive launches the Bubble Tea TUI wizard
func runInteractive() error {
	p := tea.NewProgram(NewWizardModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("wizard failed: %w", err)
	}
	return nil
}

// runNonInteractiveWithFlags runs the init process using the provided flags.
// It validates required fields, generates repository files, and prints a success message.
func runNonInteractiveWithFlags(flags InitFlags) error {
	if flags.KeyName == "" || flags.KeyEmail == "" {
		return fmt.Errorf("--key-name and --key-email are required in non-interactive mode")
	}

	// Password is read from stdin or ENV — never from a flag
	password, err := signing.GetSigningPassword(
		signing.PasswordSourceAuto,
		"Enter password to encrypt signing key",
	)
	if err != nil {
		return fmt.Errorf("key password required: use %s env var or pipe via stdin: %w",
			signing.EnvSigningPassword, err)
	}

	settings := initpkg.InitSettings{
		ArchiveLocation: flags.ArchiveLocation,
		KeyName:         flags.KeyName,
		KeyEmail:        flags.KeyEmail,
		KeyExpiry:       flags.KeyExpiry,
		KeyFormat:       flags.KeyFormat,
		HistoryFormat:   flags.HistoryFormat,
		KeyPassword:     password,
	}

	files, err := initpkg.GenerateRepoFiles(settings)
	if err != nil {
		return err
	}

	// Print success message
	theme := config.CurrentTheme
	fmt.Println(theme.SuccessMessage("Repository initialized successfully"))
	fmt.Println()
	for _, file := range files {
		fmt.Println(theme.CompleteIndicator() + " " + file)
	}
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Customize kernel configs in configs/")
	fmt.Println("  2. Build kernels: anvil build-kernel")
	fmt.Println("  3. Commit to git: git add . && git commit")

	return nil
}
