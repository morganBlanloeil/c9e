package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/wescale/claude-dashboard/internal/display"
)

// fixtureRows returns deterministic display.Row values for testing.
func fixtureRows() []display.Row {
	return []display.Row{
		{
			PID:           1001,
			SessionID:     "aabbccdd",
			FullSessionID: "aabbccdd-1111-2222-3333-444444444444",
			Status:        display.StatusActive,
			CPU:           "12.3",
			Mem:           "4.5",
			Cwd:           "~/repos/project-alpha",
			RawCwd:        "/home/user/repos/project-alpha",
			UptimeSec:     3600,
			IdleSec:       30,
			LastAction:    "implement feature X",
			Alive:         true,
			LogPath:       "/tmp/fake/logs/alpha.jsonl",
			Turns:         5,
			Cost:          "$0.42",
			CostValue:     0.42,
			InputTokens:   50000,
			OutputTokens:  10000,
			CostModel:     "claude-sonnet-4-20250514",
			HasUsageData:  true,
		},
		{
			PID:           2002,
			SessionID:     "eeff0011",
			FullSessionID: "eeff0011-5555-6666-7777-888888888888",
			Status:        display.StatusIdle,
			CPU:           "0.0",
			Mem:           "1.2",
			Cwd:           "~/repos/project-beta",
			RawCwd:        "/home/user/repos/project-beta",
			UptimeSec:     7200,
			IdleSec:       600,
			LastAction:    "fix bug in parser",
			Alive:         true,
			LogPath:       "/tmp/fake/logs/beta.jsonl",
			Turns:         12,
			Cost:          "$1.50",
			CostValue:     1.50,
			InputTokens:   200000,
			OutputTokens:  30000,
			CostModel:     "claude-sonnet-4-20250514",
			HasUsageData:  true,
		},
		{
			PID:           3003,
			SessionID:     "22334455",
			FullSessionID: "22334455-9999-aaaa-bbbb-cccccccccccc",
			Status:        display.StatusDead,
			CPU:           "0.0",
			Mem:           "0.0",
			Cwd:           "~/repos/project-gamma",
			RawCwd:        "/home/user/repos/project-gamma",
			UptimeSec:     900,
			IdleSec:       -1,
			LastAction:    "review PR",
			Alive:         false,
			LogPath:       "",
			Turns:         3,
			Cost:          "",
			CostValue:     0,
			InputTokens:   0,
			OutputTokens:  0,
			CostModel:     "",
			HasUsageData:  false,
		},
	}
}

// sessionJSON represents a session file for fixture creation.
type sessionJSON struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Cwd       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"`
}

// writeSessionFile creates a session JSON file in the temp dir.
func writeSessionFile(t *testing.T, homeDir string, s sessionJSON) {
	t.Helper()
	dir := filepath.Join(homeDir, ".claude", "sessions")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, s.SessionID+".json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeHistoryFile creates a history.jsonl file in the temp dir.
func writeHistoryFile(t *testing.T, homeDir string, lines []string) {
	t.Helper()
	dir := filepath.Join(homeDir, ".claude")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	content := strings.Join(lines, "\n") + "\n"
	path := filepath.Join(dir, "history.jsonl")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// setupFakeClaudeHome creates a temp dir with fake Claude session data
// and returns the home directory path. The data simulates two sessions:
// one active and one dead (not matching any real PID).
func setupFakeClaudeHome(t *testing.T) string {
	t.Helper()
	homeDir := t.TempDir()

	// Session 1: will appear as DEAD since PID won't match any real process
	writeSessionFile(t, homeDir, sessionJSON{
		PID:       99991,
		SessionID: "aaaabbbb-1111-2222-3333-444444444444",
		Cwd:       "/tmp/project-one",
		StartedAt: 1700000000000,
	})

	// Session 2: also DEAD
	writeSessionFile(t, homeDir, sessionJSON{
		PID:       99992,
		SessionID: "ccccdddd-5555-6666-7777-888888888888",
		Cwd:       "/tmp/project-two",
		StartedAt: 1700001000000,
	})

	// History entries
	writeHistoryFile(t, homeDir, []string{
		`{"display":"write tests","timestamp":1700000500000,"project":"/tmp/project-one","sessionId":"aaaabbbb-1111-2222-3333-444444444444"}`,
		`{"display":"refactor code","timestamp":1700001500000,"project":"/tmp/project-two","sessionId":"ccccdddd-5555-6666-7777-888888888888"}`,
	})

	return homeDir
}

// newTestModel creates a model with fixture rows injected, ready for testing.
// It sets dimensions via WindowSizeMsg and injects data via dataMsg.
func newTestModel(t *testing.T) Model {
	t.Helper()
	m := NewModel("test")
	// Set dimensions
	sized, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = sized.(Model)
	// Inject data
	updated, _ := m.Update(dataMsg{rows: fixtureRows(), err: nil})
	m = updated.(Model)
	return m
}
