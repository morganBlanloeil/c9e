package tui

import (
	"regexp"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/golden"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/wescale/claude-dashboard/internal/display"
)

// ansiRE strips ANSI escape sequences — used only for content assertions, not golden files.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

// ---------------------------------------------------------------------------
// Golden file tests — ANSI codes preserved for visual regression
// Run with -update to regenerate: go test ./internal/tui/ -update
// ---------------------------------------------------------------------------

func TestGolden_ListView(t *testing.T) {
	m := newTestModel(t)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestGolden_DetailView(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestGolden_EmptyList(t *testing.T) {
	m := NewModel("test")
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = updated.(Model)
	updated, _ = m.Update(dataMsg{rows: nil, err: nil})
	m = updated.(Model)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestGolden_FilterActive(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = updated.(Model)
	for _, ch := range "alpha" {
		updated, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		m = updated.(Model)
	}
	golden.RequireEqual(t, []byte(m.View()))
}

func TestGolden_ConfirmKill(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	m = updated.(Model)
	golden.RequireEqual(t, []byte(m.View()))
}

func TestGolden_LogView(t *testing.T) {
	m := newTestModel(t)
	m.view = viewLogs
	m.logFollow = true
	m.logShowThink = false
	m.logFrom = viewList
	golden.RequireEqual(t, []byte(m.View()))
}

func TestGolden_SortDescending(t *testing.T) {
	m := newTestModel(t)
	// Toggle sort descending
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	m = updated.(Model)
	golden.RequireEqual(t, []byte(m.View()))
}

// ---------------------------------------------------------------------------
// Teatest full-pipeline tests — real bubbletea lifecycle
// ---------------------------------------------------------------------------

func TestTeatest_ListViewPipeline(t *testing.T) {
	home := setupFakeClaudeHome(t)
	m := NewModel("test").WithHomeDir(home)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Wait for sessions to appear (both should be DEAD since fake PIDs)
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "99991") &&
			strings.Contains(string(bts), "99992")
	}, teatest.WithDuration(5*time.Second))

	// Quit the program
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTeatest_NavigateToDetail(t *testing.T) {
	home := setupFakeClaudeHome(t)
	m := NewModel("test").WithHomeDir(home)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Wait for data to load
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "99991")
	}, teatest.WithDuration(5*time.Second))

	// Navigate to detail view
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Should show detail view content
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "Session") && strings.Contains(s, "PID")
	}, teatest.WithDuration(3*time.Second))

	// Go back and quit
	tm.Send(tea.KeyMsg{Type: tea.KeyEsc})
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

func TestTeatest_FilterSessions(t *testing.T) {
	home := setupFakeClaudeHome(t)
	m := NewModel("test").WithHomeDir(home)

	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 40))

	// Wait for data
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		return strings.Contains(string(bts), "99991")
	}, teatest.WithDuration(5*time.Second))

	// Filter for "project-one"
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	tm.Type("project-one")
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	// Should show filtered results — only project-one visible
	teatest.WaitFor(t, tm.Output(), func(bts []byte) bool {
		s := string(bts)
		return strings.Contains(s, "project-one") &&
			!strings.Contains(s, "project-two")
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(3*time.Second))
}

// ---------------------------------------------------------------------------
// Content assertion tests (non-golden, verify specific fields)
// ---------------------------------------------------------------------------

func TestView_DetailFields(t *testing.T) {
	m := newTestModel(t)
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = updated.(Model)

	view := stripANSI(m.View())
	checks := []string{
		"Session aabbccdd",
		"PID 1001",
		"ACTIVE",
		"12.3%",
		"4.5%",
		"$0.42",
		"implement feature X",
	}
	for _, want := range checks {
		if !strings.Contains(view, want) {
			t.Errorf("detail view missing %q", want)
		}
	}
}

func TestView_LogViewContent(t *testing.T) {
	m := newTestModel(t)
	m.view = viewLogs
	m.logFollow = true
	m.logShowThink = false
	m.logFrom = viewList

	view := stripANSI(m.View())
	if !strings.Contains(view, "Log Tail") {
		t.Error("log view should contain 'Log Tail' title")
	}
	if !strings.Contains(view, "follow:") {
		t.Error("log view should show follow status")
	}
	if !strings.Contains(view, "thinking:") {
		t.Error("log view should show thinking status")
	}
}

func TestView_ErrorDisplay(t *testing.T) {
	m := newTestModel(t)
	m.err = &testError{msg: "something went wrong"}

	view := stripANSI(m.View())
	if !strings.Contains(view, "something went wrong") {
		t.Error("view should display error message")
	}
}

func TestView_SortIndicator(t *testing.T) {
	m := newTestModel(t)
	view := stripANSI(m.View())
	if !strings.Contains(view, "sort: PID") {
		t.Error("view should show sort indicator for PID")
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = updated.(Model)
	view = stripANSI(m.View())
	if !strings.Contains(view, "sort: STATUS") {
		t.Error("view should show sort indicator for STATUS after 's'")
	}
}

// ---------------------------------------------------------------------------
// Unit tests for formatting helpers
// ---------------------------------------------------------------------------

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{name: "negative", seconds: -1, want: emDash},
		{name: "zero minutes", seconds: 0, want: "0m"},
		{name: "30 seconds", seconds: 30, want: "0m"},
		{name: "5 minutes", seconds: 300, want: "5m"},
		{name: "1 hour 30 min", seconds: 5400, want: "1h 30m"},
		{name: "1 day 2 hours", seconds: 93600, want: "1d 2h"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatDuration(tt.seconds)
			if got != tt.want {
				t.Errorf("formatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestFormatIdle(t *testing.T) {
	tests := []struct {
		name    string
		seconds int64
		want    string
	}{
		{name: "negative", seconds: -1, want: emDash},
		{name: "30 seconds", seconds: 30, want: "30s"},
		{name: "5 minutes", seconds: 300, want: "5m"},
		{name: "2 hours", seconds: 7200, want: "2h 0m"},
		{name: "1 day", seconds: 86400, want: "1d"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatIdle(tt.seconds)
			if got != tt.want {
				t.Errorf("formatIdle(%d) = %q, want %q", tt.seconds, got, tt.want)
			}
		})
	}
}

func TestFormatTokenCount(t *testing.T) {
	tests := []struct {
		name   string
		tokens int64
		want   string
	}{
		{name: "small", tokens: 500, want: "500"},
		{name: "thousands", tokens: 1500, want: "1.5K"},
		{name: "millions", tokens: 2500000, want: "2.5M"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTokenCount(tt.tokens)
			if got != tt.want {
				t.Errorf("formatTokenCount(%d) = %q, want %q", tt.tokens, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{name: "short", input: "hello", maxLen: 10, want: "hello"},
		{name: "exact", input: "hello", maxLen: 5, want: "hello"},
		{name: "long", input: "hello world", maxLen: 8, want: "hello w…"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncate(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

func TestStatusIcon_AllStatuses(t *testing.T) {
	statuses := []display.Status{
		display.StatusActive,
		display.StatusWaiting,
		display.StatusIdle,
		display.StatusDead,
	}
	for _, s := range statuses {
		icon := stripANSI(statusIcon(s))
		if icon == "" || icon == "?" {
			t.Errorf("statusIcon(%s) returned empty or unknown", s)
		}
	}
}
