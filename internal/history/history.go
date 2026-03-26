package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Entry represents a single line from ~/.claude/history.jsonl.
type Entry struct {
	Display   string `json:"display"`
	Timestamp int64  `json:"timestamp"` // milliseconds since epoch
	Project   string `json:"project"`
	SessionID string `json:"sessionId"`
}

// LastActions returns the most recent action per session ID.
// It reads the tail of the history file for performance.
func LastActions() (result map[string]Entry, err error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolving home directory: %w", err)
	}
	return LastActionsFrom(home)
}

// LastActionsFrom returns the most recent action per session ID,
// reading from the history file under the given home directory.
func LastActionsFrom(homeDir string) (result map[string]Entry, err error) {
	path := filepath.Join(homeDir, ".claude", "history.jsonl")

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Entry), nil
		}
		return nil, fmt.Errorf("opening history file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	// Seek to last 512KB for performance
	info, err := f.Stat()
	if err != nil {
		return nil, fmt.Errorf("reading history file info: %w", err)
	}
	const tailSize = 512 * 1024
	if info.Size() > tailSize {
		if _, err := f.Seek(-tailSize, io.SeekEnd); err != nil {
			return nil, fmt.Errorf("seeking history file: %w", err)
		}
		// Skip partial first line
		reader := bufio.NewReader(f)
		if _, err := reader.ReadString('\n'); err != nil && err != io.EOF {
			return nil, fmt.Errorf("reading history file: %w", err)
		}
		return scanEntries(reader)
	}

	return scanEntries(bufio.NewReader(f))
}

func scanEntries(r *bufio.Reader) (map[string]Entry, error) {
	result := make(map[string]Entry)
	scanner := bufio.NewScanner(r)
	const scannerBufSize = 256 * 1024
	scanner.Buffer(make([]byte, scannerBufSize), scannerBufSize)

	for scanner.Scan() {
		var e Entry
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			continue
		}
		if e.SessionID == "" || e.Display == "" {
			continue
		}
		prev, exists := result[e.SessionID]
		if !exists || e.Timestamp > prev.Timestamp {
			result[e.SessionID] = e
		}
	}
	if err := scanner.Err(); err != nil {
		return result, fmt.Errorf("scanning history entries: %w", err)
	}
	return result, nil
}

