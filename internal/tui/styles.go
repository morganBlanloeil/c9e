package tui

import "github.com/charmbracelet/lipgloss"

// AdaptiveColor: {Light, Dark}
var (
	// Colors — adaptive for light/dark terminals
	cyan   = lipgloss.AdaptiveColor{Light: "#0077AA", Dark: "#5FD7D7"}
	green  = lipgloss.AdaptiveColor{Light: "#007A00", Dark: "#5FFF5F"}
	yellow = lipgloss.AdaptiveColor{Light: "#806600", Dark: "#FFD700"}
	red    = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF5F5F"}
	dim    = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#6C6C6C"}
	text   = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#FAFAFA"}
	purple = lipgloss.AdaptiveColor{Light: "#6A1B9A", Dark: "#AF87FF"}

	doneColor = lipgloss.AdaptiveColor{Light: "#CC6600", Dark: "#FFB347"}

	selectedBg = lipgloss.AdaptiveColor{Light: "#D0D0D0", Dark: "#303030"}
	selectedFg = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}

	// Title bar
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(cyan).
			Padding(0, 1)

	// Status summary
	activeCountStyle  = lipgloss.NewStyle().Foreground(green).Bold(true)
	waitingCountStyle = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	idleCountStyle    = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	deadCountStyle    = lipgloss.NewStyle().Foreground(red).Bold(true)

	// Table header
	headerStyle = lipgloss.NewStyle().
			Foreground(purple).
			Bold(true)

	// Row styles
	selectedRowStyle = lipgloss.NewStyle().
				Background(selectedBg).
				Foreground(selectedFg)

	normalRowStyle = lipgloss.NewStyle()

	// Status badges
	activeBadge  = lipgloss.NewStyle().Foreground(green).Bold(true)
	waitingBadge = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	idleBadge    = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	deadBadge    = lipgloss.NewStyle().Foreground(red).Bold(true)
	doneBadge    = lipgloss.NewStyle().Foreground(doneColor).Bold(true).Blink(true)

	// Done summary count
	doneCountStyle = lipgloss.NewStyle().Foreground(doneColor).Bold(true)

	// Footer / help bar
	helpStyle = lipgloss.NewStyle().
			Foreground(dim).
			Padding(0, 1)

	// Detail view
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(cyan).
				Padding(0, 1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(purple).
				Bold(true).
				Width(16)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(text)

	detailSepStyle = lipgloss.NewStyle().
			Foreground(dim)

	// Filter
	filterPromptStyle = lipgloss.NewStyle().
				Foreground(cyan).
				Bold(true)

	filterTextStyle = lipgloss.NewStyle().
			Foreground(text)

	// Dim style
	dimStyle = lipgloss.NewStyle().Foreground(dim)

	// Log view styles
	logUserIcon   = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	logAssistIcon = lipgloss.NewStyle().Foreground(green).Bold(true)
	logToolIcon   = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	logThinkIcon  = lipgloss.NewStyle().Foreground(dim)
	logTimestamp  = lipgloss.NewStyle().Foreground(dim)
	logFollowOn   = lipgloss.NewStyle().Foreground(green).Bold(true)
	logFollowOff  = lipgloss.NewStyle().Foreground(dim)

	// Notification styles
	notifyOnStyle    = lipgloss.NewStyle().Foreground(green).Bold(true)
	notifyOffStyle   = lipgloss.NewStyle().Foreground(dim)
	notifyFlashStyle = lipgloss.NewStyle().Foreground(doneColor).Bold(true)
)
