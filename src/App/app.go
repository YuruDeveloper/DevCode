package app

import (
	devcodeerror "DevCode/src/DevCodeError"
	"DevCode/src/config"
	"DevCode/src/events"
	"DevCode/src/service/environment"
	"DevCode/src/service/llm/ollama"
	"DevCode/src/service/mcp"
	"DevCode/src/service/message"
	"DevCode/src/service/tool"
	"DevCode/src/viewinterface"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func NewApp() (*App, error) {
	viper.SetConfigFile("env.toml")
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, devcodeerror.Wrap(err, devcodeerror.FailLoggerSetup, "Fail LoggerSetup")
	}
	if err := viper.ReadInConfig(); err != nil {
		return nil, devcodeerror.Wrap(err, devcodeerror.FailReadConfig, "Fail Read Config")
	}
	config := config.LoadConfig()
	bus, err := events.NewEventBus(config.EventBusConfig, logger)
	if err != nil {
		return nil, err
	}
	app := &App{
		bus:                bus,
		model:              viewinterface.NewMainModel(bus, config.ViewConfig),
		mcpService:         mcp.NewMcpService(bus, config.McpServiceConfig),
		toolService:        tool.NewToolService(bus, config.ToolServiceConfig),
		messageService:     message.NewMessageService(bus),
		environmentService: environment.NewEnvironmentService(bus, logger),
		ollamaService:      ollama.NewOllamaService(bus, config.OllamaServiceConfig),
		logger:             logger,
	}
	return app, nil
}

type App struct {
	bus                *events.EventBus
	model              *viewinterface.MainModel
	mcpService         *mcp.McpService
	environmentService *environment.EnvironmentService
	ollamaService      *ollama.OllamaService
	toolService        *tool.ToolService
	messageService     *message.MessageService
	logger             *zap.Logger
}

func (instance *App) Run() {
	program := tea.NewProgram(
		instance.model,
	)
	instance.model.SetProgram(program)
	defer func() {
		instance.bus.Close()
	}()
	if _, err := program.Run(); err != nil {
		instance.logger.Error("", zap.Error(devcodeerror.Wrap(err, devcodeerror.FailRunApp, "Fail Run App")))
		return
	}
}
