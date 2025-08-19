package viewinterface

import (	
	"os"
	"strings"
	"time"

	"UniCode/src/events"
	"UniCode/src/types"

	"github.com/Delta456/box-cli-maker/v2"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/mattn/go-runewidth"
	"golang.org/x/term"
)

type Status int

const (
	UserInput = iota + 1
	AssistantInput
)

func GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // 기본값
	}
	return width
}

type StreamUpdateMsg struct {
	Content    string
	IsComplete bool
}

type ToolApprovalMsg struct {
    Data types.ToolUseReportData
}

type ToolStatusMsg struct {
    Data types.ToolUseReportData
}

type Model struct {
	Input            textarea.Model
	Viewport         viewport.Model
	Spinner          spinner.Model
	MessageBuffer    []rune
	AssistantMessage string
	ModelError       error
	Bus              *events.EventBus
	SessionUUID      uuid.UUID
	MessageUUID      uuid.UUID
	Status           Status
	Insert           bool
	Program          *tea.Program
	Tool             string
}

func InitModel(bus *events.EventBus) *Model {

	text := textarea.New()
	text.CharLimit = 500
	text.SetWidth(GetTerminalWidth())
	text.SetHeight(3)
	text.Focus()
	text.ShowLineNumbers = false

	ViewPort := viewport.New(GetTerminalWidth(), 10)

	Viewspinner := spinner.New()
	Viewspinner.Spinner = spinner.Dot
	Viewspinner.Style = lipgloss.NewStyle().Foreground(types.PrimaryColor)
	model := &Model{
		Input:            text,
		Viewport:         ViewPort,
		Bus:              bus,
		MessageBuffer:    []rune("> "),
		Spinner:          Viewspinner,
		AssistantMessage: "",
		SessionUUID:      uuid.New(),
		Status:           UserInput,
		Insert:           false,
	}
	bus.Subscribe(events.StreamChunkParsedEvent, model)
	bus.Subscribe(events.StreamChunkParsedErrorEvent, model)
	bus.Subscribe(events.RequestToolUseEvent, model)
	bus.Subscribe(events.ToolUseReportEvent, model)
	return model
}

func (instance *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		instance.Spinner.Tick,
	)
}

func (instance *Model) HandleEvent(event events.Event) {
	switch event.Type {
	case events.StreamChunkParsedEvent:
		data := event.Data.(types.ParsedChunkData)
		// 현재 요청과 일치하는 응답만 처리
		if data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdateMsg{
				Content:    data.Content,
				IsComplete: data.IsComplete,
			})
		}
	case events.StreamChunkParsedErrorEvent:
		data := event.Data.(types.ParsedChunkErrorData)
		// 현재 요청과 일치하는 에러만 처리
		if data.RequestUUID == instance.MessageUUID && instance.Program != nil {
			instance.Program.Send(StreamUpdateMsg{
				Content:    data.Error,
				IsComplete: true,
			})
		}
	case events.RequestToolUseEvent:
	case events.ToolUseReportEvent:
	}
}

func (instance *Model) GetID() types.Source {
	return types.Model
}

func (instance *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if instance.Insert {
		cmd := tea.Println(instance.AssistantMessage)
		instance.AssistantMessage = ""
		instance.Status = UserInput
		instance.Insert = false
		return instance, cmd
	}
	switch msg := msg.(type) {
	case StreamUpdateMsg:
		if instance.WillExceedBorder(msg.Content) {
			instance.AssistantMessage += "\n"
		}
		instance.AssistantMessage += msg.Content
		instance.Status = AssistantInput
		if msg.IsComplete {
			instance.Insert = true
		}
		return instance, nil
	case tea.KeyMsg:
		msgkey := msg.String()
		return instance.ProcessUserInput(msgkey)
	}
	return instance, nil
}

func (instance *Model) WillExceedBorder(newContent string) bool {
	if len(instance.AssistantMessage) == 0 {
		return false
	}

	terminalWidth := GetTerminalWidth()
	lastNewlineIndex := strings.LastIndex(instance.AssistantMessage, "\n")

	var currentLineWidth int
	if lastNewlineIndex == -1 {
		// 전체 메시지가 한 줄인 경우
		currentLineWidth = runewidth.StringWidth(instance.AssistantMessage)
	} else {
		// 마지막 줄만 체크
		lastLine := instance.AssistantMessage[lastNewlineIndex+1:]
		currentLineWidth = runewidth.StringWidth(lastLine)
	}

	// 새 콘텐츠를 추가했을 때의 너비
	newContentWidth := runewidth.StringWidth(newContent)
	futureWidth := currentLineWidth + newContentWidth

	// 새 콘텐츠 추가 후 터미널 너비를 초과하면 true
	return futureWidth > terminalWidth
}

func (instance *Model) ProcessUserInput(msg string) (tea.Model, tea.Cmd) {
	switch msg {
	case tea.KeyCtrlC.String():
		return instance, tea.Quit

	case tea.KeyBackspace.String():
		if len(instance.MessageBuffer) > 2 {
			instance.MessageBuffer = instance.MessageBuffer[:len(instance.MessageBuffer)-1]
		}
	case tea.KeyEnter.String():
		if len(instance.MessageBuffer) > 2 {
			var cmd tea.Cmd
			if instance.Status == UserInput {
				cmd = tea.Println(string(instance.MessageBuffer))
				instance.PublishEvent(events.UserInputEvent)
			}
			instance.MessageBuffer = []rune("> ")
			return instance, cmd
		}
	case tea.KeyEsc.String():
		if instance.Status == AssistantInput {
			instance.PublishEvent(events.StreamCancelEvent)
		}
	default:
		if msg == "" || len([]rune(msg)) == 0 {
			return instance, nil
		}
		instance.MessageBuffer = append(instance.MessageBuffer, []rune(msg)...)
	}
	return instance, nil
}

func (instance *Model) PublishEvent(Type events.EventType) {
	var data any
	switch Type {
	case events.UserInputEvent:
		instance.MessageUUID = uuid.New()
		data = types.RequestData{
			SessionUUID: instance.SessionUUID,
			RequestUUID: instance.MessageUUID,
			Message:     string(instance.MessageBuffer[2 : len(instance.MessageBuffer)-1]),
		}
	case events.StreamCancelEvent:
		data = types.StreamCancelData{
			RequestUUID: instance.MessageUUID,
		}
		instance.MessageUUID = uuid.Nil
	}
	instance.Bus.Publish(
		events.Event{
			Type:      Type,
			Data:      data,
			Timestamp: time.Now(),
			Source:    types.Model,
		},
	)
}

func (instance *Model) View() string {
	var builder strings.Builder
	if instance.AssistantMessage != "" {
		builder.WriteString(instance.AssistantMessage + "\n")
	}
	if instance.Tool != "" {
		builder.WriteString(instance.Tool + "\n")
	}
	builder.WriteString(instance.ProcessInputRendering())
	return builder.String()
}

func (instance *Model) ProcessInputRendering() string {
	// cacluate box contentWidth

	contentWidth := max(1, GetTerminalWidth()-4) // 2(테두리) + 4(패딩)
	message := string(instance.MessageBuffer) + "▌"
	//차지하는 크기 계산하기
	displayWidth := runewidth.StringWidth(message)
	if displayWidth%contentWidth != 0 {
		left := contentWidth - displayWidth%contentWidth
		// 빈 공간으로 채울 문자열 생성
		emptyContent := strings.Repeat(" ", left)
		message += emptyContent
	}
	//box make
	textAreaBox := box.New(box.Config{
		Px:            1,
		Py:            0,
		AllowWrapping: true,
		WrappingLimit: contentWidth,
		Type:          "Round",
		Color:         "HiBlack",
	})

	return textAreaBox.String("", message)
}
