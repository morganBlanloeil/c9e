package process

import (
	"bufio"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Info holds live process stats.
type Info struct {
	PID   int
	CPU   string
	Mem   string
	Alive bool
}

// ListClaude returns process info for all running Claude Code CLI instances.
func ListClaude() (map[int]Info, error) {
	cmd := exec.Command("ps", "aux")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps aux failed: %w", err)
	}

	result := make(map[int]Info)
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	// Skip header
	if scanner.Scan() {
		_ = scanner.Text()
	}

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 11 {
			continue
		}

		cmdStr := strings.Join(fields[10:], " ")
		if !isClaudeCodeCLI(cmdStr) {
			continue
		}

		pid, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}

		result[pid] = Info{
			PID:   pid,
			CPU:   fields[2],
			Mem:   fields[3],
			Alive: true,
		}
	}

	return result, nil
}

// Kill sends SIGTERM to a process.
func Kill(pid int) error {
	cmd := exec.Command("kill", strconv.Itoa(pid))
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to kill PID %d: %w", pid, err)
	}
	return nil
}

// IsAlive checks if a specific PID is a running process.
func IsAlive(pid int) bool {
	cmd := exec.Command("kill", "-0", strconv.Itoa(pid))
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
