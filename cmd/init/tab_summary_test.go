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

func TestSummaryTab_Init(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:       "Test User",
		KeyEmail:      "test@example.com",
		KeyGenerated:  true,
		KeyPath:       "/path/to/key",
		PublicKeyPath: "/path/to/key.pub",
	}

	tab := NewSummaryTab(settings)
	cmd := tab.Init()

	if cmd == nil {
		t.Fatal("Init() should return a command")
	}

	if tab.complete {
		t.Error("Init() should not immediately set complete to true")
	}

	if tab.err != nil {
		t.Errorf("Init() should not set error, got: %v", tab.err)
	}
}

func TestSummaryTab_IsComplete(t *testing.T) {
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
			tab := &SummaryTab{complete: tt.complete, err: tt.err}
			if got := tab.IsComplete(); got != tt.want {
				t.Errorf("IsComplete() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummaryTab_GetState(t *testing.T) {
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
			tab := &SummaryTab{complete: tt.complete, err: tt.err}
			if got := tab.GetState(); got != tt.want {
				t.Errorf("GetState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSummaryTab_Update_WindowSize(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewSummaryTab(settings)

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

func TestSummaryTab_Update_GenerateFiles(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:  "Test User",
		KeyEmail: "test@example.com",
	}
	tab := NewSummaryTab(settings)

	msg := generateFilesMsg{}
	updated, cmd := tab.Update(msg)

	if cmd == nil {
		t.Fatal("Update() should return a command for generateFilesMsg")
	}

	if updated.complete {
		t.Error("complete should not be set until filesGeneratedMsg is processed")
	}
}

func TestSummaryTab_Update_FilesGenerated_Success(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:  "Test User",
		KeyEmail: "test@example.com",
	}
	tab := NewSummaryTab(settings)

	files := []string{
		"anvil.yaml",
		".gitignore",
		"configs/kernel-x86_64.config",
		"configs/kernel-aarch64.config",
	}
	msg := filesGeneratedMsg{filesCreated: files, err: nil}
	updated, cmd := tab.Update(msg)

	if cmd != nil {
		t.Error("Update() should return nil command for filesGeneratedMsg success")
	}

	if !updated.complete {
		t.Error("complete should be true after successful filesGeneratedMsg")
	}

	if updated.err != nil {
		t.Errorf("err should be nil, got: %v", updated.err)
	}

	if len(updated.filesCreated) != len(files) {
		t.Errorf("filesCreated length = %d, want %d", len(updated.filesCreated), len(files))
	}

	for i, file := range files {
		if updated.filesCreated[i] != file {
			t.Errorf("filesCreated[%d] = %q, want %q", i, updated.filesCreated[i], file)
		}
	}
}

func TestSummaryTab_Update_FilesGenerated_Error(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:  "Test User",
		KeyEmail: "test@example.com",
	}
	tab := NewSummaryTab(settings)

	testErr := fmt.Errorf("file generation failed")
	msg := filesGeneratedMsg{filesCreated: nil, err: testErr}
	updated, cmd := tab.Update(msg)

	if cmd != nil {
		t.Error("Update() should return nil command when error occurs")
	}

	if !updated.complete {
		t.Error("complete should be true even when error occurs")
	}

	if updated.err == nil {
		t.Error("err should be set when filesGeneratedMsg contains error")
	}

	if updated.err != testErr {
		t.Errorf("err = %v, want %v", updated.err, testErr)
	}

	if len(updated.filesCreated) != 0 {
		t.Errorf("filesCreated should be empty when error occurs, got %d files", len(updated.filesCreated))
	}
}

func TestSummaryTab_Update_Retry(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:  "Test User",
		KeyEmail: "test@example.com",
	}
	tab := NewSummaryTab(settings)
	tab.complete = true
	tab.err = fmt.Errorf("previous error")

	msg := tea.KeyPressMsg{Code: 'r', Text: "r"}
	updated, cmd := tab.Update(msg)

	if cmd == nil {
		t.Fatal("Update() should return a command for retry")
	}

	if updated.err != nil {
		t.Error("err should be cleared on retry")
	}

	if updated.complete {
		t.Error("complete should be reset on retry")
	}
}

func TestSummaryTab_Update_Quit_Success(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewSummaryTab(settings)
	tab.complete = true
	tab.err = nil
	tab.filesCreated = []string{"file1.txt", "file2.txt"}

	tests := []struct {
		name string
		msg  tea.KeyPressMsg
	}{
		{"enter key", tea.KeyPressMsg{Code: tea.KeyEnter}},
		{"q key", tea.KeyPressMsg{Code: 'q', Text: "q"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cmd := tab.Update(tt.msg)

			if cmd == nil {
				t.Fatal("Update() should return tea.Quit command")
			}

			result := cmd()
			if _, ok := result.(tea.QuitMsg); !ok {
				t.Error("Command should return tea.QuitMsg")
			}
		})
	}
}

func TestSummaryTab_Update_Quit_Error(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewSummaryTab(settings)
	tab.complete = true
	tab.err = fmt.Errorf("test error")

	msg := tea.KeyPressMsg{Code: 'q', Text: "q"}
	_, cmd := tab.Update(msg)

	if cmd == nil {
		t.Fatal("Update() should return tea.Quit command")
	}

	result := cmd()
	if _, ok := result.(tea.QuitMsg); !ok {
		t.Error("Command should return tea.QuitMsg")
	}
}

func TestSummaryTab_View_Generating(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewSummaryTab(settings)
	tab.complete = false

	view := tab.View()

	if !strings.Contains(view, "Generating repository files") {
		t.Error("View() should contain generating message when not complete")
	}
}

func TestSummaryTab_View_Complete_Success(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewSummaryTab(settings)
	tab.complete = true
	tab.err = nil
	tab.filesCreated = []string{
		"anvil.yaml",
		".gitignore",
		"configs/kernel-x86_64.config",
	}

	view := tab.View()

	if !strings.Contains(view, "anvil.yaml") {
		t.Error("View() should contain created file names")
	}

	if !strings.Contains(view, ".gitignore") {
		t.Error("View() should contain .gitignore")
	}

	if !strings.Contains(view, "configs/kernel-x86_64.config") {
		t.Error("View() should contain kernel config file")
	}

	if !strings.Contains(view, "✓") {
		t.Error("View() should contain checkmark for completed generation")
	}

	if !strings.Contains(view, "Enter") || !strings.Contains(view, "exit") {
		t.Error("View() should contain exit instructions")
	}
}

func TestSummaryTab_View_Error(t *testing.T) {
	settings := &initpkg.InitSettings{}
	tab := NewSummaryTab(settings)
	tab.complete = true
	tab.err = fmt.Errorf("test error message")

	view := tab.View()

	if !strings.Contains(view, "test error message") {
		t.Error("View() should contain error message when error exists")
	}

	if !strings.Contains(view, "r") || !strings.Contains(view, "retry") {
		t.Error("View() should contain retry instructions")
	}

	if !strings.Contains(view, "q") || !strings.Contains(view, "quit") {
		t.Error("View() should contain quit instructions")
	}
}

func TestSummaryTab_AsyncFlow(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:  "Test User",
		KeyEmail: "test@example.com",
	}
	tab := NewSummaryTab(settings)

	// Step 1: Init returns commands
	cmd := tab.Init()
	if cmd == nil {
		t.Fatal("Init() should return a command")
	}

	// Step 2: Send generateFilesMsg
	genTab, genCmd := tab.Update(generateFilesMsg{})
	if genCmd == nil {
		t.Fatal("generateFilesMsg should return a command")
	}

	if genTab.complete {
		t.Error("Tab should not be marked as complete yet")
	}

	// Step 3: Simulate successful file generation
	files := []string{
		"anvil.yaml",
		".gitignore",
		"configs/kernel-x86_64.config",
		"configs/kernel-aarch64.config",
	}
	filesGenMsg := filesGeneratedMsg{filesCreated: files, err: nil}

	// Step 4: Send filesGeneratedMsg
	finalTab, finalCmd := genTab.Update(filesGenMsg)
	if finalCmd != nil {
		t.Error("filesGeneratedMsg should not return a command on success")
	}

	if !finalTab.complete {
		t.Error("Tab should be marked as complete")
	}

	if finalTab.err != nil {
		t.Errorf("Tab should have no error, got: %v", finalTab.err)
	}

	if len(finalTab.filesCreated) != len(files) {
		t.Errorf("filesCreated length = %d, want %d", len(finalTab.filesCreated), len(files))
	}
}

func TestSummaryTab_ErrorRetryFlow(t *testing.T) {
	settings := &initpkg.InitSettings{
		KeyName:  "Test User",
		KeyEmail: "test@example.com",
	}
	tab := NewSummaryTab(settings)

	// Step 1: Simulate file generation error
	errMsg := filesGeneratedMsg{err: fmt.Errorf("generation failed")}

	errTab, _ := tab.Update(errMsg)

	if !errTab.complete {
		t.Error("Tab should be marked as complete after error")
	}

	if errTab.err == nil {
		t.Error("Tab should have error set")
	}

	// Step 2: Retry
	retryMsg := tea.KeyPressMsg{Code: 'r', Text: "r"}
	retryTab, retryCmd := errTab.Update(retryMsg)

	if retryCmd == nil {
		t.Fatal("Retry should return a command")
	}

	if retryTab.err != nil {
		t.Error("Error should be cleared on retry")
	}

	if retryTab.complete {
		t.Error("Complete should be reset on retry")
	}

	// Step 3: Simulate successful generation after retry
	successMsg := filesGeneratedMsg{
		filesCreated: []string{"file1.txt", "file2.txt"},
		err:          nil,
	}

	successTab, _ := retryTab.Update(successMsg)

	if !successTab.complete {
		t.Error("Tab should be complete after successful retry")
	}

	if successTab.err != nil {
		t.Errorf("Tab should have no error after successful retry, got: %v", successTab.err)
	}

	if len(successTab.filesCreated) != 2 {
		t.Errorf("filesCreated length = %d, want 2", len(successTab.filesCreated))
	}
}
