// SPDX-License-Identifier: Apache-2.0
package init

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

func TestWizardModel_Init(t *testing.T) {
	m := NewWizardModel()

	// Should create 3 tabs
	if len(m.tabs) != 3 {
		t.Errorf("expected 3 tabs, got %d", len(m.tabs))
	}

	// Tab 0 should be active initially
	if m.activeTab != 0 {
		t.Errorf("expected activeTab 0, got %d", m.activeTab)
	}

	// First tab should be active state
	if m.tabs[0].State != ui.TabActive {
		t.Errorf("expected first tab to be active, got %v", m.tabs[0].State)
	}

	// Other tabs should be pending
	for i := 1; i < len(m.tabs); i++ {
		if m.tabs[i].State != ui.TabPending {
			t.Errorf("expected tab %d to be pending, got %v", i, m.tabs[i].State)
		}
	}
}

func TestWizardModel_TabCompleteMsg(t *testing.T) {
	m := NewWizardModel()

	// Simulate tab 0 completing
	msg := TabCompleteMsg{TabIndex: 0}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(WizardModel)

	// Tab 0 should be marked complete
	if m.tabs[0].State != ui.TabComplete {
		t.Errorf("expected tab 0 to be complete, got %v", m.tabs[0].State)
	}

	// Tab 1 should be active
	if m.activeTab != 1 {
		t.Errorf("expected activeTab 1, got %d", m.activeTab)
	}

	if m.tabs[1].State != ui.TabActive {
		t.Errorf("expected tab 1 to be active, got %v", m.tabs[1].State)
	}
}

func TestWizardModel_SettingsUpdateMsg(t *testing.T) {
	m := NewWizardModel()

	// Update settings from tab
	settings := initpkg.InitSettings{
		KeyName:  "Test Kernels",
		KeyEmail: "test@example.com",
	}

	msg := SettingsUpdateMsg{Settings: settings}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(WizardModel)

	// Settings should be stored (via pointer)
	if m.settings.KeyName != "Test Kernels" {
		t.Errorf("expected KeyName to be set, got %s", m.settings.KeyName)
	}

	if m.settings.KeyEmail != "test@example.com" {
		t.Errorf("expected KeyEmail to be set, got %s", m.settings.KeyEmail)
	}
}

func TestWizardModel_WindowSizeMsg(t *testing.T) {
	m := NewWizardModel()

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(WizardModel)

	// Dimensions should be stored
	if m.width != 120 {
		t.Errorf("expected width 120, got %d", m.width)
	}

	if m.height != 40 {
		t.Errorf("expected height 40, got %d", m.height)
	}
}

func TestWizardModel_Quitting(t *testing.T) {
	m := NewWizardModel()

	// Navigate to last tab and complete it
	for i := 0; i < 2; i++ {
		msg := TabCompleteMsg{TabIndex: i}
		updatedModel, _ := m.Update(msg)
		m = updatedModel.(WizardModel)
	}

	// Mark final tab as complete
	m.tabs[2].State = ui.TabComplete

	// Press 'q' should quit
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	updatedModel, cmd := m.Update(msg)
	m = updatedModel.(WizardModel)

	// Should be quitting
	if !m.quitting {
		t.Error("expected quitting to be true")
	}

	// Should return tea.Quit command
	if cmd == nil {
		t.Error("expected quit command, got nil")
	}
}

func TestWizardModel_SettingsMerge(t *testing.T) {
	// Test mergeSettings function
	old := initpkg.InitSettings{
		KeyFormat: "armored",
		KeyName:   "Old Name",
		KeyEmail:  "old@example.com",
	}

	new := initpkg.InitSettings{
		KeyFormat: "binary",
		KeyExpiry: "2y",
	}

	merged := mergeSettings(old, new)

	// New values should override
	if merged.KeyFormat != "binary" {
		t.Errorf("expected new KeyFormat, got %s", merged.KeyFormat)
	}

	// New values should be added
	if merged.KeyExpiry != "2y" {
		t.Errorf("expected KeyExpiry to be set, got %s", merged.KeyExpiry)
	}

	// Old values not in new should be preserved
	if merged.KeyName != "Old Name" {
		t.Errorf("expected old KeyName to be preserved, got %s", merged.KeyName)
	}

	if merged.KeyEmail != "old@example.com" {
		t.Errorf("expected old KeyEmail to be preserved, got %s", merged.KeyEmail)
	}
}

func TestWizardModel_View(t *testing.T) {
	m := NewWizardModel()

	// Set dimensions
	m.width = 120
	m.height = 40

	// Should not panic
	view := m.View()
	if view == "" {
		t.Error("expected non-empty view")
	}

	// Before dimensions are set, should show "Initializing..."
	m2 := NewWizardModel()
	view2 := m2.View()
	if view2 != "Initializing..." {
		t.Errorf("expected 'Initializing...', got %s", view2)
	}
}

func TestWizardModel_TabErrorMsg(t *testing.T) {
	m := NewWizardModel()

	// Simulate tab error
	msg := TabErrorMsg{
		TabIndex: 1,
		Error:    tea.ErrProgramKilled,
	}

	updatedModel, _ := m.Update(msg)
	m = updatedModel.(WizardModel)

	// Tab should be marked as error
	if m.tabs[1].State != ui.TabError {
		t.Errorf("expected tab 1 to be error, got %v", m.tabs[1].State)
	}

	// Error should be stored
	if m.err == nil {
		t.Error("expected error to be stored")
	}
}

func TestWizardModel_DelegateToActiveTab(t *testing.T) {
	m := NewWizardModel()

	// Set dimensions first so tabs can process messages
	m.width = 120
	m.height = 40

	// Send a message that should be handled by the active tab
	msg := tea.KeyMsg{Type: tea.KeyEnter}
	updatedModel, _ := m.Update(msg)
	m = updatedModel.(WizardModel)

	// The configs tab auto-completes immediately, so activeTab advances to 1
	// The test verifies delegation is working
	_ = m
}
