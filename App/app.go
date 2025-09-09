package app

import (
	devcodeerror "DevCode/DevCodeError"
	"DevCode/config"
	"DevCode/events"
	toolManager "DevCode/manager/tool"
	"DevCode/module/environment"
	"DevCode/module/llm/ollama"
	"DevCode/module/mcp"
	"DevCode/module/message"
	"DevCode/module/tool"
	"DevCode/viewinterface"
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
	manager := toolManager.NewToolManager(bus, logger)
	app := &App{
		bus:               bus,
		toolManager:       manager,
		model:             viewinterface.NewMainModel(bus, config.ViewConfig, logger, manager),
		mcpModule:         mcp.NewMcpModule(bus, config.McpServiceConfig, logger),
		toolModule:        tool.NewToolModule(bus, config.ToolServiceConfig, logger),
		messageModule:     message.NewMessageModule(bus, logger),
		environmentModule: environment.NewEnvironmentModule(bus, logger),
		ollamaModule:      ollama.NewOllamaModule(bus, config.OllamaServiceConfig, logger),
		logger:            logger,
	}
	return app, nil
}

type App struct {
	bus               *events.EventBus
	toolManager       *toolManager.ToolManager
	model             *viewinterface.MainModel
	mcpModule         *mcp.McpModule
	environmentModule *environment.EnvironmentModule
	ollamaModule      *ollama.OllamaModule
	toolModule        *tool.ToolModule
	messageModule     *message.MessageModule
	logger            *zap.Logger
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
