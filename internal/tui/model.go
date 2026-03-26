package tui

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wescale/claude-dashboard/internal/display"
	"github.com/wescale/claude-dashboard/internal/logs"
	"github.com/wescale/claude-dashboard/internal/notify"
	"github.com/wescale/claude-dashboard/internal/process"
	"github.com/wescale/claude-dashboard/internal/terminal"
)

// SortColumn identifies which column to sort by.
type SortColumn int

const (
	SortByPID SortColumn = iota
	SortByStatus
	SortByCPU
	SortByMem
	SortByUptime
	SortByIdle
	SortByDir
	SortByAction
	SortByTurns
)

// ErrUnsupportedOS is returned when the OS does not support clipboard operations.
var ErrUnsupportedOS = errors.New("unsupported OS for clipboard")

var sortColumnNames = []string{"PID", "STATUS", "CPU%", "MEM%", "UPTIME", "IDLE", "DIR", "ACTION", "TURNS"}

const (
	refreshInterval        = 3 * time.Second
	keyEsc                 = "esc"
	keyCtrlC               = "ctrl+c"
	sessionIDLen           = 8
	doneHighlightDuration  = 30 * time.Second
	flashDuration          = 3 * time.Second
	clipboardFlashDuration = 2 * time.Second
	defaultLogTailSize     = 200
	bitSize64              = 64
	msPerSecond            = 1000
)

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

	// Notification state
	notifyEnabled bool
	notifyFlash   string    // flash message shown briefly after notification
	notifyFlashAt time.Time // when the flash was set

	// Sort state
	sortCol SortColumn
	sortAsc bool // true = ascending, false = descending

	// Clipboard flash
	clipboardFlash    string
	clipboardFlashEnd time.Time

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

	// Configurable home directory for data sources (default: os.UserHomeDir())
	homeDir string
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

// clipboardFlashMsg clears the clipboard flash message.
type clipboardFlashMsg struct{}

// jumpMsg carries the result of a jump-to-terminal attempt.
type jumpMsg struct{ err error }

// copyToClipboard copies text to system clipboard.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.CommandContext(context.Background(), "pbcopy")
	case "linux":
		cmd = exec.CommandContext(context.Background(), "xclip", "-selection", "clipboard")
	default:
		return fmt.Errorf("%w: %s", ErrUnsupportedOS, runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("copying to clipboard: %w", err)
	}
	return nil
}

// NewModel creates a new TUI model.
func NewModel(version string) Model {
	return Model{
		rows:              nil,
		filtered:          nil,
		cursor:            0,
		width:             0,
		height:            0,
		view:              viewList,
		filter:            "",
		filtering:         false,
		confirm:           nil,
		err:               nil,
		version:           version,
		prevStatus:        make(map[string]string),
		doneHighlight:     make(map[string]time.Time),
		notifyEnabled:     notify.Available(),
		notifyFlash:       "",
		notifyFlashAt:     time.Time{},
		sortCol:           SortByPID,
		sortAsc:           true,
		clipboardFlash:    "",
		clipboardFlashEnd: time.Time{},
		logEntries:        nil,
		logFiltered:       nil,
		logScroll:         0,
		logOffset:         0,
		logPath:           "",
		logFollow:         false,
		logShowThink:      false,
		logErr:            nil,
		logFrom:           viewList,
		homeDir:           "",
	}
}

// WithHomeDir returns a copy of the model with the given home directory
// for loading session data. Used for testing.
func (m Model) WithHomeDir(dir string) Model {
	m.homeDir = dir
	return m
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
			entries, off, err := logs.ReadTail(path, defaultLogTailSize)
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

func fetchDataCmd(homeDir string) tea.Cmd {
	return func() tea.Msg {
		rows, err := fetchRows(homeDir)
		return dataMsg{rows: rows, err: err}
	}
}

// Init starts the initial data fetch and tick timer.
func (m Model) Init() tea.Cmd {
	return tea.Batch(fetchDataCmd(m.homeDir), doTick())
}

// Update handles messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		cmds := []tea.Cmd{fetchDataCmd(m.homeDir), doTick()}
		if m.view == viewLogs && m.logPath != "" {
			cmds = append(cmds, fetchLogCmd(m.logPath, m.logOffset, false))
		}
		return m, tea.Batch(cmds...)

	case clipboardFlashMsg:
		m.clipboardFlash = ""
		return m, nil

	case dataMsg:
		m.err = msg.err
		if msg.err == nil {
			m.rows = msg.rows
			m.sortRows()

			// Detect sessions that transitioned from ACTIVE/WAITING to IDLE/DEAD
			now := time.Now()
			curStatus := make(map[string]string, len(m.rows))
			for _, r := range m.rows {
				curStatus[r.SessionID] = string(r.Status)
				prev, hasPrev := m.prevStatus[r.SessionID]
				wasActive := prev == string(display.StatusActive) || prev == string(display.StatusWaiting)
				nowDone := r.Status == display.StatusIdle || r.Status == display.StatusDead
				if hasPrev && wasActive && nowDone {
					m.doneHighlight[r.SessionID] = now.Add(doneHighlightDuration)
					// Send desktop notification
					if m.notifyEnabled {
						dir := r.Cwd
						if dir == "" {
							dir = r.SessionID
						}
						_ = notify.Send("c9e \u2014 Task Complete", dir+" \u2014 session finished")
						m.notifyFlash = "\U0001f514 Notified: " + dir
						m.notifyFlashAt = now
					}
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

	case jumpMsg:
		if msg.err != nil {
			m.clipboardFlash = msg.err.Error()
		} else {
			m.clipboardFlash = "Jumped to terminal"
		}
		m.clipboardFlashEnd = time.Now().Add(clipboardFlashDuration)
		return m, tea.Tick(clipboardFlashDuration, func(_ time.Time) tea.Msg {
			return clipboardFlashMsg{}
		})

	case killMsg:
		if msg.err != nil {
			m.err = msg.err
		}
		m.confirm = nil
		return m, fetchDataCmd(m.homeDir)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

// View renders the TUI.
func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	switch m.view {
	case viewList:
		return m.viewList()
	case viewDetail:
		return m.viewDetail()
	case viewLogs:
		return m.viewLogs()
	default:
		return m.viewList()
	}
}

func (m Model) isDone(sessionID string) bool {
	expiry, ok := m.doneHighlight[sessionID]
	return ok && time.Now().Before(expiry)
}

func (m Model) activeFlash() string {
	if m.notifyFlash == "" {
		return ""
	}
	if time.Since(m.notifyFlashAt) > flashDuration {
		return ""
	}
	return m.notifyFlash
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
		case "enter", keyEsc:
			m.filtering = false
			if key == keyEsc {
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
		case "q", keyCtrlC:
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
		case keyEsc:
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
		case "n":
			m.notifyEnabled = !m.notifyEnabled
		case "s":
			// Cycle sort column forward
			m.sortCol = (m.sortCol + 1) % (SortByTurns + 1)
			m.sortRows()
			m.applyFilter()
		case "S":
			// Toggle sort direction
			m.sortAsc = !m.sortAsc
			m.sortRows()
			m.applyFilter()
		case "c":
			// Copy selected session's CWD to clipboard
			if r := m.selectedRow(); r != nil && r.RawCwd != "" {
				if err := copyToClipboard(r.RawCwd); err == nil {
					m.clipboardFlash = "Copied CWD!"
					m.clipboardFlashEnd = time.Now().Add(clipboardFlashDuration)
					return m, tea.Tick(clipboardFlashDuration, func(_ time.Time) tea.Msg {
						return clipboardFlashMsg{}
					})
				}
			}
		case "o":
			// Jump to the Ghostty tab running this session
			if r := m.selectedRow(); r != nil && r.RawCwd != "" {
				cwd := r.RawCwd
				return m, func() tea.Msg {
					return jumpMsg{err: terminal.FocusByWorkdir(cwd)}
				}
			}
		case "?":
			// Could show help modal — for now just cycle
		}

	case viewDetail:
		switch key {
		case "q", keyEsc, "backspace":
			m.view = viewList
		case keyCtrlC:
			return m, tea.Quit
		case "l":
			if cmd := m.openLogs(viewDetail); cmd != nil {
				return m, cmd
			}
		}

	case viewLogs:
		switch key {
		case "q", keyEsc:
			m.view = m.logFrom
		case keyCtrlC:
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

func (m *Model) sortRows() {
	col := m.sortCol
	asc := m.sortAsc
	sort.SliceStable(m.rows, func(i, j int) bool {
		less := false
		switch col {
		case SortByPID:
			less = m.rows[i].PID < m.rows[j].PID
		case SortByStatus:
			less = string(m.rows[i].Status) < string(m.rows[j].Status)
		case SortByCPU:
			ci, _ := strconv.ParseFloat(m.rows[i].CPU, bitSize64)
			cj, _ := strconv.ParseFloat(m.rows[j].CPU, bitSize64)
			less = ci < cj
		case SortByMem:
			mi, _ := strconv.ParseFloat(m.rows[i].Mem, bitSize64)
			mj, _ := strconv.ParseFloat(m.rows[j].Mem, bitSize64)
			less = mi < mj
		case SortByUptime:
			less = m.rows[i].UptimeSec < m.rows[j].UptimeSec
		case SortByIdle:
			less = m.rows[i].IdleSec < m.rows[j].IdleSec
		case SortByDir:
			less = strings.ToLower(m.rows[i].Cwd) < strings.ToLower(m.rows[j].Cwd)
		case SortByAction:
			less = strings.ToLower(m.rows[i].LastAction) < strings.ToLower(m.rows[j].LastAction)
		case SortByTurns:
			less = m.rows[i].Turns < m.rows[j].Turns
		}
		if !asc {
			return !less
		}
		return less
	})
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
	return max(1, m.height-logFixedLines)
}

func (m *Model) logScrollToBottom() {
	visible := m.logVisibleLines()
	if len(m.logFiltered) > visible {
		m.logScroll = len(m.logFiltered) - visible
	} else {
		m.logScroll = 0
	}
}
