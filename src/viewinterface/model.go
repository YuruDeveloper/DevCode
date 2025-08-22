package viewinterface

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"strings"
	"time"

	"os"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type Status int 

const (
	UserInput = Status(iota + 1)
	AssistantInput
	ToolDecision
)

type StreamUpdate struct {
	Content string
	IsComplete bool
}

func NewModel(bus *events.EventBus) *Model {
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
	model := &Model{
		TextView: text,
		Bus:      bus,
		SessionUUID: uuid.New(),
		Status: UserInput,
	}
	bus.Subscribe(events.StreamChunkParsedEvent,model)
	bus.Subscribe(events.StreamChunkParsedErrorEvent,model)
	bus.Subscribe(events.RequestToolUseEvent,model)
	bus.Subscribe(events.ToolUseReportEvent,model)
	return model
}

type Model struct {
	Bus      *events.EventBus
	TextView textarea.Model
	Status Status
	SessionUUID uuid.UUID
	MessageUUID uuid.UUID
	AssistanceMessage string
	Program *tea.Program
}

func (instance *Model) SetProgram(program *tea.Program) {
	instance.Program = program
}

func (instance *Model) HandleEvent(event events.Event) {
	switch event.Type {
	case events.StreamChunkParsedEvent:
		data := event.Data.(types.ParsedChunkData)
		if data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdate {
				Content: data.Content,
				IsComplete: data.IsComplete,
			})
		}
	case events.StreamChunkParsedErrorEvent:
		data := event.Data.(types.ParsedChunkErrorData)
		if data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdate {
				Content: data.Error,
				IsComplete: true,
			})
		}
	}

}

func (instance *Model) GetID() types.Source {
	return types.Model
}

func (instance *Model) Init() tea.Cmd {
	return textinput.Blink
}

func (instance *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		instance.UpdateSize(msg)
	case tea.KeyMsg:
		switch msg.String() {
		case tea.KeyCtrlC.String():
			return instance, tea.Quit
		case tea.KeyEnter.String():
			if instance.Status == ToolDecision {
				return instance , cmd
			}
			if instance.Status == UserInput {
				instance.MessageUUID = uuid.New()
				instance.PublishEvent(events.UserInputEvent,types.RequestData {
					SessionUUID: instance.SessionUUID,
					RequestUUID: instance.MessageUUID,
					Message: instance.TextView.Value(),
				})
				instance.Status = AssistantInput
			} 
			cmd = tea.Println(instance.TextView.Value())
			instance.TextView.Reset()
			return instance , cmd
		}
	case StreamUpdate:
		instance.AddToAssistantMessage(msg.Content)
		if msg.IsComplete {
			cmd =  tea.Println(instance.AssistanceMessage)
			cmds = append(cmds, cmd)
			instance.AssistanceMessage = ""
			instance.Status = UserInput
		}
	}

	height := (instance.TextView.Length() + 1)/instance.TextView.Width() + 1
	height = min(height, 5)
	instance.TextView.SetHeight(height)
	instance.TextView, cmd = instance.TextView.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return instance, tea.Batch(cmds...)
}

func (instance *Model) View() string {
	return lipgloss.JoinVertical(lipgloss.Left,instance.AssistanceMessage,instance.TextView.View())
}

func (instance *Model) AddToAssistantMessage(newContent string) {
	if len(instance.AssistanceMessage) == 0 {
		instance.AssistanceMessage +=  " * " + newContent
	}
 
	lastNewlineIndex := strings.LastIndex(instance.AssistanceMessage,"\n")

	var currentLineWidth int 
	if lastNewlineIndex == -1 {
		currentLineWidth = runewidth.StringWidth(instance.AssistanceMessage)
	} else {
		currentLineWidth = runewidth.StringWidth(instance.AssistanceMessage[lastNewlineIndex+1:])
	}

	newContentWidth := runewidth.StringWidth(newContent)
	if newContentWidth + currentLineWidth > instance.GetTerminalWidth() {
		instance.AssistanceMessage += "\n"
	}
	instance.AssistanceMessage += newContent
}

func (instance *Model) GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // 기본값
	}
	return width
}

func (instance *Model) UpdateSize(msg tea.WindowSizeMsg) {
	instance.TextView.SetWidth(msg.Width)
}

func (instance *Model) PublishEvent(eventType events.EventType,data any) {
	instance.Bus.Publish(
		events.Event{
			Type: eventType,
			Data: data,
			Timestamp: time.Now(),
			Source: types.Model,
		},
	)
}
