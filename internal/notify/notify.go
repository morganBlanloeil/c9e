package notify

import (
	"fmt"
	"os/exec"
	"runtime"
)

// Send sends a macOS desktop notification using osascript.
func Send(title, message string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("notifications not supported on %s", runtime.GOOS)
	}

	script := fmt.Sprintf(`display notification %q with title %q sound name "Glass"`, message, title)
	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

// Available reports whether desktop notifications are supported on this platform.
func Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("osascript")
	return err == nil
}

// BuildCommand returns the exec.Cmd that would be used to send a notification.
// Exposed for testing purposes.
func BuildCommand(title, message string) *exec.Cmd {
	script := fmt.Sprintf(`display notification %q with title %q sound name "Glass"`, message, title)
	return exec.Command("osascript", "-e", script)
}
