package display

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ANSI colors for static (non-TUI) table output.
const (
	ansiRed    = "\033[0;31m"
	ansiGreen  = "\033[0;32m"
	ansiYellow = "\033[1;33m"
	ansiCyan   = "\033[0;36m"
	ansiDim    = "\033[2m"
	ansiBold   = "\033[1m"
	ansiReset  = "\033[0m"
)

const (
	emDash       = "—"
	msPerSecond  = 1000
	secsPerDay   = 86400
	secsPerHour  = 3600
	secsPerMin   = 60
	sepWidth     = 107
	maxCwdLen    = 40
	maxActionLen = 50
)

// CleanAction removes newlines and carriage returns from a string.
func CleanAction(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// Status represents the state of a Claude Code instance.
type Status string

const (
	StatusActive  Status = "ACTIVE"
	StatusWaiting Status = "WAITING"
	StatusIdle    Status = "IDLE"
	StatusDead    Status = "DEAD"
)

// IdleThreshold is the duration after which a session is considered idle.
const IdleThreshold = 5 * time.Minute

// Row represents a single dashboard row.
type Row struct {
	PID           int     `json:"pid"`
	SessionID     string  `json:"session_id"`
	FullSessionID string  `json:"full_session_id"`
	Status        Status  `json:"status"`
	CPU           string  `json:"cpu"`
	Mem           string  `json:"mem"`
	Cwd           string  `json:"cwd"`
	RawCwd        string  `json:"-"` // unexpanded cwd for log path resolution
	UptimeSec     int64   `json:"uptime_s"`
	IdleSec       int64   `json:"idle_s"`
	LastAction    string  `json:"last_action"`
	Alive         bool    `json:"alive"`
	LogPath       string  `json:"log_path,omitempty"`
	Turns         int     `json:"turns"`      // conversation turn count (user messages)
	Cost          string  `json:"cost"`       // pre-formatted cost string
	CostValue     float64 `json:"cost_value"` // raw cost for sorting
	InputTokens   int64   `json:"input_tokens,omitempty"`
	OutputTokens  int64   `json:"output_tokens,omitempty"`
	CostModel     string  `json:"cost_model,omitempty"`
	HasUsageData  bool    `json:"has_usage_data"`
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
		case StatusWaiting:
			activeCount++
		case StatusIdle:
			idleCount++
		case StatusDead:
			deadCount++
		}
	}

	fmt.Println()
	fmt.Printf("%s%s  Claude Code Dashboard%s  %s%s%s\n", ansiBold, ansiCyan, ansiReset, ansiDim, time.Now().Format("15:04:05"), ansiReset)
	printSep()

	fmt.Printf("  %s● %d running%s", ansiGreen, activeCount+idleCount, ansiReset)
	if idleCount > 0 {
		fmt.Printf("  %s◐ %d idle%s", ansiYellow, idleCount, ansiReset)
	}
	if deadCount > 0 {
		fmt.Printf("  %s○ %d dead%s", ansiRed, deadCount, ansiReset)
	}
	fmt.Println()
	printSep()

	fmt.Printf("  %s%-6s  %-8s  %5s  %5s  %5s  %8s  %-10s  %-9s  %-40s  %s%s\n",
		ansiDim, "PID", "STATUS", "TURNS", "CPU%", "MEM%", "COST", "UPTIME", "IDLE", "DIRECTORY", "LAST ACTION", ansiReset)
	printSep()

	for _, r := range rows {
		statusColor := colorFor(r.Status)
		icon := iconFor(r.Status)

		uptime := formatDuration(r.UptimeSec)
		idle := formatIdle(r.IdleSec)
		cwd := truncate(filepath.Base(r.Cwd), maxCwdLen)
		action := CleanAction(truncate(r.LastAction, maxActionLen))

		costStr := r.Cost
		if costStr == "" {
			costStr = emDash
		}

		fmt.Printf("  %s%s%s %s%-6d%s  %s%-8s%s  %5d  %5s  %5s  %8s  %-10s  %-9s  %s%-40s%s  %s\n",
			statusColor, icon, ansiReset,
			"", r.PID, "",
			statusColor, r.Status, ansiReset,
			r.Turns,
			r.CPU, r.Mem,
			costStr,
			uptime, idle,
			ansiDim, cwd, ansiReset,
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
	if err := enc.Encode(rows); err != nil {
		return fmt.Errorf("encoding JSON output: %w", err)
	}
	return nil
}

func printSep() {
	fmt.Printf("%s  %s%s\n", ansiDim, strings.Repeat("─", sepWidth), ansiReset)
}

func colorFor(s Status) string {
	switch s {
	case StatusActive:
		return ansiGreen
	case StatusWaiting:
		return ansiCyan
	case StatusIdle:
		return ansiYellow
	case StatusDead:
		return ansiRed
	default:
		return ansiReset
	}
}

func iconFor(s Status) string {
	switch s {
	case StatusActive:
		return "●"
	case StatusWaiting:
		return "◇"
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
		return emDash
	}
	d := seconds / secsPerDay
	h := (seconds % secsPerDay) / secsPerHour
	m := (seconds % secsPerHour) / secsPerMin
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
		return emDash
	}
	if seconds < secsPerMin {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < secsPerHour {
		return fmt.Sprintf("%dm", seconds/secsPerMin)
	}
	if seconds < secsPerDay {
		return fmt.Sprintf("%dh %dm", seconds/secsPerHour, (seconds%secsPerHour)/secsPerMin)
	}
	return fmt.Sprintf("%dd", seconds/secsPerDay)
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}
