// SPDX-License-Identifier: Apache-2.0
package init

import (
	"errors"
	"os/exec"
	"regexp"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/log"
	tea "github.com/charmbracelet/bubbletea"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

// SigningTab collects signing key metadata using huh.Form
type SigningTab struct {
	width, height int
	form          *huh.Form
	formComplete  bool
	// Collected values
	keyName            string
	keyEmail           string
	keyExpiry          string
	keyFormat          string
	histFormat         string
	keyPassword        string
	keyPasswordConfirm string
}

// NewSigningTab creates a new signing settings tab
func NewSigningTab() *SigningTab {
	return &SigningTab{}
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

// Init implements TabModel interface
func (t *SigningTab) Init() tea.Cmd {
	log.Debugf("signing.Init: called, form is nil=%v, w=%d h=%d", t.form == nil, t.width, t.height)

	// Detect git config defaults
	gitName, _ := getGitConfig("user.name")
	gitEmail, _ := getGitConfig("user.email")

	// Pre-fill with git defaults if available
	if gitName != "" {
		t.keyName = gitName
	}
	if gitEmail != "" {
		t.keyEmail = gitEmail
	}

	// Set default values for selects
	if t.keyExpiry == "" {
		t.keyExpiry = "1y"
	}
	if t.keyFormat == "" {
		t.keyFormat = "armored"
	}
	if t.histFormat == "" {
		t.histFormat = "armored"
	}

	// Create form with 7 fields (including password fields for key encryption)
	t.form = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Key Name").
				Description("Full name for the signing key").
				Value(&t.keyName).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("key name is required")
					}
					return nil
				}),

			huh.NewInput().
				Title("Key Email").
				Description("Email address for the signing key").
				Value(&t.keyEmail).
				Validate(func(s string) error {
					if s == "" {
						return errors.New("key email is required")
					}
					// Email regex validation
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
				Value(&t.keyExpiry),

			huh.NewSelect[string]().
				Title("Private Key Format").
				Description("Storage format for the private key").
				Options(
					huh.NewOption("Armored (ASCII)", "armored"),
					huh.NewOption("Binary", "binary"),
				).
				Value(&t.keyFormat),

			huh.NewSelect[string]().
				Title("Public Key History Format").
				Description("Storage format for public key history").
				Options(
					huh.NewOption("Armored (ASCII)", "armored"),
					huh.NewOption("Binary", "binary"),
				).
				Value(&t.histFormat),

			huh.NewInput().
				Title("Key Password").
				Placeholder("Enter password to encrypt private key").
				EchoMode(huh.EchoModePassword).
				Value(&t.keyPassword).
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
				Value(&t.keyPasswordConfirm).
				Validate(func(s string) error {
					if s != t.keyPassword {
						return errors.New("passwords do not match")
					}
					return nil
				}),
		),
	)

	formInitCmd := t.form.Init()
	log.Debugf("signing.Init: form created, formInitCmd is nil=%v", formInitCmd == nil)
	return formInitCmd
}

// Update implements TabModel interface
func (t *SigningTab) Update(msg tea.Msg) (*SigningTab, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		log.Debugf("signing.Update: KEY=%q formIsNil=%v", keyMsg.String(), t.form == nil)
	} else {
		log.Debugf("signing.Update: msg=%T formIsNil=%v formComplete=%v", msg, t.form == nil, t.formComplete)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		// Set form width; let huh auto-size height via its own Update(WindowSizeMsg)
		if t.form != nil {
			t.form.WithWidth(msg.Width)
		}
	}

	// Delegate to form.Update() for all input handling
	var cmd tea.Cmd
	if t.form != nil {
		formStateBefore := t.form.State
		viewBefore := t.form.View()
		form, formCmd := t.form.Update(msg)
		t.form = form.(*huh.Form)
		cmd = formCmd
		viewAfter := t.form.View()
		log.Debugf("signing.Update: form.Update done stateBefore=%v stateAfter=%v cmdIsNil=%v viewLenBefore=%d viewLenAfter=%d",
			formStateBefore, t.form.State, cmd == nil, len(viewBefore), len(viewAfter))
	}

	// Check if form completed
	if t.form != nil && t.form.State == huh.StateCompleted && !t.formComplete {
		t.formComplete = true

		// Send SettingsUpdateMsg with populated InitSettings
		settings := initpkg.InitSettings{
			KeyName:       t.keyName,
			KeyEmail:      t.keyEmail,
			KeyExpiry:     t.keyExpiry,
			KeyFormat:     t.keyFormat,
			HistoryFormat: t.histFormat,
			KeyPassword:   t.keyPassword,
		}

		return t, tea.Batch(
			func() tea.Msg { return SettingsUpdateMsg{Settings: settings} },
			func() tea.Msg { return TabCompleteMsg{TabIndex: 0} },
			cmd,
		)
	}

	return t, cmd
}

// View implements TabModel interface
func (t *SigningTab) View() string {
	if t.form == nil {
		return ""
	}
	return t.form.View()
}

// IsComplete implements TabModel interface
func (t *SigningTab) IsComplete() bool {
	return t.formComplete
}

// GetState implements TabModel interface
func (t *SigningTab) GetState() ui.TabState {
	if t.formComplete {
		return ui.TabComplete
	}
	return ui.TabActive
}
