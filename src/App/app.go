package app

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

func NewApp() (*App, error) {
	viper.SetConfigFile("env.toml")
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}
	config, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}
	bus, err := events.NewEventBus(config.EventBusConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create event bus: %w", err)
	}
	ollama, err := ollama.NewOllamaService(bus, config.OllamaServiceConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create ollama service: %w", err)
	}
	app := &App{
		bus:                bus,
		model:              viewinterface.NewMainModel(bus, config.ViewConfig),
		mcpService:         mcp.NewMcpService(bus, config.McpServiceConfig),
		toolService:        tool.NewToolService(bus),
		messageSerivce:     message.NewMessageService(bus),
		environmentService: environment.NewEnvironmentService(bus),
		ollamaService:      ollama,
	}
	return app, nil
}

type App struct{
	bus *events.EventBus
	model *viewinterface.MainModel
	mcpService *mcp.McpService
	environmentService *environment.EnvironmentService
	ollamaService *ollama.OllamaService
	toolService *tool.ToolService
	messageSerivce *message.MessageService
}

func (instance *App) Run() {
	program := tea.NewProgram(
		instance.model,
	)
	instance.model.SetProgram(program)
	defer func() {
		instance.bus.Close()
	}()
	if _ , err := program.Run() ; err != nil {
		fmt.Printf("error : %v",err)
	}
}

