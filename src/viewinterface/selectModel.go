package viewinterface

import (
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type SelectKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Choice key.Binding
	Quit   key.Binding
}

func NewDefaultSelectKeyMap() SelectKeyMap {
	return SelectKeyMap{
		Up: key.NewBinding(
			key.WithKeys(tea.KeyUp.String(), "w"),
		),
		Down: key.NewBinding(
			key.WithKeys(tea.KeyDown.String(), "s"),
		),
		Choice: key.NewBinding(
			key.WithKeys(tea.KeyEnter.String()),
		),
		Quit: key.NewBinding(
			key.WithKeys(tea.KeyEsc.String()),
		),
	}
}

func NewSelectModel(choices []string, selectCallBack func(int), quitCallBack func(), style lipgloss.Style) *SelectModel {
	return &SelectModel{
		Choices:        choices,
		SecltedIndex:   0,
		Keys:           NewDefaultSelectKeyMap(),
		SelectCallBack: selectCallBack,
		QuitCallBack:   quitCallBack,
		Style:          style,
	}
}

type SelectModel struct {
	Choices        []string
	SecltedIndex   int
	Keys           SelectKeyMap
	SelectCallBack func(int)
	QuitCallBack   func()
	Style          lipgloss.Style
}

func (instance *SelectModel) Init() tea.Cmd {
	return nil
}

func (instance *SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(tea.KeyMsg); ok {
		switch {
		case key.Matches(msg, instance.Keys.Up):
			instance.SecltedIndex++
		case key.Matches(msg, instance.Keys.Down):
			instance.SecltedIndex--
		case key.Matches(msg, instance.Keys.Choice):
			instance.SelectCallBack(instance.SecltedIndex)
		case key.Matches(msg, instance.Keys.Quit):
			instance.QuitCallBack()
		}
	}
	instance.SecltedIndex = (instance.SecltedIndex + len(instance.Choices)) % len(instance.Choices)
	return instance, nil
}

func (instance *SelectModel) View() string {
	var builder strings.Builder
	for index, choice := range instance.Choices {
		if index == instance.SecltedIndex {
			builder.WriteString("> ")
		} else {
			builder.WriteString("  ")
		}
		builder.WriteString(choice)
		builder.WriteString("\n")
	}
	return instance.Style.Render(
		builder.String(),
	)
}
