// SPDX-License-Identifier: Apache-2.0
package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/Work-Fort/Anvil/pkg/config"
)

// LayoutDimensions holds calculated dimensions for a TUI layout
type LayoutDimensions struct {
	Width             int
	Height            int
	PaneContentWidth  int
	PaneRenderedWidth int
	ShowInstructions  bool
	BlankLineCount    int
	ContentHeight     int
}

// CalculateSplitPaneDimensions calculates dimensions for a 50/50 split-pane layout
// following the width calculation pattern from cli/AGENTS.md
//
// CRITICAL: Lipgloss width behavior (as of v1.1.1):
// - Style.Width(w) sets content width INCLUDING padding (padding is inside)
// - Border is rendered OUTSIDE of Style.Width() (adds to final render)
// - Actual rendered width = Style.Width() + border width
//
// Example for terminal width 130:
// - Gap between panes: 2 chars
// - Target per pane: (130 - 2) / 2 = 64 chars rendered
// - Border overhead: 2 chars (RoundedBorder = 1 char per side)
// - Content width to set: 64 - 2 = 62 chars
// - Rendered: 62 content + 2 border = 64 chars ✓
func CalculateSplitPaneDimensions(terminalWidth, terminalHeight int) LayoutDimensions {
	const gap = 2
	paneRenderedWidth := (terminalWidth - gap) / 2
	const borderWidth = 2 // All border types are 2 chars wide (1 per side)
	paneContentWidth := paneRenderedWidth - borderWidth

	return LayoutDimensions{
		Width:             terminalWidth,
		Height:            terminalHeight,
		PaneContentWidth:  paneContentWidth,
		PaneRenderedWidth: paneRenderedWidth,
	}
}

// CalculateContentHeight calculates available content height with graceful degradation
// Uses named constants pattern from cli/AGENTS.md to avoid magic numbers
//
// Parameters:
//   - terminalHeight: total terminal height
//   - required: map of required UI elements and their line counts
//   - optional: map of optional UI elements and their line counts
//   - minContentHeight: minimum acceptable content height
//
// Returns dimensions with ShowInstructions and BlankLineCount set based on available space
func CalculateContentHeight(terminalHeight int, required, optional map[string]int, minContentHeight int) LayoutDimensions {
	// Calculate overheads
	requiredOverhead := 0
	for _, lines := range required {
		requiredOverhead += lines
	}

	optionalOverhead := 0
	for _, lines := range optional {
		optionalOverhead += lines
	}

	availableHeight := terminalHeight - requiredOverhead

	dims := LayoutDimensions{
		Height:           terminalHeight,
		ShowInstructions: false,
		BlankLineCount:   0,
	}

	// Determine what optional elements fit
	if availableHeight >= minContentHeight+optionalOverhead {
		// Enough room for everything
		dims.ShowInstructions = true
		dims.BlankLineCount = optional["blankLines"]
		dims.ContentHeight = availableHeight - optionalOverhead
	} else if availableHeight >= minContentHeight+optional["instructionsLines"]+1 {
		// Drop blank lines, keep instructions
		dims.ShowInstructions = true
		dims.BlankLineCount = 1
		dims.ContentHeight = availableHeight - optional["instructionsLines"] - 1
	} else if availableHeight >= minContentHeight {
		// Drop everything optional
		dims.ShowInstructions = false
		dims.BlankLineCount = 0
		dims.ContentHeight = availableHeight
	} else {
		// Below minimum - enforce minimum anyway
		dims.ContentHeight = minContentHeight
	}

	return dims
}

// RenderCenteredModal renders a modal overlay centered in the terminal
// Used for progress indicators, confirmations, etc.
func RenderCenteredModal(content string, width, height int, borderColor lipgloss.Color, modalWidth int) string {
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor).
		Padding(1, 2).
		Width(modalWidth).
		Render(content)

	return lipgloss.Place(
		width, height,
		lipgloss.Center, lipgloss.Center,
		modal,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
	)
}

// RenderProgressModal renders a progress modal with title, status, indicator, and help text
func RenderProgressModal(title, statusMessage, indicator, helpText string, width, height, modalWidth int) string {
	theme := config.CurrentTheme

	titleStyled := lipgloss.NewStyle().
		Foreground(theme.GetPrimaryColor()).
		Bold(true).
		Render(title)

	statusStyled := ""
	if statusMessage != "" {
		statusStyled = lipgloss.NewStyle().
			Foreground(theme.GetMutedColor()).
			Render("\n" + statusMessage)
	}

	indicatorStyled := ""
	if indicator != "" {
		indicatorStyled = "\n" + indicator
	}

	helpStyled := lipgloss.NewStyle().
		Foreground(theme.GetMutedColor()).
		Render("\n\nPlease wait...")
	if helpText != "" {
		helpStyled = lipgloss.NewStyle().
			Foreground(theme.GetMutedColor()).
			Render("\n\n" + helpText)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, titleStyled, statusStyled, indicatorStyled, helpStyled)

	return RenderCenteredModal(content, width, height, theme.GetPrimaryColor(), modalWidth)
}

// CreatePaneStyle creates a styled pane based on active state
// Uses ThickBorder for active, NormalBorder for inactive (both 2 chars wide)
func CreatePaneStyle(isActive bool, accentColor, mutedColor lipgloss.Color, contentWidth int) lipgloss.Style {
	if isActive {
		return lipgloss.NewStyle().
			Border(lipgloss.ThickBorder()).
			BorderForeground(accentColor).
			Width(contentWidth).
			Padding(0, 1)
	}

	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(mutedColor).
		Width(contentWidth).
		Padding(0, 1)
}

// FillTerminal uses lipgloss.Place to fill terminal dimensions and eliminate gaps
func FillTerminal(content string, width, height int) string {
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, content)
}

// ConfirmationForm wraps a huh.Form for reusable Yes/No confirmations
type ConfirmationForm struct {
	form *huh.Form
	key  string
}

// NewConfirmationForm creates a new confirmation form with Y/N quick keys
// key: the field key to retrieve the result
// title: the main question text
// description: optional explanation text
// affirmative: text for "Yes" option
// negative: text for "No" option
func NewConfirmationForm(key, title, description, affirmative, negative string) *ConfirmationForm {
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Key(key).
				Title(title).
				Description(description).
				Affirmative(affirmative).
				Negative(negative),
		),
	)

	return &ConfirmationForm{
		form: form,
		key:  key,
	}
}

// Init initializes the form and returns the initial command
func (cf *ConfirmationForm) Init() tea.Cmd {
	return cf.form.Init()
}

// Update handles form updates with Y/N/ESC quick key support
// Returns: (confirmed bool, shouldProceed bool, model, cmd)
// - If Y pressed: (true, true, ...)
// - If N pressed: (false, true, ...)
// - If ESC pressed: (false, false, ...) - cancelled
// - If form completed: (result, true, ...)
// - Otherwise: (false, false, ...) - still collecting input
func (cf *ConfirmationForm) Update(msg tea.Msg) (bool, bool, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "y", "Y":
			return true, true, nil
		case "n", "N":
			return false, true, nil
		case "esc":
			return false, false, nil
		}
	}

	form, cmd := cf.form.Update(msg)
	cf.form = form.(*huh.Form)

	// Check if form is complete (arrow keys + enter)
	if cf.form.State == huh.StateCompleted {
		confirmed := cf.form.GetBool(cf.key)
		return confirmed, true, cmd
	}

	return false, false, cmd
}

// View renders the form
func (cf *ConfirmationForm) View() string {
	return cf.form.View()
}
