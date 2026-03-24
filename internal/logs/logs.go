package logs

import (
	"bufio"
	"encoding/json"
	"fmt"
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

const (
	roleUser         = "user"
	roleAssistant    = "assistant"
	scannerBufSize   = 512 * 1024
	readTailSize     = 1024 * 1024
	lastRoleTailSize = 64 * 1024
	maxSummaryLen    = 120
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
func ReadTail(path string, n int) (entries []LogEntry, offset int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, 0, fmt.Errorf("opening log file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing log file: %w", cerr)
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return nil, 0, fmt.Errorf("reading log file info: %w", err)
	}

	// Seek to last 1MB for performance
	if info.Size() > readTailSize {
		if _, err := f.Seek(-readTailSize, io.SeekEnd); err != nil {
			return nil, 0, fmt.Errorf("seeking log file: %w", err)
		}
		// Skip partial first line
		reader := bufio.NewReader(f)
		if _, err := reader.ReadString('\n'); err != nil && err != io.EOF {
			return nil, 0, fmt.Errorf("reading log file: %w", err)
		}
	}

	entries = scanEntries(f)

	// Keep last n
	if len(entries) > n {
		entries = entries[len(entries)-n:]
	}

	offset = info.Size()
	return entries, offset, nil
}

// ReadFrom reads new entries from a JSONL file starting at the given offset.
// Returns new entries, the updated offset, and any error.
func ReadFrom(path string, offset int64) (entries []LogEntry, newOffset int64, err error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, offset, fmt.Errorf("opening log file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing log file: %w", cerr)
		}
	}()

	info, err := f.Stat()
	if err != nil {
		return nil, offset, fmt.Errorf("reading log file info: %w", err)
	}

	if info.Size() <= offset {
		return nil, offset, nil
	}

	if _, err := f.Seek(offset, io.SeekStart); err != nil {
		return nil, offset, fmt.Errorf("seeking log file: %w", err)
	}

	entries = scanEntries(f)
	return entries, info.Size(), nil
}

// LastRole returns the role ("user" or "assistant") of the last message in a
// session JSONL file. It reads only the tail of the file for performance.
// Returns empty string if the file cannot be read or has no valid entries.
func LastRole(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer func() { _ = f.Close() }()

	info, err := f.Stat()
	if err != nil {
		return ""
	}

	// Read last 64KB — enough to find the last message
	if info.Size() > lastRoleTailSize {
		if _, err := f.Seek(-lastRoleTailSize, io.SeekEnd); err != nil {
			return ""
		}
		// Skip partial first line
		reader := bufio.NewReader(f)
		_, _ = reader.ReadString('\n')
		return scanLastRole(reader)
	}

	return scanLastRole(bufio.NewReader(f))
}

// scanLastRole reads JSONL lines and returns the "type" field of the last
// user or assistant message.
func scanLastRole(r io.Reader) string {
	var lastRole string
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, scannerBufSize), scannerBufSize)
	for scanner.Scan() {
		var line jsonLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type == roleUser || line.Type == roleAssistant {
			lastRole = line.Type
		}
	}
	return lastRole
}

func scanEntries(r io.Reader) []LogEntry {
	var entries []LogEntry
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, scannerBufSize), scannerBufSize)

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
	Type  string `json:"type"`
	Text  string `json:"text"`
	Name  string `json:"name"`  // tool_use name
	Input any    `json:"input"` // tool_use input (ignored but present)
}

func parseLine(data []byte) (LogEntry, bool) {
	var line jsonLine
	if err := json.Unmarshal(data, &line); err != nil {
		return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
	}

	// Only process user and assistant message types
	if line.Type != roleUser && line.Type != roleAssistant {
		return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
	}

	ts, _ := time.Parse(time.RFC3339Nano, line.Timestamp)

	if line.Message == nil {
		return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
	}

	var msg messageEnvelope
	if err := json.Unmarshal(line.Message, &msg); err != nil {
		return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
	}

	// User message: content is typically a string
	if msg.Role == roleUser {
		var contentStr string
		if err := json.Unmarshal(msg.Content, &contentStr); err == nil {
			summary := cleanSummary(contentStr, maxSummaryLen)
			if summary == "" {
				return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
			}
			return LogEntry{
				Timestamp: ts,
				Type:      EntryUser,
				RawType:   roleUser,
				Summary:   summary,
				HasThink:  false,
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
						HasThink:  false,
					}, true
				}
			}
		}
		return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
	}

	// Assistant message: content is an array of blocks
	if msg.Role == roleAssistant {
		var blocks []contentBlock
		if err := json.Unmarshal(msg.Content, &blocks); err != nil {
			return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
		}
		if len(blocks) == 0 {
			return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
		}

		// Use the first block to determine type
		block := blocks[0]
		switch block.Type {
		case "text":
			summary := cleanSummary(block.Text, maxSummaryLen)
			if summary == "" {
				return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
			}
			return LogEntry{
				Timestamp: ts,
				Type:      EntryAssistant,
				RawType:   "text",
				Summary:   summary,
				HasThink:  false,
			}, true
		case "tool_use":
			return LogEntry{
				Timestamp: ts,
				Type:      EntryAssistant,
				RawType:   "tool_use",
				Summary:   "Tool: " + block.Name,
				HasThink:  false,
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

	return LogEntry{Timestamp: time.Time{}, Type: 0, RawType: "", Summary: "", HasThink: false}, false
}

// CountTurns counts user message entries in a session JSONL file.
// It reads only the necessary structure to identify user messages.
func CountTurns(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, scannerBufSize), scannerBufSize)
	for scanner.Scan() {
		var line jsonLine
		if err := json.Unmarshal(scanner.Bytes(), &line); err != nil {
			continue
		}
		if line.Type == roleUser {
			count++
		}
	}
	return count
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
