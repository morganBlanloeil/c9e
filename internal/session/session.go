package session

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// Session represents a Claude Code session from ~/.claude/sessions/*.json.
type Session struct {
	PID       int    `json:"pid"`
	SessionID string `json:"sessionId"`
	Cwd       string `json:"cwd"`
	StartedAt int64  `json:"startedAt"` // milliseconds since epoch
}

// ShortCwd returns the cwd with $HOME replaced by ~.
func (s Session) ShortCwd() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return s.Cwd
	}
	if strings.HasPrefix(s.Cwd, home) {
		return "~" + s.Cwd[len(home):]
	}
	return s.Cwd
}

// LoadAll reads all session files from the sessions directory.
func LoadAll() ([]Session, error) {
	dir, err := sessionsDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []Session
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		var s Session
		if err := json.Unmarshal(data, &s); err != nil {
			continue
		}
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func sessionsDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".claude", "sessions"), nil
}
