package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wescale/claude-dashboard/internal/display"
	"github.com/wescale/claude-dashboard/internal/process"
)

const refreshInterval = 3 * time.Second

// viewMode represents which screen is active.
type viewMode int

const (
	viewList viewMode = iota
	viewDetail
)

// Model is the bubbletea model for the TUI.
type Model struct {
	rows      []display.Row
	filtered  []display.Row
	cursor    int
	width     int
	height    int
	view      viewMode
	filter    string
	filtering bool
	confirm   *confirmAction // pending confirmation prompt
	err       error
	version   string
}

// confirmAction holds a pending action requiring user confirmation.
type confirmAction struct {
	label string
	pid   int
	onYes func(int) tea.Cmd
}

// killMsg is sent after a kill attempt.
type killMsg struct {
	pid int
	err error
}

// NewModel creates a new TUI model.
func NewModel(version string) Model {
	return Model{
		version: version,
	}
}

// tickMsg triggers a data refresh.
type tickMsg time.Time

// dataMsg carries refreshed data.
type dataMsg struct {
	rows []display.Row
	err  error
}

func doTick() tea.Cmd {
	return tea.Tick(refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func fetchDataCmd() tea.Cmd {
	return func() tea.Msg {
		rows, err := fetchRows()
		return dataMsg{rows: rows, err: err}
	}
}

// Init starts the initial data fetch and tick timer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchDataCmd(), doTick())
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tea.Batch(fetchDataCmd(), doTick())

	case dataMsg:
		m.err = msg.err
		if msg.err == nil {
			m.rows = msg.rows
			sort.Slice(m.rows, func(i, j int) bool {
				return m.rows[i].PID < m.rows[j].PID
			})
			m.applyFilter()
		}

	case killMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.confirm = nil
		return m, fetchDataCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Confirmation mode
	if m.confirm != nil {
		switch key {
		case "y", "Y":
			cmd := m.confirm.onYes(m.confirm.pid)
			return m, cmd
		default:
			m.confirm = nil
		}
		return m, nil
	}

	// Filter mode input
	if m.filtering {
		switch key {
		case "enter", "esc":
			m.filtering = false
			if key == "esc" {
				m.filter = ""
				m.applyFilter()
			}
		case "backspace":
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
		default:
			if len(key) == 1 {
				m.filter += key
				m.applyFilter()
			}
		}
		return m, nil
	}

	// Normal mode
	switch m.view {
	case viewList:
		switch key {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
		case "g", "home":
			m.cursor = 0
		case "G", "end":
			if len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
			}
		case "enter":
			if len(m.filtered) > 0 {
				m.view = viewDetail
			}
		case "/":
			m.filtering = true
			m.filter = ""
		case "esc":
			if m.filter != "" {
				m.filter = ""
				m.applyFilter()
			}
		case "d":
			if r := m.selectedRow(); r != nil && r.Alive {
				m.confirm = &confirmAction{
					label: fmt.Sprintf("Kill session PID %d? (y/N)", r.PID),
					pid:   r.PID,
					onYes: func(pid int) tea.Cmd {
						return func() tea.Msg {
							err := process.Kill(pid)
							return killMsg{pid: pid, err: err}
						}
					},
				}
			}
		case "?":
			// Could show help modal — for now just cycle
		}

	case viewDetail:
		switch key {
		case "q", "esc", "backspace":
			m.view = viewList
		case "ctrl+c":
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m *Model) applyFilter() {
	if m.filter == "" {
		m.filtered = m.rows
	} else {
		lower := strings.ToLower(m.filter)
		m.filtered = nil
		for _, r := range m.rows {
			if strings.Contains(strings.ToLower(r.Cwd), lower) ||
				strings.Contains(strings.ToLower(r.LastAction), lower) ||
				strings.Contains(strings.ToLower(string(r.Status)), lower) {
				m.filtered = append(m.filtered, r)
			}
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
}

// View renders the TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.view {
	case viewDetail:
		return m.viewDetail()
	default:
		return m.viewList()
	}
}

// selectedRow returns the currently selected row, if any.
func (m Model) selectedRow() *display.Row {
	if m.cursor >= 0 && m.cursor < len(m.filtered) {
		return &m.filtered[m.cursor]
	}
	return nil
}
