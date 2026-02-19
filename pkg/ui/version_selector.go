// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
)

type state int

const (
	stateBrowsing state = iota
	stateConfirmingDelete
	stateDownloading
	stateSettingDefault
	stateDeleting
)

// Download phases
const (
	phaseDownloading = "downloading"
	phaseProcessing  = "processing"
)

type VersionItem struct {
	version   string
	isDefault bool
}

func (v VersionItem) FilterValue() string { return v.version }
func (v VersionItem) Title() string {
	if v.isDefault {
		theme := config.CurrentTheme
		markerStyle := lipgloss.NewStyle().Foreground(theme.GetSuccessColor())
		return markerStyle.Render("●") + " " + v.version + " (default)"
	}
	return "  " + v.version
}
func (v VersionItem) Description() string { return "" }

// customDelegate is a list item delegate that renders without cursor indicators
type customDelegate struct {
	accentColor lipgloss.Color
}

func (d customDelegate) Height() int  { return 1 }
func (d customDelegate) Spacing() int { return 0 }
func (d customDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

func (d customDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	versionItem, ok := item.(VersionItem)
	if !ok {
		return
	}

	// Get the version and default status
	var displayText string
	if versionItem.isDefault {
		theme := config.CurrentTheme
		markerStyle := lipgloss.NewStyle().Foreground(theme.GetSuccessColor())
		displayText = markerStyle.Render("●") + " " + versionItem.version + " (default)"
	} else {
		displayText = "  " + versionItem.version
	}

	// DEBUG: Log what we're rendering
	if index == 0 {
		log.Debug("customDelegate.Render", "index", index, "displayText", fmt.Sprintf("%q", displayText), "isSelected", index == m.Index(), "version", versionItem.version)
	}

	// Apply accent color to selected item (entire line)
	if index == m.Index() {
		displayText = lipgloss.NewStyle().Foreground(d.accentColor).Render(displayText)
	}

	fmt.Fprint(w, displayText)
}

type VersionSelectorModel struct {
	target          string // "kernel" or "firecracker"
	downloadedList  list.Model
	availableList   list.Model
	tabs            []Tab
	activeTabIndex  int // 0 = Local, 1 = Remote
	currentState    state
	width           int
	height          int
	quitting        bool
	selectedVersion string
	confirmForm     *ConfirmationForm
	progress        progress.Model
	progressPercent float64
	spinner         spinner.Model
	currentPhase    string // "downloading" or "processing"
	statusMessage   string
	downloadFn      func(string, func(float64), func(string)) error
	setDefaultFn    func(string) error
	deleteFn        func(string) error
	reloadFn        func() ([]string, []string, error)
	getDefaultVerFn func() string
	globalKeys      KeyBindingSet
	downloadedKeys  KeyBindingSet
	availableKeys   KeyBindingSet

	// Layout state for graceful degradation
	showInstructions bool
	blankLineCount   int
}

type ActionCompleteMsg struct{}

type StatusUpdateMsg struct {
	status      string
	done        chan struct{}
	progress    chan float64
	status_chan chan string
}

func NewVersionSelector(target string, downloaded, available []string, downloadFn func(string, func(float64), func(string)) error, setDefaultFn, deleteFn func(string) error, reloadFn func() ([]string, []string, error), getDefaultVerFn func() string) VersionSelectorModel {
	// Get theme colors
	theme := config.CurrentTheme
	primaryColor := theme.GetPrimaryColor()
	secondaryColor := theme.GetSecondaryColor()

	// Get default version
	defaultVer := getDefaultVerFn()

	// Create downloaded list
	downloadedItems := make([]list.Item, len(downloaded))
	for i, v := range downloaded {
		downloadedItems[i] = VersionItem{
			version:   v,
			isDefault: v == defaultVer,
		}
	}

	downloadedDelegate := customDelegate{accentColor: primaryColor}

	downloadedList := list.New(downloadedItems, downloadedDelegate, 0, 0)
	downloadedList.Title = "" // Title will be in border
	downloadedList.SetShowTitle(false)
	downloadedList.SetShowStatusBar(false)
	downloadedList.SetShowPagination(false)
	downloadedList.SetFilteringEnabled(false)
	downloadedList.SetShowHelp(false) // Hide default help text
	// Clear any cursor-related styles
	downloadedList.Styles.FilterCursor = lipgloss.NewStyle()

	// Create available list
	availableItems := make([]list.Item, len(available))
	for i, v := range available {
		availableItems[i] = VersionItem{
			version:   v,
			isDefault: false, // Available versions can't be default
		}
	}

	availableDelegate := customDelegate{accentColor: secondaryColor}

	availableList := list.New(availableItems, availableDelegate, 0, 0)
	availableList.Title = "" // Title will be in border
	availableList.SetShowTitle(false)
	availableList.SetShowStatusBar(false)
	availableList.SetShowPagination(false)
	availableList.SetFilteringEnabled(false)
	availableList.SetShowHelp(false) // Hide default help text
	// Clear any cursor-related styles
	availableList.Styles.FilterCursor = lipgloss.NewStyle()

	// Initialize progress bar and spinner with theme colors
	prog := progress.New(progress.WithGradient(theme.Secondary, theme.Primary))
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(primaryColor)

	// Create tabs for Local and Remote
	// Spinner field is not used for these tabs (they use solid indicators)
	tabs := []Tab{
		{
			Title: "Local",
			State: TabComplete, // Will be set dynamically in View()
		},
		{
			Title: "Remote",
			State: TabComplete, // Will be set dynamically in View()
		},
	}

	return VersionSelectorModel{
		target:           target,
		downloadedList:   downloadedList,
		availableList:    availableList,
		tabs:             tabs,
		activeTabIndex:   0, // Start on Local tab
		currentState:     stateBrowsing,
		progress:         prog,
		spinner:          spin,
		currentPhase:     phaseDownloading,
		downloadFn:       downloadFn,
		setDefaultFn:     setDefaultFn,
		deleteFn:         deleteFn,
		reloadFn:         reloadFn,
		getDefaultVerFn:  getDefaultVerFn,
		globalKeys:       GlobalKeyBindings(),
		downloadedKeys:   DownloadedPaneKeyBindings(),
		availableKeys:    AvailablePaneKeyBindings(),
		showInstructions: true, // Default to full layout
		blankLineCount:   3,    // Default to all blank lines
	}
}

func (m VersionSelectorModel) Init() tea.Cmd {
	return tea.WindowSize()
}

// performDownload executes the download in a goroutine with progress tracking
func (m VersionSelectorModel) performDownload() tea.Cmd {
	version := m.selectedVersion
	downloadFn := m.downloadFn

	return func() tea.Msg {
		log.Debugf("performDownload: Starting download of %s", version)
		// Create channels for progress and status updates
		progressChan := make(chan float64, 10)
		statusChan := make(chan string, 10)
		done := make(chan struct{})

		// Start download in background goroutine
		go func() {
			lastReported := -1.0
			progressCallback := func(percent float64) {
				// Only report progress every 5% to avoid flooding the UI
				// Also allow 0.0 resets to pass through even if we just reported 1.0
				reportThreshold := 0.05
				if percent == 0.0 || percent-lastReported >= reportThreshold || percent >= 0.99 {
					log.Debugf("performDownload: Progress callback called with %.2f", percent)
					select {
					case progressChan <- percent:
						lastReported = percent
						log.Debugf("performDownload: Sent progress %.2f to channel", percent)
					default:
						log.Debugf("performDownload: Channel full, skipped progress %.2f", percent)
					}
				}
			}

			statusCallback := func(status string) {
				log.Debugf("performDownload: Status callback called with %s", status)
				select {
				case statusChan <- status:
					log.Debugf("performDownload: Sent status %s to channel", status)
				default:
					log.Debugf("performDownload: Status channel full, skipped %s", status)
				}
			}

			// Actually perform the download with progress and status tracking
			log.Debugf("performDownload: Calling downloadFn")
			downloadFn(version, progressCallback, statusCallback)

			// Download complete - signal done
			log.Debugf("performDownload: Download complete, closing channels")
			close(progressChan)
			close(statusChan)
			close(done)
		}()

		// Wait for either progress updates, status updates, or completion
		log.Debugf("performDownload: Waiting for progress, status, or completion")
		for {
			select {
			case percent, ok := <-progressChan:
				if ok {
					log.Debugf("performDownload: Received progress %.2f, returning progressUpdateMsg", percent)
					// Send progress update and continue listening
					return progressUpdateMsg{
						percent:  percent,
						done:     done,
						progress: progressChan,
						status:   statusChan,
					}
				}
				log.Debugf("performDownload: Progress channel closed")
			case status, ok := <-statusChan:
				if ok {
					log.Debugf("performDownload: Received status %s, returning StatusUpdateMsg", status)
					// Send status update and continue listening
					return StatusUpdateMsg{
						status:      status,
						done:        done,
						progress:    progressChan,
						status_chan: statusChan,
					}
				}
				log.Debugf("performDownload: Status channel closed")
			case <-done:
				log.Debugf("performDownload: Done channel signaled, returning ActionCompleteMsg")
				// Download complete
				return ActionCompleteMsg{}
			}
		}
	}
}

// progressUpdateMsg carries progress and channels for continued listening
type progressUpdateMsg struct {
	percent  float64
	done     chan struct{}
	progress chan float64
	status   chan string
}

// waitForProgress returns a command that reads the next progress update
func waitForProgress(done chan struct{}, progress chan float64, status chan string) tea.Cmd {
	return func() tea.Msg {
		log.Debugf("waitForProgress: Waiting for next progress or status update")
		select {
		case percent, ok := <-progress:
			if ok {
				log.Debugf("waitForProgress: Received progress %.2f", percent)
				return progressUpdateMsg{
					percent:  percent,
					done:     done,
					progress: progress,
					status:   status,
				}
			}
			log.Debugf("waitForProgress: Progress channel closed")
			return ActionCompleteMsg{}
		case stat, ok := <-status:
			if ok {
				log.Debugf("waitForProgress: Received status %s", stat)
				return StatusUpdateMsg{
					status:      stat,
					done:        done,
					progress:    progress,
					status_chan: status,
				}
			}
			log.Debugf("waitForProgress: Status channel closed")
			return ActionCompleteMsg{}
		case <-done:
			log.Debugf("waitForProgress: Done channel signaled")
			return ActionCompleteMsg{}
		}
	}
}

// performSetDefault executes setting the default version
func (m VersionSelectorModel) performSetDefault() tea.Cmd {
	return func() tea.Msg {
		// Actually set the default
		m.setDefaultFn(m.selectedVersion)
		// Return completion message
		return ActionCompleteMsg{}
	}
}

// performDelete executes the deletion
func (m VersionSelectorModel) performDelete() tea.Cmd {
	version := m.selectedVersion
	deleteFn := m.deleteFn
	return func() tea.Msg {
		log.Debugf("performDelete: Starting delete of version %s", version)

		// Actually perform the deletion
		err := deleteFn(version)

		if err != nil {
			log.Debugf("performDelete: Delete failed with error: %v", err)
		} else {
			log.Debugf("performDelete: Delete succeeded for %s", version)
		}

		log.Debugf("performDelete: Sending ActionCompleteMsg")
		return ActionCompleteMsg{}
	}
}

func (m VersionSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle ActionCompleteMsg first - must reload lists
	if _, ok := msg.(ActionCompleteMsg); ok {
		log.Debugf("Update: Received ActionCompleteMsg, reloading lists")

		downloaded, available, err := m.reloadFn()
		if err != nil {
			log.Debugf("Update: Reload failed with error: %v", err)
			m.currentState = stateBrowsing
			return m, nil
		}

		log.Debugf("Update: Reload returned %d downloaded, %d available", len(downloaded), len(available))
		log.Debugf("Update: Local versions: %v", downloaded)
		log.Debugf("Update: Remote versions: %v", available)

		// Get default version
		defaultVer := m.getDefaultVerFn()

		// Update lists
		downloadedItems := make([]list.Item, len(downloaded))
		for i, v := range downloaded {
			downloadedItems[i] = VersionItem{
				version:   v,
				isDefault: v == defaultVer,
			}
		}
		m.downloadedList.SetItems(downloadedItems)

		availableItems := make([]list.Item, len(available))
		for i, v := range available {
			availableItems[i] = VersionItem{
				version:   v,
				isDefault: false,
			}
		}
		m.availableList.SetItems(availableItems)

		log.Debugf("Update: Lists updated, returning to browsing state")
		m.currentState = stateBrowsing
		return m, nil
	}

	// Handle confirmation form if active
	if m.currentState == stateConfirmingDelete && m.confirmForm != nil {
		confirmed, shouldProceed, cmd := m.confirmForm.Update(msg)

		if shouldProceed {
			// User made a choice (Y/N or completed form)
			log.Debugf("Update: Delete confirmation: confirmed=%v", confirmed)
			if confirmed {
				m.currentState = stateDeleting
				return m, tea.Batch(cmd, m.performDelete())
			} else {
				log.Debugf("Update: User cancelled deletion")
				m.currentState = stateBrowsing
				return m, cmd
			}
		} else if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			// User cancelled
			log.Debugf("Update: User pressed esc to cancel deletion")
			m.currentState = stateBrowsing
			return m, nil
		}

		// Still collecting input
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// CRITICAL: Width calculation for tabbed layout
		//
		// Layout structure: [tabs] then [full-width content pane]
		// Target: Content pane spans full terminal width
		//
		// Lipgloss width behavior (as of v1.1.1):
		// - Style.Width(w) sets content width INCLUDING padding (padding is inside)
		// - Border is rendered OUTSIDE of Style.Width() (adds to final render)
		// - Actual rendered width = Style.Width() + border width
		//
		// Content pane uses:
		// - Border: NormalBorder top=false (sides + bottom = 1 char per side = 2 chars total)
		// - Padding: (1, 2) inside the content width
		//
		// Example for terminal width 130:
		// - Target: 130 chars rendered
		// - Border overhead: 2 chars
		// - Content width to set: 130 - 2 = 128 chars
		// - Rendered: 128 content + 2 border = 130 chars ✓
		borderWidth := 2 // Border sides (top border is disabled, connects to tabs)
		contentWidth := m.width - borderWidth

		// List component gets the content width (it will be rendered inside the styled pane)
		listWidth := contentWidth

		// Calculate content height with graceful degradation for small terminals
		const (
			headerLines   = 1 // Header text (required)
			tabsLines     = 3 // Tab row (required)
			helpLines     = 1 // Help footer (required)
			contentBorder = 2 // Content pane borders (top disabled, bottom + padding)
			minListHeight = 5 // Minimum viable list height

			// Optional elements (drop when height is constrained)
			instructionsLines = 1 // Instructions text (optional)
			blankLines        = 3 // Spacers (optional)
		)

		// Calculate required height (without optional elements)
		requiredOverhead := headerLines + tabsLines + helpLines + contentBorder
		optionalOverhead := instructionsLines + blankLines

		// Determine what to show based on available height
		availableHeight := m.height - requiredOverhead
		m.showInstructions = false
		m.blankLineCount = 0

		if availableHeight >= minListHeight+optionalOverhead {
			// Enough room for everything
			m.showInstructions = true
			m.blankLineCount = 3
		} else if availableHeight >= minListHeight+instructionsLines+1 {
			// Drop blank lines, keep instructions
			m.showInstructions = true
			m.blankLineCount = 1
		} else if availableHeight >= minListHeight {
			// Drop instructions and blanks, keep essentials
			m.showInstructions = false
			m.blankLineCount = 0
		}

		actualOverhead := requiredOverhead + m.blankLineCount
		if m.showInstructions {
			actualOverhead += instructionsLines
		}

		listHeight := m.height - actualOverhead
		if listHeight < minListHeight {
			listHeight = minListHeight // Enforce minimum
		}

		log.Debugf("WindowSizeMsg: terminal=%dx%d, contentWidth=%d, listSize=%dx%d",
			m.width, m.height, contentWidth, listWidth, listHeight)

		m.downloadedList.SetSize(listWidth, listHeight)
		m.availableList.SetSize(listWidth, listHeight)

		return m, nil

	case progressUpdateMsg:
		log.Debugf("Update: Received progressUpdateMsg with %.2f", msg.percent)
		m.progressPercent = msg.percent
		var cmd tea.Cmd
		if m.currentPhase == phaseDownloading {
			cmd = m.progress.SetPercent(m.progressPercent)
		}
		// Keep listening for more progress and status updates
		return m, tea.Batch(cmd, waitForProgress(msg.done, msg.progress, msg.status))

	case StatusUpdateMsg:
		log.Debugf("Update: Received StatusUpdateMsg: %s", msg.status)
		m.statusMessage = msg.status

		// Only show progress bar for the main download, everything else uses spinner
		if msg.status == "Downloading kernel..." || msg.status == "Downloading Firecracker..." {
			m.currentPhase = phaseDownloading
		} else {
			m.currentPhase = phaseProcessing
		}

		// Continue waiting for more updates
		return m, waitForProgress(msg.done, msg.progress, msg.status_chan)

	case progress.FrameMsg:
		progressModel, cmd := m.progress.Update(msg)
		m.progress = progressModel.(progress.Model)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyMsg:
		switch m.currentState {
		case stateBrowsing:
			// Check global key bindings first
			if binding := m.globalKeys.Contains(msg.String()); binding != nil {
				switch binding.Key {
				case "ESC":
					m.quitting = true
					return m, tea.Quit

				case "TAB":
					// Switch between tabs
					m.activeTabIndex = (m.activeTabIndex + 1) % len(m.tabs)
					return m, nil
				}
				return m, nil
			}

			// Check tab-specific key bindings
			if m.activeTabIndex == 0 { // Local tab
				if binding := m.downloadedKeys.Contains(msg.String()); binding != nil {
					switch binding.Key {
					case "ENTER":
						if item, ok := m.downloadedList.SelectedItem().(VersionItem); ok {
							m.selectedVersion = item.version
							m.currentState = stateSettingDefault
							return m, m.performSetDefault()
						}

					case "DEL":
						if item, ok := m.downloadedList.SelectedItem().(VersionItem); ok {
							m.selectedVersion = item.version

							// Create confirmation form
							m.confirmForm = NewConfirmationForm(
								"confirm",
								fmt.Sprintf("Delete version %s?", m.selectedVersion),
								"This action cannot be undone.",
								"Yes",
								"No",
							)

							m.currentState = stateConfirmingDelete
							return m, m.confirmForm.Init()
						}
					}
					return m, nil
				}
			} else if m.activeTabIndex == 1 { // Remote tab
				if binding := m.availableKeys.Contains(msg.String()); binding != nil {
					switch binding.Key {
					case "ENTER":
						if item, ok := m.availableList.SelectedItem().(VersionItem); ok {
							m.selectedVersion = item.version
							m.currentState = stateDownloading
							m.currentPhase = phaseDownloading
							m.progressPercent = 0
							return m, tea.Batch(m.spinner.Tick, m.performDownload())
						}
					}
					return m, nil
				}
			}

		case stateDownloading, stateSettingDefault, stateDeleting:
			// Don't accept input during operations
			return m, nil
		}
	}

	// Update the active list only when browsing
	if m.currentState == stateBrowsing {
		var cmd tea.Cmd
		if m.activeTabIndex == 0 { // Local tab
			m.downloadedList, cmd = m.downloadedList.Update(msg)
		} else { // Remote tab
			m.availableList, cmd = m.availableList.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m VersionSelectorModel) View() string {
	if m.quitting {
		return ""
	}

	// Handle race condition: View() may be called before WindowSizeMsg arrives
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Get theme
	theme := config.CurrentTheme

	// Header using theme helper
	var targetDisplay string
	if m.target == "firecracker" {
		targetDisplay = "FIRECRACKER"
	} else {
		targetDisplay = "KERNEL"
	}
	header := theme.RenderHeader(m.width, "VERSION MANAGER", targetDisplay)

	// Update tab states: active tab = TabActive (cyan), inactive = TabComplete (green)
	for i := range m.tabs {
		if i == m.activeTabIndex {
			m.tabs[i].State = TabActive
		} else {
			m.tabs[i].State = TabComplete
		}
	}

	// Render tabs
	tabsRow := RenderTabs(m.tabs, TabsConfig{
		ActiveIndex: m.activeTabIndex,
		Width:       m.width,
	})

	// Get the content for the active tab
	var tabContent string
	var tabKeys KeyBindingSet
	if m.activeTabIndex == 0 { // Local tab
		tabContent = m.downloadedList.View()
		tabKeys = m.downloadedKeys
	} else { // Remote tab
		tabContent = m.availableList.View()
		tabKeys = m.availableKeys
	}

	// Add help text to bottom of content
	helpStyle := lipgloss.NewStyle().Foreground(theme.GetMutedColor())
	tabHelp := tabKeys.RenderInline(helpStyle)
	contentWithHelp := lipgloss.JoinVertical(lipgloss.Left, tabContent, "", tabHelp)

	// Calculate content pane height (tabs are already rendered above)
	// This must match the calculation in WindowSizeMsg
	borderWidth := 2 // Border sides
	contentWidth := m.width - borderWidth

	// Render content pane using RenderTabContent
	contentPane := RenderTabContent(contentWithHelp, contentWidth, 0)

	// Footer using theme helper
	help := theme.RenderFooter(m.width, m.globalKeys.Render(lipgloss.NewStyle()))

	// Build layout with graceful degradation
	// CRITICAL: Use JoinVertical, NOT string concatenation
	layoutParts := []string{header}

	// Add blank line after header if we have room
	if m.blankLineCount >= 1 {
		layoutParts = append(layoutParts, "")
	}

	// Add instructions if we have room
	if m.showInstructions {
		instructionText := "Browse available versions and download what you need. Set a default version or remove versions you no longer use."
		instructions := lipgloss.NewStyle().
			Foreground(theme.GetMutedColor()).
			Width(m.width).
			Align(lipgloss.Center).
			Render(instructionText)
		layoutParts = append(layoutParts, instructions)
	}

	// Add blank line after instructions if we have room
	if m.blankLineCount >= 2 {
		layoutParts = append(layoutParts, "")
	}

	// Add tabs and content pane (required)
	layoutParts = append(layoutParts, tabsRow, contentPane)

	// Add blank line before help if we have room
	if m.blankLineCount >= 3 {
		layoutParts = append(layoutParts, "")
	}

	// Add help (required)
	layoutParts = append(layoutParts, help)

	baseView := lipgloss.JoinVertical(lipgloss.Left, layoutParts...)

	log.Debugf("View: terminal=%dx%d, activeTab=%d",
		m.width, m.height, m.activeTabIndex)

	// Show modals/overlays based on state
	switch m.currentState {
	case stateConfirmingDelete:
		// Render huh form centered with max width constraint
		formView := m.confirmForm.View()
		// Use MaxWidth to prevent expansion while allowing natural sizing
		constrainedForm := lipgloss.NewStyle().
			MaxWidth(60).
			Render(formView)
		formWidth := lipgloss.Width(constrainedForm)
		formHeight := lipgloss.Height(constrainedForm)
		log.Debugf("VersionSelector confirm modal: terminal=%dx%d, form=%dx%d", m.width, m.height, formWidth, formHeight)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, constrainedForm, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))

	case stateDownloading:
		modal := m.renderProgressModal(theme.Secondary, theme.Muted, "Downloading")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))

	case stateSettingDefault:
		modal := m.renderProgressModal(theme.Primary, theme.Muted, "Setting default")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))

	case stateDeleting:
		modal := m.renderProgressModal(theme.Primary, theme.Muted, "Deleting")
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
	}

	// Place baseView to fill terminal dimensions and eliminate bottom gap
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, baseView)
}

func (m VersionSelectorModel) renderProgressModal(accentColor, mutedColor, action string) string {
	title := lipgloss.NewStyle().
		Foreground(lipgloss.Color(accentColor)).
		Bold(true).
		Render(fmt.Sprintf("%s %s", action, m.selectedVersion))

	// Status message
	statusText := ""
	if m.statusMessage != "" {
		statusText = lipgloss.NewStyle().
			Foreground(lipgloss.Color(mutedColor)).
			Render("\n" + m.statusMessage)
	}

	// Show either progress bar or spinner based on current phase
	var indicator string
	if m.currentState == stateDownloading {
		if m.currentPhase == phaseDownloading {
			// Show progress bar for downloads
			indicator = "\n" + m.progress.View()
		} else {
			// Show spinner for processing
			indicator = "\n" + m.spinner.View()
		}
	}

	helpText := lipgloss.NewStyle().
		Foreground(lipgloss.Color(mutedColor)).
		Render("\n\nPlease wait...")

	modal := lipgloss.JoinVertical(lipgloss.Left, title, statusText, indicator, helpText)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(accentColor)).
		Padding(1, 2).
		Width(50).
		Render(modal)
}

// RunVersionSelector runs the version selector with the provided callbacks
func RunVersionSelector(
	target string,
	downloaded, available []string,
	downloadFn func(string, func(float64), func(string)) error,
	setDefaultFn, deleteFn func(string) error,
	reloadFn func() ([]string, []string, error),
	getDefaultVerFn func() string,
) error {
	model := NewVersionSelector(target, downloaded, available, downloadFn, setDefaultFn, deleteFn, reloadFn, getDefaultVerFn)
	p := tea.NewProgram(model, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
