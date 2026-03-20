package history

import (
	"bufio"
	"encoding/json"
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
func LastActions() (map[string]Entry, error) {
	path, err := historyPath()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]Entry), nil
		}
		return nil, err
	}
	defer f.Close()

	// Seek to last 512KB for performance
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	const tailSize = 512 * 1024
	if info.Size() > tailSize {
		if _, err := f.Seek(-tailSize, io.SeekEnd); err != nil {
			return nil, err
		}
		// Skip partial first line
		reader := bufio.NewReader(f)
		if _, err := reader.ReadString('\n'); err != nil && err != io.EOF {
			return nil, err
		}
		return scanEntries(reader)
	}

	return scanEntries(bufio.NewReader(f))
}

func scanEntries(r *bufio.Reader) (map[string]Entry, error) {
	result := make(map[string]Entry)
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 256*1024), 256*1024)

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
	return result, scanner.Err()
}

func historyPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "history.jsonl"), nil
}
