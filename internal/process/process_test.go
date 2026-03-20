package process

import "testing"

func TestIsClaudeCodeCLI(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		want bool
	}{
		{name: "claude CLI node process", cmd: "node /usr/local/bin/claude --session abc", want: true},
		{name: "claude with path", cmd: "/home/user/.local/bin/claude code review", want: true},
		{name: "Claude.app helper", cmd: "/Applications/Claude.app/Contents/MacOS/Claude Helper (GPU)", want: false},
		{name: "Claude.app main", cmd: "/Applications/Claude.app/Contents/MacOS/Claude", want: false},
		{name: "unrelated process", cmd: "/usr/bin/vim main.go", want: false},
		{name: "case insensitive match", cmd: "CLAUDE --version", want: true},
		{name: "claude in path", cmd: "/Users/me/.claude/bin/c9e", want: true},
		{name: "empty command", cmd: "", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isClaudeCodeCLI(tt.cmd)
			if got != tt.want {
				t.Errorf("isClaudeCodeCLI(%q) = %v, want %v", tt.cmd, got, tt.want)
			}
		})
	}
}
