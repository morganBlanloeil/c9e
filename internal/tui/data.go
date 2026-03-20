package tui

import (
	"time"

	"github.com/wescale/claude-dashboard/internal/display"
	"github.com/wescale/claude-dashboard/internal/history"
	"github.com/wescale/claude-dashboard/internal/process"
	"github.com/wescale/claude-dashboard/internal/session"
)

// fetchRows collects all session data and returns display rows.
func fetchRows() ([]display.Row, error) {
	sessions, err := session.LoadAll()
	if err != nil {
		return nil, err
	}

	actions, err := history.LastActions()
	if err != nil {
		return nil, err
	}

	procs, err := process.ListClaude()
	if err != nil {
		return nil, err
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

		status := display.StatusActive
		if !alive {
			status = display.StatusDead
		} else if idleSec > int64(display.IdleThreshold.Seconds()) {
			status = display.StatusIdle
		}

		sid := s.SessionID
		if len(sid) > 8 {
			sid = sid[:8]
		}

		rows = append(rows, display.Row{
			PID:        s.PID,
			SessionID:  sid,
			Status:     status,
			CPU:        cpu,
			Mem:        mem,
			Cwd:        s.ShortCwd(),
			UptimeSec:  uptimeSec,
			IdleSec:    idleSec,
			LastAction: lastAction,
			Alive:      alive,
		})
	}

	return rows, nil
}
