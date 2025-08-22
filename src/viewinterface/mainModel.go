package viewinterface

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"fmt"
	"strings"
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

type MainKeyMap struct {
	Choice key.Binding
	Cancel key.Binding
	Exit   key.Binding
}

type ToolStatusUpdate struct{}

type ToolStatusEnd struct {
	Info string
}

func NewDefualtMainKeyMap() MainKeyMap {
	return MainKeyMap{
		Choice: key.NewBinding(
			key.WithKeys(tea.KeyEnter.String()),
		),
		Cancel: key.NewBinding(
			key.WithKeys(tea.KeyEsc.String()),
		),
		Exit: key.NewBinding(
			key.WithKeys(tea.KeyCtrlC.String()),
		),
	}
}
func NewMainModel(bus *events.EventBus) *MainModel {
	text := textarea.New()
	text.Focus()

	text.SetHeight(1)
	text.ShowLineNumbers = false
	inputStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		PaddingLeft(1).
		PaddingRight(2).
		BorderForeground(lipgloss.ANSIColor(8))
	text.FocusedStyle.Base = inputStyle
	text.BlurredStyle.Base = inputStyle
	text.SetPromptFunc(2, func(lineIdx int) string {
		if lineIdx == 0 {
			return "> "
		}
		return ""
	})
	view := viewport.New(1, 0)
	model := &MainModel{
		InputPort:   text,
		Bus:         bus,
		SessionUUID: uuid.New(),
		Status:      UserInput,
		MessagePort: view,
		Keys:        NewDefualtMainKeyMap(),
		ActiveTools: make(map[string]types.ToolUseReportData),
		ToolBlinkShow: true,
	}
	bus.Subscribe(events.StreamChunkParsedEvent, model)
	bus.Subscribe(events.StreamChunkParsedErrorEvent, model)
	bus.Subscribe(events.RequestToolUseEvent, model)
	bus.Subscribe(events.ToolUseReportEvent, model)
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
	ActiveTools map[string]types.ToolUseReportData
	ToolBlinkShow bool
}

func (instance *MainModel) SetProgram(program *tea.Program) {
	instance.Program = program
}

func (instance *MainModel) HandleEvent(event events.Event) {
	switch event.Type {
	case events.StreamChunkParsedEvent:
		data := event.Data.(types.ParsedChunkData)
		if data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdate{
				Content:    data.Content,
				IsComplete: data.IsComplete,
			})
		}
	case events.StreamChunkParsedErrorEvent:
		data := event.Data.(types.ParsedChunkErrorData)
		if data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdate{
				Content:    data.Error,
				IsComplete: true,
			})
		}
	case events.ToolUseReportEvent:
		data := event.Data.(types.ToolUseReportData)
		if data.ToolStatus != types.Call {
			delete(instance.ActiveTools,data.ToolCall.String())
			statusSymbol := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10))
			if data.ToolStatus == types.Error {
				statusSymbol.Foreground(lipgloss.ANSIColor(9))
			}
			if instance.Program != nil {
				instance.Program.Send(ToolStatusEnd {
					Info: fmt.Sprintf("%s %s",statusSymbol.Render("●"),data.ToolInfo),
				})
			}
		} else {
			instance.ActiveTools[data.ToolCall.String()] = data
			if instance.Program != nil {
				instance.Program.Send(ToolStatusUpdate{})
			}
		}
	case events.RequestToolUseEvent:
		//creet tool stauts view
		//show tool use desision add 
	}

}

func (instance *MainModel) GetID() types.Source {
	return types.Model
}

func (instance *MainModel) Init() tea.Cmd {
	return tea.Batch(textinput.Blink,tea.Tick(time.Millisecond * 500, func (time.Time) tea.Msg { return  ToolStatusUpdate{}}))
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
				instance.PublishEvent(events.UserInputEvent, types.RequestData{
					SessionUUID: instance.SessionUUID,
					RequestUUID: instance.MessageUUID,
					Message:     instance.InputPort.Value(),
				})
				instance.Status = AssistantInput
			}
			cmd = tea.Println(instance.InputPort.Value())
			instance.InputPort.Reset()
			return instance, cmd
		case key.Matches(msg, instance.Keys.Cancel) && instance.Status != ToolDecision:
			instance.PublishEvent(events.StreamCancelEvent, types.StreamCancelData{
				RequestUUID: instance.MessageUUID,
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
	case ToolStatusEnd:
		cmd = tea.Println(msg.Info)
		cmds = append(cmds, cmd)
	case ToolStatusUpdate:
	}
	instance.MessagePort, cmd = instance.MessagePort.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	height := (instance.InputPort.Length()+1)/instance.InputPort.Width() + 1
	height = min(height, 5)
	instance.InputPort.SetHeight(height)
	instance.InputPort, cmd = instance.InputPort.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return instance, tea.Batch(cmds...)
}

func (instance *MainModel) View() string {
	return lipgloss.JoinVertical(lipgloss.Left, instance.MessagePort.View(), instance.ToolCallView(),instance.InputPort.View())
}

func (instance *MainModel) ToolCallView() string {
	if len(instance.ActiveTools) == 0 {
		return ""
	}
	var builder strings.Builder
	for _ , tool := range instance.ActiveTools {
		symbol := " "
		if instance.ToolBlinkShow {
			symbol = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8)).Render("●")
			instance.ToolBlinkShow = false
		} else {
			instance.ToolBlinkShow = true
		}
		builder.WriteString(fmt.Sprintf("%s %s\n",symbol,tool.ToolInfo))
	}
	return builder.String()
}

func (instance *MainModel) AddToAssistantMessage(newContent string) {
	if len(instance.AssistantMessage) == 0 {
		instance.AssistantMessage = "● " + newContent
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

func (instance *MainModel) PublishEvent(eventType events.EventType, data any) {
	instance.Bus.Publish(
		events.Event{
			Type:      eventType,
			Data:      data,
			Timestamp: time.Now(),
			Source:    types.Model,
		},
	)
}
