package tui

import (
	"fmt"
	"time"

	"github.com/wescale/claude-dashboard/internal/cost"
	"github.com/wescale/claude-dashboard/internal/display"
	"github.com/wescale/claude-dashboard/internal/history"
	"github.com/wescale/claude-dashboard/internal/logs"
	"github.com/wescale/claude-dashboard/internal/process"
	"github.com/wescale/claude-dashboard/internal/session"
)

const roleAssistant = "assistant"

// fetchRows collects all session data and returns display rows.
func fetchRows() ([]display.Row, error) {
	sessions, err := session.LoadAll()
	if err != nil {
		return nil, fmt.Errorf("loading sessions: %w", err)
	}

	actions, err := history.LastActions()
	if err != nil {
		return nil, fmt.Errorf("loading history: %w", err)
	}

	procs, err := process.ListClaude()
	if err != nil {
		return nil, fmt.Errorf("listing processes: %w", err)
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
			uptimeSec = (nowMs - s.StartedAt) / msPerSecond
		}

		idleSec := int64(-1)
		lastAction := emDash
		if action, ok := actions[s.SessionID]; ok {
			lastAction = action.Display
			if action.Timestamp > 0 {
				idleSec = (nowMs - action.Timestamp) / msPerSecond
			}
		}

		logPath := logs.ResolvePath(s.SessionID, s.Cwd)

		status := display.StatusActive
		switch {
		case !alive:
			status = display.StatusDead
		case idleSec > int64(display.IdleThreshold.Seconds()):
			status = display.StatusIdle
		case logPath != "" && logs.LastRole(logPath) == roleAssistant:
			status = display.StatusWaiting
		}

		sid := s.SessionID
		if len(sid) > sessionIDLen {
			sid = sid[:sessionIDLen]
		}

		// Count conversation turns
		turns := 0
		if logPath != "" {
			turns = logs.CountTurns(logPath)
		}

		// Estimate cost from session log
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
			PID:           s.PID,
			SessionID:     sid,
			FullSessionID: s.SessionID,
			Status:        status,
			CPU:           cpu,
			Mem:           mem,
			Cwd:           s.ShortCwd(),
			RawCwd:        s.Cwd,
			UptimeSec:     uptimeSec,
			IdleSec:       idleSec,
			LastAction:    lastAction,
			Alive:         alive,
			LogPath:       logPath,
			Turns:         turns,
			Cost:          costStr,
			CostValue:     costValue,
			InputTokens:   inputTokens,
			OutputTokens:  outputTokens,
			CostModel:     costModel,
			HasUsageData:  hasUsageData,
		})
	}

	return rows, nil
}
