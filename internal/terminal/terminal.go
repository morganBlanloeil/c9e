package terminal

import (
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

type backend int

const (
	backendNone backend = iota
	backendTmux
	backendITerm2
	backendTerminalApp
	backendGhostty
)

// Available returns true if jump-to-terminal is potentially possible.
func Available() bool {
	// tmux
	if os.Getenv("TMUX") != "" {
		if _, err := exec.LookPath("tmux"); err == nil {
			return true
		}
	}

	// macOS: any of the supported terminals can be detected per-PID
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("osascript"); err == nil {
			return true
		}
	}

	return false
}

// JumpTo detects the terminal hosting the given PID and switches focus to it.
func JumpTo(pid int) error {
	be := detectForPID(pid)

	switch be {
	case backendTmux:
		return jumpTmux(pid)
	case backendITerm2:
		return jumpDarwin(pid, backendITerm2)
	case backendTerminalApp:
		return jumpDarwin(pid, backendTerminalApp)
	case backendGhostty:
		return jumpGhostty(pid)
	default:
		return nil
	}
}

// detectForPID walks the ancestor chain of a PID to find which terminal hosts it.
func detectForPID(pid int) backend {
	ancestors, err := Ancestors(pid)
	if err != nil {
		return backendNone
	}

	candidates := append([]int{pid}, ancestors...)
	for _, cpid := range candidates {
		comm := getComm(cpid)
		commLower := strings.ToLower(comm)
		switch {
		case strings.Contains(commLower, "ghostty"):
			return backendGhostty
		case strings.Contains(commLower, "iterm"):
			return backendITerm2
		case commLower == "terminal" || strings.HasSuffix(commLower, "/terminal"):
			return backendTerminalApp
		case strings.Contains(commLower, "tmux"):
			return backendTmux
		}
	}

	return backendNone
}

// getComm returns the command name for a PID.
func getComm(pid int) string {
	out, err := exec.Command("ps", "-o", "comm=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
