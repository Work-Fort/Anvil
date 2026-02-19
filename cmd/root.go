// SPDX-License-Identifier: Apache-2.0
package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/cmd/buildkernel"
	"github.com/Work-Fort/Anvil/cmd/clean"
	configCmd "github.com/Work-Fort/Anvil/cmd/config"
	"github.com/Work-Fort/Anvil/cmd/firecracker"
	initcmd "github.com/Work-Fort/Anvil/cmd/init"
	"github.com/Work-Fort/Anvil/cmd/kernel"
	"github.com/Work-Fort/Anvil/cmd/signing"
	"github.com/Work-Fort/Anvil/cmd/update"
	"github.com/Work-Fort/Anvil/cmd/version"
	"github.com/Work-Fort/Anvil/cmd/vsock"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	// Version is set at build time via ldflags
	// -ldflags "-X github.com/Work-Fort/Anvil/cmd.Version=x.y.z"
	Version string

	// DisableUpdate is set at build time by package managers via ldflags
	// -ldflags "-X github.com/Work-Fort/Anvil/cmd.DisableUpdate=true"
	DisableUpdate string

	logLevel    string
	useTUI      bool
	debugLogger *log.Logger
)

var rootCmd = &cobra.Command{
	Use:   "anvil",
	Short: "Firecracker kernel and tooling manager",
	Long: `Anvil - Firecracker kernel and tooling manager

A CLI tool to download, manage, and switch between Firecracker kernels
and Firecracker binary versions. Implements XDG Base Directory specification
for organized file storage.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Initialize directories before any command runs
		if err := config.InitDirs(); err != nil {
			return err
		}

		// Load config files now that directories exist
		if err := config.LoadConfig(); err != nil {
			return err
		}

		// Update flag values from Viper (respects config file and env vars)
		useTUI = config.GetUseTUI()

		// Handle disabled logging first
		if logLevel == "disabled" {
			// Disable all logging
			log.SetOutput(io.Discard)
			return nil
		}

		// Configure log level from flag
		var level log.Level
		switch logLevel {
		case "debug":
			level = log.DebugLevel
		case "info":
			level = log.InfoLevel
		case "warn":
			level = log.WarnLevel
		case "error":
			level = log.ErrorLevel
		default:
			level = log.DebugLevel // Default to debug
		}

		// Always log to file in JSON format
		logFile := filepath.Join(config.GlobalPaths.DataDir, "debug.log")
		f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		// Create file logger with JSON formatting
		debugLogger = log.NewWithOptions(f, log.Options{
			ReportTimestamp: true,
			TimeFormat:      "2006-01-02T15:04:05.000Z07:00",
			Level:           level,
			ReportCaller:    true,
			Formatter:       log.JSONFormatter,
		})

		// Set as default logger
		log.SetDefault(debugLogger)

		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		// Print error with styling
		theme := config.CurrentTheme
		errorStyle := theme.ErrorStyle()
		fmt.Fprintf(os.Stderr, "%s %s\n", errorStyle.Render("Error:"), err.Error())
		os.Exit(1)
	}
}

func init() {
	// Configure logging - will be redirected to file in PersistentPreRunE
	log.SetReportTimestamp(false)
	log.SetLevel(log.InfoLevel)

	// Initialize Viper configuration
	config.InitViper()

	// Add global flags
	rootCmd.PersistentFlags().StringVarP(&logLevel, "log-level", "l", "debug", "Log level: disabled, debug, info, warn, error")
	rootCmd.PersistentFlags().BoolVar(&useTUI, "use-tui", true, "Enable terminal UI mode")

	// Bind flags to Viper for config file and environment variable support
	config.BindFlags(rootCmd.PersistentFlags())

	// Add subcommands using factory functions
	rootCmd.AddCommand(buildkernel.NewBuildKernelCmd())
	rootCmd.AddCommand(clean.NewCleanCmd())
	rootCmd.AddCommand(configCmd.NewConfigCmd())
	rootCmd.AddCommand(firecracker.NewFirecrackerCmd())
	rootCmd.AddCommand(initcmd.GetInitCmd())
	rootCmd.AddCommand(kernel.NewKernelCmd())
	rootCmd.AddCommand(signing.NewSigningCmd())
	rootCmd.AddCommand(update.NewUpdateCmd(Version, DisableUpdate))
	rootCmd.AddCommand(version.NewVersionCmd(Version))
	rootCmd.AddCommand(vsock.NewVsockCmd())

	// Set custom help, usage, and error functions
	rootCmd.SetHelpFunc(styledHelpFunc)
	rootCmd.SetUsageFunc(styledUsageFunc)
	rootCmd.SilenceUsage = true  // Don't show usage on errors
	rootCmd.SilenceErrors = true // We'll handle error printing ourselves

	// Disable default completion and provide custom one (Linux only - no powershell)
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	initCompletionCmd()
}

// initCompletionCmd creates a custom completion command for Linux shells only.
// This mirrors Cobra's default implementation from completions.go but excludes PowerShell.
func initCompletionCmd() {
	completionCmd := &cobra.Command{
		Use:   "completion",
		Short: "Generate the autocompletion script for the specified shell",
		Long: fmt.Sprintf(`Generate the autocompletion script for %s for the specified shell.
See each sub-command's help for details on how to use the generated script.
`, rootCmd.Name()),
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
	}

	// Flags for shell-specific options
	noDesc := rootCmd.CompletionOptions.DisableDescriptions
	haveNoDescFlag := !rootCmd.CompletionOptions.DisableNoDescFlag && !rootCmd.CompletionOptions.DisableDescriptions
	shortDesc := "Generate the autocompletion script for %s"

	// Bash completion (copied from Cobra's default, Linux paths only)
	bash := &cobra.Command{
		Use:   "bash",
		Short: fmt.Sprintf(shortDesc, "bash"),
		Long: fmt.Sprintf(`Generate the autocompletion script for the bash shell.

This script depends on the 'bash-completion' package.
If it is not installed already, you can install it via your OS's package manager.

To load completions in your current shell session:

	source <(%[1]s completion bash)

To load completions for every new session, execute once:

	%[1]s completion bash > /etc/bash_completion.d/%[1]s

You will need to start a new shell for this setup to take effect.
`, rootCmd.Name()),
		Args:                  cobra.NoArgs,
		DisableFlagsInUseLine: true,
		ValidArgsFunction:     cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenBashCompletionV2(os.Stdout, !noDesc)
		},
	}
	if haveNoDescFlag {
		bash.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")
	}

	// Zsh completion (copied from Cobra's default, Linux paths only)
	zsh := &cobra.Command{
		Use:   "zsh",
		Short: fmt.Sprintf(shortDesc, "zsh"),
		Long: fmt.Sprintf(`Generate the autocompletion script for the zsh shell.

If shell completion is not already enabled in your environment you will need
to enable it.  You can execute the following once:

	echo "autoload -U compinit; compinit" >> ~/.zshrc

To load completions in your current shell session:

	source <(%[1]s completion zsh)

To load completions for every new session, execute once:

	%[1]s completion zsh > "${fpath[1]}/_%[1]s"

You will need to start a new shell for this setup to take effect.
`, rootCmd.Name()),
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			if noDesc {
				return cmd.Root().GenZshCompletionNoDesc(os.Stdout)
			}
			return cmd.Root().GenZshCompletion(os.Stdout)
		},
	}
	if haveNoDescFlag {
		zsh.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")
	}

	// Fish completion (copied from Cobra's default)
	fish := &cobra.Command{
		Use:   "fish",
		Short: fmt.Sprintf(shortDesc, "fish"),
		Long: fmt.Sprintf(`Generate the autocompletion script for the fish shell.

To load completions in your current shell session:

	%[1]s completion fish | source

To load completions for every new session, execute once:

	%[1]s completion fish > ~/.config/fish/completions/%[1]s.fish

You will need to start a new shell for this setup to take effect.
`, rootCmd.Name()),
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Root().GenFishCompletion(os.Stdout, !noDesc)
		},
	}
	if haveNoDescFlag {
		fish.Flags().BoolVar(&noDesc, "no-descriptions", false, "disable completion descriptions")
	}

	// Add only Linux shells (no PowerShell)
	completionCmd.AddCommand(bash, zsh, fish)
	rootCmd.AddCommand(completionCmd)
}

// styledHelpFunc renders help output as markdown through glamour
func styledHelpFunc(cmd *cobra.Command, args []string) {
	markdown := generateHelpMarkdown(cmd)
	renderMarkdown(markdown)
}

// styledUsageFunc renders usage output as markdown through glamour
func styledUsageFunc(cmd *cobra.Command) error {
	markdown := generateUsageMarkdown(cmd)
	renderMarkdown(markdown)
	return nil
}

// GenerateHelpMarkdown creates markdown for the help output (exported for man page generation)
func GenerateHelpMarkdown(cmd *cobra.Command) string {
	return generateHelpMarkdown(cmd)
}

// generateHelpMarkdown creates markdown for the help output
func generateHelpMarkdown(cmd *cobra.Command) string {
	var md strings.Builder

	// Command name and description
	md.WriteString(fmt.Sprintf("# %s\n\n", cmd.Name()))

	if cmd.Long != "" {
		md.WriteString(fmt.Sprintf("%s\n\n", cmd.Long))
	} else if cmd.Short != "" {
		md.WriteString(fmt.Sprintf("%s\n\n", cmd.Short))
	}

	// Usage
	if cmd.Runnable() {
		md.WriteString("## Usage\n\n")
		md.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.UseLine()))
	}

	// Aliases
	if len(cmd.Aliases) > 0 {
		md.WriteString("## Aliases\n\n")
		md.WriteString(fmt.Sprintf("`%s`\n\n", strings.Join(cmd.Aliases, "`, `")))
	}

	// Available Commands
	if hasSubCommands(cmd) {
		md.WriteString("## Available Commands\n\n")
		for _, subCmd := range cmd.Commands() {
			if !subCmd.IsAvailableCommand() || subCmd.IsAdditionalHelpTopicCommand() {
				continue
			}
			md.WriteString(fmt.Sprintf("- **%s** - %s\n", subCmd.Name(), subCmd.Short))
		}
		md.WriteString("\n")
	}

	// Flags
	if cmd.HasAvailableLocalFlags() {
		md.WriteString("## Flags\n\n")
		md.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.LocalFlags().FlagUsages()))
	}

	// Global Flags
	if cmd.HasAvailableInheritedFlags() {
		md.WriteString("## Global Flags\n\n")
		md.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.InheritedFlags().FlagUsages()))
	}

	// Additional help topics
	if hasHelpSubCommands(cmd) {
		md.WriteString("## Additional Help Topics\n\n")
		for _, subCmd := range cmd.Commands() {
			if subCmd.IsAdditionalHelpTopicCommand() {
				md.WriteString(fmt.Sprintf("- **%s** - %s\n", subCmd.CommandPath(), subCmd.Short))
			}
		}
		md.WriteString("\n")
	}

	// Footer
	md.WriteString(fmt.Sprintf("Use `%s [command] --help` for more information about a command.\n", cmd.CommandPath()))

	return md.String()
}

// generateUsageMarkdown creates markdown for the usage output
func generateUsageMarkdown(cmd *cobra.Command) string {
	var md strings.Builder

	md.WriteString("## Usage\n\n")

	if cmd.Runnable() {
		md.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.UseLine()))
	}

	// Available Commands
	if hasSubCommands(cmd) {
		md.WriteString("### Available Commands\n\n")
		for _, subCmd := range cmd.Commands() {
			if !subCmd.IsAvailableCommand() || subCmd.IsAdditionalHelpTopicCommand() {
				continue
			}
			md.WriteString(fmt.Sprintf("- **%s** - %s\n", subCmd.Name(), subCmd.Short))
		}
		md.WriteString("\n")
	}

	// Flags
	if cmd.HasAvailableLocalFlags() {
		md.WriteString("### Flags\n\n")
		md.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.LocalFlags().FlagUsages()))
	}

	// Global Flags
	if cmd.HasAvailableInheritedFlags() {
		md.WriteString("### Global Flags\n\n")
		md.WriteString(fmt.Sprintf("```\n%s\n```\n\n", cmd.InheritedFlags().FlagUsages()))
	}

	return md.String()
}

// renderMarkdown renders markdown through glamour and wraps with lipgloss
func renderMarkdown(markdown string) {
	// Get terminal width if stdout is a terminal
	width := 100 // Default fallback
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			width = w
		}
	}

	// Create glamour renderer with custom style
	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		// Fallback to plain text if glamour fails
		fmt.Println(markdown)
		return
	}

	// Render markdown
	rendered, err := r.Render(markdown)
	if err != nil {
		// Fallback to plain text if rendering fails
		fmt.Println(markdown)
		return
	}

	// Trim trailing whitespace and print
	fmt.Print(strings.TrimRight(rendered, " \n"))
	fmt.Println() // Single newline at end
}

// hasSubCommands checks if command has available subcommands
func hasSubCommands(cmd *cobra.Command) bool {
	for _, subCmd := range cmd.Commands() {
		if subCmd.IsAvailableCommand() && !subCmd.IsAdditionalHelpTopicCommand() {
			return true
		}
	}
	return false
}

// hasHelpSubCommands checks if command has help subcommands
func hasHelpSubCommands(cmd *cobra.Command) bool {
	for _, subCmd := range cmd.Commands() {
		if subCmd.IsAdditionalHelpTopicCommand() {
			return true
		}
	}
	return false
}

// GetDebugLogger returns the file-based debug logger if available, otherwise returns the default logger
func GetDebugLogger() *log.Logger {
	if debugLogger != nil {
		return debugLogger
	}
	return log.Default()
}

// GetRootCommand returns the root command for external use (e.g., man page generation)
func GetRootCommand() *cobra.Command {
	return rootCmd
}

// RenderMarkdownToString renders markdown through glamour and returns the string
func RenderMarkdownToString(markdown string) (string, error) {
	// Get terminal width if stdout is a terminal
	width := 100 // Default fallback
	if term.IsTerminal(int(os.Stdout.Fd())) {
		if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
			width = w
		}
	}

	r, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(width),
	)
	if err != nil {
		return "", err
	}

	return r.Render(markdown)
}
