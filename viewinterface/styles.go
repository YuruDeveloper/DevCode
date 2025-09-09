package viewinterface

import (
	"github.com/charmbracelet/lipgloss"
)

type Styles struct {
	Input       lipgloss.Style
	Select      lipgloss.Style
	Message     lipgloss.Style
	ToolPending lipgloss.Style
	ToolError   lipgloss.Style
	ToolSuccess lipgloss.Style
	ToolDefault lipgloss.Style
}

func NewStyles() *Styles {
	return &Styles{
		Input: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			PaddingLeft(1).
			PaddingRight(2).
			BorderForeground(lipgloss.ANSIColor(8)),

		Select: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.ANSIColor(32)).
			Height(10).
			Width(20),

		Message: lipgloss.NewStyle().
			Width(0),

		ToolPending: lipgloss.NewStyle().
			Foreground(lipgloss.ANSIColor(8)),

		ToolError: lipgloss.NewStyle().
			Foreground(lipgloss.ANSIColor(9)),

		ToolSuccess: lipgloss.NewStyle().
			Foreground(lipgloss.ANSIColor(10)),

		ToolDefault: lipgloss.NewStyle().
			Foreground(lipgloss.ANSIColor(11)),
	}
}

var DefaultStyles = NewStyles()
