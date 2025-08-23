package viewinterface

import (
	"UniCode/src/constants"
	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	StatusLight = "●"
)

type UpdateStatus struct {
	NewStauts constants.ToolStatus
}

func NewToolModel(data string) *ToolModel {
	light := cursor.New()
	light.SetChar(lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render(StatusLight))
	return &ToolModel{
		Status:   light,
		ToolInfo: data,
	}
}

type ToolModel struct {
	Status   cursor.Model
	ToolInfo string
}

func (instance *ToolModel) Init() tea.Cmd {
	return instance.Status.Focus()
}

func (instance *ToolModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	if msg, ok := msg.(UpdateStatus); ok {
		if msg.NewStauts != constants.Call {
			var newChar string
			switch msg.NewStauts {
			case constants.Error:
				newChar = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9)).Render(StatusLight)
			case constants.Success:
				newChar = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10)).Render(StatusLight)
			default:
				// 예상치 못한 상태인 경우 노란색으로 표시
				newChar = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(11)).Render(StatusLight)
			}
			instance.Status.SetChar(newChar)
			instance.Status.SetMode(cursor.CursorStatic)
		}
	}
	instance.Status, cmd = instance.Status.Update(msg)
	return instance, cmd
}

func (instance *ToolModel) View() string {
	return lipgloss.JoinHorizontal(lipgloss.Left, instance.Status.View(), " ", instance.ToolInfo, "\n")
}
