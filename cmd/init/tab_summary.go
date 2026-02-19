// SPDX-License-Identifier: Apache-2.0
package init

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Work-Fort/Anvil/pkg/config"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

// Custom messages for async file generation
type generateFilesMsg struct{}
type filesGeneratedMsg struct {
	filesCreated []string
	err          error
}

// SummaryTab handles final file generation
type SummaryTab struct {
	width        int
	height       int
	settings     *initpkg.InitSettings
	filesCreated []string
	complete     bool
	err          error
	spinner      spinner.Model
}

// NewSummaryTab creates a new summary tab
func NewSummaryTab(settings *initpkg.InitSettings) *SummaryTab {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(config.CurrentTheme.GetSecondaryColor())

	return &SummaryTab{
		settings: settings,
		spinner:  s,
	}
}

// Init implements TabModel interface
// Auto-generates repository files using async message flow
func (t *SummaryTab) Init() tea.Cmd {
	return tea.Batch(
		t.spinner.Tick,
		func() tea.Msg { return generateFilesMsg{} },
	)
}

// Update implements TabModel interface
func (t *SummaryTab) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.width = msg.Width
		t.height = msg.Height
		return t, nil

	case generateFilesMsg:
		// Trigger actual file generation
		return t, t.generateFiles()

	case filesGeneratedMsg:
		t.err = msg.err
		t.complete = true
		if t.err == nil {
			// Store created files
			t.filesCreated = msg.filesCreated
		}
		return t, nil

	case spinner.TickMsg:
		if !t.complete {
			var cmd tea.Cmd
			t.spinner, cmd = t.spinner.Update(msg)
			return t, cmd
		}

	case tea.KeyMsg:
		// Handle user input after generation
		if t.complete {
			if t.err != nil {
				// Error state: allow retry or quit
				switch msg.String() {
				case "r":
					// Retry generation
					t.err = nil
					t.complete = false
					t.filesCreated = nil
					return t, tea.Batch(
						t.spinner.Tick,
						func() tea.Msg { return generateFilesMsg{} },
					)
				case "q", "ctrl+c":
					return t, tea.Quit
				}
			} else {
				// Success state: allow exit
				switch msg.String() {
				case "enter", "q":
					return t, tea.Quit
				}
			}
		}
	}

	return t, nil
}

// generateFiles performs the actual file generation
func (t *SummaryTab) generateFiles() tea.Cmd {
	return func() tea.Msg {
		files, err := initpkg.GenerateRepoFiles(*t.settings)
		return filesGeneratedMsg{
			filesCreated: files,
			err:          err,
		}
	}
}

// View implements TabModel interface
func (t *SummaryTab) View() string {
	theme := config.CurrentTheme

	// Show error if generation failed
	if t.err != nil {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			theme.ErrorMessage("Failed to generate repository files"),
			"",
			theme.SubtleStyle().Render(t.err.Error()),
			"",
			theme.SubtleStyle().Render("Press r to retry, q to quit"),
			"",
		)
	}

	// Show generating state
	if !t.complete {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			"",
			t.spinner.View()+" Generating repository files...",
			"",
			theme.SubtleStyle().Render("Creating configuration files and directory structure."),
		)
	}

	// Show complete state with created files
	if t.complete && t.err == nil {
		title := lipgloss.NewStyle().
			Bold(true).
			Foreground(theme.GetPrimaryColor()).
			Render("Repository Initialized Successfully")

		// Build list of created files
		fileLines := make([]string, 0, len(t.filesCreated))
		for _, file := range t.filesCreated {
			fileLines = append(fileLines, theme.CompleteIndicator()+" "+file)
		}

		contentParts := []string{
			"",
			title,
			"",
			"The following files have been created:",
			"",
		}
		contentParts = append(contentParts, fileLines...)
		contentParts = append(contentParts,
			"",
			theme.SubtleStyle().Render("Press Enter or q to exit"),
			"",
		)

		content := lipgloss.JoinVertical(lipgloss.Left, contentParts...)
		return content
	}

	// Default state (should not be reached)
	return ""
}

// IsComplete implements TabModel interface
func (t *SummaryTab) IsComplete() bool {
	return t.complete && t.err == nil
}

// GetState implements TabModel interface
func (t *SummaryTab) GetState() ui.TabState {
	if t.err != nil {
		return ui.TabError
	}
	if t.complete && t.err == nil {
		return ui.TabComplete
	}
	return ui.TabActive
}
