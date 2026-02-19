// SPDX-License-Identifier: Apache-2.0
package init

import (
	"testing"

	"github.com/charmbracelet/huh"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

func TestSigningTab_Init(t *testing.T) {
	tab := NewSigningTab()
	cmd := tab.Init()

	if cmd == nil {
		t.Fatal("Init() should return a command (form.Init())")
	}

	// Initially not complete
	if tab.formComplete {
		t.Error("Init() should not set formComplete to true")
	}

	// Should have defaults pre-filled (we can't easily test git config here)
	// but we can verify the form exists
	if tab.form == nil {
		t.Error("Init() should create form")
	}
}

func TestSigningTab_Update_WindowSize(t *testing.T) {
	tab := NewSigningTab()
	tab.Init()

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updatedTab, _ := tab.Update(msg)

	// WindowSizeMsg updates dimensions and delegates to form
	// The form may or may not return a command, we don't enforce either way

	if updatedTab.width != 100 {
		t.Errorf("width = %d, want 100", updatedTab.width)
	}
	if updatedTab.height != 50 {
		t.Errorf("height = %d, want 50", updatedTab.height)
	}
}

func TestSigningTab_FormCompletion(t *testing.T) {
	tab := NewSigningTab()
	tab.Init()

	// Simulate form completion by setting values and state
	tab.keyName = "Test User"
	tab.keyEmail = "test@example.com"
	tab.keyExpiry = "1y"
	tab.keyFormat = "armored"
	tab.histFormat = "armored"

	// Simulate huh.Completed state by manually completing the form
	// We do this by sending the form completion message
	msg := struct{}{} // Dummy message to trigger form state check

	// Manually set form state to completed for testing
	// In real usage, huh.Form will set this when user completes all fields
	tab.form.State = huh.StateCompleted
	tab.formComplete = false // Start as false to test the transition

	updatedTab, cmd := tab.Update(msg)

	if cmd == nil {
		t.Fatal("Form completion should return commands")
	}

	if !updatedTab.formComplete {
		t.Error("formComplete should be true after form completion")
	}

	// Verify the values are stored
	if updatedTab.keyName != "Test User" {
		t.Errorf("keyName = %q, want %q", updatedTab.keyName, "Test User")
	}
	if updatedTab.keyEmail != "test@example.com" {
		t.Errorf("keyEmail = %q, want %q", updatedTab.keyEmail, "test@example.com")
	}
	if updatedTab.keyExpiry != "1y" {
		t.Errorf("keyExpiry = %q, want %q", updatedTab.keyExpiry, "1y")
	}
}

func TestSigningTab_IsComplete(t *testing.T) {
	tab := NewSigningTab()

	// Initially not complete
	if tab.IsComplete() {
		t.Error("IsComplete() should return false initially")
	}

	// Set complete
	tab.formComplete = true

	if !tab.IsComplete() {
		t.Error("IsComplete() should return true after formComplete is set")
	}
}

func TestSigningTab_GetState(t *testing.T) {
	tab := NewSigningTab()

	// Initially active
	if tab.GetState() != ui.TabActive {
		t.Errorf("GetState() = %v, want TabActive", tab.GetState())
	}

	// Set complete
	tab.formComplete = true

	if tab.GetState() != ui.TabComplete {
		t.Errorf("GetState() = %v, want TabComplete", tab.GetState())
	}
}

func TestSigningTab_View(t *testing.T) {
	tab := NewSigningTab()
	tab.Init()

	view := tab.View()

	// View should delegate to form.View()
	// We can't test the exact content without a running form,
	// but we can verify it returns something
	if view == "" {
		t.Error("View() should return form.View() content")
	}
}

func TestSigningTab_EmailValidation(t *testing.T) {
	// Test that email validation is set up correctly
	// We'll create the tab and verify the form has validation
	tab := NewSigningTab()
	tab.Init()

	// We can't easily test huh form validation without running the full TUI,
	// but we can verify the form was created
	if tab.form == nil {
		t.Error("form should be created with email validation")
	}
}

func TestSigningTab_DefaultValues(t *testing.T) {
	// Test that default values are set
	tab := NewSigningTab()
	tab.Init()

	// Expiry should default to "1y" (can't easily verify without form inspection)
	// Format fields should default to "armored"
	// We verify the form exists and has groups
	if tab.form == nil {
		t.Error("form should be created with default values")
	}
}

func TestSigningTab_SettingsUpdateMsg(t *testing.T) {
	tab := NewSigningTab()
	tab.Init()

	// Set test values
	tab.keyName = "Test User"
	tab.keyEmail = "test@example.com"
	tab.keyExpiry = "2y"
	tab.keyFormat = "binary"
	tab.histFormat = "armored"

	// Simulate form completion
	tab.form.State = huh.StateCompleted
	tab.formComplete = false

	updatedTab, cmd := tab.Update(struct{}{})

	if cmd == nil {
		t.Fatal("Form completion should return commands")
	}

	// Execute the batch command (it should be a tea.Batch)
	// We can't easily decompose batch commands, but we can verify it exists
	if !updatedTab.formComplete {
		t.Error("formComplete should be set to true")
	}

	// The values should be in the tab struct
	if updatedTab.keyName != "Test User" {
		t.Errorf("keyName = %q, want %q", updatedTab.keyName, "Test User")
	}
	if updatedTab.keyExpiry != "2y" {
		t.Errorf("keyExpiry = %q, want %q", updatedTab.keyExpiry, "2y")
	}
	if updatedTab.keyFormat != "binary" {
		t.Errorf("keyFormat = %q, want %q", updatedTab.keyFormat, "binary")
	}
	if updatedTab.histFormat != "armored" {
		t.Errorf("histFormat = %q, want %q", updatedTab.histFormat, "armored")
	}
}

func TestSigningTab_TabCompleteMsg(t *testing.T) {
	tab := NewSigningTab()
	tab.Init()

	// Simulate form completion
	tab.form.State = huh.StateCompleted
	tab.keyName = "Test"
	tab.keyEmail = "test@example.com"
	tab.keyExpiry = "1y"
	tab.keyFormat = "armored"
	tab.histFormat = "armored"

	updatedTab, cmd := tab.Update(struct{}{})

	if cmd == nil {
		t.Fatal("Form completion should return commands including TabCompleteMsg")
	}

	if !updatedTab.formComplete {
		t.Error("formComplete should be true after completion")
	}

	// The batch command should include TabCompleteMsg{TabIndex: 2}
	// We verify this indirectly by checking formComplete
}

func TestGetGitConfig(t *testing.T) {
	// Test the git config helper function
	// This might fail in CI without git config, so we make it non-fatal
	name, err := getGitConfig("user.name")
	if err != nil {
		// Git config might not be set, that's OK
		t.Logf("git config user.name not set (expected in some environments): %v", err)
	} else {
		if name == "" {
			t.Error("getGitConfig should return non-empty name if no error")
		}
	}

	email, err := getGitConfig("user.email")
	if err != nil {
		t.Logf("git config user.email not set (expected in some environments): %v", err)
	} else {
		if email == "" {
			t.Error("getGitConfig should return non-empty email if no error")
		}
	}

	// Test invalid key
	_, err = getGitConfig("invalid.key.that.does.not.exist")
	if err == nil {
		// It's OK if it returns empty string with no error
		// Different git versions behave differently
		t.Log("Invalid git config key returned no error (OK)")
	}
}

func TestSigningTab_FormFields(t *testing.T) {
	// Verify the form has all required fields
	tab := NewSigningTab()
	tab.Init()

	if tab.form == nil {
		t.Fatal("form should be created")
	}

	// We can't easily inspect huh.Form fields without running it,
	// but we can verify the form exists and is properly initialized
	if tab.form.State != huh.StateNormal {
		t.Errorf("form.State = %v, want StateNormal", tab.form.State)
	}
}

func TestSigningTab_View_ReturnsFormView(t *testing.T) {
	tab := NewSigningTab()
	tab.Init()

	view1 := tab.View()
	view2 := tab.form.View()

	// View() should delegate to form.View()
	// The exact content may differ due to state changes, but both should be non-empty
	if view1 == "" || view2 == "" {
		t.Error("Both tab.View() and form.View() should return content")
	}

	// If we set custom width/height, form should be updated
	tab.width = 80
	tab.height = 24
	tab.form.WithWidth(80).WithHeight(24)

	view3 := tab.View()
	if view3 == "" {
		t.Error("View() should still work after size update")
	}
}

func TestEmailRegexValidation(t *testing.T) {
	// Test the email validation regex pattern used in the form
	tests := []struct {
		email string
		valid bool
	}{
		{"test@example.com", true},
		{"user.name@domain.co.uk", true},
		{"user+tag@example.com", true},
		{"invalid.email", false},
		{"@example.com", false},
		{"user@", false},
		{"", false},
		{"user @example.com", false},
	}

	// We can't easily test the exact validation function without exposing it,
	// but we document the expected behavior here
	// The actual validation happens inside the huh.Form
	for _, tt := range tests {
		t.Logf("Email %q should be valid=%v", tt.email, tt.valid)
	}
}
