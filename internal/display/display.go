package display

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// Colors
const (
	Red    = "\033[0;31m"
	Green  = "\033[0;32m"
	Yellow = "\033[1;33m"
	Cyan   = "\033[0;36m"
	Dim    = "\033[2m"
	Bold   = "\033[1m"
	Reset  = "\033[0m"
)

// Status represents the state of a Claude Code instance.
type Status string

const (
	StatusActive Status = "ACTIVE"
	StatusIdle   Status = "IDLE"
	StatusDead   Status = "DEAD"
)

// IdleThreshold is the duration after which a session is considered idle.
const IdleThreshold = 5 * time.Minute

// Row represents a single dashboard row.
type Row struct {
	PID        int    `json:"pid"`
	SessionID  string `json:"session_id"`
	Status     Status `json:"status"`
	CPU        string `json:"cpu"`
	Mem        string `json:"mem"`
	Cwd        string `json:"cwd"`
	UptimeSec  int64  `json:"uptime_s"`
	IdleSec    int64  `json:"idle_s"`
	LastAction string `json:"last_action"`
	Alive      bool   `json:"alive"`
}

// RenderTable prints the dashboard table to stdout.
func RenderTable(rows []Row) {
	activeCount := 0
	idleCount := 0
	deadCount := 0
	for _, r := range rows {
		switch r.Status {
		case StatusActive:
			activeCount++
		case StatusIdle:
			idleCount++
		case StatusDead:
			deadCount++
		}
	}

	fmt.Println()
	fmt.Printf("%s%s  Claude Code Dashboard%s  %s%s%s\n", Bold, Cyan, Reset, Dim, time.Now().Format("15:04:05"), Reset)
	printSep()

	fmt.Printf("  %s● %d running%s", Green, activeCount+idleCount, Reset)
	if idleCount > 0 {
		fmt.Printf("  %s◐ %d idle%s", Yellow, idleCount, Reset)
	}
	if deadCount > 0 {
		fmt.Printf("  %s○ %d dead%s", Red, deadCount, Reset)
	}
	fmt.Println()
	printSep()

	fmt.Printf("  %s%-6s  %-8s  %5s  %5s  %-10s  %-9s  %-40s  %s%s\n",
		Dim, "PID", "STATUS", "CPU%", "MEM%", "UPTIME", "IDLE", "DIRECTORY", "LAST ACTION", Reset)
	printSep()

	for _, r := range rows {
		statusColor := colorFor(r.Status)
		icon := iconFor(r.Status)

		uptime := formatDuration(r.UptimeSec)
		idle := formatIdle(r.IdleSec)
		cwd := truncate(r.Cwd, 40)
		action := cleanAction(truncate(r.LastAction, 50))

		fmt.Printf("  %s%s%s %s%-6d%s  %s%-8s%s  %5s  %5s  %-10s  %-9s  %s%-40s%s  %s\n",
			statusColor, icon, Reset,
			"", r.PID, "",
			statusColor, r.Status, Reset,
			r.CPU, r.Mem,
			uptime, idle,
			Dim, cwd, Reset,
			action,
		)
	}

	printSep()
	fmt.Println()
}

// RenderJSON prints the dashboard data as JSON.
func RenderJSON(rows []Row) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(rows)
}

func printSep() {
	fmt.Printf("%s  %s%s\n", Dim, strings.Repeat("─", 97), Reset)
}

func colorFor(s Status) string {
	switch s {
	case StatusActive:
		return Green
	case StatusIdle:
		return Yellow
	case StatusDead:
		return Red
	default:
		return Reset
	}
}

func iconFor(s Status) string {
	switch s {
	case StatusActive:
		return "●"
	case StatusIdle:
		return "◐"
	case StatusDead:
		return "○"
	default:
		return "?"
	}
}

func formatDuration(seconds int64) string {
	if seconds < 0 {
		return "—"
	}
	d := seconds / 86400
	h := (seconds % 86400) / 3600
	m := (seconds % 3600) / 60
	if d > 0 {
		return fmt.Sprintf("%dd %dh", d, h)
	}
	if h > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

func formatIdle(seconds int64) string {
	if seconds < 0 {
		return "—"
	}
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh %dm", seconds/3600, (seconds%3600)/60)
	}
	return fmt.Sprintf("%dd", seconds/86400)
}

func truncate(s string, max int) string {
	if len(s) > max {
		return s[:max-1] + "…"
	}
	return s
}

func cleanAction(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
