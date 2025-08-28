package main

import (
	"DevCode/src/events"
	"DevCode/src/service/environment"
	"DevCode/src/service/llm/ollama"
	"DevCode/src/service/mcp"
	"DevCode/src/service/message"
	"DevCode/src/service/tool"
	"DevCode/src/viewinterface"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	viper.SetConfigFile("env.toml")
	viper.ReadInConfig()
file, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        panic(err)
    }
    defer file.Close()

    // Encoder 설정 (JSON 또는 Console 형식)
    encoderConfig := zapcore.EncoderConfig{
        TimeKey:        "timestamp",
        LevelKey:       "level",
        NameKey:        "logger",
        CallerKey:      "caller",
        FunctionKey:    zapcore.OmitKey,
        MessageKey:     "msg",
        StacktraceKey:  "stacktrace",
        LineEnding:     zapcore.DefaultLineEnding,
        EncodeLevel:    zapcore.LowercaseLevelEncoder,
        EncodeTime:     zapcore.ISO8601TimeEncoder,
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }

    // File Writer 설정
    fileWriter := zapcore.AddSync(file)
    
    // Core 생성
    core := zapcore.NewCore(
        zapcore.NewJSONEncoder(encoderConfig),  // JSON 형식
        fileWriter,
        zapcore.InfoLevel,
    )

    // Logger 생성
    logger  := zap.New(core, zap.AddCaller())
	bus, err := events.NewEventBus(logger)
	if err != nil {
		panic(fmt.Sprintf("Failed to config event bus: %v", err))
	}
	mcp.NewMcpService(bus, logger)
	ollama.NewOllamaService(bus, logger)
	environment.NewEnvironmentService(bus, logger)
	message.NewMessageService(bus, logger)
	tool.NewToolService(bus, logger)
	model := viewinterface.NewMainModel(bus)
	program := tea.NewProgram(
		model,
	)
	model.SetProgram(program)
	defer func() {
		bus.Close()
		logger.Sync()
		file.Close()
	}()
	if _, err := program.Run(); err != nil {
		fmt.Printf("오류 발생: %v", err)
	}
}
