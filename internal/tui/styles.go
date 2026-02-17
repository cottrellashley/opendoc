// Package tui implements the Bubble Tea terminal UI for OpenDoc console.
package tui

import "github.com/charmbracelet/lipgloss"

// OpenDoc theme colours — matches the site's dark mode palette.
var (
	colorBg      = lipgloss.Color("#0d1117")
	colorSurface = lipgloss.Color("#161b22")
	colorBorder  = lipgloss.Color("#21262d")
	colorText    = lipgloss.Color("#e6edf3")
	colorMuted   = lipgloss.Color("#7d8590")
	colorAccent  = lipgloss.Color("#79a5f2")
	colorGreen   = lipgloss.Color("#3fb950")
	colorOrange  = lipgloss.Color("#d29922")
	colorRed     = lipgloss.Color("#f85149")
)

// Layout styles.
var (
	headerStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorAccent).
			Bold(true).
			Padding(0, 1)

	statusBarStyle = lipgloss.NewStyle().
			Background(colorSurface).
			Foreground(colorMuted).
			Padding(0, 1)

	contentStyle = lipgloss.NewStyle().
			Padding(1, 2)

	titleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	subtitleStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	selectedStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)

	normalItemStyle = lipgloss.NewStyle().
			Foreground(colorText)

	mutedStyle = lipgloss.NewStyle().
			Foreground(colorMuted)

	accentStyle = lipgloss.NewStyle().
			Foreground(colorAccent)

	successStyle = lipgloss.NewStyle().
			Foreground(colorGreen)

	warningStyle = lipgloss.NewStyle().
			Foreground(colorOrange)

	errorStyle = lipgloss.NewStyle().
			Foreground(colorRed)

	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorBorder).
			Padding(1, 2)
)
