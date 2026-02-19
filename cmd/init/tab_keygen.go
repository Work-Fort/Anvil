// SPDX-License-Identifier: Apache-2.0
package init

import (
	"fmt"
	"path/filepath"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Work-Fort/Anvil/pkg/config"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/signing"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

// keysDir is the repo-local directory where wizard-generated keys are stored.
const keysDir = "keys"

// Custom messages for async flow
type generateKeyMsg struct{}
type keyGeneratedMsg struct {
	keyPath       string
	publicKeyPath string
	err           error
}

// KeygenTab handles auto-generation of encrypted signing key
type KeygenTab struct {
	width      int
	height     int
	settings   *initpkg.InitSettings
	generating bool
	complete   bool
	err        error
	spinner    spinner.Model
}

// NewKeygenTab creates a new key generation tab
func NewKeygenTab(settings *initpkg.InitSettings) *KeygenTab {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(config.CurrentTheme.GetSecondaryColor())

	return &KeygenTab{
		settings: settings,
		spinner:  s,
	}
}

// Init implements TabModel interface
// Auto-generates signing key using async message flow
func (t *KeygenTab) Init() tea.Cmd {
	return tea.Batch(
		t.spinner.Tick,
		func() tea.Msg { return generateKeyMsg{} },
	)
}

// Update implements TabModel interface
func (t *KeygenTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case generateKeyMsg:
		// Mark as generating and trigger actual key generation
		t.generating = true
		return t, t.generateKey()

	case keyGeneratedMsg:
		t.err = msg.err
		t.complete = true
		if t.err == nil {
			// Update settings with generated key paths
			t.settings.KeyPath = msg.keyPath
			t.settings.PublicKeyPath = msg.publicKeyPath
			t.settings.KeyGenerated = true

			// Send settings update and completion notification
			return t, tea.Batch(
				func() tea.Msg { return SettingsUpdateMsg{Settings: *t.settings} },
				func() tea.Msg { return TabCompleteMsg{TabIndex: 1} },
			)
		}
		return t, nil

	case spinner.TickMsg:
		if t.generating && !t.complete {
			var cmd tea.Cmd
			t.spinner, cmd = t.spinner.Update(msg)
			return t, cmd
		}
	}

	return t, nil
}

// generateKey performs the actual key generation
func (t *KeygenTab) generateKey() tea.Cmd {
	return func() tea.Msg {
		// Convert format string to KeyFormat
		format := signing.KeyFormatArmored
		if t.settings.KeyFormat == "binary" {
			format = signing.KeyFormatBinary
		}

		opts := signing.GenerateKeyOptions{
			Name:       t.settings.KeyName,
			Email:      t.settings.KeyEmail,
			Expiry:     t.settings.KeyExpiry,
			Format:     format,
			Password:   t.settings.KeyPassword,
			OutputDir:  keysDir,
			SkipBackup: true, // keys live in the repo; no separate backup needed
		}

		// Generate the key
		if _, err := signing.GenerateKey(opts); err != nil {
			return keyGeneratedMsg{err: fmt.Errorf("failed to generate key: %w", err)}
		}

		return keyGeneratedMsg{
			keyPath:       filepath.Join(keysDir, "signing-key-private.asc"),
			publicKeyPath: filepath.Join(keysDir, "signing-key.asc"),
		}
	}
}

// View implements TabModel interface
func (t *KeygenTab) View() string {
	theme := config.CurrentTheme

	// Show error if generation failed
	if t.err != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			theme.ErrorMessage("Failed to generate signing key"),
			"",
			theme.SubtleStyle().Render(t.err.Error()),
			"",
		)
	}

	// Show generating state
	if t.generating && !t.complete {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			t.spinner.View()+" Generating encrypted signing key...",
			"",
			theme.SubtleStyle().Render("This may take a moment while generating a secure 4096-bit RSA key."),
		)
	}

	// Show complete state with key paths
	if t.complete && t.err == nil {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.GetPrimaryColor()).
			Render("Signing Key Generated")

		content := lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			title,
			"",
			"Your signing key has been generated and saved:",
			"",
			theme.CompleteIndicator()+" Private key: "+t.settings.KeyPath,
			theme.CompleteIndicator()+" Public key:  "+t.settings.PublicKeyPath,
			"",
			theme.SubtleStyle().Render("The private key is stored securely with restricted permissions."),
			"",
		)

		return content
	}

	// Default state (should not be reached)
	return ""
}

// IsComplete implements TabModel interface
func (t *KeygenTab) IsComplete() bool {
	return t.complete && t.err == nil
}

// GetState implements TabModel interface
func (t *KeygenTab) GetState() ui.TabState {
	if t.err != nil {
		return ui.TabError
	}
	if t.complete && t.err == nil {
		return ui.TabComplete
	}
	return ui.TabActive
}
