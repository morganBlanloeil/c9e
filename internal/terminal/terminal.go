package terminal

import (
	"os"
	"os/exec"
	"sync"
)

type backend int

const (
	backendNone backend = iota
	backendTmux
	backendITerm2
	backendTerminalApp
)

var (
	detectedBackend backend
	detectOnce      sync.Once
)

func detect() backend {
	// 1. tmux
	if os.Getenv("TMUX") != "" {
		if _, err := exec.LookPath("tmux"); err == nil {
			return backendTmux
		}
	}

	termProgram := os.Getenv("TERM_PROGRAM")
	hasOsascript := false
	if _, err := exec.LookPath("osascript"); err == nil {
		hasOsascript = true
	}

	// 2. iTerm2
	if termProgram == "iTerm.app" && hasOsascript {
		return backendITerm2
	}

	// 3. Terminal.app
	if termProgram == "Apple_Terminal" && hasOsascript {
		return backendTerminalApp
	}

	return backendNone
}

// Available returns true if the current terminal supports jump-to-pane.
func Available() bool {
	detectOnce.Do(func() {
		detectedBackend = detect()
	})
	return detectedBackend != backendNone
}

// JumpTo switches focus to the terminal pane/tab running the given PID.
func JumpTo(pid int) error {
	detectOnce.Do(func() {
		detectedBackend = detect()
	})

	switch detectedBackend {
	case backendTmux:
		return jumpTmux(pid)
	case backendITerm2:
		return jumpDarwin(pid, backendITerm2)
	case backendTerminalApp:
		return jumpDarwin(pid, backendTerminalApp)
	default:
		return nil
	}
}
