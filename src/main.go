package main

import (
	"UniCode/src/service"
	"UniCode/src/viewinterface"
	"UniCode/src/events"
	"fmt"
	"github.com/spf13/viper"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	viper.SetConfigFile("env.toml")
	viper.ReadInConfig()
	bus := events.NewEventBus()
	service.NewMcpService(bus)
	service.NewOllamaService(bus)
	service.NewEnvironmentService(bus)
	model := viewinterface.InitModel(bus)
	program := tea.NewProgram(
		model,
	)
	
	if _, err := program.Run(); err != nil {
		fmt.Printf("오류 발생: %v", err)
	}
}