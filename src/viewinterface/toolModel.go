package viewinterface

import (
	"DevCode/src/config"
	"DevCode/src/constants"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type UpdateStatus struct {
	NewStauts constants.ToolStatus
}

func NewToolModel(data string, config config.ViewConfig) *ToolModel {
	light := cursor.New()
	light.SetChar(DefaultStyles.ToolPending.Render(config.Dot))
	return &ToolModel{
		Status:   light,
		ToolInfo: data,
	}
}

type ToolModel struct {
	Status   cursor.Model
	ToolInfo string
	Config   config.ViewConfig
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
				newChar = DefaultStyles.ToolError.Render(instance.Config.Dot)
			case constants.Success:
				newChar = DefaultStyles.ToolSuccess.Render(instance.Config.Dot)
			default:
				// 예상치 못한 상태인 경우 노란색으로 표시
				newChar = DefaultStyles.ToolDefault.Render(instance.Config.Dot)
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
