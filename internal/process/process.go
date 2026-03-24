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
	CPU   string
	Mem   string
	Alive bool
}

// ps aux field layout constants.
const (
	psMinFields = 11 // minimum fields in a ps aux line
	psCmdIndex  = 10 // index where command string starts
	psPIDIndex  = 1
	psCPUIndex  = 2
	psMemIndex  = 3
)

// ListClaude returns process info for all running Claude Code CLI instances.
func ListClaude() (map[int]Info, error) {
	cmd := exec.CommandContext(context.Background(), "ps", "aux")
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
		if len(fields) < psMinFields {
			continue
		}

		cmdStr := strings.Join(fields[psCmdIndex:], " ")
		if !isClaudeCodeCLI(cmdStr) {
			continue
		}

		pid, err := strconv.Atoi(fields[psPIDIndex])
		if err != nil {
			continue
		}

		result[pid] = Info{
			PID:   pid,
			CPU:   fields[psCPUIndex],
			Mem:   fields[psMemIndex],
			Alive: true,
		}
	}

	return result, nil
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
