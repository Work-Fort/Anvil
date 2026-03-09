// SPDX-License-Identifier: Apache-2.0
package init

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
	"github.com/Work-Fort/Anvil/pkg/ui"
)

func TestKeygenTab_Init(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:       "Test User",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}

	tab := NewKeygenTab(settings)
	cmd := tab.Init()

	if cmd == nil {
		t.Fatal("Init() should return a command")
	}

	if tab.complete {
		t.Error("Init() should not immediately set complete to true")
	}

	if tab.generating {
		t.Error("Init() should not immediately set generating to true")
	}

	if tab.err != nil {
		t.Errorf("Init() should not set error, got: %v", tab.err)
	}
}

func TestKeygenTab_IsComplete(t *testing.T) {
	tests := []struct {
		name     string
		complete bool
		err      error
		want     bool
	}{
		{"complete when complete and no error", true, nil, true},
		{"not complete when not complete", false, nil, false},
		{"not complete when error exists", true, fmt.Errorf("test error"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tab := &KeygenTab{complete: tt.complete, err: tt.err}
			if got := tab.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeygenTab_GetState(t *testing.T) {
	tests := []struct {
		name     string
		complete bool
		err      error
		want     ui.TabState
	}{
		{"returns TabComplete when complete and no error", true, nil, ui.TabComplete},
		{"returns TabError when error exists", true, fmt.Errorf("test error"), ui.TabError},
		{"returns TabActive when not complete", false, nil, ui.TabActive},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tab := &KeygenTab{complete: tt.complete, err: tt.err}
			if got := tab.GetState(); got != tt.want {
				t.Errorf("GetState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestKeygenTab_Update_WindowSize(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewKeygenTab(settings)

	msg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updated, cmd := tab.Update(msg)

	if cmd != nil {
		t.Error("WindowSizeMsg should return nil command")
	}

	if updated.width != 100 {
		t.Errorf("width = %d, want 100", updated.width)
	}
	if updated.height != 50 {
		t.Errorf("height = %d, want 50", updated.height)
	}
}

func TestKeygenTab_Update_GenerateKey(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:       "Test User",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}
	tab := NewKeygenTab(settings)

	msg := generateKeyMsg{}
	updated, cmd := tab.Update(msg)

	if cmd == nil {
		t.Fatal("Update() should return a command for generateKeyMsg")
	}

	if !updated.generating {
		t.Error("generating should be set to true when generateKeyMsg is received")
	}

	if updated.complete {
		t.Error("complete should not be set until keyGeneratedMsg is processed")
	}
}

func TestKeygenTab_Update_KeyGenerated_Success(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:       "Test User",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}
	tab := NewKeygenTab(settings)
	tab.generating = true

	msg := keyGeneratedMsg{
		keyPath:       "/path/to/private.key",
		publicKeyPath: "/path/to/public.key",
		err:           nil,
	}
	updated, cmd := tab.Update(msg)

	if cmd == nil {
		t.Fatal("Update() should return commands for keyGeneratedMsg")
	}

	if !updated.complete {
		t.Error("complete should be true after successful keyGeneratedMsg")
	}

	if updated.err != nil {
		t.Errorf("err should be nil, got: %v", updated.err)
	}

	if updated.settings.KeyPath != "/path/to/private.key" {
		t.Errorf("KeyPath = %q, want %q", updated.settings.KeyPath, "/path/to/private.key")
	}
	if updated.settings.PublicKeyPath != "/path/to/public.key" {
		t.Errorf("PublicKeyPath = %q, want %q", updated.settings.PublicKeyPath, "/path/to/public.key")
	}
	if !updated.settings.KeyGenerated {
		t.Error("KeyGenerated should be true")
	}
}

func TestKeygenTab_Update_KeyGenerated_Error(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:       "Test User",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}
	tab := NewKeygenTab(settings)
	tab.generating = true

	testErr := fmt.Errorf("key generation failed")
	msg := keyGeneratedMsg{err: testErr}
	updated, cmd := tab.Update(msg)

	if cmd != nil {
		t.Error("Update() should return nil command when error occurs")
	}

	if !updated.complete {
		t.Error("complete should be true even when error occurs")
	}

	if updated.err == nil {
		t.Error("err should be set when keyGeneratedMsg contains error")
	}

	if updated.err != testErr {
		t.Errorf("err = %v, want %v", updated.err, testErr)
	}

	if updated.settings.KeyGenerated {
		t.Error("KeyGenerated should be false when error occurs")
	}
}

func TestKeygenTab_View_Generating(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewKeygenTab(settings)
	tab.generating = true
	tab.complete = false

	view := tab.View()

	if !strings.Contains(view, "Generating encrypted signing key") {
		t.Error("View() should contain generating message when generating")
	}
}

func TestKeygenTab_View_Complete(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyPath:       "/path/to/private.key",
		PublicKeyPath: "/path/to/public.key",
		KeyGenerated:  true,
	}
	tab := NewKeygenTab(settings)
	tab.complete = true
	tab.err = nil

	view := tab.View()

	if !strings.Contains(view, "/path/to/private.key") {
		t.Error("View() should contain private key path")
	}
	if !strings.Contains(view, "/path/to/public.key") {
		t.Error("View() should contain public key path")
	}
	if !strings.Contains(view, "✓") {
		t.Error("View() should contain checkmark for completed generation")
	}
}

func TestKeygenTab_View_Error(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewKeygenTab(settings)
	tab.complete = true
	tab.err = fmt.Errorf("test error message")

	view := tab.View()

	if !strings.Contains(view, "test error message") {
		t.Error("View() should contain error message when error exists")
	}

	if !strings.Contains(view, "Failed to generate signing key") {
		t.Error("View() should contain error header")
	}
}

func TestKeygenTab_AsyncFlow(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:       "Test User",
		KeyEmail:      "test@example.com",
		KeyExpiry:     "1y",
		KeyFormat:     "armored",
		HistoryFormat: "armored",
	}
	tab := NewKeygenTab(settings)

	// Step 1: Init returns commands
	cmd := tab.Init()
	if cmd == nil {
		t.Fatal("Init() should return a command")
	}

	// Step 2: Send generateKeyMsg
	genTab, genCmd := tab.Update(generateKeyMsg{})
	if genCmd == nil {
		t.Fatal("generateKeyMsg should return a command")
	}

	if !genTab.generating {
		t.Error("Tab should be marked as generating")
	}

	// Step 3: Simulate successful key generation response
	keyGenMsg := keyGeneratedMsg{
		keyPath:       "/test/private.key",
		publicKeyPath: "/test/public.key",
		err:           nil,
	}

	// Step 4: Send keyGeneratedMsg
	finalTab, finalCmd := genTab.Update(keyGenMsg)
	if finalCmd == nil {
		t.Fatal("keyGeneratedMsg should return commands")
	}

	if !finalTab.complete {
		t.Error("Tab should be marked as complete")
	}

	if finalTab.err != nil {
		t.Errorf("Tab should have no error, got: %v", finalTab.err)
	}

	if !finalTab.settings.KeyGenerated {
		t.Error("Settings should have KeyGenerated=true")
	}

	if finalTab.settings.KeyPath != "/test/private.key" {
		t.Errorf("KeyPath = %q, want %q", finalTab.settings.KeyPath, "/test/private.key")
	}

	if finalTab.settings.PublicKeyPath != "/test/public.key" {
		t.Errorf("PublicKeyPath = %q, want %q", finalTab.settings.PublicKeyPath, "/test/public.key")
	}
}
