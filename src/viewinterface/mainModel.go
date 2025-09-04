package viewinterface

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"time"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
)

type Status int

const (
	UserInput = Status(iota + 1)
	AssistantInput
	ToolDecision
)

type StreamUpdate struct {
	Content    string
	IsComplete bool
}

type PendingTool struct {
	RequestUUID  uuid.UUID
	ToolCallUUID uuid.UUID
}

func NewMainModel(bus *events.EventBus, config config.ViewConfig) *MainModel {
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
		InputPort:        text,
		Bus:              bus,
		SessionUUID:      uuid.New(),
		Status:           UserInput,
		MessagePort:      view,
		Keys:             NewDefaultMainKeyMap(),
		ActiveTools:      make(map[uuid.UUID]*ToolModel),
		PendingToolStack: make([]*PendingTool, 0, 5),
		SelectModel:      selectModel,
		Config:           config, // Add this line
	}
	model.SelectModel.SelectCallBack = model.Select
	model.SelectModel.QuitCallBack = model.Quit
	model.Subscribe()
	return model
}

type MainModel struct {
	Bus              *events.EventBus
	InputPort        textarea.Model
	MessagePort      viewport.Model
	Status           Status
	SessionUUID      uuid.UUID
	MessageUUID      uuid.UUID
	Program          *tea.Program
	AssistantMessage string
	Keys             MainKeyMap
	ActiveTools      map[uuid.UUID]*ToolModel
	PendingToolStack []*PendingTool
	SelectModel      *SelectModel
	Config           config.ViewConfig
}

func (instance *MainModel) SetProgram(program *tea.Program) {
	instance.Program = program
}

func (instance *MainModel) Subscribe() {
	instance.Bus.StreamChunkParsedEvent.Subscribe(constants.Model, func(event events.Event[dto.ParsedChunkData]) {
		if event.Data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdate{
				Content:    event.Data.Content,
				IsComplete: event.Data.IsComplete,
			})
		}
	})
	instance.Bus.StreamChunkParsedErrorEvent.Subscribe(constants.Model, func(event events.Event[dto.ParsedChunkErrorData]) {
		if event.Data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdate{
				Content:    event.Data.Error,
				IsComplete: true,
			})
		}
	})
	instance.Bus.ToolUseReportEvent.Subscribe(constants.Model, func(event events.Event[dto.ToolUseReportData]) {
		instance.Program.Send(event.Data)
	})
	instance.Bus.RequestToolUseEvent.Subscribe(constants.Model, func(event events.Event[dto.ToolUseReportData]) {
		instance.Program.Send(event.Data)
		instance.Program.Send(PendingTool{
			RequestUUID:  event.Data.RequestUUID,
			ToolCallUUID: event.Data.ToolCallUUID,
		})
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
		case key.Matches(msg, instance.Keys.Choice) && instance.Status != ToolDecision:
			if instance.Status == UserInput {
				instance.MessageUUID = uuid.New()
				instance.Bus.UserInputEvent.Publish(
					events.Event[dto.UserRequestData]{
						Data: dto.UserRequestData{
							SessionUUID: instance.SessionUUID,
							RequestUUID: instance.MessageUUID,
							Message:     instance.InputPort.Value(),
						},
						TimeStamp: time.Now(),
						Source:    constants.Model,
					},
				)
				instance.Status = AssistantInput
			}
			cmd = tea.Println(instance.InputPort.Value())
			instance.InputPort.Reset()
			return instance, cmd
		case key.Matches(msg, instance.Keys.Cancel) && instance.Status != ToolDecision:
			instance.Bus.StreamCancelEvent.Publish(events.Event[dto.StreamCancelData]{
				Data: dto.StreamCancelData{
					RequestUUID: instance.MessageUUID,
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
			instance.Status = UserInput
		}
	case dto.ToolUseReportData:
		if msg.ToolStatus != constants.Call {
			model := instance.ActiveTools[msg.ToolCallUUID]
			if model != nil {
				updatedModel, _ := model.Update(UpdateStatus{NewStauts: msg.ToolStatus})
				instance.AssistantMessage += updatedModel.View() + "\n"
				delete(instance.ActiveTools, msg.ToolCallUUID)
			}
		} else {
			instance.ActiveTools[msg.ToolCallUUID] = NewToolModel(msg.ToolInfo, instance.Config)
		}
	case PendingTool:
		instance.PendingToolStack = append(instance.PendingToolStack, &msg)
	}
	if len(instance.ActiveTools) != 0 {
		for _, model := range instance.ActiveTools {
			model.Update(msg)
		}
	}
	instance.MessagePort, cmd = instance.MessagePort.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	if len(instance.PendingToolStack) != 0 {
		instance.Status = ToolDecision
		instance.SelectModel.Update(msg)
	}
	height := (instance.InputPort.Length()+1)/instance.InputPort.Width() + 1
	height = min(height, 5)
	instance.InputPort.SetHeight(height)
	if instance.Status != ToolDecision {
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
	if len(instance.ActiveTools) > 0 {
		for _, toolview := range instance.ActiveTools {
			list = append(list, toolview.View())
		}
	}
	if instance.Status == ToolDecision {
		list = append(list, instance.SelectModel.View())
	}
	list = append(list, instance.InputPort.View())
	return lipgloss.JoinVertical(lipgloss.Left, list...)
}

func (instance *MainModel) Select(selectIndex int) {
	var accept bool
	if selectIndex == 0 {
		accept = true
	} else {
		accept = false
	}
	instance.Bus.UserDecisionEvent.Publish(
		events.Event[dto.UserDecisionData]{
			Data: dto.UserDecisionData{
				RequestUUID:  instance.PendingToolStack[0].RequestUUID,
				ToolCallUUID: instance.PendingToolStack[0].ToolCallUUID,
				Accept:       accept,
			},
			TimeStamp: time.Now(),
			Source:    constants.Model,
		},
	)
	instance.PendingToolStack = instance.PendingToolStack[1:]
	if len(instance.PendingToolStack) == 0 {
		instance.Status = AssistantInput
	}
}

func (instance *MainModel) Quit() {
	instance.Bus.UserDecisionEvent.Publish(
		events.Event[dto.UserDecisionData]{
			Data: dto.UserDecisionData{
				RequestUUID:  instance.PendingToolStack[0].RequestUUID,
				ToolCallUUID: instance.PendingToolStack[0].ToolCallUUID,
				Accept:       false,
			},
			TimeStamp: time.Now(),
			Source:    constants.Model,
		},
	)
	instance.PendingToolStack = instance.PendingToolStack[1:]
	if len(instance.PendingToolStack) == 0 {
		instance.Status = AssistantInput
	}
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
