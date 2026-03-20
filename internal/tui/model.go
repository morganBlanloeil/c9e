package tui

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wescale/claude-dashboard/internal/display"
	"github.com/wescale/claude-dashboard/internal/logs"
	"github.com/wescale/claude-dashboard/internal/process"
)

const refreshInterval = 3 * time.Second

// viewMode represents which screen is active.
type viewMode int

const (
	viewList viewMode = iota
	viewDetail
	viewLogs
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

	// Task-finished highlight state
	prevStatus    map[string]string    // previous refresh status keyed by session ID
	doneHighlight map[string]time.Time // sessions that just finished, with expiry time

	// Log view state
	logEntries   []logs.LogEntry
	logFiltered  []logs.LogEntry
	logScroll    int
	logOffset    int64
	logPath      string
	logFollow    bool
	logShowThink bool
	logErr       error
	logFrom      viewMode // view to return to on esc
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
		version:       version,
		prevStatus:    make(map[string]string),
		doneHighlight: make(map[string]time.Time),
	}
}

// isDone returns true if the session recently finished its task.
func (m Model) isDone(sessionID string) bool {
	expiry, ok := m.doneHighlight[sessionID]
	return ok && time.Now().Before(expiry)
}

// tickMsg triggers a data refresh.
type tickMsg time.Time

// dataMsg carries refreshed data.
type dataMsg struct {
	rows []display.Row
	err  error
}

// logDataMsg carries log entries from a fetch.
type logDataMsg struct {
	entries []logs.LogEntry
	offset  int64
	err     error
	initial bool
}

func fetchLogCmd(path string, offset int64, initial bool) tea.Cmd {
	return func() tea.Msg {
		if initial {
			entries, off, err := logs.ReadTail(path, 200)
			return logDataMsg{entries: entries, offset: off, err: err, initial: true}
		}
		entries, off, err := logs.ReadFrom(path, offset)
		return logDataMsg{entries: entries, offset: off, err: err, initial: false}
	}
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
		cmds := []tea.Cmd{fetchDataCmd(), doTick()}
		if m.view == viewLogs && m.logPath != "" {
			cmds = append(cmds, fetchLogCmd(m.logPath, m.logOffset, false))
		}
		return m, tea.Batch(cmds...)

	case dataMsg:
		m.err = msg.err
		if msg.err == nil {
			m.rows = msg.rows
			sort.Slice(m.rows, func(i, j int) bool {
				return m.rows[i].PID < m.rows[j].PID
			})

			// Detect sessions that transitioned from ACTIVE to IDLE/DEAD
			now := time.Now()
			curStatus := make(map[string]string, len(m.rows))
			for _, r := range m.rows {
				curStatus[r.SessionID] = string(r.Status)
				prev, hasPrev := m.prevStatus[r.SessionID]
				if hasPrev && prev == string(display.StatusActive) &&
					(r.Status == display.StatusIdle || r.Status == display.StatusDead) {
					m.doneHighlight[r.SessionID] = now.Add(30 * time.Second)
				}
			}
			// Prune expired highlights
			for sid, expiry := range m.doneHighlight {
				if now.After(expiry) {
					delete(m.doneHighlight, sid)
				}
			}
			m.prevStatus = curStatus

			m.applyFilter()
		}

	case logDataMsg:
		m.logErr = msg.err
		if msg.err == nil {
			if msg.initial {
				m.logEntries = msg.entries
			} else if len(msg.entries) > 0 {
				m.logEntries = append(m.logEntries, msg.entries...)
			}
			m.logOffset = msg.offset
			m.filterLogEntries()
			if m.logFollow {
				// Scroll to bottom
				m.logScrollToBottom()
			}
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
		case "l":
			if cmd := m.openLogs(viewList); cmd != nil {
				return m, cmd
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
		case "l":
			if cmd := m.openLogs(viewDetail); cmd != nil {
				return m, cmd
			}
		}

	case viewLogs:
		switch key {
		case "q", "esc":
			m.view = m.logFrom
		case "ctrl+c":
			return m, tea.Quit
		case "j", "down":
			visibleLines := m.logVisibleLines()
			if m.logScroll < len(m.logFiltered)-visibleLines {
				m.logScroll++
				m.logFollow = false
			}
		case "k", "up":
			if m.logScroll > 0 {
				m.logScroll--
				m.logFollow = false
			}
		case "G":
			m.logScrollToBottom()
			m.logFollow = true
		case "g":
			m.logScroll = 0
			m.logFollow = false
		case "f":
			m.logFollow = !m.logFollow
			if m.logFollow {
				m.logScrollToBottom()
			}
		case "t":
			m.logShowThink = !m.logShowThink
			m.filterLogEntries()
			if m.logFollow {
				m.logScrollToBottom()
			}
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
	case viewLogs:
		return m.viewLogs()
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

func (m *Model) openLogs(from viewMode) tea.Cmd {
	r := m.selectedRow()
	if r == nil || r.LogPath == "" {
		return nil
	}
	m.logPath = r.LogPath
	m.logEntries = nil
	m.logFiltered = nil
	m.logScroll = 0
	m.logOffset = 0
	m.logFollow = true
	m.logShowThink = false
	m.logErr = nil
	m.logFrom = from
	m.view = viewLogs
	return fetchLogCmd(m.logPath, 0, true)
}

func (m *Model) filterLogEntries() {
	if m.logShowThink {
		m.logFiltered = m.logEntries
	} else {
		m.logFiltered = nil
		for _, e := range m.logEntries {
			if !e.HasThink {
				m.logFiltered = append(m.logFiltered, e)
			}
		}
	}
}

func (m Model) logVisibleLines() int {
	v := m.height - 7 // title, 2 separators, status bar, separator, help, padding
	if v < 1 {
		return 1
	}
	return v
}

func (m *Model) logScrollToBottom() {
	visible := m.logVisibleLines()
	if len(m.logFiltered) > visible {
		m.logScroll = len(m.logFiltered) - visible
	} else {
		m.logScroll = 0
	}
}
