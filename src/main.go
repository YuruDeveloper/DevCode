package main

import (
	"DevCode/src/config"
	"DevCode/src/events"
	"DevCode/src/service/environment"
	"DevCode/src/service/llm/ollama"
	"DevCode/src/service/mcp"
	"DevCode/src/service/message"
	"DevCode/src/service/tool"
	"DevCode/src/viewinterface"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigFile("env.toml")
	viper.ReadInConfig()
	config  , err:= config.LoadConfig()
	bus, err := events.NewEventBus()
	if err != nil {
		panic(fmt.Sprintf("Failed to config event bus: %v", err))
	}
	mcp.NewMcpService(bus,config.McpServiceConfig)
	ollama.NewOllamaService(bus,config.OllamaServiceConfig)
	environment.NewEnvironmentService(bus)
	message.NewMessageService(bus)
	tool.NewToolService(bus)
	model := viewinterface.NewMainModel(bus,config.ViewConfig)
	program := tea.NewProgram(
		model,
	)
	model.SetProgram(program)
	defer func() {
		bus.Close()
	}()
	if _, err := program.Run(); err != nil {
		fmt.Printf("오류 발생: %v", err)
	}
}
