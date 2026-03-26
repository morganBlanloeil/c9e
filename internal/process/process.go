package process

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Info holds live process stats.
type Info struct {
	PID   int
	PPID  int
	CPU   string
	Mem   string
	Alive bool
}

// ps -eo field layout constants.
const (
	psMinFields = 5 // minimum fields in a ps -eo line
	psCmdIndex  = 4 // index where command string starts
	psPIDIndex  = 0
	psPPIDIndex = 1
	psCPUIndex  = 2
	psMemIndex  = 3
)

// ProcessTree holds the full PID→PPID mapping for all system processes.
// It is used to walk the ancestry chain when detecting sub-agent descendants.
type ProcessTree map[int]int // pid → ppid

// ListClaude returns process info for all running Claude Code CLI instances,
// along with the full process tree needed to detect descendant relationships.
func ListClaude() (map[int]Info, error) {
	claudeProcs, tree, err := listClaudeWithTree()
	if err != nil {
		return nil, err
	}
	// Store tree for later use by HasClaudeDescendants.
	lastTree = tree
	return claudeProcs, nil
}

// lastTree caches the process tree from the most recent ListClaude call.
var lastTree ProcessTree

func listClaudeWithTree() (map[int]Info, ProcessTree, error) {
	cmd := exec.CommandContext(context.Background(), "ps", "-eo", "pid,ppid,%cpu,%mem,args")
	out, err := cmd.Output()
	if err != nil {
		return nil, nil, fmt.Errorf("ps failed: %w", err)
	}

	claudeProcs := make(map[int]Info)
	tree := make(ProcessTree)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	// Skip header
	if scanner.Scan() {
		_ = scanner.Text()
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < psMinFields {
			continue
		}

		pid, err := strconv.Atoi(fields[psPIDIndex])
		if err != nil {
			continue
		}

		ppid, err := strconv.Atoi(fields[psPPIDIndex])
		if err != nil {
			continue
		}

		// Record every process in the tree for ancestry walks.
		tree[pid] = ppid

		cmdStr := strings.Join(fields[psCmdIndex:], " ")
		if !isClaudeCodeCLI(cmdStr) {
			continue
		}

		claudeProcs[pid] = Info{
			PID:   pid,
			PPID:  ppid,
			CPU:   fields[psCPUIndex],
			Mem:   fields[psMemIndex],
			Alive: true,
		}
	}

	return claudeProcs, tree, nil
}

// HasClaudeChildren reports whether any Claude Code process is a descendant
// of pid, walking the full process tree (not just direct children).
func HasClaudeChildren(pid int, procs map[int]Info) bool {
	for cpid := range procs {
		if cpid == pid {
			continue
		}
		if isDescendantOf(cpid, pid, lastTree) {
			return true
		}
	}
	return false
}

// isDescendantOf walks the process tree upward from child to see if it
// reaches ancestor. Guards against cycles with a visited set.
func isDescendantOf(child, ancestor int, tree ProcessTree) bool {
	visited := make(map[int]bool)
	current := child
	for {
		ppid, ok := tree[current]
		if !ok || ppid == 0 || ppid == current {
			return false
		}
		if ppid == ancestor {
			return true
		}
		if visited[ppid] {
			return false
		}
		visited[current] = true
		current = ppid
	}
}

// CountClaudeChildren returns the number of Claude Code processes that are
// children of the given pid. Sub-agents spawned by a session appear as
// separate claude processes whose PPID is the session's PID.
func CountClaudeChildren(pid int, procs map[int]Info) int {
	count := 0
	for _, p := range procs {
		if p.PPID == pid {
			count++
		}
	}
	return count
}

// Kill sends SIGTERM to a process.
func Kill(pid int) error {
	cmd := exec.CommandContext(context.Background(), "kill", strconv.Itoa(pid))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill PID %d: %w", pid, err)
	}
	return nil
}

// IsAlive checks if a specific PID is a running process.
func IsAlive(pid int) bool {
	cmd := exec.CommandContext(context.Background(), "kill", "-0", strconv.Itoa(pid))
	return cmd.Run() == nil
}

func isClaudeCodeCLI(cmd string) bool {
	lower := strings.ToLower(cmd)
	if !strings.Contains(lower, "claude") {
		return false
	}
	// Match "claude" CLI processes, not Claude.app helpers
	if strings.Contains(cmd, "Claude.app") || strings.Contains(cmd, "Claude Helper") {
		return false
	}
	return true
}
