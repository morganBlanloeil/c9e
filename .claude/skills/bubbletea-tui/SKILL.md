---
name: bubbletea-tui
description: >-
  Charmbracelet Bubbletea TUI expert for building terminal user interfaces.
  Use when working with bubbletea models, views, updates, lipgloss styles,
  or bubbles components. Triggers on internal/tui/ files, lipgloss, bubbletea imports,
  or when user mentions TUI, terminal UI, or Charmbracelet.
allowed-tools: Read, Edit, Write, Grep, Glob, Bash
---

# Bubbletea TUI Expert

Expert in building terminal UIs with the Charmbracelet stack: bubbletea, lipgloss, and bubbles.

## Architecture: Model-View-Update

Bubbletea follows the Elm architecture:

```
User Input → Update(msg) → Model → View() → Terminal Output
                ↑                      |
                └──── Cmd (async) ─────┘
```

### Model

```go
type Model struct {
    // State
    rows     []display.Row
    selected int
    view     viewState

    // Dimensions
    width  int
    height int
}
```

- Keep models flat — avoid deep nesting
- Use typed constants for view states: `viewList`, `viewDetail`, `viewLogs`
- Store terminal dimensions for responsive layouts

### Update

```go
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        return m.handleKey(msg)
    case tea.WindowSizeMsg:
        m.width = msg.Width
        m.height = msg.Height
        return m, nil
    case tickMsg:
        return m, m.fetchData()
    }
    return m, nil
}
```

- Type-switch on `tea.Msg` — never use raw strings
- Return `tea.Cmd` for async operations (data fetching, timers)
- Use `tea.Batch()` to combine multiple commands
- Use `tea.Tick()` for periodic refresh

### View

```go
func (m Model) View() string {
    switch m.view {
    case viewList:
        return m.renderList()
    case viewDetail:
        return m.renderDetail()
    }
    return ""
}
```

- View is a pure function — no side effects
- Build strings with `lipgloss` — never raw ANSI codes
- Respect `m.width` and `m.height` for layout

## Lipgloss Styling

### Adaptive Colors (mandatory in this project)

```go
// DO: Always use AdaptiveColor
var statusActive = lipgloss.AdaptiveColor{Light: "28", Dark: "46"}

// DON'T: Hardcoded colors
var statusActive = lipgloss.Color("46")
```

### Common Patterns

```go
// Base style with padding
var baseStyle = lipgloss.NewStyle().
    Padding(0, 1)

// Table header
var headerStyle = lipgloss.NewStyle().
    Bold(true).
    Foreground(lipgloss.AdaptiveColor{Light: "236", Dark: "252"})

// Selected row
var selectedStyle = lipgloss.NewStyle().
    Background(lipgloss.AdaptiveColor{Light: "253", Dark: "236"})
```

- Chain style methods fluently
- Use `.Width()` and `.MaxWidth()` for column alignment
- Use `lipgloss.JoinHorizontal()` / `lipgloss.JoinVertical()` for layout
- Use `lipgloss.Place()` for centering

## Key Handling

```go
func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
    switch msg.String() {
    case "q", "esc":
        if m.view != viewList {
            m.view = viewList
            return m, nil
        }
        return m, tea.Quit
    case "j", "down":
        m.selected = min(m.selected+1, len(m.rows)-1)
    case "k", "up":
        m.selected = max(m.selected-1, 0)
    case "enter":
        m.view = viewDetail
    }
    return m, nil
}
```

- Use `msg.String()` for key matching
- Support both vim keys (j/k) and arrows
- Use `tea.Quit` to exit cleanly

## Destructive Actions

```go
// Confirmation pattern used in this project
func (m *Model) confirmAction(action string) {
    m.confirming = true
    m.confirmMsg = fmt.Sprintf("Really %s? (y/n)", action)
}
```

- Always confirm before kill/delete operations
- Show confirmation in the help bar area
- Cancel on any key except `y`

## Async Data Fetching

```go
type dataMsg struct {
    rows []display.Row
    err  error
}

func fetchDataCmd() tea.Msg {
    rows, err := fetchRows()
    return dataMsg{rows: rows, err: err}
}

// In Update:
case dataMsg:
    if msg.err != nil {
        m.err = msg.err
        return m, nil
    }
    m.rows = msg.rows
```

- Wrap results in typed messages
- Always handle errors in the message
- Use `tea.Tick` for periodic refresh, not goroutines

## Bubbles Components

Available in this project's dependencies:

| Component | Use For |
|-----------|---------|
| `viewport` | Scrollable content (detail view, logs) |
| `textinput` | Filter/search input |
| `spinner` | Loading indicators |
| `key` | Key binding definitions |
