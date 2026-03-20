package logs

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogEntryType classifies a log entry.
type LogEntryType int

const (
	EntryUser LogEntryType = iota
	EntryAssistant
	EntrySystem
)

// LogEntry is a parsed line from a session JSONL file.
type LogEntry struct {
	Timestamp time.Time
	Type      LogEntryType
	RawType   string // original "type" field from JSONL
	Summary   string // human-readable summary
	HasThink  bool   // true if this is a thinking block
}

// ResolvePath constructs the JSONL path for a session.
// Claude Code stores session logs at ~/.claude/projects/{slug}/{sessionID}.jsonl
// where slug is the cwd with "/" and "." replaced by "-".
func ResolvePath(sessionID, cwd string) string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	slug := strings.ReplaceAll(cwd, "/", "-")
	slug = strings.ReplaceAll(slug, ".", "-")
	return filepath.Join(home, ".claude", "projects", slug, sessionID+".jsonl")
}

// ReadTail reads the last n entries from a JSONL file.
// Returns entries, the file offset after reading, and any error.
func ReadTail(path string, n int) ([]LogEntry, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, err
	}

	// Seek to last 1MB for performance
	const tailSize = 1024 * 1024
	if info.Size() > tailSize {
		if _, err := f.Seek(-tailSize, io.SeekEnd); err != nil {
			return nil, 0, err
		}
		// Skip partial first line
		reader := bufio.NewReader(f)
		if _, err := reader.ReadString('\n'); err != nil && err != io.EOF {
			return nil, 0, err
		}
	}

	entries := scanEntries(f)

	// Keep last n
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}

	offset := info.Size()
	return entries, offset, nil
}

// ReadFrom reads new entries from a JSONL file starting at the given offset.
// Returns new entries, the updated offset, and any error.
func ReadFrom(path string, offset int64) ([]LogEntry, int64, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, offset, err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return nil, offset, err
	}

	if info.Size() <= offset {
		return nil, offset, nil
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, err
	}

	entries := scanEntries(f)
	return entries, info.Size(), nil
}

func scanEntries(r io.Reader) []LogEntry {
	var entries []LogEntry
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 512*1024), 512*1024)

	for scanner.Scan() {
		if entry, ok := parseLine(scanner.Bytes()); ok {
			entries = append(entries, entry)
		}
	}
	return entries
}

// jsonLine is the minimal structure we need from each JSONL line.
type jsonLine struct {
	Type      string          `json:"type"`
	Timestamp string          `json:"timestamp"`
	Message   json.RawMessage `json:"message"`
}

type messageEnvelope struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`
}

type contentBlock struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Name    string `json:"name"`    // tool_use name
	Input   any    `json:"input"`   // tool_use input (ignored but present)
}

func parseLine(data []byte) (LogEntry, bool) {
	var line jsonLine
	if err := json.Unmarshal(data, &line); err != nil {
		return LogEntry{}, false
	}

	// Only process user and assistant message types
	if line.Type != "user" && line.Type != "assistant" {
		return LogEntry{}, false
	}

	ts, _ := time.Parse(time.RFC3339Nano, line.Timestamp)

	if line.Message == nil {
		return LogEntry{}, false
	}

	var msg messageEnvelope
	if err := json.Unmarshal(line.Message, &msg); err != nil {
		return LogEntry{}, false
	}

	// User message: content is typically a string
	if msg.Role == "user" {
		var contentStr string
		if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
			summary := cleanSummary(contentStr, 120)
			if summary == "" {
				return LogEntry{}, false
			}
			return LogEntry{
				Timestamp: ts,
				Type:      EntryUser,
				RawType:   "user",
				Summary:   summary,
			}, true
		}
		// Content might be an array (tool_result)
		var blocks []contentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err == nil {
			for _, b := range blocks {
				if b.Type == "tool_result" {
					return LogEntry{
						Timestamp: ts,
						Type:      EntryUser,
						RawType:   "tool_result",
						Summary:   "Tool result",
					}, true
				}
			}
		}
		return LogEntry{}, false
	}

	// Assistant message: content is an array of blocks
	if msg.Role == "assistant" {
		var blocks []contentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			return LogEntry{}, false
		}
		if len(blocks) == 0 {
			return LogEntry{}, false
		}

		// Use the first block to determine type
		block := blocks[0]
		switch block.Type {
		case "text":
			summary := cleanSummary(block.Text, 120)
			if summary == "" {
				return LogEntry{}, false
			}
			return LogEntry{
				Timestamp: ts,
				Type:      EntryAssistant,
				RawType:   "text",
				Summary:   summary,
			}, true
		case "tool_use":
			return LogEntry{
				Timestamp: ts,
				Type:      EntryAssistant,
				RawType:   "tool_use",
				Summary:   "Tool: " + block.Name,
			}, true
		case "thinking":
			return LogEntry{
				Timestamp: ts,
				Type:      EntryAssistant,
				RawType:   "thinking",
				Summary:   "[thinking]",
				HasThink:  true,
			}, true
		}
	}

	return LogEntry{}, false
}

func cleanSummary(s string, maxLen int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.TrimSpace(s)
	runes := []rune(s)
	if len(runes) > maxLen {
		return string(runes[:maxLen-1]) + "…"
	}
	return s
}
