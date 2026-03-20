package terminal

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// jumpTmux finds the tmux pane running the given PID and switches to it.
func jumpTmux(pid int) error {
	// List all panes with their IDs and root shell PIDs.
	out, err := exec.Command("tmux", "list-panes", "-a", "-F", "#{pane_id} #{pane_pid}").Output()
	if err != nil {
		return fmt.Errorf("tmux list-panes: %w", err)
	}

	// Build a map of shell PID → pane ID.
	panePIDs := make(map[int]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		parts := strings.Fields(line)
		if len(parts) != 2 {
			continue
		}
		shellPID, err := strconv.Atoi(parts[1])
		if err != nil {
			continue
		}
		panePIDs[shellPID] = parts[0]
	}

	// Walk ancestors of the target PID to find a matching pane.
	ancestors, err := Ancestors(pid)
	if err != nil {
		return fmt.Errorf("ancestor lookup: %w", err)
	}

	// Also check the PID itself.
	candidates := append([]int{pid}, ancestors...)
	for _, cpid := range candidates {
		if paneID, ok := panePIDs[cpid]; ok {
			if err := exec.Command("tmux", "select-window", "-t", paneID).Run(); err != nil {
				return fmt.Errorf("tmux select-window: %w", err)
			}
			if err := exec.Command("tmux", "select-pane", "-t", paneID).Run(); err != nil {
				return fmt.Errorf("tmux select-pane: %w", err)
			}
			return nil
		}
	}

	return fmt.Errorf("no tmux pane found for PID %d", pid)
}
