package main

import (
	"UniCode/src/events"
	"UniCode/src/service/eniroment"
	"UniCode/src/service/llm/ollama"
	"UniCode/src/service/mcp"
	"UniCode/src/service/message"
	"UniCode/src/service/tool"
	"UniCode/src/viewinterface"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	viper.SetConfigFile("env.toml")
	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Sprintf("Failed to read config file: %v", err))
	}
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("Failed to config logger: %v", err))
	}
	bus, err := events.NewEventBus(logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to config event bus: %v", err))
	}
	mcp.NewMcpService(bus, logger)
	ollama.NewOllamaService(bus, logger)
	eniroment.NewEnvironmentService(bus, logger)
	message.NewMessageService(bus, logger)
	tool.NewToolService(bus, logger)
	model := viewinterface.NewMainModel(bus)
	program := tea.NewProgram(
		model,
	)
	model.SetProgram(program)

	if _, err := program.Run(); err != nil {
		fmt.Printf("오류 발생: %v", err)
	}
}
