package viewinterface

import (
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type MainKeyMap struct {
	Choice key.Binding
	Cancel key.Binding
	Exit   key.Binding
}

func NewDefaultMainKeyMap() MainKeyMap {
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

type SelectKeyMap struct {
	Up     key.Binding
	Down   key.Binding
	Choice key.Binding
	Quit   key.Binding
}

func NewDefaultSelectKeyMap() SelectKeyMap {
	return SelectKeyMap{
		Up: key.NewBinding(
			key.WithKeys(tea.KeyUp.String(), "w", "ㅈ"),
		),
		Down: key.NewBinding(
			key.WithKeys(tea.KeyDown.String(), "s", "ㄴ"),
		),
		Choice: key.NewBinding(
			key.WithKeys(tea.KeyEnter.String()),
		),
		Quit: key.NewBinding(
			key.WithKeys(tea.KeyEsc.String()),
		),
	}
}
