---
name: boba-master
description: Expert guidance for building Bubble Tea TUIs with Charm libraries - covers MVU architecture, Lipgloss layout patterns, state management, and best practices
---

# Bubble Tea TUI Development Master Guide

Use this skill when building or debugging Bubble Tea (bubbletea) terminal user interfaces.

## When to Use This Skill

- Building new TUI applications with Bubble Tea
- Debugging layout issues (width/height calculations, alignment)
- Implementing state machines for complex workflows
- Integrating async operations (downloads, builds, long-running tasks)
- Creating reusable TUI components
- Fixing terminal resizing issues

## Core Architecture: The Elm Pattern (MVU)

Bubble Tea follows **Model-View-Update** architecture:

```go
type Model struct {
    // ALL state lives here
    width, height int
    currentState  State
    data          []Item
}

func (m Model) Init() tea.Cmd {
    // Return initial commands (async operations)
    return fetchDataCmd
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    // Handle messages, update state, return new commands
    // This is the ONLY place state changes
    switch msg := msg.(type) {
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
    case tea.KeyMsg:
        // Handle input
    case CustomMsg:
        // Handle custom events
    }
    return m, nil
}

func (m Model) View() string {
    // Render UI based on current state
    // MUST be pure - no side effects, no state mutation
    return lipgloss.JoinVertical(lipgloss.Left, header, content, footer)
}
```

**Key Principles:**
- **Model** contains ALL state
- **Update** is the ONLY place state changes
- **View** is pure (no side effects)
- Use **value receivers** (documented pattern, though pointers work)
- Commands run in goroutines and return messages

**Message Flow:**

1. User input or events generate messages (tea.Msg)
2. `Update()` processes message, updates state, returns new model + commands
3. Commands (tea.Cmd) generate more messages asynchronously
4. `View()` renders current state (called after each Update)

**Key insight:** View() is called frequently. Keep it pure (no side effects, no state mutation). All I/O and state changes happen in Update() or via Commands.

## Critical Lipgloss Layout Patterns

### Understanding lipgloss.Width() Function

**CRITICAL:** `Width()` returns the MAX width across all lines, NOT the sum:

```go
func Width(str string) int {
    // Returns MAX width across all lines
    for _, line := range strings.Split(str, "\n") {
        w := ansi.StringWidth(line)
        if w > width {
            width = w
        }
    }
    return width
}
```

This is why you MUST use `JoinVertical`/`JoinHorizontal` and never string concatenation (`+`):

```go
// ❌ WRONG - Width() will sum widths, not take max
baseView := panes + help  // No newline, becomes one long line

// ✅ CORRECT - Proper vertical stacking with newlines
baseView := lipgloss.JoinVertical(lipgloss.Left, panes, "", help)
```

### The Width Calculation Formula

**CRITICAL: Understanding Lipgloss Width Behavior (v1.1.1+)**

```go
// Style.Width(w) sets content width INCLUDING padding (padding is inside)
// Border is rendered OUTSIDE of Style.Width() (adds to final render)
// Actual rendered width = Style.Width() + border width

// Example: Split-pane layout for terminal width 130
gap := 2
paneRenderedWidth := (130 - gap) / 2  // = 64 chars per pane
borderWidth := 2                       // All border types = 2 chars (1 per side)
paneContentWidth := paneRenderedWidth - borderWidth  // = 62

// Set Style.Width(62), rendered width will be 64 ✓
style := lipgloss.NewStyle().
    Width(62).        // Content area (padding inside this)
    Padding(0, 1).    // Inside the 62
    Border(lipgloss.RoundedBorder())  // Adds 2 to rendered width
```

**How padding and borders work:**
- `Style.Width(w)` sets width to `w` (includes padding INSIDE this width)
- **Padding:** Rendered INSIDE the Style.Width() value (doesn't add to rendered width)
- **Border:** Rendered OUTSIDE the Style.Width() value (adds to rendered width)
- Actual rendered width = `Style.Width()` + border width

**Always measure actual rendered dimensions:**
```go
renderedWidth := lipgloss.Width(styledComponent)
log.Debugf("Target=%d Actual=%d Diff=%d", targetWidth, renderedWidth, targetWidth-renderedWidth)
```

**All border types are 2 chars wide:**
- `lipgloss.NormalBorder()` = 2 chars (light)
- `lipgloss.RoundedBorder()` = 2 chars (rounded corners)
- `lipgloss.ThickBorder()` = 2 chars (heavy)
- `lipgloss.DoubleBorder()` = 2 chars (double lines)
- `lipgloss.BlockBorder()` = 2 chars (solid blocks)

You can swap them for visual effect without breaking layouts! Use ThickBorder for active panes, NormalBorder for inactive.

### Proper Joining Functions

```go
// Horizontal joining (side-by-side)
// First arg is VERTICAL alignment (Top/Center/Bottom)
panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "  ", rightPane)

// Vertical stacking
// First arg is HORIZONTAL alignment (Left/Center/Right)
layout := lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
```

**Gotcha:** `JoinHorizontal` uses vertical alignment (Top/Bottom), `JoinVertical` uses horizontal alignment (Left/Right). This is correct but counterintuitive.

### Split-Pane Layout Pattern (Full-Width)

**Problem:** Create two equal-width panes that fill terminal width with no gaps.

**Solution Pattern:**
```go
// In Update() - WindowSizeMsg handler
case tea.WindowSizeMsg:
    m.width = msg.Width
    m.height = msg.Height

    // Step 1: Calculate target rendered width per pane
    gap := 2  // Space between panes
    paneRenderedWidth := (m.width - gap) / 2

    // Step 2: Account for border overhead (NOT padding - it's inside)
    borderWidth := 2  // RoundedBorder = 1 char per side
    paneContentWidth := paneRenderedWidth - borderWidth

    // Step 3: Set component sizes
    m.leftList.SetSize(paneContentWidth, m.height - 4)
    m.rightList.SetSize(paneContentWidth, m.height - 4)

// In View()
func (m Model) View() string {
    // Use same calculation as Update()
    gap := 2
    paneRenderedWidth := (m.width - gap) / 2
    borderWidth := 2
    paneContentWidth := paneRenderedWidth - borderWidth

    // Create styled panes
    leftPane := lipgloss.NewStyle().
        Width(paneContentWidth).  // Content width (padding inside)
        Padding(0, 1).
        Border(lipgloss.RoundedBorder()).  // Adds 2 to rendered width
        Render(m.leftList.View())

    rightPane := lipgloss.NewStyle().
        Width(paneContentWidth).
        Padding(0, 1).
        Border(lipgloss.RoundedBorder()).
        Render(m.rightList.View())

    // Join horizontally with gap
    panes := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, "  ", rightPane)

    // Stack vertically (use JoinVertical, NOT string concatenation!)
    return lipgloss.JoinVertical(lipgloss.Left, panes, "", helpText)
}
```

**Key Points:**
- Calculate dimensions in **both** Update() and View() (keep them synchronized)
- Remember: `Style.Width()` includes padding, excludes border
- Use `JoinHorizontal` for side-by-side, `JoinVertical` for stacking
- Always use proper join functions, never string concatenation

**Verification:**
```go
renderedLeft := lipgloss.Width(leftPane)   // Should equal paneRenderedWidth
renderedRight := lipgloss.Width(rightPane) // Should equal paneRenderedWidth
total := renderedLeft + gap + renderedRight // Should equal m.width
```

### Height Calculation with Named Constants

**NEVER use magic numbers - use named constants:**

```go
// Self-documenting height calculation
const (
    headerLines       = 1 // Header text
    instructionsLines = 1 // Instructions text (optional)
    helpLines         = 1 // Help footer
    blankLines        = 3 // Spacers between sections (optional)
    paneBorders       = 2 // Top and bottom
    paneTitle         = 1 // Title inside pane
    paneSeparator     = 1 // Separator line
    paneFooter        = 1 // Footer (reserved even if empty)
)

uiOverhead := headerLines + helpLines + paneBorders + paneTitle + paneSeparator + paneFooter
optionalOverhead := instructionsLines + blankLines

// Graceful degradation
availableHeight := m.height - uiOverhead
if availableHeight >= minContentHeight + optionalOverhead {
    // Show everything
    showInstructions = true
    actualBlankLines = 3
} else if availableHeight >= minContentHeight {
    // Drop optional elements
    showInstructions = false
    actualBlankLines = 0
}

contentHeight := m.height - uiOverhead - actualOptionalOverhead
```

**Critical: Align Pane Heights**

If panes have different content amounts, set explicit height to align bottom borders:

```go
// Force both panes to same height
paneHeight := listHeight + 2  // list content + top/bottom borders

leftPane := lipgloss.NewStyle().
    Width(paneContentWidth).
    Height(paneHeight).        // Explicit height ensures alignment
    Border(border).
    Render(m.leftList.View())

rightPane := lipgloss.NewStyle().
    Width(paneContentWidth).
    Height(paneHeight).        // Same height = aligned borders
    Border(border).
    Render(m.rightList.View())
```

**Why:** Without explicit height, panes render to content height, causing misaligned borders.

**Accounting Checklist:**
```
✓ Header rows (if any)
✓ Blank spacer rows
✓ Pane borders (2 rows: top + bottom)
✓ Footer rows (if any)
= Total overhead to subtract from terminal height
```

### The Golden Rules

1. **NEVER use string concatenation for layout** - Always use `JoinVertical` or `JoinHorizontal`
   ```go
   // ❌ WRONG - Width() will sum widths, not take max
   baseView := panes + help

   // ✅ CORRECT - Proper vertical stacking
   baseView := lipgloss.JoinVertical(lipgloss.Left, panes, "", help)
   ```

2. **Always calculate dimensions in BOTH Update() and View()** - Keep them synchronized

3. **Handle WindowSizeMsg race condition:**
   ```go
   func (m Model) View() string {
       if m.width == 0 || m.height == 0 {
           return "Initializing..."
       }
       // ... rest of view
   }
   ```

4. **Use `lipgloss.Place()` to fill terminal and eliminate gaps:**
   ```go
   // Fill terminal dimensions exactly, preventing gaps on edges
   return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, content)

   // For centered modals/dialogs
   return lipgloss.Place(
       m.width, m.height,
       lipgloss.Center, lipgloss.Center,
       modal,
       lipgloss.WithWhitespaceChars(" "),
       lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
   )
   ```

## State Management Patterns

### State Machine Pattern

For complex workflows (wizards, multi-step processes):

```go
type State int

const (
    StateSelectVersion State = iota
    StateDownloading
    StateProcessing
    StateComplete
)

type Model struct {
    currentState State
    // ... other fields
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case CustomCompleteMsg:
        // State transition
        m.currentState = StateComplete
        return m, nil
    }

    // Different behavior based on state
    switch m.currentState {
    case StateSelectVersion:
        // Handle version selection input
    case StateDownloading:
        // Show progress, block input
    }
    return m, nil
}
```

### Message Passing for Async Operations

```go
// Define custom messages
type DownloadProgressMsg struct {
    Percent float64
}

type DownloadCompleteMsg struct {
    Error error
}

// Async command pattern
func downloadFile(url string) tea.Cmd {
    return func() tea.Msg {
        progressChan := make(chan float64, 10)
        done := make(chan error, 1)

        go func() {
            // Actual download
            err := doDownload(url, func(p float64) {
                progressChan <- p
            })
            close(progressChan)
            done <- err
        }()

        // Listen for progress
        select {
        case p := <-progressChan:
            return DownloadProgressMsg{Percent: p}
        case err := <-done:
            return DownloadCompleteMsg{Error: err}
        }
    }
}
```

## Theme Integration

**ALWAYS use theme helpers for consistency:**

```go
// Example theme pattern (adjust package/struct to your project)
theme := yourapp.CurrentTheme  // or however your theme is accessed

// Use theme colors
style := lipgloss.NewStyle().Foreground(theme.GetPrimaryColor())

// Use theme helpers for messages
fmt.Println(theme.SuccessMessage("Done!"))      // ✓ Done!
fmt.Println(theme.ErrorMessage("Failed!"))      // ✗ Failed!
fmt.Println(theme.WarningMessage("Caution!"))   // ⚠ Caution!

// Use theme helpers for indicators
activeIndicator := theme.ActiveIndicator()      // ●
pendingIndicator := theme.PendingIndicator()    // ○
completeIndicator := theme.CompleteIndicator()  // ✓
errorIndicator := theme.ErrorIndicator()        // ✗

// Extract reusable rendering to theme if used across multiple TUIs
header := theme.RenderHeader(width, "SECTION", "CONTEXT")
footer := theme.RenderFooter(width, helpContent)
```

## Debugging TUI Applications

**CRITICAL for AI Agents:** You do NOT have an interactive terminal and CANNOT see stdin/stdout. TUI applications take over the terminal, making it impossible for you to see output. You MUST use file-based logging for ALL debugging.

**Your debugging workflow:**
1. Clear the log file BEFORE test runs: `echo "" > /path/to/app/debug.log`
2. Make code changes
3. Build and install your application
4. User tests the TUI (you cannot see this)
5. Read the log file: `tail -50 /path/to/app/debug.log` or use `Read` tool
6. Analyze issues from logs and user reports

TUI applications take over stdout/stderr, so use **file-based logging:**

```go
// Setup in main or init (adjust path to your project's data directory)
logFile := "/path/to/app/data/debug.log"  // e.g., ~/.local/share/yourapp/debug.log
f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)

logger := log.NewWithOptions(f, log.Options{
    ReportTimestamp: true,
    TimeFormat:      "2006-01-02T15:04:05.000Z07:00",
    Level:           log.DebugLevel,
    ReportCaller:    true,
    Formatter:       log.JSONFormatter,
})
log.SetDefault(logger)

// Use throughout code
log.Debugf("WindowSize: %dx%d, calculated=%d", m.width, m.height, calculated)
```

**Debugging workflow:**
1. Clear log: `echo "" > /path/to/app/debug.log`
2. Run TUI
3. Check logs: `tail -50 /path/to/app/debug.log`
4. Analyze dimension mismatches

**Common debug checks:**
```go
actualWidth := lipgloss.Width(component)
log.Debugf("Component: expected=%d actual=%d gap=%d", m.width, actualWidth, m.width-actualWidth)
```

## Reusable Components Pattern

Extract common patterns to helpers:

```go
// ui/helpers.go

// Calculate split-pane dimensions
func CalculateSplitPaneDimensions(terminalWidth, terminalHeight int) LayoutDimensions {
    gap := 2
    paneRenderedWidth := (terminalWidth - gap) / 2
    borderWidth := 2
    paneContentWidth := paneRenderedWidth - borderWidth

    return LayoutDimensions{
        Width:             terminalWidth,
        Height:            terminalHeight,
        PaneContentWidth:  paneContentWidth,
        PaneRenderedWidth: paneRenderedWidth,
    }
}

// Create styled panes with active/inactive states
func CreatePaneStyle(isActive bool, accentColor, mutedColor lipgloss.Color, contentWidth int) lipgloss.Style {
    if isActive {
        return lipgloss.NewStyle().
            Border(lipgloss.ThickBorder()).
            BorderForeground(accentColor).
            Width(contentWidth).
            Padding(0, 1)
    }
    return lipgloss.NewStyle().
        Border(lipgloss.NormalBorder()).
        BorderForeground(mutedColor).
        Width(contentWidth).
        Padding(0, 1)
}
```

## Huh Forms and Confirmations

### Quick Confirmation Dialog

```go
var confirmed bool
form := huh.NewForm(
    huh.NewGroup(
        huh.NewConfirm().
            Title("Delete this version?").
            Description("This cannot be undone.").
            Affirmative("Yes").
            Negative("No").
            Value(&confirmed),
    ),
)
err := form.Run()
if err != nil {
    return err
}
if confirmed {
    // Perform action
}
```

### Inline Key Handling (Without Forms)

For instant y/n response without form submission:

```go
// In Update():
if keyMsg, ok := msg.(tea.KeyMsg); ok {
    switch keyMsg.String() {
    case "y", "Y":
        // Immediate action
        return m, performActionCmd
    case "n", "N", "esc":
        // Cancel
        m.state = StatePrevious
        return m, nil
    }
}
```

Use this pattern when you want immediate response without Enter key confirmation.

## Interactive vs Non-Interactive Mode

CLI applications should detect if they're running in an interactive terminal:

```go
import "golang.org/x/term"

func isInteractive() bool {
    return term.IsTerminal(int(os.Stdin.Fd()))
}

// In command implementation
func runCommand() error {
    if isInteractive() {
        // Show TUI
        p := tea.NewProgram(NewModel())
        return p.Start()
    }
    // Simple text output for pipes/scripts
    return simpleOutput()
}
```

**Why:** TUIs break when stdin/stdout are redirected. Provide both modes for flexibility.

## Common Patterns

### Progress Indicators

```go
// Use progress bar for known progress (downloads)
if m.phase == PhaseDownloading {
    indicator = m.progress.View()
} else {
    // Use spinner for unknown duration (processing)
    indicator = m.spinner.View()
}

// Initialize in NewModel()
prog := progress.New(progress.WithGradient(theme.Secondary, theme.Primary))
spinner := spinner.New()
spinner.Spinner = spinner.Dot
spinner.Style = lipgloss.NewStyle().Foreground(theme.GetPrimaryColor())
```

### Tabbed Interfaces

```go
type Tab struct {
    Title   string
    State   TabState  // Active, Pending, Complete, Error
    Spinner spinner.Model
}

// Render tabs with proper border connections
func RenderTabs(tabs []Tab, cfg TabsConfig) string {
    // Custom border connections
    inactiveTabBorder := tabBorderWithBottom("┴", "─", "┴")
    activeTabBorder := tabBorderWithBottom("┘", " ", "└")

    // Render each tab with appropriate style
    // Join horizontally
    // Extend to full width if needed
}
```

### Modal Overlays

```go
// Render modal over existing content
func renderModal(content string, width, height int) string {
    modal := lipgloss.NewStyle().
        Border(lipgloss.RoundedBorder()).
        BorderForeground(theme.GetPrimaryColor()).
        Padding(1, 2).
        Width(40).
        Render(content)

    return lipgloss.Place(
        width, height,
        lipgloss.Center, lipgloss.Center,
        modal,
        lipgloss.WithWhitespaceChars(" "),
        lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
    )
}
```

## Anti-Patterns to Avoid

1. ❌ **String concatenation for layout** - Use `JoinVertical`/`JoinHorizontal`
2. ❌ **Mutating state in View()** - View must be pure
3. ❌ **Magic numbers in height calculations** - Use named constants
4. ❌ **Hardcoded colors** - Use theme helpers
5. ❌ **Ignoring WindowSizeMsg** - Handle terminal resizing
6. ❌ **Blocking operations in Update()** - Use commands for async work
7. ❌ **Using stdout/stderr for debugging** - Use file-based logging
8. ❌ **Forgetting border width in calculations** - Always account for 2-char borders

## Testing Workflow

**AI Agent Responsibilities:**

1. **Before test runs:** Clear logs with `echo "" > /path/to/app/debug.log`
2. **After code changes:** Build and install the application
3. **After user tests:** Read log file to diagnose issues
4. **Never:** Kill running processes without asking first
5. **Always:** Use log file for debugging - you cannot see the TUI

**Testing Steps:**

1. Clear logs: `echo "" > /path/to/app/debug.log`
2. Build and install your application
3. User tests TUI functionality (you cannot see this)
4. Check logs for dimension issues: `tail -50 /path/to/app/debug.log`
5. Analyze actual rendered output vs expected from logs and user feedback

## Key Takeaways (Critical Reminders)

1. **Never use `+` to combine UI elements** - Always use `JoinVertical` or `JoinHorizontal`, never string concatenation. `Width()` measures max line width, not sum.

2. **Style.Width() includes padding, excludes border** - Only border adds to rendered width (as of v1.1.1). Actual rendered width = Style.Width() + border width.

3. **All border types are 2 chars wide** - Thick, Normal, Rounded, Double borders all use same width; swap freely for visual effect without breaking layouts.

4. **Split-pane calculation pattern** - `contentWidth = (terminalWidth - gap) / 2 - borderWidth`. Calculate in both Update() and View().

5. **Account for every row with named constants** - Use const block to document height overhead, no magic numbers. Makes code self-documenting.

6. **Use explicit Height() for pane alignment** - Force same height to align bottom borders when content amounts differ.

7. **Measure rendered output** - `Width()` and `Height()` measure actual display size. Use for verification and debugging.

8. **Log to files, not stdout** - TUI owns stdout/stderr. Use file-based logging for debugging. Clear log before tests.

9. **WindowSizeMsg is async** - Handle zero dimensions gracefully in initial View() calls. Check for `m.width == 0`.

10. **Use theme consistently** - Centralized color scheme. Use theme helpers for all colors and messages.

11. **Keep View() pure** - No side effects, no state mutation. View is called frequently and must be fast.

12. **Calculate dimensions in both Update() and View()** - Keep sizing logic synchronized between state updates and rendering.

## Resources

- **Bubble Tea State Machine Pattern**: https://zackproser.com/blog/bubbletea-state-machine
- **Building Bubble Tea Programs**: https://leg100.github.io/en/posts/building-bubbletea-programs/
- **Commands in Bubble Tea**: https://charm.land/blog/commands-in-bubbletea/
- **Lipgloss Examples**: https://github.com/charmbracelet/lipgloss/tree/master/examples
- **Bubble Tea Examples**: https://github.com/charmbracelet/bubbletea/tree/main/examples
- **Bubble Tea Tutorial**: https://github.com/charmbracelet/bubbletea/tree/master/tutorials
- **Huh Documentation**: https://github.com/charmbracelet/huh

## Quick Checklist

Before committing TUI code, verify:

- [ ] WindowSizeMsg handled properly
- [ ] Width/height calculations use named constants
- [ ] All layout uses JoinVertical/JoinHorizontal (no string concat)
- [ ] Border widths accounted for (all borders = 2 chars)
- [ ] Theme helpers used consistently
- [ ] View() is pure (no side effects)
- [ ] File-based logging for debug output
- [ ] Terminal resizing works smoothly
- [ ] Small terminals handled gracefully (optional elements drop)
- [ ] Async operations use commands and messages
- [ ] State machine clearly defined if multi-step process
