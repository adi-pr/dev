package output

import (
	"strings"

	"charm.land/lipgloss/v2"
)

var (
	Text = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#DCE8E6"))

	Primary = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#E8CFA8")).
		Bold(true)

	Secondary = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9BD0CC")).
		Bold(true)

	Muted = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#A2ADAC"))

	Subtle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6D7876"))

	Success = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B5CCBA"))

	Warning = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#D3FAE8"))

	Error = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FA746F"))

	Branch = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#B0CCC9"))

	Rule = lipgloss.NewStyle().
		Foreground(lipgloss.Color("#3F4A49"))
)

func PadRight(value string, width int) string {
	visible := lipgloss.Width(value)

	if visible >= width {
		return value
	}

	return value + strings.Repeat(" ", width-visible)
}
