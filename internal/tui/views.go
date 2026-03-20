package tui

import (
	"fmt"
	"strings"

	"github.com/wescale/claude-dashboard/internal/display"
)

func (m Model) viewList() string {
	var b strings.Builder

	// Title bar
	title := titleStyle.Render("  Claude Code Dashboard")
	ver := dimStyle.Render(fmt.Sprintf(" %s", m.version))
	b.WriteString(title + ver + "\n")

	// Separator
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Summary counts
	active, idle, dead := 0, 0, 0
	for _, r := range m.rows {
		switch r.Status {
		case display.StatusActive:
			active++
		case display.StatusIdle:
			idle++
		case display.StatusDead:
			dead++
		}
	}
	summary := "  " + activeCountStyle.Render(fmt.Sprintf("● %d active", active))
	if idle > 0 {
		summary += "  " + idleCountStyle.Render(fmt.Sprintf("◐ %d idle", idle))
	}
	if dead > 0 {
		summary += "  " + deadCountStyle.Render(fmt.Sprintf("○ %d dead", dead))
	}
	b.WriteString(summary + "\n")

	// Filter bar
	if m.filtering {
		b.WriteString(filterPromptStyle.Render("  /") + filterTextStyle.Render(m.filter+"█") + "\n")
	} else if m.filter != "" {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  filter: %s", m.filter)) + "\n")
	}

	// Separator
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Header
	header := fmt.Sprintf("  %-6s  %-8s  %5s  %5s  %-10s  %-9s  %-40s  %s",
		"PID", "STATUS", "CPU%", "MEM%", "UPTIME", "IDLE", "DIRECTORY", "LAST ACTION")
	b.WriteString(headerStyle.Render(header) + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Rows
	visibleRows := m.height - 8 // account for header, footer, separators
	if m.filtering || m.filter != "" {
		visibleRows--
	}
	if visibleRows < 1 {
		visibleRows = 1
	}

	// Scroll offset
	scrollOffset := 0
	if m.cursor >= visibleRows {
		scrollOffset = m.cursor - visibleRows + 1
	}

	for i := scrollOffset; i < len(m.filtered) && i < scrollOffset+visibleRows; i++ {
		r := m.filtered[i]
		line := m.renderRow(r)
		if i == m.cursor {
			line = selectedRowStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}

	// Pad empty space
	rendered := len(m.filtered) - scrollOffset
	if rendered > visibleRows {
		rendered = visibleRows
	}
	for i := rendered; i < visibleRows; i++ {
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")

	if m.err != nil {
		b.WriteString(deadBadge.Render(fmt.Sprintf("  error: %v", m.err)) + "\n")
	}

	if m.confirm != nil {
		b.WriteString(deadBadge.Render("  "+m.confirm.label) + "\n")
	} else {
		help := "  j/k: navigate  enter: detail  d: kill  /: filter  q: quit"
		b.WriteString(helpStyle.Render(help))
	}

	return b.String()
}

func (m Model) renderRow(r display.Row) string {
	icon := statusIcon(r.Status)
	status := statusText(r.Status)
	uptime := formatDuration(r.UptimeSec)
	idle := formatIdle(r.IdleSec)
	cwd := truncate(r.Cwd, 40)
	action := cleanAction(truncate(r.LastAction, 50))

	return fmt.Sprintf("  %s %-6d  %s  %5s  %5s  %-10s  %-9s  %-40s  %s",
		icon, r.PID, status, r.CPU, r.Mem, uptime, idle, dimStyle.Render(cwd), action)
}

func (m Model) viewDetail() string {
	r := m.selectedRow()
	if r == nil {
		return "No session selected"
	}

	var b strings.Builder

	// Title
	title := detailTitleStyle.Render(fmt.Sprintf("  Session %s — PID %d", r.SessionID, r.PID))
	b.WriteString(title + "\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n\n")

	// Fields
	fields := []struct {
		label string
		value string
	}{
		{"Status", string(r.Status)},
		{"PID", fmt.Sprintf("%d", r.PID)},
		{"Session ID", r.SessionID},
		{"Directory", r.Cwd},
		{"CPU", r.CPU + "%"},
		{"Memory", r.Mem + "%"},
		{"Uptime", formatDuration(r.UptimeSec)},
		{"Idle", formatIdle(r.IdleSec)},
		{"Last Action", cleanAction(r.LastAction)},
	}

	for _, f := range fields {
		label := detailLabelStyle.Render("  " + f.label)
		value := detailValueStyle.Render(f.value)
		if f.label == "Status" {
			value = statusIcon(r.Status) + " " + statusText(r.Status)
		}
		b.WriteString(label + "  " + value + "\n")
	}

	// Pad
	usedLines := len(fields) + 4
	for i := usedLines; i < m.height-2; i++ {
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(dimStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(helpStyle.Render("  esc/q: back to list  ctrl+c: quit"))

	return b.String()
}

func statusIcon(s display.Status) string {
	switch s {
	case display.StatusActive:
		return activeBadge.Render("●")
	case display.StatusIdle:
		return idleBadge.Render("◐")
	case display.StatusDead:
		return deadBadge.Render("○")
	default:
		return "?"
	}
}

func statusText(s display.Status) string {
	padded := fmt.Sprintf("%-8s", string(s))
	switch s {
	case display.StatusActive:
		return activeBadge.Render(padded)
	case display.StatusIdle:
		return idleBadge.Render(padded)
	case display.StatusDead:
		return deadBadge.Render(padded)
	default:
		return padded
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

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}

func cleanAction(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}
