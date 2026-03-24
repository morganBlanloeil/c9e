package terminal

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrNotGhostty  = errors.New("jump-to-terminal is only supported in Ghostty")
	ErrNoTermFound = errors.New("no Ghostty terminal found for directory")
)

func FocusByWorkdir(cwd string) error {
	if !IsGhostty() {
		return ErrNotGhostty
	}

	expanded := ExpandHome(cwd)
	cmd := BuildFocusCommand(expanded)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("AppleScript failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	result := strings.TrimSpace(string(out))
	if result == "no match" {
		return fmt.Errorf("%w: %s", ErrNoTermFound, cwd)
	}

	return nil
}

func BuildFocusCommand(cwd string) *exec.Cmd {
	script := BuildFocusScript(cwd)
	return exec.CommandContext(context.Background(), "osascript", "-e", script)
}

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

func ExpandHome(path string) string {
	if strings.HasPrefix(path, "~") {
		home, err := os.UserHomeDir()
		if err == nil {
			return home + path[1:]
		}
	}
	return path
}

func IsGhostty() bool {
	return os.Getenv("TERM_PROGRAM") == "ghostty"
}
