package main

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/viewinterface"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigFile("env.toml")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Failed to read config file: %v", err))
	}
	bus := events.NewEventBus()
	service.NewMcpService(bus)
	service.NewOllamaService(bus)
	service.NewEnvironmentService(bus)
	service.NewMessageService(bus)
	service.NewToolService(bus)
	model := viewinterface.NewModel(bus)
	program := tea.NewProgram(
		model,
	)
	model.SetProgram(program)

	if _, err := program.Run(); err != nil {
		fmt.Printf("오류 발생: %v", err)
	}
}
