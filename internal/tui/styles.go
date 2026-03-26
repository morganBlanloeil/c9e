package tui

import "github.com/charmbracelet/lipgloss"

const detailLabelWidth = 16

// AdaptiveColor: {Light, Dark}
var (
	// Colors — adaptive for light/dark terminals
	cyan   = lipgloss.AdaptiveColor{Light: "#0077AA", Dark: "#5FD7D7"}
	green  = lipgloss.AdaptiveColor{Light: "#007A00", Dark: "#5FFF5F"}
	yellow = lipgloss.AdaptiveColor{Light: "#806600", Dark: "#FFD700"}
	red    = lipgloss.AdaptiveColor{Light: "#CC0000", Dark: "#FF5F5F"}
	dim    = lipgloss.AdaptiveColor{Light: "#888888", Dark: "#9E9E9E"}
	border = lipgloss.AdaptiveColor{Light: "#AAAAAA", Dark: "#606060"}
	text   = lipgloss.AdaptiveColor{Light: "#1A1A1A", Dark: "#FAFAFA"}
	muted  = lipgloss.AdaptiveColor{Light: "#555555", Dark: "#C0C0C0"}
	purple = lipgloss.AdaptiveColor{Light: "#6A1B9A", Dark: "#D7AFFF"}

	doneColor = lipgloss.AdaptiveColor{Light: "#CC6600", Dark: "#FFB347"}

	selectedBg = lipgloss.AdaptiveColor{Light: "#D0D0D0", Dark: "#4A4A4A"}
	selectedFg = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}

	doneBg = lipgloss.AdaptiveColor{Light: "#FFF3E0", Dark: "#3D2800"}
	doneFg = lipgloss.AdaptiveColor{Light: "#CC6600", Dark: "#FFD080"}

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

	// Done row highlight — full-row background for recently finished sessions
	doneRowStyle = lipgloss.NewStyle().
			Background(doneBg).
			Foreground(doneFg).
			Bold(true)

	// Status badges
	activeBadge  = lipgloss.NewStyle().Foreground(green).Bold(true)
	waitingBadge = lipgloss.NewStyle().Foreground(cyan).Bold(true)
	idleBadge    = lipgloss.NewStyle().Foreground(yellow).Bold(true)
	deadBadge    = lipgloss.NewStyle().Foreground(red).Bold(true)
	doneBadge    = lipgloss.NewStyle().Foreground(doneColor).Bold(true).Blink(true)

	// Done summary count
	doneCountStyle = lipgloss.NewStyle().Foreground(doneColor).Bold(true)

	// Separator lines
	borderStyle = lipgloss.NewStyle().Foreground(border)

	// Muted style (brighter than dim, for secondary content like CWD)
	mutedStyle = lipgloss.NewStyle().Foreground(muted)

	// Footer / help bar
	helpStyle = lipgloss.NewStyle().
			Foreground(muted).
			Padding(0, 1)

	// Detail view
	detailTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(cyan).
				Padding(0, 1)

	detailLabelStyle = lipgloss.NewStyle().
				Foreground(purple).
				Bold(true).
				Width(detailLabelWidth)

	detailValueStyle = lipgloss.NewStyle().
				Foreground(text)

	// Filter
	filterPromptStyle = lipgloss.NewStyle().
				Foreground(cyan).
				Bold(true)

	filterTextStyle = lipgloss.NewStyle().
			Foreground(text)

	// Dim style
	dimStyle = lipgloss.NewStyle().Foreground(dim)

	// Cost column styles
	costLow    = lipgloss.NewStyle().Foreground(green)          // < $0.10
	costMedium = lipgloss.NewStyle().Foreground(yellow)         // $0.10 - $1.00
	costHigh   = lipgloss.NewStyle().Foreground(red).Bold(true) // > $1.00

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
