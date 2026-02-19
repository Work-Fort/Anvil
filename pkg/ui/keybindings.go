// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// KeyBinding represents a single key action
type KeyBinding struct {
	Key         string   // Display name: "ENTER", "TAB", "DEL"
	Keys        []string // Actual keys to match: ["enter"], ["tab"], ["delete", "backspace"]
	Description string   // What it does
}

// KeyBindingSet is a collection of related key bindings
type KeyBindingSet struct {
	Bindings []KeyBinding
}

// Contains checks if a key press matches any binding in the set
func (kbs KeyBindingSet) Contains(key string) *KeyBinding {
	for i := range kbs.Bindings {
		for _, k := range kbs.Bindings[i].Keys {
			if k == key {
				return &kbs.Bindings[i]
			}
		}
	}
	return nil
}

// Render formats key bindings for display
// Format: "[KEY] Action  •  [KEY] Action"
func (kbs KeyBindingSet) Render(style lipgloss.Style) string {
	if len(kbs.Bindings) == 0 {
		return ""
	}

	parts := make([]string, len(kbs.Bindings))
	for i, binding := range kbs.Bindings {
		parts[i] = fmt.Sprintf("[%s] %s", binding.Key, binding.Description)
	}

	return style.Render(strings.Join(parts, "  •  "))
}

// RenderInline formats key bindings for inline display (more compact)
// Format: "Key: action | Key: action"
func (kbs KeyBindingSet) RenderInline(style lipgloss.Style) string {
	if len(kbs.Bindings) == 0 {
		return ""
	}

	parts := make([]string, len(kbs.Bindings))
	caser := cases.Title(language.Und, cases.NoLower)
	for i, binding := range kbs.Bindings {
		// Use first key alias for display (e.g., "enter" instead of showing all)
		keyName := caser.String(binding.Keys[0])
		parts[i] = fmt.Sprintf("%s: %s", keyName, strings.ToLower(binding.Description))
	}

	return style.Render(strings.Join(parts, " | "))
}

// Common key binding sets for version selector

// GlobalKeyBindings returns the global key bindings (work in any pane)
func GlobalKeyBindings() KeyBindingSet {
	return KeyBindingSet{
		Bindings: []KeyBinding{
			{Key: "TAB", Keys: []string{"tab"}, Description: "Switch Panes"},
			{Key: "ESC", Keys: []string{"esc", "ctrl+c"}, Description: "Exit"},
		},
	}
}

// DownloadedPaneKeyBindings returns key bindings for the downloaded pane
func DownloadedPaneKeyBindings() KeyBindingSet {
	return KeyBindingSet{
		Bindings: []KeyBinding{
			{Key: "ENTER", Keys: []string{"enter"}, Description: "Set Default"},
			{Key: "DEL", Keys: []string{"delete", "backspace"}, Description: "Remove"},
		},
	}
}

// AvailablePaneKeyBindings returns key bindings for the available pane
func AvailablePaneKeyBindings() KeyBindingSet {
	return KeyBindingSet{
		Bindings: []KeyBinding{
			{Key: "ENTER", Keys: []string{"enter"}, Description: "Download"},
		},
	}
}
