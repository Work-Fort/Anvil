// SPDX-License-Identifier: Apache-2.0
package init

import (
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Work-Fort/Anvil/pkg/config"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/ui"
	"github.com/charmbracelet/log"
)

// WizardModel orchestrates the init tabs (Key Gen + Summary).
// Signing config is collected via standalone huh form before this runs.
type WizardModel struct {
	width  int
	height int

	tabs      []ui.Tab
	activeTab int
	settings  *initpkg.InitSettings
	quitting  bool
	err       error

	keygenTab  *KeygenTab
	summaryTab *SummaryTab
}

// NewWizardModel creates the init wizard with pre-collected settings.
func NewWizardModel(settings *initpkg.InitSettings) WizardModel {
	tabs := []ui.Tab{
		{Title: "Key Gen", State: ui.TabActive},
		{Title: "Summary", State: ui.TabPending},
	}

	for i := range tabs {
		s := spinner.New()
		s.Spinner = spinner.Dot
		s.Style = lipgloss.NewStyle().Foreground(config.CurrentTheme.GetSecondaryColor())
		tabs[i].Spinner = s
	}

	return WizardModel{
		tabs:       tabs,
		activeTab:  0,
		settings:   settings,
		keygenTab:  NewKeygenTab(settings),
		summaryTab: NewSummaryTab(settings),
	}
}

func (m WizardModel) Init() tea.Cmd {
	return m.keygenTab.Init()
}

func (m WizardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Debugf("wizard.Update: msg=%T activeTab=%d w=%d h=%d", msg, m.activeTab, m.width, m.height)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		var cmds []tea.Cmd
		var cmd tea.Cmd

		m.keygenTab, cmd = m.keygenTab.Update(msg)
		cmds = append(cmds, cmd)
		m.summaryTab, cmd = m.summaryTab.Update(msg)
		cmds = append(cmds, cmd)

		return m, tea.Batch(cmds...)

	case tea.KeyPressMsg:
		// Allow quitting from final tab when complete
		if msg.String() == "q" && m.activeTab == 1 && m.tabs[1].State == ui.TabComplete {
			m.quitting = true
			return m, tea.Quit
		}

		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

	case SettingsUpdateMsg:
		*m.settings = mergeSettings(*m.settings, msg.Settings)
		return m, nil

	case TabCompleteMsg:
		m.tabs[msg.TabIndex].State = ui.TabComplete
		log.Debugf("wizard.TabCompleteMsg: tabIndex=%d advancing to %d", msg.TabIndex, msg.TabIndex+1)

		if msg.TabIndex < len(m.tabs)-1 {
			m.activeTab = msg.TabIndex + 1
			m.tabs[m.activeTab].State = ui.TabActive

			var initCmd tea.Cmd
			switch m.activeTab {
			case 1: // SummaryTab
				m.summaryTab = NewSummaryTab(m.settings)
				initCmd = m.summaryTab.Init()
			}

			return m, initCmd
		}
		return m, nil

	case TabErrorMsg:
		m.tabs[msg.TabIndex].State = ui.TabError
		m.err = msg.Error
		return m, nil
	}

	// Delegate to active tab
	var cmd tea.Cmd
	switch m.activeTab {
	case 0:
		m.keygenTab, cmd = m.keygenTab.Update(msg)
	case 1:
		m.summaryTab, cmd = m.summaryTab.Update(msg)
	}

	// Update spinner state
	activeTabModel := m.getActiveTabModel()
	if activeTabModel != nil {
		m.tabs[m.activeTab].State = activeTabModel.GetState()
	}

	return m, cmd
}

func (m WizardModel) View() tea.View {
	if m.width == 0 || m.height == 0 {
		return tea.NewView("Initializing...")
	}

	tabsCfg := ui.TabsConfig{
		ActiveIndex: m.activeTab,
		Width:       m.width,
	}
	tabsView := ui.RenderTabs(m.tabs, tabsCfg)

	contentHeight := m.height - 4
	var activeContent string
	switch m.activeTab {
	case 0:
		activeContent = m.keygenTab.View()
	case 1:
		activeContent = m.summaryTab.View()
	}

	content := ui.RenderTabContent(
		activeContent,
		m.width-2,
		contentHeight,
	)

	v := tea.NewView(lipgloss.JoinVertical(
		lipgloss.Left,
		tabsView,
		content,
	))
	v.AltScreen = true
	return v
}

func (m WizardModel) getActiveTabModel() interface {
	GetState() ui.TabState
} {
	switch m.activeTab {
	case 0:
		return m.keygenTab
	case 1:
		return m.summaryTab
	default:
		return nil
	}
}

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
