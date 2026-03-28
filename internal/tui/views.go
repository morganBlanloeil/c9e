package tui

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/morganBlanloeil/c9e/internal/cost"
	"github.com/morganBlanloeil/c9e/internal/display"
	"github.com/morganBlanloeil/c9e/internal/logs"
)

// Layout constants: fixed lines consumed by UI chrome.
const (
	listFixedLines    = 9 // title, 2 separators, summary, header, separator, footer separator, stats, help
	logFixedLines     = 7 // title, 2 separators, status bar, separator, footer separator, help
	emDash            = "—"
	maxCwdLen         = 40
	maxActionLen      = 50
	statusPadWidth    = 8
	secsPerDay        = 86400
	secsPerHour       = 3600
	secsPerMin        = 60
	tokensPerMillion  = 1_000_000
	tokensPerThousand = 1_000
	logPrefixWidth    = 18
	minSummaryWidth   = 20
	highCostThreshold = 1.0
	medCostThreshold  = 0.10
	tokenFormatBase   = 10
)

func (m Model) viewList() string {
	var b strings.Builder

	// Title bar
	title := titleStyle.Render("  Claude Code Dashboard")
	ver := dimStyle.Render(" " + m.version)
	b.WriteString(title + ver + "\n")

	// Separator
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Summary counts
	active, waiting, idle, dead := 0, 0, 0, 0
	var totalCost float64
	for _, r := range m.rows {
		switch r.Status {
		case display.StatusActive:
			active++
		case display.StatusWaiting:
			waiting++
		case display.StatusIdle:
			idle++
		case display.StatusDead:
			dead++
		}
		totalCost += r.CostValue
	}
	doneCount := 0
	for sid := range m.doneHighlight {
		if m.isDone(sid) {
			doneCount++
		}
	}
	summary := "  " + activeCountStyle.Render(fmt.Sprintf("● %d active", active))
	if waiting > 0 {
		summary += "  " + waitingCountStyle.Render(fmt.Sprintf("◇ %d waiting", waiting))
	}
	if idle > 0 {
		summary += "  " + idleCountStyle.Render(fmt.Sprintf("◐ %d idle", idle))
	}
	if dead > 0 {
		summary += "  " + deadCountStyle.Render(fmt.Sprintf("○ %d dead", dead))
	}
	if doneCount > 0 {
		summary += "  " + doneCountStyle.Render(fmt.Sprintf("★ %d done", doneCount))
	}
	if totalCost > 0 {
		summary += "  " + styledCost(totalCost).Render("Total: "+cost.Format(totalCost))
	}
	b.WriteString(summary + "\n")

	// Filter bar
	if m.filtering {
		b.WriteString(filterPromptStyle.Render("  /") + filterTextStyle.Render(m.filter+"█") + "\n")
	} else if m.filter != "" {
		b.WriteString(dimStyle.Render("  filter: "+m.filter) + "\n")
	}

	// Separator
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Sort indicator
	sortInfo := dimStyle.Render("  sort: " + sortColumnNames[m.sortCol])
	if m.sortAsc {
		sortInfo += dimStyle.Render(" ▲")
	} else {
		sortInfo += dimStyle.Render(" ▼")
	}

	// Header with sort indicator on the right
	header := fmt.Sprintf("  %-6s  %-8s  %5s  %5s  %5s  %8s  %-10s  %-9s  %-40s  %s",
		"PID", "STATUS", "TURNS", "CPU%", "MEM%", "COST", "UPTIME", "IDLE", "DIRECTORY", "LAST ACTION")
	headerLine := headerStyle.Render(header)
	b.WriteString(headerLine + "  " + sortInfo + "\n")
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Rows
	visibleRows := max(1, m.height-listFixedLines)
	if m.filtering || m.filter != "" {
		visibleRows = max(1, visibleRows-1)
	}

	// Scroll offset
	scrollOffset := 0
	if m.cursor >= visibleRows {
		scrollOffset = m.cursor - visibleRows + 1
	}

	for i := scrollOffset; i < len(m.filtered) && i < scrollOffset+visibleRows; i++ {
		r := m.filtered[i]
		line := m.renderRow(r)
		switch {
		case i == m.cursor:
			line = selectedRowStyle.Render(line)
		case m.isDone(r.SessionID):
			line = doneRowStyle.Render(line)
		}
		b.WriteString(line + "\n")
	}

	// Pad empty space
	rendered := min(len(m.filtered)-scrollOffset, visibleRows)
	for i := rendered; i < visibleRows; i++ {
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Aggregate stats bar
	var totalCPU, totalMem float64
	for _, r := range m.rows {
		c, _ := strconv.ParseFloat(r.CPU, bitSize64)
		totalCPU += c
		me, _ := strconv.ParseFloat(r.Mem, bitSize64)
		totalMem += me
	}
	stats := fmt.Sprintf("  %d sessions  |  CPU: %.1f%%  MEM: %.1f%%  |  %d active  %d idle  %d dead",
		len(m.rows), totalCPU, totalMem, active, idle, dead)
	b.WriteString(mutedStyle.Render(stats) + "\n")

	if m.err != nil {
		b.WriteString(deadBadge.Render(fmt.Sprintf("  error: %v", m.err)) + "\n")
	}

	if flash := m.activeFlash(); flash != "" {
		b.WriteString(notifyFlashStyle.Render("  "+flash) + "\n")
	}

	// Clipboard flash
	if m.clipboardFlash != "" {
		b.WriteString(activeCountStyle.Render("  "+m.clipboardFlash) + "\n")
	}

	if m.confirm != nil {
		b.WriteString(deadBadge.Render("  "+m.confirm.label) + "\n")
	} else {
		notifyState := notifyOffStyle.Render("OFF")
		if m.notifyEnabled {
			notifyState = notifyOnStyle.Render("ON")
		}
		help := fmt.Sprintf("  j/k: navigate  enter: detail  l: logs  o: jump  d: kill  /: filter  s/S: sort  c: copy cwd  n: notify %s  q: quit", notifyState)
		b.WriteString(helpStyle.Render(help))
	}

	return b.String()
}

func (m Model) renderRow(r display.Row) string {
	icon := statusIcon(r.Status)
	status := statusText(r.Status)
	if m.isDone(r.SessionID) {
		icon = doneBadge.Render("★")
		status = doneBadge.Render(fmt.Sprintf("%-*s", statusPadWidth, "DONE"))
	}
	uptime := fmt.Sprintf("%-10s", formatDuration(r.UptimeSec))
	idle := fmt.Sprintf("%-9s", formatIdle(r.IdleSec))
	cwd := fmt.Sprintf("%-40s", truncate(filepath.Base(r.Cwd), maxCwdLen))
	action := display.CleanAction(truncate(r.LastAction, maxActionLen))
	turns := fmt.Sprintf("%5d", r.Turns)

	costStr := fmt.Sprintf("%8s", emDash)
	if r.Cost != "" {
		costStr = styledCost(r.CostValue).Render(fmt.Sprintf("%8s", r.Cost))
	}

	// Build row by concatenating styled strings with literal spacing.
	// Using fmt.Sprintf width specifiers on ANSI-styled strings breaks alignment
	// because Go counts invisible escape codes as part of the string width.
	return fmt.Sprintf("  %s %-6d  %s  %s  %5s  %5s  %s  %s  %s  %s  %s",
		icon, r.PID, status, turns, r.CPU, r.Mem, costStr, uptime, idle, mutedStyle.Render(cwd), action)
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
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n\n")

	// Fields
	costDisplay := emDash
	if r.Cost != "" {
		costDisplay = r.Cost
	}
	tokenInfo := ""
	if r.InputTokens > 0 || r.OutputTokens > 0 {
		tokenInfo = fmt.Sprintf(" (in: %s, out: %s)", formatTokenCount(r.InputTokens), formatTokenCount(r.OutputTokens))
	}
	costLabel := "Cost"
	if !r.HasUsageData {
		costLabel = "Cost (est.)"
	}
	modelDisplay := r.CostModel
	if modelDisplay == "" {
		modelDisplay = emDash
	}

	fields := []struct {
		label string
		value string
	}{
		{"Status", string(r.Status)},
		{"PID", strconv.Itoa(r.PID)},
		{"Session ID", r.SessionID},
		{"Directory", r.RawCwd},
		{"CPU", r.CPU + "%"},
		{"Memory", r.Mem + "%"},
		{"Uptime", formatDuration(r.UptimeSec)},
		{"Idle", formatIdle(r.IdleSec)},
		{"Turns", strconv.Itoa(r.Turns)},
		{costLabel, costDisplay + tokenInfo},
		{"Model", modelDisplay},
		{"Last Action", display.CleanAction(r.LastAction)},
	}

	for _, f := range fields {
		label := detailLabelStyle.Render("  " + f.label)
		value := detailValueStyle.Render(f.value)
		if f.label == "Status" {
			if m.isDone(r.SessionID) {
				value = doneBadge.Render("★") + " " + doneBadge.Render("DONE")
			} else {
				value = statusIcon(r.Status) + " " + statusText(r.Status)
			}
		}
		b.WriteString(label + "  " + value + "\n")
	}

	// Pad
	const detailHeaderLines = 4 // title + separator + blank + footer
	usedLines := len(fields) + detailHeaderLines
	for i := usedLines; i < m.height-2; i++ {
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(helpStyle.Render("  esc/q: back to list  l: logs  ctrl+c: quit"))

	return b.String()
}

func statusIcon(s display.Status) string {
	switch s {
	case display.StatusActive:
		return activeBadge.Render("●")
	case display.StatusWaiting:
		return waitingBadge.Render("◇")
	case display.StatusIdle:
		return idleBadge.Render("◐")
	case display.StatusDead:
		return deadBadge.Render("○")
	default:
		return "?"
	}
}

func statusText(s display.Status) string {
	padded := fmt.Sprintf("%-*s", statusPadWidth, string(s))
	switch s {
	case display.StatusActive:
		return activeBadge.Render(padded)
	case display.StatusWaiting:
		return waitingBadge.Render(padded)
	case display.StatusIdle:
		return idleBadge.Render(padded)
	case display.StatusDead:
		return deadBadge.Render(padded)
	default:
		return padded
	}
}

// styledCost returns the appropriate lipgloss style for a cost value.
func styledCost(costValue float64) lipgloss.Style {
	switch {
	case costValue > highCostThreshold:
		return costHigh
	case costValue >= medCostThreshold:
		return costMedium
	default:
		return costLow
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

// formatTokenCount formats a token count as a human-readable string (e.g. "1.2K", "3.4M").
func formatTokenCount(tokens int64) string {
	switch {
	case tokens >= tokensPerMillion:
		return fmt.Sprintf("%.1fM", float64(tokens)/tokensPerMillion)
	case tokens >= tokensPerThousand:
		return fmt.Sprintf("%.1fK", float64(tokens)/tokensPerThousand)
	default:
		return strconv.FormatInt(tokens, tokenFormatBase)
	}
}

func truncate(s string, maxLen int) string {
	if len(s) > maxLen {
		return s[:maxLen-1] + "…"
	}
	return s
}

func (m Model) viewLogs() string {
	r := m.selectedRow()
	if r == nil {
		return "No session selected"
	}

	var b strings.Builder

	// Title
	sid := r.FullSessionID
	if sid == "" {
		sid = r.SessionID
	}
	title := detailTitleStyle.Render("  Log Tail — Session " + sid)
	b.WriteString(title + "\n")
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Status bar
	followStr := logFollowOff.Render("OFF")
	if m.logFollow {
		followStr = logFollowOn.Render("ON")
	}
	thinkStr := logFollowOff.Render("OFF")
	if m.logShowThink {
		thinkStr = logFollowOn.Render("ON")
	}

	statusLine := fmt.Sprintf("  %d entries  follow: %s  thinking: %s",
		len(m.logFiltered), followStr, thinkStr)
	b.WriteString(statusLine + "\n")
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")

	// Log entries
	visibleLines := m.logVisibleLines()

	if m.logErr != nil {
		b.WriteString(deadBadge.Render(fmt.Sprintf("  error: %v", m.logErr)) + "\n")
	} else if len(m.logFiltered) == 0 {
		b.WriteString(dimStyle.Render("  No log entries found") + "\n")
	}

	rendered := 0
	for i := m.logScroll; i < len(m.logFiltered) && rendered < visibleLines; i++ {
		entry := m.logFiltered[i]
		b.WriteString(renderLogEntry(entry, m.width) + "\n")
		rendered++
	}

	// Pad empty space
	for i := rendered; i < visibleLines; i++ {
		b.WriteString("\n")
	}

	// Footer
	b.WriteString(borderStyle.Render(strings.Repeat("─", m.width)) + "\n")
	b.WriteString(helpStyle.Render("  j/k: scroll  G: bottom  g: top  f: follow  t: thinking  esc: back"))

	return b.String()
}

func renderLogEntry(e logs.LogEntry, width int) string {
	ts := logTimestamp.Render(e.Timestamp.Format("15:04:05"))

	var icon string
	switch {
	case e.HasThink:
		icon = logThinkIcon.Render("~")
	case e.Type == logs.EntryUser:
		if e.RawType == "tool_result" {
			icon = logToolIcon.Render("T")
		} else {
			icon = logUserIcon.Render(">")
		}
	case e.RawType == "tool_use":
		icon = logToolIcon.Render("T")
	default:
		icon = logAssistIcon.Render("<")
	}

	// Truncate summary to fit width (account for "  HH:MM:SS  X  " prefix ~18 chars)
	maxSummary := width - logPrefixWidth
	maxSummary = max(maxSummary, minSummaryWidth)
	summary := e.Summary
	runes := []rune(summary)
	if len(runes) > maxSummary {
		summary = string(runes[:maxSummary-1]) + "…"
	}

	return fmt.Sprintf("  %s  %s  %s", ts, icon, summary)
}
