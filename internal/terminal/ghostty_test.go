package terminal

import (
	"os"
	"strings"
	"testing"
)

func TestBuildFocusScript(t *testing.T) {
	script := BuildFocusScript("/Users/alice/dev/myproject")

	if !strings.Contains(script, `tell application "Ghostty"`) {
		t.Error("expected script to target Ghostty application")
	}
	if !strings.Contains(script, "working directory contains") {
		t.Error("expected script to match on working directory")
	}
	if !strings.Contains(script, "/Users/alice/dev/myproject") {
		t.Error("expected script to contain the provided cwd")
	}
	if !strings.Contains(script, "focus item 1") {
		t.Error("expected script to focus the matched terminal")
	}
	if !strings.Contains(script, `return "no match"`) {
		t.Error("expected script to handle no-match case")
	}
}

func TestBuildFocusScriptEscaping(t *testing.T) {
	// Paths with special characters should be properly quoted by %q
	script := BuildFocusScript(`/Users/alice/my "project"`)

	if !strings.Contains(script, "working directory contains") {
		t.Error("expected valid script even with special chars in path")
	}
	// %q escapes double quotes, so the script should still be valid AppleScript
	if !strings.Contains(script, `my \"project\"`) {
		t.Errorf("expected escaped quotes in script, got:\n%s", script)
	}
}

func TestBuildFocusCommand(t *testing.T) {
	cmd := BuildFocusCommand("/Users/alice/dev/myproject")

	if cmd.Path == "" {
		t.Fatal("expected non-empty command path")
	}

	args := cmd.Args
	if len(args) != 3 {
		t.Fatalf("expected 3 args (osascript -e <script>), got %d: %v", len(args), args)
	}
	if args[1] != "-e" {
		t.Errorf("expected second arg to be '-e', got %q", args[1])
	}
	if !strings.Contains(args[2], "Ghostty") {
		t.Error("expected script arg to mention Ghostty")
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home directory")
	}

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"tilde prefix", "~/dev/project", home + "/dev/project"},
		{"tilde only", "~", home},
		{"absolute path", "/usr/local/bin", "/usr/local/bin"},
		{"relative path", "dev/project", "dev/project"},
		{"empty string", "", ""},
		{"tilde in middle", "/home/~user", "/home/~user"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandHome(tt.input)
			if result != tt.expected {
				t.Errorf("ExpandHome(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsGhostty(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "ghostty")
	if !IsGhostty() {
		t.Error("expected IsGhostty() = true when TERM_PROGRAM=ghostty")
	}

	t.Setenv("TERM_PROGRAM", "iterm2")
	if IsGhostty() {
		t.Error("expected IsGhostty() = false when TERM_PROGRAM=iterm2")
	}

	t.Setenv("TERM_PROGRAM", "")
	if IsGhostty() {
		t.Error("expected IsGhostty() = false when TERM_PROGRAM is empty")
	}
}

func TestFocusByWorkdirNotGhostty(t *testing.T) {
	t.Setenv("TERM_PROGRAM", "iterm2")
	err := FocusByWorkdir("/some/path")
	if err == nil {
		t.Fatal("expected error when not running in Ghostty")
	}
	if !strings.Contains(err.Error(), "only supported in Ghostty") {
		t.Errorf("unexpected error message: %v", err)
	}
}
