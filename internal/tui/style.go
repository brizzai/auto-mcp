package tui

import "github.com/charmbracelet/lipgloss"

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#15202b")).
			Background(lipgloss.Color("#f56a96")).
			Padding(0, 1)

	editHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#f56a96")).
			Padding(0, 1)

	statusMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#f56a96", Dark: "#f23a74"}).
				Render

	completeMessageStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#56FF4E")).
				Render
)
var docStyle = lipgloss.NewStyle().Margin(1, 2)
