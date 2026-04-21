package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorBase    = lipgloss.AdaptiveColor{Light: "#1a1a2e", Dark: "#e0e0e0"}
	colorSubtle  = lipgloss.AdaptiveColor{Light: "#8a8fa8", Dark: "#8a8fa8"}
	colorActive  = lipgloss.AdaptiveColor{Light: "#2196f3", Dark: "#29b6f6"}
	colorGreen   = lipgloss.AdaptiveColor{Light: "#00c853", Dark: "#00e676"}
	colorYellow  = lipgloss.AdaptiveColor{Light: "#ffab00", Dark: "#ffd600"}
	colorRed     = lipgloss.AdaptiveColor{Light: "#f44336", Dark: "#ff5252"}
	colorGray    = lipgloss.AdaptiveColor{Light: "#757575", Dark: "#9e9e9e"}

	styleTab = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(colorSubtle)

	styleActiveTab = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(colorActive).
			Bold(true).
			Underline(true)

	styleTabBar = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderBottom(true).
			BorderForeground(colorSubtle).
			MarginBottom(1)

	styleRunning = lipgloss.NewStyle().Foreground(colorGreen)
	stylePaused  = lipgloss.NewStyle().Foreground(colorYellow)
	styleDone    = lipgloss.NewStyle().Foreground(colorRed)
	styleIdle    = lipgloss.NewStyle().Foreground(colorGray)

	styleHelp = lipgloss.NewStyle().
			Foreground(colorSubtle).
			MarginTop(1)

	styleModal = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorActive).
			Padding(1, 2)

	styleLabel = lipgloss.NewStyle().
			Foreground(colorSubtle).
			Width(12)

	styleError = lipgloss.NewStyle().Foreground(colorRed)
)
