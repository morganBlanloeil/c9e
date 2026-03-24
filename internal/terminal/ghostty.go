package terminal

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// FocusByWorkdir finds a Ghostty terminal whose working directory matches
// the given path and focuses it (switching to its tab/window).
// Returns an error if Ghostty is not the current terminal or no match is found.
func FocusByWorkdir(cwd string) error {
	if !IsGhostty() {
		return fmt.Errorf("jump-to-terminal is only supported in Ghostty")
	}

	expanded := ExpandHome(cwd)
	cmd := BuildFocusCommand(expanded)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("AppleScript failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	result := strings.TrimSpace(string(out))
	if result == "no match" {
		return fmt.Errorf("no Ghostty terminal found for directory: %s", cwd)
	}

	return nil
}

// BuildFocusCommand creates the osascript command that finds and focuses
// a Ghostty terminal by working directory. The cwd should already be expanded.
func BuildFocusCommand(cwd string) *exec.Cmd {
	script := BuildFocusScript(cwd)
	return exec.Command("osascript", "-e", script)
}

// BuildFocusScript returns the AppleScript source that finds a Ghostty
// terminal whose working directory contains cwd and focuses it.
func BuildFocusScript(cwd string) string {
	return fmt.Sprintf(`tell application "Ghostty"
	set matches to every terminal whose working directory contains %q
	if (count of matches) > 0 then
		focus item 1 of matches
		return "ok"
	else
		return "no match"
	end if
end tell`, cwd)
}

// ExpandHome replaces a leading ~ with the user's home directory.
func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}

// IsGhostty checks if the current terminal is Ghostty.
func IsGhostty() bool {
	return os.Getenv("TERM_PROGRAM") == "ghostty"
}
