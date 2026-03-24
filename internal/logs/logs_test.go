package logs

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testRoleUser = "user"

func TestResolvePath(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := ResolvePath("abc-123", "/Users/me/projects/foo")
	want := filepath.Join(home, ".claude", "projects", "-Users-me-projects-foo", "abc-123.jsonl")
	if got != want {
		t.Errorf("ResolvePath = %q, want %q", got, want)
	}
}

func TestResolvePath_DotsReplacedWithHyphens(t *testing.T) {
	home, _ := os.UserHomeDir()
	got := ResolvePath("abc-123", "/Users/morgan.blanloeil/repos/my-project")
	want := filepath.Join(home, ".claude", "projects", "-Users-morgan-blanloeil-repos-my-project", "abc-123.jsonl")
	if got != want {
		t.Errorf("ResolvePath = %q, want %q", got, want)
	}
}

func TestParseLine_User(t *testing.T) {
	line := `{"type":"user","timestamp":"2026-03-20T10:23:02.940Z","message":{"role":"user","content":"Hello world, how are you doing today?"}}`
	entry, ok := parseLine([]byte(line))
	if !ok {
		t.Fatal("parseLine returned false for user message")
	}
	if entry.Type != EntryUser {
		t.Errorf("Type = %d, want EntryUser", entry.Type)
	}
	if entry.RawType != testRoleUser {
		t.Errorf("RawType = %q, want %q", entry.RawType, testRoleUser)
	}
	if entry.Summary != "Hello world, how are you doing today?" {
		t.Errorf("Summary = %q", entry.Summary)
	}
	if entry.Timestamp.IsZero() {
		t.Error("Timestamp is zero")
	}
}

func TestParseLine_AssistantText(t *testing.T) {
	line := `{"type":"assistant","timestamp":"2026-03-20T10:23:06.719Z","message":{"role":"assistant","content":[{"type":"text","text":"I will help you with that."}]}}`
	entry, ok := parseLine([]byte(line))
	if !ok {
		t.Fatal("parseLine returned false")
	}
	if entry.Type != EntryAssistant {
		t.Errorf("Type = %d, want EntryAssistant", entry.Type)
	}
	if entry.Summary != "I will help you with that." {
		t.Errorf("Summary = %q", entry.Summary)
	}
}

func TestParseLine_ToolUse(t *testing.T) {
	line := `{"type":"assistant","timestamp":"2026-03-20T10:23:10.643Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Read","input":{"path":"/foo"}}]}}`
	entry, ok := parseLine([]byte(line))
	if !ok {
		t.Fatal("parseLine returned false")
	}
	if entry.Summary != "Tool: Read" {
		t.Errorf("Summary = %q, want %q", entry.Summary, "Tool: Read")
	}
	if entry.RawType != "tool_use" {
		t.Errorf("RawType = %q", entry.RawType)
	}
}

func TestParseLine_Thinking(t *testing.T) {
	line := `{"type":"assistant","timestamp":"2026-03-20T10:23:06.161Z","message":{"role":"assistant","content":[{"type":"thinking","thinking":"Let me think about this..."}]}}`
	entry, ok := parseLine([]byte(line))
	if !ok {
		t.Fatal("parseLine returned false")
	}
	if !entry.HasThink {
		t.Error("HasThink should be true")
	}
	if entry.Summary != "[thinking]" {
		t.Errorf("Summary = %q", entry.Summary)
	}
}

func TestParseLine_ToolResult(t *testing.T) {
	line := `{"type":"user","timestamp":"2026-03-20T10:23:12.000Z","message":{"role":"user","content":[{"type":"tool_result","tool_use_id":"123","content":"ok"}]}}`
	entry, ok := parseLine([]byte(line))
	if !ok {
		t.Fatal("parseLine returned false for tool_result")
	}
	if entry.Type != EntryUser {
		t.Errorf("Type = %d, want EntryUser", entry.Type)
	}
	if entry.Summary != "Tool result" {
		t.Errorf("Summary = %q", entry.Summary)
	}
}

func TestParseLine_SkipsProgress(t *testing.T) {
	line := `{"type":"progress","timestamp":"2026-03-20T10:23:10.645Z"}`
	_, ok := parseLine([]byte(line))
	if ok {
		t.Error("parseLine should skip progress lines")
	}
}

func TestParseLine_TruncatesLongSummary(t *testing.T) {
	long := strings.Repeat("x", 200)
	line := `{"type":"user","timestamp":"2026-03-20T10:00:00Z","message":{"role":"user","content":"` + long + `"}}`
	entry, ok := parseLine([]byte(line))
	if !ok {
		t.Fatal("parseLine returned false")
	}
	runes := []rune(entry.Summary)
	if len(runes) > maxSummaryLen {
		t.Errorf("Summary rune length = %d, want <= %d", len(runes), maxSummaryLen)
	}
}

func TestReadTailAndReadFrom(t *testing.T) {
	// Create a temp JSONL file
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	lines := []string{
		`{"type":"user","timestamp":"2026-03-20T10:00:00Z","message":{"role":"user","content":"First message"}}`,
		`{"type":"assistant","timestamp":"2026-03-20T10:00:01Z","message":{"role":"assistant","content":[{"type":"text","text":"Response one"}]}}`,
		`{"type":"progress","timestamp":"2026-03-20T10:00:02Z"}`,
		`{"type":"user","timestamp":"2026-03-20T10:00:03Z","message":{"role":"user","content":"Second message"}}`,
		`{"type":"assistant","timestamp":"2026-03-20T10:00:04Z","message":{"role":"assistant","content":[{"type":"tool_use","name":"Bash","input":{}}]}}`,
	}
	content := strings.Join(lines, "\n") + "\n"
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// ReadTail
	entries, offset, err := ReadTail(path, 100)
	if err != nil {
		t.Fatal(err)
	}
	// 5 lines, 1 progress skipped = 4 entries
	if len(entries) != 4 {
		t.Fatalf("ReadTail got %d entries, want 4", len(entries))
	}
	if entries[0].Summary != "First message" {
		t.Errorf("entries[0].Summary = %q", entries[0].Summary)
	}
	if entries[3].Summary != "Tool: Bash" {
		t.Errorf("entries[3].Summary = %q", entries[3].Summary)
	}

	// ReadTail with limit
	entries2, _, err := ReadTail(path, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries2) != 2 {
		t.Fatalf("ReadTail(2) got %d entries, want 2", len(entries2))
	}

	// ReadFrom at offset (nothing new)
	newEntries, newOffset, err := ReadFrom(path, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(newEntries) != 0 {
		t.Errorf("ReadFrom got %d new entries, want 0", len(newEntries))
	}
	if newOffset != offset {
		t.Errorf("offset changed")
	}

	// Append a new line and ReadFrom
	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString(`{"type":"user","timestamp":"2026-03-20T10:00:05Z","message":{"role":"user","content":"Third message"}}` + "\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}

	newEntries, _, err = ReadFrom(path, offset)
	if err != nil {
		t.Fatal(err)
	}
	if len(newEntries) != 1 {
		t.Fatalf("ReadFrom got %d new entries, want 1", len(newEntries))
	}
	if newEntries[0].Summary != "Third message" {
		t.Errorf("newEntries[0].Summary = %q", newEntries[0].Summary)
	}
}
