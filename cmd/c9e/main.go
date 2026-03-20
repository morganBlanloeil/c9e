package main

import (
	"fmt"
	"os"
	"sort"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"golang.org/x/term"

	"github.com/wescale/claude-dashboard/internal/cost"
	"github.com/wescale/claude-dashboard/internal/display"
	"github.com/wescale/claude-dashboard/internal/history"
	"github.com/wescale/claude-dashboard/internal/logs"
	"github.com/wescale/claude-dashboard/internal/process"
	"github.com/wescale/claude-dashboard/internal/session"
	"github.com/wescale/claude-dashboard/internal/tui"
)

var version = "dev"

const helpText = `
  Claude Code Dashboard — Monitor running Claude Code instances

  USAGE
    c9e [OPTIONS]

  MODES
    (default)         Interactive TUI (like k9s)
    --table, -t       Static table output (one-shot)
    --json,  -j       JSON output for scripting

  OPTIONS
    --help,    -h     Show this help
    --version, -v     Show version

  TUI KEYS
    j/k, ↑/↓         Navigate sessions
    enter             Drill into session detail
    esc, q            Back / quit
    /                 Filter by directory, status, or action
    g / G             Jump to first / last
    ctrl+c            Force quit

  COLUMNS
    PID               Process ID of the Claude Code instance
    STATUS            Current state of the instance:
                        ● ACTIVE  — interaction within the last 5 minutes
                        ◐ IDLE    — no interaction for more than 5 minutes
                        ○ DEAD    — session file exists but process is no longer running
    CPU%              Current CPU usage of the process
    MEM%              Current memory usage of the process
    UPTIME            Time since the instance was started
    IDLE              Time since the last user message in that session
    DIRECTORY         Working directory (project) of the instance
    LAST ACTION       Last user message sent in the session

  DATA SOURCES
    ~/.claude/sessions/*.json   Session metadata (pid, cwd, startedAt)
    ~/.claude/history.jsonl     User action log (last message per session)
    ps aux                      Live process stats (cpu, mem, alive check)

  EXAMPLES
    c9e                  Interactive TUI (default)
    c9e --table          One-shot table view
    c9e --json           JSON output for scripting
    c9e --json | jq '.[] | select(.alive and .idle_s > 600)'
                                      Find sessions idle for 10+ minutes
`

func main() {
	tableMode := false
	jsonOutput := false

	for _, arg := range os.Args[1:] {
		switch arg {
		case "--table", "-t":
			tableMode = true
		case "--json", "-j":
			jsonOutput = true
		case "--help", "-h":
			fmt.Print(helpText)
			os.Exit(0)
		case "--version", "-v":
			fmt.Printf("c9e %s\n", version)
			os.Exit(0)
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\nRun c9e --help for usage.\n", arg)
			os.Exit(1)
		}
	}

	// Auto-detect: if stdout is not a TTY, fallback to table mode
	if !tableMode && !jsonOutput && !term.IsTerminal(int(os.Stdout.Fd())) {
		tableMode = true
	}

	if tableMode || jsonOutput {
		if err := runStatic(jsonOutput); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// Default: interactive TUI
	m := tui.NewModel(version)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func runStatic(jsonOutput bool) error {
	sessions, err := session.LoadAll()
	if err != nil {
		return fmt.Errorf("loading sessions: %w", err)
	}
	if len(sessions) == 0 {
		fmt.Println("No active sessions found.")
		return nil
	}

	actions, err := history.LastActions()
	if err != nil {
		return fmt.Errorf("loading history: %w", err)
	}

	procs, err := process.ListClaude()
	if err != nil {
		return fmt.Errorf("listing processes: %w", err)
	}

	nowMs := time.Now().UnixMilli()
	rows := make([]display.Row, 0, len(sessions))

	for _, s := range sessions {
		proc, alive := procs[s.PID]
		cpu := "0.0"
		mem := "0.0"
		if alive {
			cpu = proc.CPU
			mem = proc.Mem
		}

		uptimeSec := int64(0)
		if s.StartedAt > 0 {
			uptimeSec = (nowMs - s.StartedAt) / 1000
		}

		idleSec := int64(-1)
		lastAction := "—"
		if action, ok := actions[s.SessionID]; ok {
			lastAction = action.Display
			if action.Timestamp > 0 {
				idleSec = (nowMs - action.Timestamp) / 1000
			}
		}

		logPath := logs.ResolvePath(s.SessionID, s.Cwd)

		status := display.StatusActive
		if !alive {
			status = display.StatusDead
		} else if idleSec > int64(display.IdleThreshold.Seconds()) {
			status = display.StatusIdle
		} else if logPath != "" && logs.LastRole(logPath) == "assistant" {
			status = display.StatusWaiting
		}

		// Estimate cost
		var costStr string
		var costValue float64
		var inputTokens, outputTokens int64
		var costModel string
		var hasUsageData bool
		if logPath != "" {
			if c, err := cost.EstimateFromLog(logPath); err == nil {
				costValue = c.EstimatedCost
				costStr = cost.Format(c.EstimatedCost)
				inputTokens = c.InputTokens
				outputTokens = c.OutputTokens
				costModel = c.Model
				hasUsageData = c.HasUsageData
			}
		}

		rows = append(rows, display.Row{
			PID:          s.PID,
			SessionID:    s.SessionID[:8],
			Status:       status,
			CPU:          cpu,
			Mem:          mem,
			Cwd:          s.ShortCwd(),
			UptimeSec:    uptimeSec,
			IdleSec:      idleSec,
			LastAction:   lastAction,
			Alive:        alive,
			LogPath:      logPath,
			Cost:         costStr,
			CostValue:    costValue,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			CostModel:    costModel,
			HasUsageData: hasUsageData,
		})
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].PID < rows[j].PID
	})

	if jsonOutput {
		return display.RenderJSON(rows)
	}
	display.RenderTable(rows)
	return nil
}
