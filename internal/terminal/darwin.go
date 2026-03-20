package terminal

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// getTTY returns the tty device name for the given PID (e.g. "ttys003").
func getTTY(pid int) (string, error) {
	out, err := exec.Command("ps", "-o", "tty=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return "", fmt.Errorf("ps tty lookup for PID %d: %w", pid, err)
	}
	tty := strings.TrimSpace(string(out))
	if tty == "" || tty == "??" {
		return "", fmt.Errorf("no tty for PID %d", pid)
	}
	return tty, nil
}

// jumpDarwin uses AppleScript to focus the iTerm2 or Terminal.app tab with the given PID's tty.
func jumpDarwin(pid int, be backend) error {
	tty, err := getTTY(pid)
	if err != nil {
		return err
	}

	var script string
	switch be {
	case backendITerm2:
		script = fmt.Sprintf(`
tell application "iTerm2"
	repeat with w in windows
		repeat with t in tabs of w
			repeat with s in sessions of t
				if tty of s contains %q then
					select t
					select w
					return
				end if
			end repeat
		end repeat
	end repeat
end tell`, tty)
	case backendTerminalApp:
		script = fmt.Sprintf(`
tell application "Terminal"
	repeat with w in windows
		repeat with t in tabs of w
			if tty of t contains %q then
				set selected tab of w to t
				set frontmost of w to true
				return
			end if
		end repeat
	end repeat
end tell`, tty)
	default:
		return fmt.Errorf("unsupported darwin backend")
	}

	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, string(out))
	}
	return nil
}
