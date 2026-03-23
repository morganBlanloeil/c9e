package terminal

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// jumpGhostty focuses the Ghostty tab containing the given PID.
// It matches the tab by checking if the tab title contains the CWD basename.
// Falls back to just activating Ghostty if no match is found.
func jumpGhostty(pid int) error {
	cwd, err := getCWD(pid)
	if err != nil {
		return activateApp("ghostty")
	}

	base := filepath.Base(cwd)

	// Try exact match first, then normalized (dashes → spaces, case-insensitive)
	script := fmt.Sprintf(`
set targetBase to %q
set targetNorm to do shell script "echo " & quoted form of targetBase & " | sed 's/-/ /g' | tr '[:upper:]' '[:lower:]'"

tell application "System Events"
	tell process "ghostty"
		set w to first window
		set tg to first tab group of w
		set tabList to radio buttons of tg

		-- Pass 1: exact substring match on basename
		repeat with t in tabList
			if name of t contains targetBase then
				click t
				tell application "ghostty" to activate
				return
			end if
		end repeat

		-- Pass 2: normalized match (dashes→spaces, case-insensitive)
		repeat with t in tabList
			set tabNorm to do shell script "echo " & quoted form of (name of t) & " | tr '[:upper:]' '[:lower:]'"
			if tabNorm contains targetNorm then
				click t
				tell application "ghostty" to activate
				return
			end if
		end repeat
	end tell
end tell

-- No match found, just activate
tell application "ghostty" to activate
`, base)

	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("osascript: %w: %s", err, string(out))
	}
	return nil
}

// getCWD returns the working directory of a process using lsof.
func getCWD(pid int) (string, error) {
	out, err := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "cwd", "-Fn").Output()
	if err != nil {
		return "", fmt.Errorf("lsof cwd for PID %d: %w", pid, err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "n") {
			return line[1:], nil
		}
	}
	return "", fmt.Errorf("no cwd found for PID %d", pid)
}

// activateApp brings the named application to the foreground.
func activateApp(name string) error {
	script := fmt.Sprintf(`tell application %q to activate`, name)
	cmd := exec.Command("osascript", "-e", script)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("activate %s: %w: %s", name, err, string(out))
	}
	return nil
}
