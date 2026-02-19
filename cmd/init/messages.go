// SPDX-License-Identifier: Apache-2.0
package init

import (
	initpkg "github.com/Work-Fort/Anvil/pkg/init"
)

// TabCompleteMsg signals a tab has completed
type TabCompleteMsg struct {
	TabIndex int
}

// SettingsUpdateMsg carries settings from a tab to parent
type SettingsUpdateMsg struct {
	Settings initpkg.InitSettings
}

// TabErrorMsg carries error from a tab to parent
type TabErrorMsg struct {
	TabIndex int
	Error    error
}

// GenerationCompleteMsg signals file generation finished
type GenerationCompleteMsg struct {
	FilesCreated []string
	Error        error
}
