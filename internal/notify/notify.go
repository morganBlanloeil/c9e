package notify

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"runtime"
)

var ErrUnsupportedPlatform = errors.New("notifications not supported")

func Send(title, message string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%w: %s", ErrUnsupportedPlatform, runtime.GOOS)
	}

	script := fmt.Sprintf(`display notification %q with title %q sound name "Glass"`, message, title)
	cmd := exec.CommandContext(context.Background(), "osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("sending notification: %w", err)
	}
	return nil
}

func Available() bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	_, err := exec.LookPath("osascript")
	return err == nil
}

func BuildCommand(title, message string) *exec.Cmd {
	script := fmt.Sprintf(`display notification %q with title %q sound name "Glass"`, message, title)
	return exec.CommandContext(context.Background(), "osascript", "-e", script)
}
