// SPDX-License-Identifier: Apache-2.0
package ui

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/log"
	"github.com/Work-Fort/Anvil/pkg/config"
	"github.com/Work-Fort/Anvil/pkg/kernel"
)

// BuildKernelPhase represents a phase in the kernel build process
type BuildKernelPhase int

const (
	PhaseSelectVersion BuildKernelPhase = iota
	PhaseDownload
	PhaseVerify
	PhaseExtract
	PhaseConfigure
	PhaseCompile
	PhasePackage
	PhaseComplete
)

// BuildKernelWizard is the unified tabbed wizard for kernel building
type BuildKernelWizard struct {
	width  int
	height int

	// Tabs configuration
	tabs              []Tab
	activePhase       BuildKernelPhase // Which tab the user is viewing
	currentBuildPhase BuildKernelPhase // Which phase the build is actually on

	// Version selection (Phase 0)
	versions        []list.Item
	versionList     list.Model
	selectedVersion string

	// Build options
	arch              string
	verificationLevel string
	configFile        string

	// Build state
	buildStarted bool
	buildOutput  []string                      // All build output lines (for viewport)
	phaseOutput  map[BuildKernelPhase][]string // Output per phase

	// Progress tracking
	progressBar     progress.Model
	progressPercent float64
	progressChan    chan float64
	phaseChan       chan kernel.BuildPhase

	// Scrollable viewport for build output
	viewport      viewport.Model
	viewportReady bool

	// Build statistics
	buildStats BuildStats

	// Installation state
	kernelInstalled    bool
	installedVersion   string
	kernelSetAsDefault bool
	installingKernel   bool
	installError       error

	// UI state
	quitting           bool
	err                error
	confirmingNewBuild bool
	confirmingInstall  bool
	confirmForm        *ConfirmationForm
	loadingCachedBuild bool
	forceRebuild       bool
	isCachedBuild      bool // True when showing a cached build (no actual build ran)
	manualTabMode      bool // True when user manually switched tabs (don't auto-follow build)

	// Build goroutine communication
	buildOutputChan chan string
	buildDoneChan   chan error
	buildStatsChan  chan kernel.BuildStats
	buildCtx        context.Context
	buildCancel     context.CancelFunc
}

// versionItem implements list.Item
type versionItem struct {
	version     string
	isLatest    bool
	description string
}

func (v versionItem) FilterValue() string { return v.version }
func (v versionItem) Title() string {
	if v.isLatest {
		return v.version + " (latest)"
	}
	return v.version
}
func (v versionItem) Description() string { return v.description }

// FetchVersionsMsg contains available kernel versions
type FetchVersionsMsg struct {
	Versions []string
	Error    error
}

// VersionSelectedMsg indicates user selected a version
type VersionSelectedMsg struct {
	Version string
}

// BuildOutputMsg contains output from a build phase
type BuildOutputMsg struct {
	Output string
}

// BuildPhaseTransitionMsg signals a phase transition
type BuildPhaseTransitionMsg struct {
	Phase kernel.BuildPhase
}

// BuildCompleteMsg signals the entire build is complete
type BuildCompleteMsg struct {
	Success bool
	Error   error
	Stats   kernel.BuildStats
}

// BuildStats contains statistics about the build
type BuildStats struct {
	TotalDuration     time.Duration
	DownloadDuration  time.Duration
	ExtractDuration   time.Duration
	ConfigureDuration time.Duration
	CompileDuration   time.Duration
	PackageDuration   time.Duration
	UncompressedSize  int64
	CompressedSize    int64
	UncompressedHash  string
	CompressedHash    string
	KernelVersion     string
	OutputPath        string
	CompressedPath    string
	BuildTimestamp    time.Time // Timestamp when build completed
}

// DownloadProgressMsg contains download progress updates
type DownloadProgressMsg struct {
	Percent float64
}

// InstallKernelMsg signals kernel installation has completed
type InstallKernelMsg struct {
	Success          bool
	InstalledVersion string
	SetAsDefault     bool
	Error            error
}

// CachedBuildLoadedMsg signals a cached build was loaded
type CachedBuildLoadedMsg struct {
	Stats kernel.BuildStats
	Error error
}

// NewBuildStartedMsg signals the cache was cleared and a new build can start
type NewBuildStartedMsg struct {
	Error error
}

// NewBuildKernelWizard creates a new kernel build wizard with tabs
func NewBuildKernelWizard(arch, verificationLevel, configFile string, forceRebuild bool) *BuildKernelWizard {
	theme := config.CurrentTheme

	// Create spinners for each tab
	spinners := make([]spinner.Model, 8)
	for i := range spinners {
		spinners[i] = spinner.New()
		spinners[i].Spinner = spinner.Dot
		spinners[i].Style = lipgloss.NewStyle().Foreground(theme.GetPrimaryColor())
	}

	// Create list delegate for version selection
	delegate := list.NewDefaultDelegate()
	delegate.Styles.SelectedTitle = delegate.Styles.SelectedTitle.
		Foreground(theme.GetPrimaryColor()).
		BorderForeground(theme.GetPrimaryColor())
	delegate.Styles.SelectedDesc = delegate.Styles.SelectedDesc.
		Foreground(theme.GetSecondaryColor())

	// Create list
	l := list.New([]list.Item{}, delegate, 0, 0)
	l.Title = ""
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(true)
	l.SetShowHelp(false)

	// Create progress bar with theme colors
	prog := progress.New(progress.WithGradient(theme.Secondary, theme.Primary))

	// Create viewport for scrollable build output
	vp := viewport.New(0, 0)

	return &BuildKernelWizard{
		tabs: []Tab{
			{Title: "Select", State: TabActive, Spinner: spinners[0]},
			{Title: "Download", State: TabPending, Spinner: spinners[1]},
			{Title: "Verify", State: TabPending, Spinner: spinners[2]},
			{Title: "Extract", State: TabPending, Spinner: spinners[3]},
			{Title: "Configure", State: TabPending, Spinner: spinners[4]},
			{Title: "Compile", State: TabPending, Spinner: spinners[5]},
			{Title: "Package", State: TabPending, Spinner: spinners[6]},
			{Title: "Complete", State: TabPending, Spinner: spinners[7]},
		},
		activePhase:       PhaseSelectVersion,
		currentBuildPhase: PhaseSelectVersion,
		versionList:       l,
		arch:              arch,
		verificationLevel: verificationLevel,
		configFile:        configFile,
		buildOutput:       []string{},
		phaseOutput:       make(map[BuildKernelPhase][]string),
		progressBar:       prog,
		viewport:          vp,
		forceRebuild:      forceRebuild,
	}
}

// Init initializes the wizard
func (m *BuildKernelWizard) Init() tea.Cmd {
	// Skip cached build if force rebuild is requested
	if !m.forceRebuild {
		// Check if a cached build exists
		hasCached, statsFile, err := kernel.CheckCachedBuild("")
		if err != nil {
			log.Debugf("Error checking cached build: %v", err)
		}

		if hasCached && statsFile != "" {
			// Load cached build stats - skip version selection
			log.Debugf("Found cached build, loading stats from: %s", statsFile)
			m.loadingCachedBuild = true
			return m.loadCachedBuild(statsFile)
		}
	} else {
		log.Debugf("Force rebuild requested, skipping cached build check")
	}

	// Start all spinners and fetch versions
	cmds := make([]tea.Cmd, len(m.tabs)+1)
	for i := range m.tabs {
		cmds[i] = m.tabs[i].Spinner.Tick
	}
	cmds[len(m.tabs)] = fetchKernelVersions
	return tea.Batch(cmds...)
}

// fetchKernelVersions fetches available kernel versions
func fetchKernelVersions() tea.Msg {
	versions, err := getKernelVersions()
	if err != nil {
		return FetchVersionsMsg{Error: err}
	}
	return FetchVersionsMsg{Versions: versions}
}

// getKernelVersions fetches kernel versions from kernel.org
func getKernelVersions() ([]string, error) {
	resp, err := http.Get("https://www.kernel.org/releases.json")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch kernel.org API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("kernel.org API returned status: %s", resp.Status)
	}

	// Parse the full releases structure
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to parse kernel.org API response: %w", err)
	}

	// Extract releases array
	releases, ok := data["releases"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected API structure")
	}

	// Extract version strings
	var versions []string
	for _, release := range releases {
		releaseMap, ok := release.(map[string]interface{})
		if !ok {
			continue
		}
		if ver, ok := releaseMap["version"].(string); ok {
			// Filter out "next-" versions for cleaner list
			if !strings.HasPrefix(ver, "next-") {
				versions = append(versions, ver)
			}
		}
	}

	if len(versions) == 0 {
		return nil, fmt.Errorf("no versions found")
	}

	return versions, nil
}

// Update handles messages
func (m *BuildKernelWizard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle confirmation form if active (must be before type switch)
	if m.confirmingInstall && m.confirmForm != nil {
		confirmed, shouldProceed, cmd := m.confirmForm.Update(msg)

		if shouldProceed {
			// User made a choice (Y/N or completed form)
			log.Debugf("Install confirmation: confirmed=%v", confirmed)
			m.confirmingInstall = false
			m.installingKernel = true
			return m, tea.Batch(cmd, m.installKernel(confirmed))
		}

		// Check for ESC to cancel
		if keyMsg, ok := msg.(tea.KeyMsg); ok && keyMsg.String() == "esc" {
			log.Debugf("User cancelled installation")
			m.confirmingInstall = false
			return m, nil
		}

		// Still collecting input
		return m, cmd
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate dimensions for content using helpers
		dims := CalculateSplitPaneDimensions(m.width, m.height)

		// Calculate content height
		const (
			headerLines   = 3
			tabLines      = 3
			helpLines     = 1
			blankLines    = 3
			borderLines   = 2
			minListHeight = 10
		)

		contentHeight := m.height - headerLines - tabLines - helpLines - blankLines - borderLines
		if contentHeight < minListHeight {
			contentHeight = minListHeight
		}

		// Update list size for version selection
		m.versionList.SetSize(dims.PaneContentWidth, contentHeight)

		// Update viewport size for build output (use full content area)
		m.viewport.Width = dims.PaneContentWidth
		m.viewport.Height = contentHeight
		m.viewportReady = true

		log.Debugf("BuildKernelWizard WindowSize: %dx%d, contentHeight=%d, viewportSize=%dx%d", m.width, m.height, contentHeight, m.viewport.Width, m.viewport.Height)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q", "esc":
			// If confirming install, cancel the confirmation
			if m.confirmingInstall {
				log.Debugf("User cancelled install confirmation")
				m.confirmingInstall = false
				return m, nil
			}
			// If confirming new build, cancel the confirmation
			if m.confirmingNewBuild {
				log.Debugf("User cancelled new build confirmation")
				m.confirmingNewBuild = false
				return m, nil
			}

			m.quitting = true
			log.Debugf("User quit during phase=%d, buildStarted=%v", m.activePhase, m.buildStarted)

			// If on completion screen, just quit
			if m.activePhase == PhaseComplete {
				return m, tea.Quit
			}

			// Cancel the build goroutine if it's running
			if m.buildCancel != nil {
				log.Debugf("Cancelling build context")
				m.buildCancel()
			}
			return m, tea.Quit

		case "i", "I":
			// Install kernel (only on completion screen)
			if m.activePhase == PhaseComplete && !m.kernelInstalled && !m.installingKernel && !m.confirmingInstall {
				log.Debugf("User requested kernel installation")
				m.confirmingInstall = true
				// Create confirmation form
				m.confirmForm = NewConfirmationForm(
					"setAsDefault",
					"Set as default kernel?",
					"Install this kernel and optionally set it as the default.",
					"Yes - Set as default",
					"No - Install only",
				)
				return m, m.confirmForm.Init()
			}
			return m, nil

		case "n", "N":
			// Handle N key on completion screen (only for new build confirmation)
			if m.activePhase == PhaseComplete {
				// If confirming new build, N means cancel
				if m.confirmingNewBuild {
					log.Debugf("User cancelled new build")
					m.confirmingNewBuild = false
					return m, nil
				}
				// Otherwise, ask for new build confirmation
				log.Debugf("User requested new build, asking for confirmation")
				m.confirmingNewBuild = true
				return m, nil
			}
			return m, nil

		case "y", "Y":
			// Handle Y key on completion screen (only for new build confirmation)
			if m.activePhase == PhaseComplete && m.confirmingNewBuild {
				log.Debugf("User confirmed new build, clearing cache")
				m.confirmingNewBuild = false
				return m, m.startNewBuild()
			}
			return m, nil

		case "enter":
			if m.activePhase == PhaseSelectVersion && !m.buildStarted {
				// Get selected version
				selected := m.versionList.SelectedItem()
				if selected != nil {
					if vItem, ok := selected.(versionItem); ok {
						m.selectedVersion = vItem.version
						m.buildStarted = true

						// Transition to download phase
						m.tabs[PhaseSelectVersion].State = TabComplete
						m.tabs[PhaseDownload].State = TabActive
						m.activePhase = PhaseDownload
						m.currentBuildPhase = PhaseDownload

						log.Debugf("Version selected: %s, starting build", m.selectedVersion)

						// Start build process
						return m, m.startBuild()
					}
				}
			}

		case "tab", "right":
			// Switch to next tab (only during build, not for cached builds)
			if m.buildStarted && !m.isCachedBuild && int(m.activePhase) < len(m.tabs)-1 {
				m.activePhase++
				m.manualTabMode = true // User manually switched tabs
				log.Debugf("User switched to next tab, activePhase=%d, manualTabMode=true", m.activePhase)
			}
			return m, nil

		case "shift+tab", "left":
			// Switch to previous tab (only during build, not for cached builds)
			if m.buildStarted && !m.isCachedBuild && m.activePhase > PhaseDownload {
				m.activePhase--
				m.manualTabMode = true // User manually switched tabs
				log.Debugf("User switched to previous tab, activePhase=%d, manualTabMode=true", m.activePhase)
			}
			return m, nil

		case "up", "k":
			if m.activePhase == PhaseSelectVersion && !m.buildStarted {
				var cmd tea.Cmd
				m.versionList, cmd = m.versionList.Update(msg)
				return m, cmd
			}
			// Scroll viewport up (when build is running and viewport is active)
			if m.buildStarted && m.viewportReady {
				m.viewport.ScrollUp(1)
			}
			return m, nil

		case "down", "j":
			if m.activePhase == PhaseSelectVersion && !m.buildStarted {
				var cmd tea.Cmd
				m.versionList, cmd = m.versionList.Update(msg)
				return m, cmd
			}
			// Scroll viewport down (when build is running and viewport is active)
			if m.buildStarted && m.viewportReady {
				m.viewport.ScrollDown(1)
			}
			return m, nil

		case "pgup":
			if m.activePhase == PhaseSelectVersion && !m.buildStarted {
				var cmd tea.Cmd
				m.versionList, cmd = m.versionList.Update(msg)
				return m, cmd
			}
			// Scroll viewport page up
			if m.buildStarted && m.viewportReady {
				m.viewport.PageUp()
			}
			return m, nil

		case "pgdown":
			if m.activePhase == PhaseSelectVersion && !m.buildStarted {
				var cmd tea.Cmd
				m.versionList, cmd = m.versionList.Update(msg)
				return m, cmd
			}
			// Scroll viewport page down
			if m.buildStarted && m.viewportReady {
				m.viewport.PageDown()
			}
			return m, nil

		case "home":
			// Scroll to top
			if m.buildStarted && m.viewportReady {
				m.viewport.GotoTop()
			}
			return m, nil

		case "end":
			// Scroll to bottom
			if m.buildStarted && m.viewportReady {
				m.viewport.GotoBottom()
			}
			return m, nil
		}

	case FetchVersionsMsg:
		if msg.Error != nil {
			m.err = msg.Error
			return m, tea.Quit
		}

		// Convert versions to list items
		items := make([]list.Item, len(msg.Versions))
		for i, v := range msg.Versions {
			items[i] = versionItem{
				version:     v,
				isLatest:    i == 0,
				description: fmt.Sprintf("Kernel version %s", v),
			}
		}
		m.versions = items
		return m, m.versionList.SetItems(items)

	case BuildStreamMsg:
		// Build started, store channels and begin listening
		log.Debugf("BuildStreamMsg received, starting output listener")
		m.buildOutputChan = msg.outputChan
		m.buildDoneChan = msg.doneChan
		m.progressChan = msg.progressChan
		m.phaseChan = msg.phaseChan
		m.buildStatsChan = msg.statsChan
		return m, waitForBuildOutput(m.buildOutputChan, m.buildDoneChan, m.progressChan, m.phaseChan, m.buildStatsChan)

	case BuildOutputMsg:
		// Append output line to build output (global for viewport)
		m.buildOutput = append(m.buildOutput, msg.Output)
		// Also append to current BUILD phase's output (not viewed phase)
		m.phaseOutput[m.currentBuildPhase] = append(m.phaseOutput[m.currentBuildPhase], msg.Output)
		log.Debugf("BuildOutputMsg: output=%s (buildPhase=%d, viewedPhase=%d)", msg.Output, m.currentBuildPhase, m.activePhase)

		// Update viewport content with all build output
		if m.viewportReady {
			m.viewport.SetContent(strings.Join(m.buildOutput, "\n"))
			// Auto-scroll to bottom (follow mode)
			m.viewport.GotoBottom()
		}

		// Continue listening for more output if channels are active
		if m.buildOutputChan != nil {
			return m, waitForBuildOutput(m.buildOutputChan, m.buildDoneChan, m.progressChan, m.phaseChan, m.buildStatsChan)
		}
		return m, nil

	case BuildPhaseTransitionMsg:
		// Phase transition - update UI to reflect new phase
		log.Debugf("BuildPhaseTransitionMsg: phase=%d, currentBuildPhase=%d, manualTabMode=%v", msg.Phase, m.currentBuildPhase, m.manualTabMode)

		// Mark previous BUILD phase complete (not the viewed phase)
		if m.currentBuildPhase >= PhaseDownload && m.currentBuildPhase <= PhasePackage {
			m.tabs[m.currentBuildPhase].State = TabComplete
			log.Debugf("Marked build phase %d as complete", m.currentBuildPhase)
		}

		// Map kernel.BuildPhase to UI phase (kernel phases are 0-indexed, UI has SelectVersion at 0)
		uiPhase := BuildKernelPhase(msg.Phase + 1) // +1 to skip PhaseSelectVersion

		// Update current build phase
		m.currentBuildPhase = uiPhase

		// Auto-advance user if they're viewing the phase that just completed
		wasViewingCompletedPhase := false
		if m.currentBuildPhase >= PhaseDownload && m.currentBuildPhase <= PhasePackage {
			// Check if user is viewing the phase that just completed (previous build phase)
			prevPhase := uiPhase - 1
			if prevPhase >= PhaseDownload && m.activePhase == prevPhase {
				wasViewingCompletedPhase = true
			}
		}

		if wasViewingCompletedPhase {
			// User was watching the phase that just finished - advance them
			m.activePhase = uiPhase
			// Clear manual mode since we're keeping them in sync
			m.manualTabMode = false
			log.Debugf("User was viewing completed phase, advancing to phase %d and clearing manual mode", uiPhase)
		} else if !m.manualTabMode {
			// Normal auto-follow mode - switch to new phase
			m.activePhase = uiPhase
			log.Debugf("Auto-switching viewed tab to phase %d", uiPhase)
		} else {
			log.Debugf("Manual tab mode active, staying on phase %d (build is now on %d)", m.activePhase, uiPhase)
		}

		// Set new build phase as active
		m.tabs[uiPhase].State = TabActive
		log.Debugf("Set build phase %d to TabActive", uiPhase)

		// Continue listening
		return m, waitForBuildOutput(m.buildOutputChan, m.buildDoneChan, m.progressChan, m.phaseChan, m.buildStatsChan)

	case DownloadProgressMsg:
		// Update progress bar with download progress
		m.progressPercent = msg.Percent
		log.Debugf("DownloadProgressMsg: percent=%.2f", msg.Percent)

		cmd := m.progressBar.SetPercent(m.progressPercent)

		// Continue listening for more updates
		return m, tea.Batch(cmd, waitForBuildOutput(m.buildOutputChan, m.buildDoneChan, m.progressChan, m.phaseChan, m.buildStatsChan))

	case BuildCompleteMsg:
		log.Debugf("BuildCompleteMsg: success=%v, error=%v", msg.Success, msg.Error)

		// Mark ALL build phases as complete (except Select and Complete)
		m.tabs[PhaseSelectVersion].State = TabComplete
		m.tabs[PhaseDownload].State = TabComplete
		m.tabs[PhaseVerify].State = TabComplete
		m.tabs[PhaseExtract].State = TabComplete
		m.tabs[PhaseConfigure].State = TabComplete
		m.tabs[PhaseCompile].State = TabComplete
		m.tabs[PhasePackage].State = TabComplete

		// Convert kernel.BuildStats to UI BuildStats
		m.buildStats = BuildStats{
			TotalDuration:     msg.Stats.TotalDuration,
			DownloadDuration:  msg.Stats.DownloadDuration,
			ExtractDuration:   msg.Stats.ExtractDuration,
			ConfigureDuration: msg.Stats.ConfigureDuration,
			CompileDuration:   msg.Stats.CompileDuration,
			PackageDuration:   msg.Stats.PackageDuration,
			UncompressedSize:  msg.Stats.UncompressedSize,
			CompressedSize:    msg.Stats.CompressedSize,
			UncompressedHash:  msg.Stats.UncompressedHash,
			CompressedHash:    msg.Stats.CompressedHash,
			KernelVersion:     msg.Stats.KernelVersion,
			OutputPath:        msg.Stats.OutputPath,
			CompressedPath:    msg.Stats.CompressedPath,
			BuildTimestamp:    msg.Stats.BuildTimestamp,
		}

		m.activePhase = PhaseComplete
		m.currentBuildPhase = PhaseComplete
		m.tabs[PhaseComplete].State = TabComplete

		if msg.Error != nil {
			m.err = msg.Error
			m.tabs[PhaseComplete].State = TabError
		}

		return m, nil

	case spinner.TickMsg:
		// Update all spinners
		cmds := make([]tea.Cmd, len(m.tabs))
		for i := range m.tabs {
			var cmd tea.Cmd
			m.tabs[i].Spinner, cmd = m.tabs[i].Spinner.Update(msg)
			cmds[i] = cmd
		}
		return m, tea.Batch(cmds...)

	case progress.FrameMsg:
		// Animate progress bar
		progressModel, cmd := m.progressBar.Update(msg)
		m.progressBar = progressModel.(progress.Model)
		return m, cmd

	case InstallKernelMsg:
		// Kernel installation complete
		m.installingKernel = false
		if msg.Success {
			m.kernelInstalled = true
			m.installedVersion = msg.InstalledVersion
			m.kernelSetAsDefault = msg.SetAsDefault
			log.Debugf("Kernel installed successfully: %s (setAsDefault=%v)", msg.InstalledVersion, msg.SetAsDefault)
		} else {
			m.installError = msg.Error
			log.Debugf("Kernel installation failed: %v", msg.Error)
		}
		return m, nil

	case CachedBuildLoadedMsg:
		// Cached build stats loaded
		m.loadingCachedBuild = false

		if msg.Error != nil {
			log.Debugf("Failed to load cached build: %v", msg.Error)
			// Continue with normal flow - fetch versions
			cmds := make([]tea.Cmd, len(m.tabs)+1)
			for i := range m.tabs {
				cmds[i] = m.tabs[i].Spinner.Tick
			}
			cmds[len(m.tabs)] = fetchKernelVersions
			return m, tea.Batch(cmds...)
		}

		log.Debugf("Cached build loaded: version=%s", msg.Stats.KernelVersion)

		// Mark all phases as complete
		m.tabs[PhaseSelectVersion].State = TabComplete
		m.tabs[PhaseDownload].State = TabComplete
		m.tabs[PhaseVerify].State = TabComplete
		m.tabs[PhaseExtract].State = TabComplete
		m.tabs[PhaseConfigure].State = TabComplete
		m.tabs[PhaseCompile].State = TabComplete
		m.tabs[PhasePackage].State = TabComplete

		// Convert kernel.BuildStats to UI BuildStats
		m.buildStats = BuildStats{
			TotalDuration:     msg.Stats.TotalDuration,
			DownloadDuration:  msg.Stats.DownloadDuration,
			ExtractDuration:   msg.Stats.ExtractDuration,
			ConfigureDuration: msg.Stats.ConfigureDuration,
			CompileDuration:   msg.Stats.CompileDuration,
			PackageDuration:   msg.Stats.PackageDuration,
			UncompressedSize:  msg.Stats.UncompressedSize,
			CompressedSize:    msg.Stats.CompressedSize,
			UncompressedHash:  msg.Stats.UncompressedHash,
			CompressedHash:    msg.Stats.CompressedHash,
			KernelVersion:     msg.Stats.KernelVersion,
			OutputPath:        msg.Stats.OutputPath,
			CompressedPath:    msg.Stats.CompressedPath,
			BuildTimestamp:    msg.Stats.BuildTimestamp,
		}

		// Set to completion screen
		m.activePhase = PhaseComplete
		m.currentBuildPhase = PhaseComplete
		m.tabs[PhaseComplete].State = TabComplete
		m.buildStarted = true
		m.selectedVersion = msg.Stats.KernelVersion
		m.isCachedBuild = true // Mark as cached build (no actual build ran)

		// Check if this build is already installed
		if isInstalled, installedVer, err := kernel.CheckKernelInstalled(msg.Stats); err == nil && isInstalled {
			m.kernelInstalled = true
			m.installedVersion = installedVer
			log.Debugf("Cached build is already installed: %s", installedVer)
		}

		return m, nil

	case NewBuildStartedMsg:
		// Cache cleared, restart wizard
		if msg.Error != nil {
			log.Debugf("Failed to clear cache: %v", msg.Error)
			m.err = msg.Error
			return m, nil
		}

		log.Debugf("Starting new build, resetting wizard")

		// Reset wizard state to initial
		m.activePhase = PhaseSelectVersion
		m.currentBuildPhase = PhaseSelectVersion
		m.buildStarted = false
		m.buildOutput = []string{}
		m.phaseOutput = make(map[BuildKernelPhase][]string)
		m.manualTabMode = false
		m.selectedVersion = ""
		m.kernelInstalled = false
		m.installedVersion = ""
		m.installingKernel = false
		m.installError = nil
		m.progressPercent = 0
		m.buildStats = BuildStats{}

		// Reset all tab states
		m.tabs[PhaseSelectVersion].State = TabActive
		for i := PhaseDownload; i <= PhaseComplete; i++ {
			m.tabs[i].State = TabPending
		}

		// Fetch versions again
		return m, fetchKernelVersions
	}

	// Update list if on version select phase
	if m.activePhase == PhaseSelectVersion && !m.buildStarted {
		var cmd tea.Cmd
		m.versionList, cmd = m.versionList.Update(msg)
		return m, cmd
	}

	// Update viewport if build is running
	if m.buildStarted && m.viewportReady {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return m, cmd
	}

	return m, nil
}

// BuildStreamMsg contains channels for streaming build output
type BuildStreamMsg struct {
	outputChan   chan string
	doneChan     chan error
	progressChan chan float64
	phaseChan    chan kernel.BuildPhase
	statsChan    chan kernel.BuildStats
}

// startBuild initiates the build process in a goroutine
func (m *BuildKernelWizard) startBuild() tea.Cmd {
	return func() tea.Msg {
		log.Debugf("Starting actual kernel build for version %s", m.selectedVersion)

		// Create cancellable context for build
		ctx, cancel := context.WithCancel(context.Background())
		m.buildCtx = ctx
		m.buildCancel = cancel

		outputChan := make(chan string, 100)
		doneChan := make(chan error, 1)
		progressChan := make(chan float64, 10)
		phaseChan := make(chan kernel.BuildPhase, 10)

		// Channel for build stats
		statsChan := make(chan kernel.BuildStats, 1)

		// Start build in goroutine
		go func() {
			defer close(outputChan)
			defer close(doneChan)
			defer close(progressChan)
			defer close(phaseChan)
			defer close(statsChan)

			// Create pipe for capturing build output
			pr, pw := io.Pipe()

			// Run kernel build in another goroutine
			go func() {
				defer pw.Close()

				// Progress callback for downloads
				progressCallback := func(percent float64) {
					select {
					case progressChan <- percent:
					default:
						// Don't block if channel is full
					}
				}

				// Phase callback for phase transitions
				phaseCallback := func(phase kernel.BuildPhase) {
					select {
					case phaseChan <- phase:
					default:
						// Don't block if channel is full
					}
				}

				// Stats callback for final statistics
				statsCallback := func(stats kernel.BuildStats) {
					select {
					case statsChan <- stats:
					default:
						// Don't block if channel is full
					}
				}

				// Build options with custom writer for streaming output
				opts := kernel.BuildOptions{
					Version:           m.selectedVersion,
					Arch:              m.arch,
					VerificationLevel: m.verificationLevel,
					ConfigFile:        m.configFile,
					Interactive:       false,            // Non-interactive (we're in TUI)
					Writer:            pw,               // Stream output to pipe for TUI
					ProgressCallback:  progressCallback, // Download progress callback
					PhaseCallback:     phaseCallback,    // Phase transition callback
					StatsCallback:     statsCallback,    // Build stats callback
					Context:           ctx,              // Context for cancellation
				}

				log.Debugf("Build options: %+v", opts)

				// Run actual kernel build - output will stream through pw
				if err := kernel.Build(opts); err != nil {
					// Write error to pipe so it gets captured
					pw.Write([]byte(fmt.Sprintf("[ERROR] Build failed: %s\n", err.Error())))
				}
			}()

			// Read output line by line and send to channel
			scanner := bufio.NewScanner(pr)
			for scanner.Scan() {
				outputChan <- scanner.Text()
			}

			if err := scanner.Err(); err != nil {
				doneChan <- err
			} else {
				doneChan <- nil
			}
		}()

		return BuildStreamMsg{
			outputChan:   outputChan,
			doneChan:     doneChan,
			progressChan: progressChan,
			phaseChan:    phaseChan,
			statsChan:    statsChan,
		}
	}
}

// waitForBuildOutput waits for the next build output, progress, phase transition, or completion
func waitForBuildOutput(outputChan chan string, doneChan chan error, progressChan chan float64, phaseChan chan kernel.BuildPhase, statsChan chan kernel.BuildStats) tea.Cmd {
	return func() tea.Msg {
		select {
		case line, ok := <-outputChan:
			if !ok {
				// Channel closed, check if done
				var buildErr error
				var buildStats kernel.BuildStats

				select {
				case buildErr = <-doneChan:
				default:
				}

				// Try to get stats (non-blocking)
				select {
				case buildStats = <-statsChan:
				default:
				}

				return BuildCompleteMsg{
					Success: buildErr == nil,
					Error:   buildErr,
					Stats:   buildStats,
				}
			}

			return BuildOutputMsg{
				Output: line,
			}

		case percent, ok := <-progressChan:
			if ok {
				return DownloadProgressMsg{Percent: percent}
			}
			// Progress channel closed, continue listening to other channels
			return waitForBuildOutput(outputChan, doneChan, progressChan, phaseChan, statsChan)()

		case phase, ok := <-phaseChan:
			if ok {
				return BuildPhaseTransitionMsg{Phase: phase}
			}
			// Phase channel closed, continue listening to other channels
			return waitForBuildOutput(outputChan, doneChan, progressChan, phaseChan, statsChan)()

		case err := <-doneChan:
			// Try to get stats (non-blocking)
			var buildStats kernel.BuildStats
			select {
			case buildStats = <-statsChan:
			default:
			}

			return BuildCompleteMsg{
				Success: err == nil,
				Error:   err,
				Stats:   buildStats,
			}
		}
	}
}

// View renders the wizard
func (m *BuildKernelWizard) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 || m.height == 0 {
		return "Initializing..."
	}

	// Show loading screen when loading cached build
	if m.loadingCachedBuild {
		theme := config.CurrentTheme
		header := theme.RenderHeader(m.width, "BUILD WIZARD", "KERNEL")
		loadingMsg := lipgloss.NewStyle().
			Foreground(theme.GetPrimaryColor()).
			Bold(true).
			Render("Loading cached build...")

		content := lipgloss.Place(
			m.width,
			m.height-3, // Account for header
			lipgloss.Center,
			lipgloss.Center,
			loadingMsg,
		)

		return lipgloss.JoinVertical(lipgloss.Left, header, "", content)
	}

	theme := config.CurrentTheme

	// Header using theme helper
	header := theme.RenderHeader(m.width, "BUILD WIZARD", "KERNEL")

	// Get content for active phase
	content := m.getPhaseContent(m.activePhase)

	// Calculate content pane dimensions
	const (
		headerLines = 1
		tabLines    = 3
		helpLines   = 1
		blankLines  = 3
		borderLines = 2
	)
	contentHeight := m.height - headerLines - tabLines - helpLines - blankLines - borderLines
	// CRITICAL: Width() includes padding inside, only borders add to rendered width
	// Content width = terminal width - border width (2 chars)
	contentWidth := m.width - 2

	// Render content pane FIRST to get its actual width
	contentPane := RenderTabContent(content, contentWidth, contentHeight)
	actualContentWidth := lipgloss.Width(contentPane)

	// Render tabs to match content pane's actual rendered width
	tabsRow := RenderTabs(m.tabs, TabsConfig{
		ActiveIndex: int(m.activePhase),
		Width:       actualContentWidth,
	})

	// Help footer using theme helper
	var helpContent string
	if m.activePhase == PhaseSelectVersion && !m.buildStarted {
		helpContent = "[↑↓] Navigate  •  [/] Filter  •  [ENTER] Select  •  [ESC] Quit"
	} else if m.activePhase == PhaseComplete {
		if m.confirmingNewBuild {
			helpContent = "⚠ Clear build cache and start new? [Y] Yes  •  [N] No"
		} else if m.kernelInstalled {
			helpContent = "[N] Start New Build  •  [Q/ESC] Exit"
		} else if m.installingKernel {
			helpContent = "Installing kernel..."
		} else {
			helpContent = "[I] Install Kernel  •  [N] Start New Build  •  [Q/ESC] Exit"
		}
	} else {
		helpContent = "[↑↓] Scroll  •  [PgUp/PgDn] Page  •  [Home/End] Top/Bottom  •  [TAB] Switch Tabs  •  [ESC] Quit"
	}
	help := theme.RenderFooter(m.width, helpContent)

	// Combine all elements using JoinVertical (not string concatenation!)
	baseView := lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		"",
		tabsRow,
		contentPane,
		"",
		help,
	)

	// If showing install confirmation modal, render huh form centered
	if m.confirmingInstall && m.confirmForm != nil {
		formView := m.confirmForm.View()
		// Use MaxWidth to prevent expansion while allowing natural sizing
		constrainedForm := lipgloss.NewStyle().
			MaxWidth(60).
			Render(formView)
		formWidth := lipgloss.Width(constrainedForm)
		formHeight := lipgloss.Height(constrainedForm)
		log.Debugf("BuildKernelWizard confirm modal: terminal=%dx%d, form=%dx%d", m.width, m.height, formWidth, formHeight)
		return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, constrainedForm, lipgloss.WithWhitespaceChars(" "), lipgloss.WithWhitespaceForeground(lipgloss.Color("0")))
	}

	return baseView
}

// getPhaseContent returns the content for a given phase
func (m *BuildKernelWizard) getPhaseContent(phase BuildKernelPhase) string {
	theme := config.CurrentTheme

	switch phase {
	case PhaseSelectVersion:
		if m.buildStarted {
			return theme.SuccessMessage("✓ Version selected: " + m.selectedVersion)
		}

		// Show version list
		return m.versionList.View()

	case PhaseDownload, PhaseVerify, PhaseExtract, PhaseConfigure, PhaseCompile, PhasePackage:
		// Check tab state first - pending tabs should never show output
		if m.tabs[phase].State == TabPending {
			return theme.WaitingIndicator() + " Waiting to start..."
		}

		// Check for errors
		if m.tabs[phase].State == TabError {
			if m.err != nil {
				return theme.ErrorIndicator() + " Error: " + m.err.Error()
			}
			return theme.ErrorIndicator() + " Failed"
		}

		// Special case: Show progress bar for download phase
		if phase == PhaseDownload && m.tabs[phase].State == TabActive && m.progressPercent > 0 {
			// Create progress bar with label
			progressLabel := lipgloss.NewStyle().
				Foreground(theme.GetPrimaryColor()).
				Render("Downloading kernel source...")
			progressView := m.progressBar.View()

			var content string
			if len(m.buildOutput) > 0 {
				// Show recent output lines + labeled progress bar
				recentLines := m.buildOutput
				if len(m.buildOutput) > 5 {
					recentLines = m.buildOutput[len(m.buildOutput)-5:]
				}
				content = strings.Join(recentLines, "\n") + "\n\n" + progressLabel + "\n" + progressView
			} else {
				content = progressLabel + "\n" + progressView
			}
			return content
		}

		// Special case: Show progress bar for extraction phase
		if phase == PhaseExtract && m.tabs[phase].State == TabActive && m.progressPercent > 0 {
			// Create progress bar with label
			progressLabel := lipgloss.NewStyle().
				Foreground(theme.GetPrimaryColor()).
				Render("Extracting kernel source...")
			progressView := m.progressBar.View()

			var content string
			if len(m.buildOutput) > 0 {
				// Show recent output lines + labeled progress bar
				recentLines := m.buildOutput
				if len(m.buildOutput) > 5 {
					recentLines = m.buildOutput[len(m.buildOutput)-5:]
				}
				content = strings.Join(recentLines, "\n") + "\n\n" + progressLabel + "\n" + progressView
			} else {
				content = progressLabel + "\n" + progressView
			}
			return content
		}

		// Special case: Use scrollable viewport for Configure and Compile phases
		if (phase == PhaseConfigure || phase == PhaseCompile) && len(m.buildOutput) > 0 && m.viewportReady {
			return m.viewport.View()
		}

		// If no output yet, show status message
		if len(m.buildOutput) == 0 {
			switch m.tabs[phase].State {
			case TabActive:
				return theme.RunningIndicator() + " Running..."
			case TabComplete:
				return theme.CompleteIndicator() + " Complete"
			}
		}

		// Show phase-specific output for completed phases, or recent global output for active phase
		var outputLines []string
		if m.tabs[phase].State == TabComplete && len(m.phaseOutput[phase]) > 0 {
			// Completed phase - show that phase's output
			outputLines = m.phaseOutput[phase]
		} else {
			// Active phase - show recent global output
			outputLines = m.buildOutput
		}

		// Show recent lines (last 20)
		recentLines := outputLines
		if len(outputLines) > 20 {
			recentLines = outputLines[len(outputLines)-20:]
		}
		return strings.Join(recentLines, "\n")

	case PhaseComplete:
		// Check tab state first - pending tabs should never show output
		if m.tabs[phase].State == TabPending {
			return theme.WaitingIndicator() + " Waiting to start..."
		}

		// Check for errors
		if m.tabs[phase].State == TabError {
			if m.err != nil {
				return theme.ErrorIndicator() + " Error: " + m.err.Error()
			}
			return theme.ErrorIndicator() + " Failed"
		}

		return m.renderBuildStats()
	}

	return ""
}

// renderBuildStats renders the build completion statistics
func (m *BuildKernelWizard) renderBuildStats() string {
	theme := config.CurrentTheme
	stats := m.buildStats

	// Format file sizes
	formatSize := func(bytes int64) string {
		const unit = 1024
		if bytes < unit {
			return fmt.Sprintf("%d B", bytes)
		}
		div, exp := int64(unit), 0
		for n := bytes / unit; n >= unit; n /= unit {
			div *= unit
			exp++
		}
		return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
	}

	// Format duration
	formatDuration := func(d time.Duration) string {
		if d < time.Second {
			return fmt.Sprintf("%dms", d.Milliseconds())
		}
		if d < time.Minute {
			return fmt.Sprintf("%.1fs", d.Seconds())
		}
		return fmt.Sprintf("%dm %ds", int(d.Minutes()), int(d.Seconds())%60)
	}

	// Title
	titleStyle := lipgloss.NewStyle().
		Foreground(theme.GetPrimaryColor()).
		Bold(true).
		MarginBottom(1)

	title := titleStyle.Render("🎉 Build Complete!")

	// Stats sections
	labelStyle := lipgloss.NewStyle().
		Foreground(theme.GetSecondaryColor()).
		Bold(true)

	valueStyle := lipgloss.NewStyle().
		Foreground(theme.GetPrimaryColor())

	// Build timing section
	timingTitle := labelStyle.Render("Build Timing:")
	timing := []string{
		fmt.Sprintf("  Total:     %s", valueStyle.Render(formatDuration(stats.TotalDuration))),
		fmt.Sprintf("  Download:  %s", formatDuration(stats.DownloadDuration)),
		fmt.Sprintf("  Extract:   %s", formatDuration(stats.ExtractDuration)),
		fmt.Sprintf("  Configure: %s", formatDuration(stats.ConfigureDuration)),
		fmt.Sprintf("  Compile:   %s", formatDuration(stats.CompileDuration)),
		fmt.Sprintf("  Package:   %s", formatDuration(stats.PackageDuration)),
	}

	// File info section
	fileTitle := labelStyle.Render("\nKernel Artifacts:")
	files := []string{
		fmt.Sprintf("  Version:     %s", valueStyle.Render(stats.KernelVersion)),
		fmt.Sprintf("  Uncompressed: %s (%s)", formatSize(stats.UncompressedSize), valueStyle.Render(stats.OutputPath)),
		fmt.Sprintf("    SHA256:    %s", stats.UncompressedHash),
		fmt.Sprintf("  Compressed:   %s (%s)", formatSize(stats.CompressedSize), valueStyle.Render(stats.CompressedPath)),
		fmt.Sprintf("    SHA256:    %s", stats.CompressedHash),
	}

	// Compression ratio
	compressionRatio := float64(stats.CompressedSize) / float64(stats.UncompressedSize) * 100
	compressionInfo := fmt.Sprintf("  Compression:  %.1f%% of original size", compressionRatio)

	// Installation status
	installStatusStyle := lipgloss.NewStyle().
		Foreground(theme.GetPrimaryColor()).
		MarginTop(2)

	var installStatus string
	if m.installingKernel {
		installStatus = installStatusStyle.Render(theme.WaitingIndicator() + " Installing kernel...")
	} else if m.kernelInstalled {
		if m.kernelSetAsDefault {
			installStatus = theme.SuccessMessage(fmt.Sprintf("Kernel installed: %s (set as default)", m.installedVersion))
		} else {
			installStatus = theme.SuccessMessage(fmt.Sprintf("Kernel installed: %s", m.installedVersion))
		}
	} else if m.installError != nil {
		installStatus = theme.ErrorMessage(fmt.Sprintf("Installation failed: %s", m.installError.Error()))
	} else {
		installStatus = lipgloss.NewStyle().
			Foreground(theme.GetMutedColor()).
			Render(theme.WarningIndicator() + " Kernel not installed (press I to install)")
	}

	// Footer
	footerStyle := lipgloss.NewStyle().
		Foreground(theme.GetMutedColor()).
		MarginTop(1)

	warningStyle := lipgloss.NewStyle().
		Foreground(theme.GetWarningColor()).
		MarginTop(1)

	var footer string
	if m.confirmingInstall {
		footer = warningStyle.Render(theme.WarningIndicator() + " Set as default kernel? [Y] Yes / [N] No (install only)")
	} else if m.confirmingNewBuild {
		footer = warningStyle.Render(theme.WarningIndicator() + " This will clear the build cache. Continue? [Y/N]")
	} else if m.kernelInstalled {
		footer = footerStyle.Render("Press N to start a new build, or ESC/Q to exit")
	} else if m.installingKernel {
		footer = ""
	} else {
		footer = footerStyle.Render("Press I to install kernel, N to start new build, or ESC/Q to exit")
	}

	// Combine all sections
	content := []string{
		title,
		timingTitle,
	}
	content = append(content, timing...)
	content = append(content, fileTitle)
	content = append(content, files...)
	content = append(content, compressionInfo)
	content = append(content, installStatus)
	if footer != "" {
		content = append(content, footer)
	}

	return strings.Join(content, "\n")
}

// ErrUserCancelled is returned when the user cancels the wizard
var ErrUserCancelled = fmt.Errorf("cancelled by user")

// loadCachedBuild loads a cached build from stats file
func (m *BuildKernelWizard) loadCachedBuild(statsFile string) tea.Cmd {
	return func() tea.Msg {
		stats, err := kernel.ReadBuildStats(statsFile)
		if err != nil {
			return CachedBuildLoadedMsg{Error: err}
		}
		return CachedBuildLoadedMsg{Stats: stats}
	}
}

// startNewBuild clears the build cache and restarts the wizard
func (m *BuildKernelWizard) startNewBuild() tea.Cmd {
	return func() tea.Msg {
		// Clear the build cache
		buildDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "build")
		artifactsDir := filepath.Join(config.GlobalPaths.KernelBuildDir, "artifacts")

		// Remove build and artifacts directories
		if err := os.RemoveAll(buildDir); err != nil {
			return NewBuildStartedMsg{Error: fmt.Errorf("failed to clear build directory: %w", err)}
		}
		if err := os.RemoveAll(artifactsDir); err != nil {
			return NewBuildStartedMsg{Error: fmt.Errorf("failed to clear artifacts directory: %w", err)}
		}

		log.Debugf("Build cache cleared")
		return NewBuildStartedMsg{}
	}
}

// installKernel installs the built kernel to the kernels directory
func (m *BuildKernelWizard) installKernel(setAsDefault bool) tea.Cmd {
	return func() tea.Msg {
		// Convert UI BuildStats to kernel.BuildStats
		kernelStats := kernel.BuildStats{
			TotalDuration:     m.buildStats.TotalDuration,
			DownloadDuration:  m.buildStats.DownloadDuration,
			ExtractDuration:   m.buildStats.ExtractDuration,
			ConfigureDuration: m.buildStats.ConfigureDuration,
			CompileDuration:   m.buildStats.CompileDuration,
			PackageDuration:   m.buildStats.PackageDuration,
			UncompressedSize:  m.buildStats.UncompressedSize,
			CompressedSize:    m.buildStats.CompressedSize,
			UncompressedHash:  m.buildStats.UncompressedHash,
			CompressedHash:    m.buildStats.CompressedHash,
			KernelVersion:     m.buildStats.KernelVersion,
			OutputPath:        m.buildStats.OutputPath,
			CompressedPath:    m.buildStats.CompressedPath,
			BuildTimestamp:    m.buildStats.BuildTimestamp,
		}

		// Install kernel with timestamp
		installedVersion, err := kernel.InstallBuiltKernel(kernelStats, setAsDefault)
		if err != nil {
			return InstallKernelMsg{
				Success: false,
				Error:   err,
			}
		}

		// Archive to repo-local directory if configured
		if archiveDir := config.GetKernelsArchiveLocation(); archiveDir != "" {
			if err := kernel.ArchiveInstalledKernel(kernelStats, archiveDir); err != nil {
				return InstallKernelMsg{
					Success: false,
					Error:   fmt.Errorf("install succeeded but archiving failed: %w", err),
				}
			}
		}

		return InstallKernelMsg{
			Success:          true,
			InstalledVersion: installedVersion,
			SetAsDefault:     setAsDefault,
		}
	}
}

// RunBuildKernelWizard runs the kernel build wizard
// This handles the ENTIRE build process: selection + build + progress
func RunBuildKernelWizard(arch, verificationLevel, configFile string, forceRebuild bool) error {
	m := NewBuildKernelWizard(arch, verificationLevel, configFile, forceRebuild)
	p := tea.NewProgram(m, tea.WithAltScreen())

	finalModel, err := p.Run()
	if err != nil {
		return err
	}

	if wizard, ok := finalModel.(*BuildKernelWizard); ok {
		// Check if user cancelled before starting build
		if wizard.quitting && !wizard.buildStarted {
			return ErrUserCancelled
		}

		if wizard.err != nil {
			return wizard.err
		}

		// Build completed successfully
		return nil
	}

	return fmt.Errorf("unexpected model type")
}
