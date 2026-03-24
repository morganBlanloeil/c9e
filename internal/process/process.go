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

// ListClaude returns process info for all running Claude Code CLI instances.
// It uses "ps -eo pid,ppid,%cpu,%mem,args" to capture parent PIDs.
func ListClaude() (map[int]Info, error) {
	cmd := exec.CommandContext(context.Background(), "ps", "-eo", "pid,ppid,%cpu,%mem,args")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ps failed: %w", err)
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

		ppid, err := strconv.Atoi(fields[psPPIDIndex])
		if err != nil {
			continue
		}

		result[pid] = Info{
			PID:   pid,
			PPID:  ppid,
			CPU:   fields[psCPUIndex],
			Mem:   fields[psMemIndex],
			Alive: true,
		}
	}

	return result, nil
}

// HasClaudeChildren reports whether any Claude Code process has pid as its parent.
func HasClaudeChildren(pid int, procs map[int]Info) bool {
	for _, p := range procs {
		if p.PPID == pid {
			return true
		}
	}
	return false
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
