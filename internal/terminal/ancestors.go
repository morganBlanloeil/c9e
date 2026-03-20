package terminal

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// Ancestors returns the chain of parent PIDs from the given PID up to PID 1.
func Ancestors(pid int) ([]int, error) {
	var chain []int
	current := pid
	for current > 1 {
		out, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(current)).Output()
		if err != nil {
			return chain, fmt.Errorf("ps lookup for PID %d: %w", current, err)
		}
		ppid, err := strconv.Atoi(strings.TrimSpace(string(out)))
		if err != nil {
			return chain, fmt.Errorf("parse ppid for PID %d: %w", current, err)
		}
		chain = append(chain, ppid)
		current = ppid
	}
	return chain, nil
}
