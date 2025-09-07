package viewinterface

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/types"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"go.uber.org/zap"
)

type StreamUpdate struct {
	Content    string
	IsComplete bool
}

func NewMainModel(bus *events.EventBus, config config.ViewConfig, logger *zap.Logger, toolManager types.ToolManager) *MainModel {
	text := textarea.New()
	text.Focus()

	text.SetHeight(1)
	text.ShowLineNumbers = false
	text.FocusedStyle.Base = DefaultStyles.Input
	text.BlurredStyle.Base = DefaultStyles.Input
	text.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return config.SelectChar + " "
		}
		return ""
	})
	view := viewport.New(1, 0)

	selectModel := NewSelectModel(
		[]string{"yes", "no"},
		nil,
		nil,
		DefaultStyles.Select,
		config.SelectChar)
	model := &MainModel{
		InputPort:   text,
		Bus:         bus,
		SessionID:   types.NewSessionID(),
		Status:      constants.UserInput,
		MessagePort: view,
		Keys:        NewDefaultMainKeyMap(),
		SelectModel: selectModel,
		Config:      config,
		logger:      logger,
		toolManager: toolManager,
	}
	model.SelectModel.SelectCallBack = model.toolManager.Select
	model.SelectModel.QuitCallBack = model.toolManager.Quit
	model.Subscribe()
	return model
}

type MainModel struct {
	Bus              *events.EventBus
	InputPort        textarea.Model
	MessagePort      viewport.Model
	Status           constants.UserStatus
	SessionID        types.SessionID
	MessageID        types.RequestID
	Program          *tea.Program
	AssistantMessage string
	Keys             MainKeyMap
	SelectModel      *SelectModel
	Config           config.ViewConfig
	logger           *zap.Logger
	toolManager      types.ToolManager
	toolModels       map[types.ToolCallID]*ToolModel
}

func (instance *MainModel) SetProgram(program *tea.Program) {
	instance.Program = program
}

func (instance *MainModel) Subscribe() {
	instance.Bus.StreamChunkParsedEvent.Subscribe(constants.Model, func(event events.Event[dto.ParsedChunkData]) {
		if event.Data.RequestID == instance.MessageID && instance.Program != nil {
			instance.Program.Send(StreamUpdate{
				Content:    event.Data.Content,
				IsComplete: event.Data.IsComplete,
			})
		}
	})
	instance.Bus.StreamChunkParsedErrorEvent.Subscribe(constants.Model, func(event events.Event[dto.ParsedChunkErrorData]) {
		if event.Data.RequestID == instance.MessageID && instance.Program != nil {
			instance.Program.Send(StreamUpdate{
				Content:    event.Data.Error,
				IsComplete: true,
			})
		}
	})
	instance.Bus.UpdaetUserStatusEvent.Subscribe(constants.Model, func(event events.Event[dto.UpdateUserStatusData]) {
		instance.Status = event.Data.Status
	})
	instance.Bus.UpdateViewEvent.Subscribe(constants.Model, func(event events.Event[dto.UpdateViewData]) {
		instance.Program.Send(event.Data)
	})
}

func (instance *MainModel) Init() tea.Cmd {
	return textinput.Blink
}

func (instance *MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		instance.UpdateSize(msg)
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, instance.Keys.Exit):
			return instance, tea.Quit
		case key.Matches(msg, instance.Keys.Choice) && instance.Status != constants.ToolDecision:
			if instance.Status == constants.UserInput {
				instance.MessageID = types.NewRequestID()
				userMessage := instance.InputPort.Value()
				instance.Bus.UserInputEvent.Publish(
					events.Event[dto.UserRequestData]{
						Data: dto.UserRequestData{
							SessionID: instance.SessionID,
							RequestID: instance.MessageID,
							Message:   userMessage,
						},
						TimeStamp: time.Now(),
						Source:    constants.Model,
					},
				)
				instance.Status = constants.AssistantInput
			}
			cmd = tea.Println(instance.InputPort.Value())
			instance.InputPort.Reset()
			return instance, cmd
		case key.Matches(msg, instance.Keys.Cancel) && instance.Status != constants.ToolDecision:
			instance.Bus.StreamCancelEvent.Publish(events.Event[dto.StreamCancelData]{
				Data: dto.StreamCancelData{
					RequestID: instance.MessageID,
				},
				TimeStamp: time.Now(),
				Source:    constants.Model,
			})
		}
	case StreamUpdate:
		instance.AddToAssistantMessage(msg.Content)
		if msg.IsComplete {
			cmd = tea.Println(instance.AssistantMessage)
			cmds = append(cmds, cmd)
			instance.AssistantMessage = ""
			instance.MessagePort.SetContent("")
			instance.MessagePort.Height = 0
			instance.Status = constants.UserInput
		}
	case dto.UpdateViewData:
		list := instance.toolManager.ChangedActiveTool()

		for _, activeTool := range list {
			if activeTool.ToolStatus == constants.Call {
				if model, exist := instance.toolModels[activeTool.ToolCallID]; exist {
					model.ToolInfo = activeTool.ToolInfo
				} else {
					instance.toolModels[activeTool.ToolCallID] = NewToolModel(activeTool.ToolInfo, instance.Config)
				}
			} else {
				if model, exist := instance.toolModels[activeTool.ToolCallID]; exist {
					updatedModel, _ := model.Update(UpdateStatus{NewStauts: activeTool.ToolStatus})
					instance.AssistantMessage += fmt.Sprintf("%s\n", updatedModel.View())
					delete(instance.toolModels, activeTool.ToolCallID)
				}
			}
		}
	}
	if len(instance.toolModels) != 0 {
		for _, model := range instance.toolModels {
			model.Update(msg)
		}
	}
	instance.MessagePort, cmd = instance.MessagePort.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if instance.toolManager.IsPedding() {
		instance.Status = constants.ToolDecision
		instance.SelectModel.Update(msg)
	}
	height := (instance.InputPort.Length()+1)/instance.InputPort.Width() + 1
	height = min(height, 5)
	instance.InputPort.SetHeight(height)
	if instance.Status != constants.ToolDecision {
		instance.InputPort, cmd = instance.InputPort.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}
	return instance, tea.Batch(cmds...)
}

func (instance *MainModel) View() string {
	list := make([]string, 0, 3)
	if instance.MessagePort.Height != 0 {
		list = append(list, instance.MessagePort.View())
	}
	if len(instance.toolModels) > 0 {
		for _, toolview := range instance.toolModels {
			list = append(list, toolview.View())
		}
	}
	if instance.Status == constants.ToolDecision {
		list = append(list, instance.SelectModel.View())
	}
	list = append(list, instance.InputPort.View())
	return lipgloss.JoinVertical(lipgloss.Left, list...)
}

func (instance *MainModel) AddToAssistantMessage(newContent string) {
	if len(instance.AssistantMessage) == 0 {
		instance.AssistantMessage = instance.Config.Dot + " " + newContent
	} else {
		instance.AssistantMessage += newContent
	}
	warped := lipgloss.NewStyle().Width(instance.MessagePort.Width).Render(instance.AssistantMessage)
	instance.MessagePort.SetContent(warped)
	instance.MessagePort.Height = instance.MessagePort.TotalLineCount()
}

func (instance *MainModel) UpdateSize(msg tea.WindowSizeMsg) {
	instance.InputPort.SetWidth(msg.Width)
	instance.MessagePort.Width = msg.Width
}
