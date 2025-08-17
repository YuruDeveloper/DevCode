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


func  GetTerminalWidth() int {
	width, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		width = 80 // 기본값
	}
	return width
}

type Model struct{
	Input textarea.Model
	Viewport viewport.Model
	Spinner spinner.Model
	MessageBuffer []rune
	InsertMessage string
	ModelError error
	Bus *events.EventBus
	SessionUUID uuid.UUID
}

func InitModel(bus *events.EventBus) *Model {
	
	text := textarea.New()
	text.CharLimit = 500
	text.SetWidth(GetTerminalWidth())
	text.SetHeight(3)
	text.Focus()
	text.ShowLineNumbers = false

	ViewPort := viewport.New(GetTerminalWidth(),3)

	Viewspinner := spinner.New()
	Viewspinner.Spinner = spinner.Dot
	Viewspinner.Style = lipgloss.NewStyle().Foreground(types.PrimaryColor)
	return &Model{
		Input: text,
		Viewport: ViewPort,
		Bus: bus,
		MessageBuffer: []rune("> "),
		Spinner: Viewspinner,
		InsertMessage: "",
		SessionUUID: uuid.New(),
	}
}

func (instance *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		instance.Spinner.Tick,
	)
}

func (instance *Model) HandleEvent(event events.Event) {
	if event.Type == events.AssistantOutputEvent {
		instance.InsertMessage = event.Data.(string)
	}
}

func (instance *Model) GetID() types.Source {
	return types.Model
}

func (instance *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if instance.InsertMessage != "" {
		cmd := tea.Println("*  " + instance.InsertMessage)
		instance.InsertMessage = ""
		return instance , cmd
	}
	switch  msg:= msg.(type) {
		case tea.KeyMsg:
			msgkey := msg.String()
			switch msgkey {
				case tea.KeyCtrlC.String():
					return instance , tea.Quit
				case tea.KeyBackspace.String():
					if len(instance.MessageBuffer) > 2 {
						instance.MessageBuffer = instance.MessageBuffer[:len(instance.MessageBuffer) -1]
					}
				case tea.KeyEnter.String():
					if len(instance.MessageBuffer) > 2 {
						cmd := tea.Println(string(instance.MessageBuffer))
						instance.Bus.Publish(events.Event{
							Type: events.UserInputEvent,
							Data: types.RequestData {
								SessionUUID: instance.SessionUUID,
								RequestUUID: uuid.New(),
								Message: string(instance.MessageBuffer[2:]),
							},
							Timestamp: time.Now(),
							Source: types.Model,
						})
						instance.MessageBuffer = []rune("> ")
						return instance,cmd
					}
				default : 
					instance.MessageBuffer = append(instance.MessageBuffer, []rune(msgkey)...)
			}
			
	}
	return instance , nil
}

func (instance *Model) View() string {
	var builder strings.Builder
	builder.WriteString(instance.ProcessInput())
	return builder.String()
}


func (instance *Model) ProcessInput() string {
	// cacluate box contentWidth

	contentWidth := GetTerminalWidth() - 4 // 2(테두리) + 4(패딩)
	if contentWidth < 1 {
		contentWidth = 1
	}
	message := string(instance.MessageBuffer)
	//차지하는 크기 계산하기
	displayWidth := runewidth.StringWidth(message)
	if displayWidth % contentWidth != 0 {
		left := contentWidth - displayWidth % contentWidth
		// 빈 공간으로 채울 문자열 생성
		emptyContent := strings.Repeat(" ", left)
		message += emptyContent
	}
	//box make 
	textAreaBox := box.New(box.Config{
		Px: 1,
		Py: 0,
		AllowWrapping: true,
		WrappingLimit: contentWidth,
		Type: "Round",
		Color: "HiBlack",
	})

	return textAreaBox.String("",message) 
}

