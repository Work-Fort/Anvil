// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"image/color"

	"charm.land/lipgloss/v2"
)

// Theme holds the application color palette and provides style builders.
// Fields use color.Color (lipgloss v2 native type) for direct use in styles.
type Theme struct {
	Primary   color.Color // Active elements, selection highlights
	Secondary color.Color // Borders, inactive but visible elements
	Muted     color.Color // Dimmed elements, help text
	Accent    color.Color // Alerts, unread indicators
	Text      color.Color // Primary text
	TextDim   color.Color // Secondary/dimmed text
	BgDark    color.Color // Dark background areas
	Success   color.Color // Success states, checkmarks
	Info      color.Color // Informational states
	Warning   color.Color // Warning states
	Error     color.Color // Error states, destructive actions
}

// CurrentTheme is the active theme used throughout the application.
// Defaults to hackerman-inspired colors; overridden by Omarchy loader on Omarchy systems.
var CurrentTheme = Theme{
	Primary:   lipgloss.Color("#82FB9C"), // Bright mint green
	Secondary: lipgloss.Color("#7cf8f7"), // Bright cyan
	Muted:     lipgloss.Color("#6a6e95"), // Purple-gray
	Accent:    lipgloss.Color("#82FB9C"), // Same as primary
	Text:      lipgloss.Color("#C8CCD4"), // Light gray text
	TextDim:   lipgloss.Color("#6B7080"), // Dimmed text
	BgDark:    lipgloss.Color("#1A1C24"), // Dark panel bg
	Success:   lipgloss.Color("#82FB9C"), // Bright mint green
	Info:      lipgloss.Color("#7cf8f7"), // Bright cyan
	Warning:   lipgloss.Color("#FFD700"), // Gold
	Error:     lipgloss.Color("#FF6B6B"), // Soft red
}

// --- Color getters (backward compat — thin wrappers over fields) ---

func (t Theme) GetPrimaryColor() color.Color   { return t.Primary }
func (t Theme) GetSecondaryColor() color.Color  { return t.Secondary }
func (t Theme) GetMutedColor() color.Color      { return t.Muted }
func (t Theme) GetAccentColor() color.Color     { return t.Accent }
func (t Theme) GetTextColor() color.Color       { return t.Text }
func (t Theme) GetTextDimColor() color.Color    { return t.TextDim }
func (t Theme) GetBgDarkColor() color.Color     { return t.BgDark }
func (t Theme) GetSuccessColor() color.Color    { return t.Success }
func (t Theme) GetInfoColor() color.Color       { return t.Info }
func (t Theme) GetWarningColor() color.Color    { return t.Warning }
func (t Theme) GetErrorColor() color.Color      { return t.Error }

// --- Style builders ---

func (t Theme) PrimaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Primary)
}

func (t Theme) SecondaryStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Secondary)
}

func (t Theme) MutedStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

func (t Theme) AccentStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Accent)
}

func (t Theme) TextStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Text)
}

func (t Theme) TextDimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.TextDim)
}

func (t Theme) SuccessStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Success).Bold(true)
}

func (t Theme) InfoStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Info)
}

func (t Theme) WarningStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Warning)
}

func (t Theme) ErrorStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Error)
}

func (t Theme) SubtleStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Muted)
}

// --- Message formatters ---

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

// --- Indicator helpers ---

func (t Theme) ActiveIndicator() string {
	return t.SuccessStyle().Render("●")
}

func (t Theme) PendingIndicator() string {
	return t.SubtleStyle().Render("○")
}

func (t Theme) WaitingIndicator() string {
	return t.SubtleStyle().Render("⏳")
}

func (t Theme) RunningIndicator() string {
	return t.InfoStyle().Render("🔄")
}

func (t Theme) CompleteIndicator() string {
	return t.SuccessStyle().Render("✓")
}

func (t Theme) ErrorIndicator() string {
	return t.ErrorStyle().Render("✗")
}

func (t Theme) WarningIndicator() string {
	return t.WarningStyle().Render("⚠")
}

func (t Theme) InfoIndicator() string {
	return t.InfoStyle().Render("ℹ")
}

// --- Key/help styles ---

func (t Theme) KeyStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.Primary).Bold(true)
}

func (t Theme) KeyDescStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(t.TextDim)
}

// --- Pane styles ---

// PaneStyleConfig holds configuration for styled panes.
type PaneStyleConfig struct {
	Border      lipgloss.Border
	BorderColor color.Color
	Width       int
	Height      int
	Padding     [2]int // [vertical, horizontal]
}

func (t Theme) PaneStyle(cfg PaneStyleConfig) lipgloss.Style {
	return lipgloss.NewStyle().
		Border(cfg.Border).
		BorderForeground(cfg.BorderColor).
		Width(cfg.Width).
		Height(cfg.Height).
		Padding(cfg.Padding[0], cfg.Padding[1])
}

func (t Theme) ActivePaneStyle(width, height int, accentColor color.Color) lipgloss.Style {
	return t.PaneStyle(PaneStyleConfig{
		Border:      lipgloss.ThickBorder(),
		BorderColor: accentColor,
		Width:       width,
		Height:      height,
		Padding:     [2]int{0, 1},
	})
}

func (t Theme) InactivePaneStyle(width, height int) lipgloss.Style {
	return t.PaneStyle(PaneStyleConfig{
		Border:      lipgloss.NormalBorder(),
		BorderColor: t.Muted,
		Width:       width,
		Height:      height,
		Padding:     [2]int{0, 1},
	})
}

// --- Layout styles ---

// RenderHeader renders a consistent header banner.
// Format: "  ANVIL  ▸  SECTION  ▸  [CONTEXT]  "
func (t Theme) RenderHeader(width int, section, context string) string {
	headerText := fmt.Sprintf("  ANVIL  ▸  %s  ▸  [%s]  ", section, context)
	return lipgloss.NewStyle().
		Foreground(t.Secondary).
		Bold(true).
		Width(width).
		Align(lipgloss.Center).
		Render(headerText)
}

// RenderFooter renders a consistent footer with box characters.
func (t Theme) RenderFooter(width int, content string) string {
	footerText := "╰─ " + content + " ─╯"
	return lipgloss.NewStyle().
		Foreground(t.Muted).
		Width(width).
		Align(lipgloss.Center).
		Render(footerText)
}
