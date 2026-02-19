// SPDX-License-Identifier: Apache-2.0
package init

import (
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/log"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/Work-Fort/Anvil/pkg/config"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

// WizardModel orchestrates all init tabs
type WizardModel struct {
	width  int
	height int

	tabs      []ui.Tab
	activeTab int
	settings  *initpkg.InitSettings
	quitting  bool
	err       error

	// Tab instances - stored separately since they're different types
	signingTab *SigningTab
	keygenTab  *KeygenTab
	summaryTab *SummaryTab
}

// NewWizardModel creates the init wizard
func NewWizardModel() WizardModel {
	// Create tab metadata
	tabs := []ui.Tab{
		{Title: "Config", State: ui.TabActive},
		{Title: "Key Gen", State: ui.TabPending},
		{Title: "Summary", State: ui.TabPending},
	}

	// Initialize spinners for each tab
	for i := range tabs {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(config.CurrentTheme.GetSecondaryColor())
		tabs[i].Spinner = s
	}

	// Create tab instances with default settings (will be updated as we progress)
	// Settings stored as pointer so tabs always see latest values
	emptySettings := &initpkg.InitSettings{
		ArchiveLocation: "archive",
	}

	return WizardModel{
		tabs:       tabs,
		activeTab:  0,
		settings:   emptySettings,
		signingTab: NewSigningTab(),
		keygenTab:  NewKeygenTab(emptySettings),
		summaryTab: NewSummaryTab(emptySettings),
	}
}

// Init implements tea.Model
func (m WizardModel) Init() tea.Cmd {
	return m.signingTab.Init()
}

// Update implements tea.Model
func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Debugf("wizard.Update: msg=%T activeTab=%d w=%d h=%d", msg, m.activeTab, m.width, m.height)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Forward to all tabs, collecting any commands they return
		var cmds []tea.Cmd
		var cmd tea.Cmd

		m.signingTab, cmd = m.signingTab.Update(msg)
		cmds = append(cmds, cmd)
		var model tea.Model
		model, cmd = m.keygenTab.Update(msg)
		m.keygenTab = model.(*KeygenTab)
		cmds = append(cmds, cmd)
		model, cmd = m.summaryTab.Update(msg)
		m.summaryTab = model.(*SummaryTab)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		// Only allow quitting from final tab when complete
		if msg.String() == "q" && m.activeTab == 2 && m.tabs[2].State == ui.TabComplete {
			m.quitting = true
			return m, tea.Quit
		}

		// Allow Ctrl+C to quit anytime
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case SettingsUpdateMsg:
		// Merge settings from tab (in-place update to pointer)
		*m.settings = mergeSettings(*m.settings, msg.Settings)
		return m, nil

	case TabCompleteMsg:
		// Mark tab as complete
		m.tabs[msg.TabIndex].State = ui.TabComplete
		log.Debugf("wizard.TabCompleteMsg: tabIndex=%d advancing to %d", msg.TabIndex, msg.TabIndex+1)

		// Advance to next tab if not at the end
		if msg.TabIndex < len(m.tabs)-1 {
			m.activeTab = msg.TabIndex + 1
			m.tabs[m.activeTab].State = ui.TabActive

			// Initialize the next tab and pass settings if needed
			var initCmd tea.Cmd
			switch m.activeTab {
			case 1: // KeygenTab - recreate with pointer to latest settings
				m.keygenTab = NewKeygenTab(m.settings)
				initCmd = m.keygenTab.Init()
			case 2: // SummaryTab - recreate with pointer to latest settings
				m.summaryTab = NewSummaryTab(m.settings)
				initCmd = m.summaryTab.Init()
			}

			return m, initCmd
		}
		return m, nil

	case TabErrorMsg:
		// Mark tab as error
		m.tabs[msg.TabIndex].State = ui.TabError
		m.err = msg.Error
		return m, nil
	}

	// Delegate to active tab
	var cmd tea.Cmd
	switch m.activeTab {
	case 0:
		m.signingTab, cmd = m.signingTab.Update(msg)
	case 1:
		var model tea.Model
		model, cmd = m.keygenTab.Update(msg)
		m.keygenTab = model.(*KeygenTab)
	case 2:
		var model tea.Model
		model, cmd = m.summaryTab.Update(msg)
		m.summaryTab = model.(*SummaryTab)
	}

	// Update spinner for active tab based on tab state
	activeTabModel := m.getActiveTabModel()
	if activeTabModel != nil {
		m.tabs[m.activeTab].State = activeTabModel.GetState()
	}

	return m, cmd
}

// View implements tea.Model
func (m WizardModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Render tabs
	tabsCfg := ui.TabsConfig{
		ActiveIndex: m.activeTab,
		Width:       m.width,
	}
	tabsView := ui.RenderTabs(m.tabs, tabsCfg)
	log.Debugf("wizard.View: activeTab=%d tabsViewLen=%d w=%d h=%d", m.activeTab, len(tabsView), m.width, m.height)

	// Render active tab content
	contentHeight := m.height - 4 // Account for tabs and padding
	var activeContent string
	switch m.activeTab {
	case 0:
		activeContent = m.signingTab.View()
	case 1:
		activeContent = m.keygenTab.View()
	case 2:
		activeContent = m.summaryTab.View()
	}

	content := ui.RenderTabContent(
		activeContent,
		m.width-2,
		contentHeight,
	)

	// Stack vertically
	return lipgloss.JoinVertical(
		lipgloss.Left,
		tabsView,
		content,
	)
}

// getActiveTabModel returns the active tab model for state checking
func (m WizardModel) getActiveTabModel() interface {
	GetState() ui.TabState
} {
	switch m.activeTab {
	case 0:
		return m.signingTab
	case 1:
		return m.keygenTab
	case 2:
		return m.summaryTab
	default:
		return nil
	}
}

// mergeSettings combines old and new settings (new overrides old)
func mergeSettings(old, new initpkg.InitSettings) initpkg.InitSettings {
	if new.KeyName != "" {
		old.KeyName = new.KeyName
	}
	if new.KeyEmail != "" {
		old.KeyEmail = new.KeyEmail
	}
	if new.KeyExpiry != "" {
		old.KeyExpiry = new.KeyExpiry
	}
	if new.KeyFormat != "" {
		old.KeyFormat = new.KeyFormat
	}
	if new.HistoryFormat != "" {
		old.HistoryFormat = new.HistoryFormat
	}
	if new.KeyPassword != "" {
		old.KeyPassword = new.KeyPassword
	}
	if new.KeyGenerated {
		old.KeyGenerated = true
		old.KeyPath = new.KeyPath
		old.PublicKeyPath = new.PublicKeyPath
	}
	if len(new.FilesCreated) > 0 {
		old.FilesCreated = new.FilesCreated
	}
	return old
}
