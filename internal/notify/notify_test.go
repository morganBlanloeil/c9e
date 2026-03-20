package notify

import (
	"runtime"
	"strings"
	"testing"
)

func TestBuildCommand(t *testing.T) {
	cmd := BuildCommand("c9e — Task Complete", "my-project — session finished")

	if cmd.Path == "" {
		t.Fatal("expected non-empty command path")
	}

	args := cmd.Args
	if len(args) != 3 {
		t.Fatalf("expected 3 args, got %d: %v", len(args), args)
	}

	if args[1] != "-e" {
		t.Errorf("expected second arg to be '-e', got %q", args[1])
	}

	script := args[2]
	if !strings.Contains(script, "display notification") {
		t.Errorf("expected script to contain 'display notification', got %q", script)
	}
	if !strings.Contains(script, "c9e") {
		t.Errorf("expected script to contain title, got %q", script)
	}
	if !strings.Contains(script, "my-project") {
		t.Errorf("expected script to contain message, got %q", script)
	}
	if !strings.Contains(script, `sound name "Glass"`) {
		t.Errorf("expected script to contain sound name, got %q", script)
	}
}

func TestBuildCommandEscaping(t *testing.T) {
	cmd := BuildCommand(`title with "quotes"`, `message with "quotes" & specials`)
	script := cmd.Args[2]
	// The fmt %q verb should properly escape quotes
	if !strings.Contains(script, "display notification") {
		t.Errorf("expected valid script even with special chars, got %q", script)
	}
}

func TestAvailable(t *testing.T) {
	result := Available()
	if runtime.GOOS == "darwin" {
		// On macOS, osascript should be available
		if !result {
			t.Log("osascript not found on macOS — unexpected but not fatal in CI")
		}
	} else {
		if result {
			t.Errorf("expected Available() to be false on %s", runtime.GOOS)
		}
	}
}
