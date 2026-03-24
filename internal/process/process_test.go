package process

import "testing"

func TestIsDescendantOf(t *testing.T) {
	// Process tree:
	//   1 (init)
	//   └── 100 (claude main)
	//       └── 200 (node)
	//           └── 300 (claude sub-agent)
	//   └── 400 (unrelated)
	tree := ProcessTree{
		100: 1,
		200: 100,
		300: 200,
		400: 1,
	}

	tests := []struct {
		name     string
		child    int
		ancestor int
		want     bool
	}{
		{"direct child", 200, 100, true},
		{"grandchild", 300, 100, true},
		{"not a descendant", 400, 100, false},
		{"self", 100, 100, false},
		{"unknown child", 999, 100, false},
		{"unknown ancestor", 300, 999, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDescendantOf(tt.child, tt.ancestor, tree)
			if got != tt.want {
				t.Errorf("isDescendantOf(%d, %d) = %v, want %v", tt.child, tt.ancestor, got, tt.want)
			}
		})
	}
}

func TestHasClaudeChildren_DeepTree(t *testing.T) {
	// Simulate: claude(100) -> node(200) -> claude sub-agent(300)
	// Only claude processes are in procs, but full tree is in lastTree.
	tree := ProcessTree{
		100: 1,
		200: 100,
		300: 200,
	}
	lastTree = tree

	procs := map[int]Info{
		100: {PID: 100, PPID: 1},
		300: {PID: 300, PPID: 200}, // sub-agent, NOT direct child of 100
	}

	if !HasClaudeChildren(100, procs) {
		t.Error("HasClaudeChildren(100) = false, want true (sub-agent 300 is a descendant)")
	}
	if HasClaudeChildren(300, procs) {
		t.Error("HasClaudeChildren(300) = true, want false (no descendants)")
	}
}

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
