// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fmt"
	"image/color"
	"io"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/charmbracelet/log"
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
	version      string
	isDefault    bool
	successColor color.Color
}

func (v VersionItem) FilterValue() string { return v.version }
func (v VersionItem) Title() string {
	if v.isDefault {
		markerStyle := lipgloss.NewStyle().Foreground(v.successColor)
		return markerStyle.Render("●") + " " + v.version + " (default)"
	}
	return "  " + v.version
}
func (v VersionItem) Description() string { return "" }

// customDelegate is a list item delegate that renders without cursor indicators
type customDelegate struct {
	accentColor  color.Color
	successColor color.Color
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

	var displayText string
	if versionItem.isDefault {
		markerStyle := lipgloss.NewStyle().Foreground(d.successColor)
		displayText = markerStyle.Render("●") + " " + versionItem.version + " (default)"
	} else {
		displayText = "  " + versionItem.version
	}

	if index == 0 {
		log.Debug("customDelegate.Render", "index", index, "displayText", fmt.Sprintf("%q", displayText), "isSelected", index == m.Index(), "version", versionItem.version)
	}

	if index == m.Index() {
		displayText = lipgloss.NewStyle().Foreground(d.accentColor).Render(displayText)
	}

	fmt.Fprint(w, displayText)
}

type VersionSelectorModel struct {
	theme           config.Theme
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

func NewVersionSelector(theme config.Theme, target string, downloaded, available []string, downloadFn func(string, func(float64), func(string)) error, setDefaultFn, deleteFn func(string) error, reloadFn func() ([]string, []string, error), getDefaultVerFn func() string) VersionSelectorModel {
	primaryColor := theme.GetPrimaryColor()
	secondaryColor := theme.GetSecondaryColor()
	successColor := theme.GetSuccessColor()

	defaultVer := getDefaultVerFn()

	downloadedItems := make([]list.Item, len(downloaded))
	for i, v := range downloaded {
		downloadedItems[i] = VersionItem{
			version:      v,
			isDefault:    v == defaultVer,
			successColor: successColor,
		}
	}

	downloadedDelegate := customDelegate{accentColor: primaryColor, successColor: successColor}

	downloadedList := list.New(downloadedItems, downloadedDelegate, 0, 0)
	downloadedList.Title = ""
	downloadedList.SetShowTitle(false)
	downloadedList.SetShowStatusBar(false)
	downloadedList.SetShowPagination(false)
	downloadedList.SetFilteringEnabled(false)
	downloadedList.SetShowHelp(false)

	availableItems := make([]list.Item, len(available))
	for i, v := range available {
		availableItems[i] = VersionItem{
			version:      v,
			isDefault:    false,
			successColor: successColor,
		}
	}

	availableDelegate := customDelegate{accentColor: secondaryColor, successColor: successColor}

	availableList := list.New(availableItems, availableDelegate, 0, 0)
	availableList.Title = ""
	availableList.SetShowTitle(false)
	availableList.SetShowStatusBar(false)
	availableList.SetShowPagination(false)
	availableList.SetFilteringEnabled(false)
	availableList.SetShowHelp(false)

	// Progress bar and spinner with theme colors
	prog := progress.New(progress.WithColors(theme.Primary, theme.Secondary))
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	spin.Style = lipgloss.NewStyle().Foreground(primaryColor)

	tabs := []Tab{
		{Title: "Local", State: TabComplete},
		{Title: "Remote", State: TabComplete},
	}

	return VersionSelectorModel{
		theme:            theme,
		target:           target,
		downloadedList:   downloadedList,
		availableList:    availableList,
		tabs:             tabs,
		activeTabIndex:   0,
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
		showInstructions: true,
		blankLineCount:   3,
	}
}

func (m VersionSelectorModel) Init() tea.Cmd {
	return nil // v2 sends WindowSizeMsg automatically
}

// performDownload executes the download in a goroutine with progress tracking
func (m VersionSelectorModel) performDownload() tea.Cmd {
	version := m.selectedVersion
	downloadFn := m.downloadFn

	return func() tea.Msg {
		log.Debugf("performDownload: Starting download of %s", version)
		progressChan := make(chan float64, 10)
		statusChan := make(chan string, 10)
		done := make(chan struct{})

		go func() {
			lastReported := -1.0
			progressCallback := func(percent float64) {
				reportThreshold := 0.05
				if percent == 0.0 || percent-lastReported >= reportThreshold || percent >= 0.99 {
					log.Debugf("performDownload: Progress callback called with %.2f", percent)
					select {
					case progressChan <- percent:
						lastReported = percent
					default:
					}
				}
			}

			statusCallback := func(status string) {
				log.Debugf("performDownload: Status callback called with %s", status)
				select {
				case statusChan <- status:
				default:
				}
			}

			downloadFn(version, progressCallback, statusCallback)

			close(progressChan)
			close(statusChan)
			close(done)
		}()

		for {
			select {
			case percent, ok := <-progressChan:
				if ok {
					return progressUpdateMsg{
						percent:  percent,
						done:     done,
						progress: progressChan,
						status:   statusChan,
					}
				}
			case status, ok := <-statusChan:
				if ok {
					return StatusUpdateMsg{
						status:      status,
						done:        done,
						progress:    progressChan,
						status_chan: statusChan,
					}
				}
			case <-done:
				return ActionCompleteMsg{}
			}
		}
	}
}

type progressUpdateMsg struct {
	percent  float64
	done     chan struct{}
	progress chan float64
	status   chan string
}

func waitForProgress(done chan struct{}, progress chan float64, status chan string) tea.Cmd {
	return func() tea.Msg {
		select {
		case percent, ok := <-progress:
			if ok {
				return progressUpdateMsg{
					percent:  percent,
					done:     done,
					progress: progress,
					status:   status,
				}
			}
			return ActionCompleteMsg{}
		case stat, ok := <-status:
			if ok {
				return StatusUpdateMsg{
					status:      stat,
					done:        done,
					progress:    progress,
					status_chan: status,
				}
			}
			return ActionCompleteMsg{}
		case <-done:
			return ActionCompleteMsg{}
		}
	}
}

func (m VersionSelectorModel) performSetDefault() tea.Cmd {
	return func() tea.Msg {
		m.setDefaultFn(m.selectedVersion)
		return ActionCompleteMsg{}
	}
}

func (m VersionSelectorModel) performDelete() tea.Cmd {
	version := m.selectedVersion
	deleteFn := m.deleteFn
	return func() tea.Msg {
		log.Debugf("performDelete: Starting delete of version %s", version)
		err := deleteFn(version)
		if err != nil {
			log.Debugf("performDelete: Delete failed with error: %v", err)
		} else {
			log.Debugf("performDelete: Delete succeeded for %s", version)
		}
		return ActionCompleteMsg{}
	}
}

func (m VersionSelectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle ActionCompleteMsg first
	if _, ok := msg.(ActionCompleteMsg); ok {
		log.Debugf("Update: Received ActionCompleteMsg, reloading lists")

		downloaded, available, err := m.reloadFn()
		if err != nil {
			m.currentState = stateBrowsing
			return m, nil
		}

		defaultVer := m.getDefaultVerFn()

		successColor := m.theme.GetSuccessColor()

		downloadedItems := make([]list.Item, len(downloaded))
		for i, v := range downloaded {
			downloadedItems[i] = VersionItem{
				version:      v,
				isDefault:    v == defaultVer,
				successColor: successColor,
			}
		}
		m.downloadedList.SetItems(downloadedItems)

		availableItems := make([]list.Item, len(available))
		for i, v := range available {
			availableItems[i] = VersionItem{
				version:      v,
				isDefault:    false,
				successColor: successColor,
			}
		}
		m.availableList.SetItems(availableItems)

		m.currentState = stateBrowsing
		return m, nil
	}

	// Handle confirmation form if active
	if m.currentState == stateConfirmingDelete && m.confirmForm != nil {
		confirmed, shouldProceed, cmd := m.confirmForm.Update(msg)

		if shouldProceed {
			log.Debugf("Update: Delete confirmation: confirmed=%v", confirmed)
			if confirmed {
				m.currentState = stateDeleting
				return m, tea.Batch(cmd, m.performDelete())
			}
			m.currentState = stateBrowsing
			return m, cmd
		} else if keyMsg, ok := msg.(tea.KeyPressMsg); ok && keyMsg.String() == "esc" {
			m.currentState = stateBrowsing
			return m, nil
		}

		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		borderWidth := 2
		contentWidth := m.width - borderWidth
		listWidth := contentWidth

		const (
			headerLines       = 1
			tabsLines         = 3
			helpLines         = 1
			contentBorder     = 2
			minListHeight     = 5
			instructionsLines = 1
			blankLines        = 3
		)

		requiredOverhead := headerLines + tabsLines + helpLines + contentBorder
		optionalOverhead := instructionsLines + blankLines

		availableHeight := m.height - requiredOverhead
		m.showInstructions = false
		m.blankLineCount = 0

		if availableHeight >= minListHeight+optionalOverhead {
			m.showInstructions = true
			m.blankLineCount = 3
		} else if availableHeight >= minListHeight+instructionsLines+1 {
			m.showInstructions = true
			m.blankLineCount = 1
		}

		actualOverhead := requiredOverhead + m.blankLineCount
		if m.showInstructions {
			actualOverhead += instructionsLines
		}

		listHeight := m.height - actualOverhead
		if listHeight < minListHeight {
			listHeight = minListHeight
		}

		log.Debugf("WindowSizeMsg: terminal=%dx%d, contentWidth=%d, listSize=%dx%d",
			m.width, m.height, contentWidth, listWidth, listHeight)

		m.downloadedList.SetSize(listWidth, listHeight)
		m.availableList.SetSize(listWidth, listHeight)

		return m, nil

	case progressUpdateMsg:
		m.progressPercent = msg.percent
		var cmd tea.Cmd
		if m.currentPhase == phaseDownloading {
			cmd = m.progress.SetPercent(m.progressPercent)
		}
		return m, tea.Batch(cmd, waitForProgress(msg.done, msg.progress, msg.status))

	case StatusUpdateMsg:
		m.statusMessage = msg.status
		if msg.status == "Downloading kernel..." || msg.status == "Downloading Firecracker..." {
			m.currentPhase = phaseDownloading
		} else {
			m.currentPhase = phaseProcessing
		}
		return m, waitForProgress(msg.done, msg.progress, msg.status_chan)

	case progress.FrameMsg:
		var cmd tea.Cmd
		m.progress, cmd = m.progress.Update(msg)
		return m, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd

	case tea.KeyPressMsg:
		switch m.currentState {
		case stateBrowsing:
			if binding := m.globalKeys.Contains(msg.String()); binding != nil {
				switch binding.Key {
				case "ESC":
					m.quitting = true
					return m, tea.Quit
				case "TAB":
					m.activeTabIndex = (m.activeTabIndex + 1) % len(m.tabs)
					return m, nil
				}
				return m, nil
			}

			if m.activeTabIndex == 0 {
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
			} else if m.activeTabIndex == 1 {
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
			return m, nil
		}
	}

	// Update the active list only when browsing
	if m.currentState == stateBrowsing {
		var cmd tea.Cmd
		if m.activeTabIndex == 0 {
			m.downloadedList, cmd = m.downloadedList.Update(msg)
		} else {
			m.availableList, cmd = m.availableList.Update(msg)
		}
		return m, cmd
	}

	return m, nil
}

func (m VersionSelectorModel) View() tea.View {
	if m.quitting {
		return tea.NewView("")
	}

	if m.width == 0 || m.height == 0 {
		return tea.NewView("Loading...")
	}

	theme := m.theme

	var targetDisplay string
	if m.target == "firecracker" {
		targetDisplay = "FIRECRACKER"
	} else {
		targetDisplay = "KERNEL"
	}
	header := theme.RenderHeader(m.width, "VERSION MANAGER", targetDisplay)

	for i := range m.tabs {
		if i == m.activeTabIndex {
			m.tabs[i].State = TabActive
		} else {
			m.tabs[i].State = TabComplete
		}
	}

	tabsRow := RenderTabs(m.tabs, TabsConfig{
		ActiveIndex: m.activeTabIndex,
		Width:       m.width,
	}, theme)

	var tabContent string
	var tabKeys KeyBindingSet
	if m.activeTabIndex == 0 {
		tabContent = m.downloadedList.View()
		tabKeys = m.downloadedKeys
	} else {
		tabContent = m.availableList.View()
		tabKeys = m.availableKeys
	}

	helpStyle := lipgloss.NewStyle().Foreground(theme.GetMutedColor())
	tabHelp := tabKeys.RenderInline(helpStyle)
	contentWithHelp := lipgloss.JoinVertical(lipgloss.Left, tabContent, "", tabHelp)

	contentPane := RenderTabContent(contentWithHelp, m.width, 0, theme)

	help := theme.RenderFooter(m.width, m.globalKeys.Render(lipgloss.NewStyle()))

	layoutParts := []string{header}

	if m.blankLineCount >= 1 {
		layoutParts = append(layoutParts, "")
	}

	if m.showInstructions {
		instructionText := "Browse available versions and download what you need. Set a default version or remove versions you no longer use."
		instructions := lipgloss.NewStyle().
			Foreground(theme.GetMutedColor()).
			Width(m.width).
			Align(lipgloss.Center).
			Render(instructionText)
		layoutParts = append(layoutParts, instructions)
	}

	if m.blankLineCount >= 2 {
		layoutParts = append(layoutParts, "")
	}

	layoutParts = append(layoutParts, tabsRow, contentPane)

	if m.blankLineCount >= 3 {
		layoutParts = append(layoutParts, "")
	}

	layoutParts = append(layoutParts, help)

	baseView := lipgloss.JoinVertical(lipgloss.Left, layoutParts...)

	wsStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("0"))

	switch m.currentState {
	case stateConfirmingDelete:
		formView := m.confirmForm.View()
		constrainedForm := lipgloss.NewStyle().MaxWidth(60).Render(formView)
		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, constrainedForm, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceStyle(wsStyle)))
		v.AltScreen = true
		return v

	case stateDownloading:
		modal := m.renderProgressModal(theme.Secondary, theme.Muted, "Downloading")
		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceStyle(wsStyle)))
		v.AltScreen = true
		return v

	case stateSettingDefault:
		modal := m.renderProgressModal(theme.Primary, theme.Muted, "Setting default")
		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceStyle(wsStyle)))
		v.AltScreen = true
		return v

	case stateDeleting:
		modal := m.renderProgressModal(theme.Primary, theme.Muted, "Deleting")
		v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, modal, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceStyle(wsStyle)))
		v.AltScreen = true
		return v
	}

	v := tea.NewView(lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, baseView))
	v.AltScreen = true
	return v
}

func (m VersionSelectorModel) renderProgressModal(accentColor, mutedColor color.Color, action string) string {
	title := lipgloss.NewStyle().
		Foreground(accentColor).
		Bold(true).
		Render(fmt.Sprintf("%s %s", action, m.selectedVersion))

	statusText := ""
	if m.statusMessage != "" {
		statusText = lipgloss.NewStyle().
			Foreground(mutedColor).
			Render("\n" + m.statusMessage)
	}

	var indicator string
	if m.currentState == stateDownloading {
		if m.currentPhase == phaseDownloading {
			indicator = "\n" + m.progress.View()
		} else {
			indicator = "\n" + m.spinner.View()
		}
	}

	helpText := lipgloss.NewStyle().
		Foreground(mutedColor).
		Render("\n\nPlease wait...")

	modal := lipgloss.JoinVertical(lipgloss.Left, title, statusText, indicator, helpText)
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(accentColor).
		Padding(1, 2).
		Width(50).
		Render(modal)
}

// RunVersionSelector runs the version selector with the provided callbacks
func RunVersionSelector(
	theme config.Theme,
	target string,
	downloaded, available []string,
	downloadFn func(string, func(float64), func(string)) error,
	setDefaultFn, deleteFn func(string) error,
	reloadFn func() ([]string, []string, error),
	getDefaultVerFn func() string,
) error {
	model := NewVersionSelector(theme, target, downloaded, available, downloadFn, setDefaultFn, deleteFn, reloadFn, getDefaultVerFn)
	p := tea.NewProgram(model)

	_, err := p.Run()
	return err
}
