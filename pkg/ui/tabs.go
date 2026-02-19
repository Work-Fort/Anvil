// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/Work-Fort/Anvil/pkg/config"
)

// TabState represents the state of a tab
type TabState int

const (
	TabPending TabState = iota
	TabActive
	TabComplete
	TabError
)

// Tab represents a single tab with state and content
type Tab struct {
	Title   string
	State   TabState
	Content string
	Spinner spinner.Model
}

// TabsConfig holds configuration for tab rendering
type TabsConfig struct {
	ActiveIndex int
	Width       int // Total width available for all tabs
}

// RenderTabs renders a set of tabs with the cyberpunk aesthetic
func RenderTabs(tabs []Tab, cfg TabsConfig) string {
	theme := config.CurrentTheme

	// Tab border style with bottom connection
	inactiveTabBorder := tabBorderWithBottom("┴", "─", "┴")

	// Base tab styles
	// Pending tabs (not yet started) - purple/muted color
	pendingTabStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(theme.GetMutedColor()).
		Padding(0, 1)

	// Active build phase - cyan/secondary color with spinner
	activeTabStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(theme.GetSecondaryColor()).
		Padding(0, 1)

	// Complete tabs - green/success color with checkmark
	completeTabStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(theme.GetSuccessColor()).
		Padding(0, 1)

	// Error tabs - red/error color
	errorTabStyle := lipgloss.NewStyle().
		Border(inactiveTabBorder, true).
		BorderForeground(theme.GetErrorColor()).
		Padding(0, 1)

	var renderedTabs []string

	for i, tab := range tabs {
		isFirst := i == 0
		isLast := i == len(tabs)-1
		isActive := i == cfg.ActiveIndex

		// Choose style based on state
		var style lipgloss.Style
		var titleText string

		// Check if this is a non-build tab (Select, Local, Remote) - use solid dot instead of spinner
		// Build tabs (Download, Verify, Extract, etc.) use spinners
		isNonBuildTab := tab.Title == "Select" || tab.Title == "Local" || tab.Title == "Remote"

		switch tab.State {
		case TabActive:
			style = activeTabStyle
			// Use solid dot for non-build tabs, spinner for build phases
			if isNonBuildTab {
				titleText = theme.ActiveIndicator() + " " + tab.Title
			} else {
				titleText = tab.Spinner.View() + " " + tab.Title
			}
		case TabComplete:
			style = completeTabStyle
			titleText = theme.CompleteIndicator() + " " + tab.Title
		case TabError:
			style = errorTabStyle
			titleText = theme.ErrorIndicator() + " " + tab.Title
		default: // TabPending
			style = pendingTabStyle
			titleText = theme.PendingIndicator() + " " + tab.Title
		}

		// Adjust borders based on viewing state and position
		border, _, _, _, _ := style.GetBorder()

		if isActive {
			// Tab being viewed - remove bottom border
			border.BottomLeft = "┘"
			border.Bottom = " "
			border.BottomRight = "└"

			// Adjust first tab's left border when viewing
			if isFirst {
				border.BottomLeft = "│"
			}
		} else {
			// Tab not being viewed - has bottom border
			if isFirst {
				border.BottomLeft = "├"
			}
		}

		// Last tab always connects to extension line (whether viewing or not)
		if isLast {
			if isActive {
				// Don't override - keep the "└" from isActive block above
				// This connects naturally to the extension line
			} else {
				border.BottomRight = "┴"
			}
		}

		style = style.Border(border)

		renderedTabs = append(renderedTabs, style.Render(titleText))
	}

	tabsRow := lipgloss.JoinHorizontal(lipgloss.Top, renderedTabs...)

	// Measure tabs width and add horizontal line to fill remaining width
	tabsWidth := lipgloss.Width(tabsRow)

	if cfg.Width > tabsWidth {
		remainingWidth := cfg.Width - tabsWidth

		// Create extension lines for each row of the tabs (3 lines total)
		// Line 1 (top): spaces
		// Line 2 (middle): spaces only (last tab already has the right border connection)
		// Line 3 (bottom): horizontal line ending with corner to connect to content pane's right border
		topLine := strings.Repeat(" ", remainingWidth)
		middleLine := strings.Repeat(" ", remainingWidth)

		// Bottom line is dashes, ending with ┐ to connect to the content pane's right border │
		// The ┐ sits above where the content pane's right │ will be
		// The last tab's "└" (when active) connects naturally to this dash line
		bottomLineContent := strings.Repeat("─", remainingWidth-1) + "┐"
		bottomLine := lipgloss.NewStyle().
			Foreground(theme.GetPrimaryColor()).
			Render(bottomLineContent)

		// Join extension as a 3-line block
		extension := lipgloss.JoinVertical(lipgloss.Left, topLine, middleLine, bottomLine)

		// Join tabs and extension horizontally
		return lipgloss.JoinHorizontal(lipgloss.Top, tabsRow, extension)
	}

	return tabsRow
}

// RenderTabContent renders the content pane for the active tab
func RenderTabContent(content string, width, height int) string {
	theme := config.CurrentTheme

	windowStyle := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder()).
		BorderForeground(theme.GetPrimaryColor()).
		BorderTop(false). // No top border - connects to tabs
		Width(width).
		Height(height).
		Padding(1, 2)

	return windowStyle.Render(content)
}

// tabBorderWithBottom creates a custom border with specified bottom characters
func tabBorderWithBottom(left, middle, right string) lipgloss.Border {
	border := lipgloss.RoundedBorder()
	border.BottomLeft = left
	border.Bottom = middle
	border.BottomRight = right
	return border
}
