// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds the application color scheme
type Theme struct {
	Primary   string // Bright mint green
	Secondary string // Bright cyan
	Muted     string // Muted purple-gray
	Success   string // Success/affirmative color
	Info      string // Info/neutral color
	Warning   string // Warning color
	Error     string // Error/destructive color
}

// CurrentTheme is the active theme used throughout the application
var CurrentTheme = Theme{
	Primary:   "#82FB9C", // Hackerman accent - bright mint green
	Secondary: "#7cf8f7", // Hackerman color6 - bright cyan
	Muted:     "#6a6e95", // Hackerman muted - purple-gray
	Success:   "#82FB9C", // Same as primary for consistency
	Info:      "#7cf8f7", // Same as secondary for consistency
	Warning:   "#FFD700", // Gold/yellow for warnings
	Error:     "#FF6B6B", // Soft red for errors
}

// Color getters return lipgloss.Color for easy styling

func (t Theme) GetPrimaryColor() lipgloss.Color {
	return lipgloss.Color(t.Primary)
}

func (t Theme) GetSecondaryColor() lipgloss.Color {
	return lipgloss.Color(t.Secondary)
}

func (t Theme) GetMutedColor() lipgloss.Color {
	return lipgloss.Color(t.Muted)
}

func (t Theme) GetSuccessColor() lipgloss.Color {
	return lipgloss.Color(t.Success)
}

func (t Theme) GetInfoColor() lipgloss.Color {
	return lipgloss.Color(t.Info)
}

func (t Theme) GetWarningColor() lipgloss.Color {
	return lipgloss.Color(t.Warning)
}

func (t Theme) GetErrorColor() lipgloss.Color {
	return lipgloss.Color(t.Error)
}

// Common style builders for consistent UI

func (t Theme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.GetSuccessColor()).Bold(true)
}

func (t Theme) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.GetInfoColor())
}

func (t Theme) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.GetWarningColor())
}

func (t Theme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.GetErrorColor())
}

func (t Theme) SubtleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.GetMutedColor())
}

// Message formatters with theme-appropriate icons

func (t Theme) SuccessMessage(text string) string {
	return t.SuccessStyle().Render("✓ " + text)
}

func (t Theme) InfoMessage(text string) string {
	return t.InfoStyle().Render("ℹ " + text)
}

func (t Theme) WarningMessage(text string) string {
	return t.WarningStyle().Render("⚠ " + text)
}

func (t Theme) ErrorMessage(text string) string {
	return t.ErrorStyle().Render("✗ " + text)
}

// Indicator helpers for consistent symbols across UI

// ActiveIndicator returns a solid dot for active states
func (t Theme) ActiveIndicator() string {
	return t.SuccessStyle().Render("●")
}

// PendingIndicator returns an empty circle for pending states
func (t Theme) PendingIndicator() string {
	return t.SubtleStyle().Render("○")
}

// WaitingIndicator returns an hourglass for waiting states
func (t Theme) WaitingIndicator() string {
	return t.SubtleStyle().Render("⏳")
}

// RunningIndicator returns a spinner/arrows for running states
func (t Theme) RunningIndicator() string {
	return t.InfoStyle().Render("🔄")
}

// CompleteIndicator returns a checkmark for completed states
func (t Theme) CompleteIndicator() string {
	return t.SuccessStyle().Render("✓")
}

// ErrorIndicator returns an X for error states
func (t Theme) ErrorIndicator() string {
	return t.ErrorStyle().Render("✗")
}

// WarningIndicator returns a warning symbol for warning states
func (t Theme) WarningIndicator() string {
	return t.WarningStyle().Render("⚠")
}

// InfoIndicator returns an info symbol for informational states
func (t Theme) InfoIndicator() string {
	return t.InfoStyle().Render("ℹ")
}

// PaneStyleConfig holds configuration for styled panes
type PaneStyleConfig struct {
	Border      lipgloss.Border
	BorderColor lipgloss.Color
	Width       int
	Height      int
	Padding     [2]int // [vertical, horizontal]
}

// PaneStyle creates a styled border for panes
func (t Theme) PaneStyle(config PaneStyleConfig) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(config.Border).
		BorderForeground(config.BorderColor).
		Width(config.Width).
		Height(config.Height).
		Padding(config.Padding[0], config.Padding[1])
}

// ActivePaneStyle returns styling for the active pane
func (t Theme) ActivePaneStyle(width, height int, accentColor lipgloss.Color) lipgloss.Style {
	return t.PaneStyle(PaneStyleConfig{
		Border:      lipgloss.ThickBorder(),
		BorderColor: accentColor,
		Width:       width,
		Height:      height,
		Padding:     [2]int{0, 1},
	})
}

// InactivePaneStyle returns styling for inactive panes
func (t Theme) InactivePaneStyle(width, height int) lipgloss.Style {
	return t.PaneStyle(PaneStyleConfig{
		Border:      lipgloss.NormalBorder(),
		BorderColor: t.GetMutedColor(),
		Width:       width,
		Height:      height,
		Padding:     [2]int{0, 1},
	})
}

// RenderHeader renders a consistent header banner across all TUIs
// Format: "  CRACKER-BARREL  ▸  SECTION  ▸  [CONTEXT]  "
func (t Theme) RenderHeader(width int, section, context string) string {
	headerText := fmt.Sprintf("  CRACKER-BARREL  ▸  %s  ▸  [%s]  ", section, context)
	return lipgloss.NewStyle().
		Foreground(t.GetSecondaryColor()).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render(headerText)
}

// RenderFooter renders a consistent footer with box characters
// Format: "╰─ [content] ─╯"
func (t Theme) RenderFooter(width int, content string) string {
	footerText := "╰─ " + content + " ─╯"
	return lipgloss.NewStyle().
		Foreground(t.GetMutedColor()).
		Width(width).
		Align(lipgloss.Center).
		Render(footerText)
}

// TODO: Investigate dynamically pulling theme from Omarchy terminal theme
// Omarchy themes are defined in ~/.config/omarchy/themes/*.toml
// We could detect the active theme and parse it to extract colors:
//   - Read current theme from Omarchy config
//   - Parse TOML to extract color values
//   - Map Omarchy color names to our theme struct
// This would allow the CLI to automatically match the user's terminal theme
// See: https://github.com/get-virgil/omarchy for theme file format
