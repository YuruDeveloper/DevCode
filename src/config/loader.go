package config

import (
	"github.com/spf13/viper"
)

const (
	BackupDot                        = "●"
	BackupSelectChar                 = ">"
	BackupMessageLimit               = 100
	BackupDefaultSystemMessageLength = 10
	BackupEnviromentInfo             = "Here is useful information about the environment you are running in:\n"
	BackupDefaultToolSize            = 10
	BackupDefaultRequestContentsSize = 10
	BackupToolCallSize               = 5
	BackupDefaultActiveStreamSzie    = 10
	BackupName                       = "DevCode"
	BackupVersion                    = "0.0.1"
	BackupUrl                        = "http://127.0.0.1:11434"
	BackupModel                      = "llama3.1:8b"
	BackupPoolSize                   = 10000
)

func LoadConfig() (*Config, error) {
	viewConfig := ViewConfig{
		Dot:        viper.GetString("view.dot"),
		SelectChar: viper.GetString("view.select"),
	}

	mcpConfig := McpServiceConfig{
		Name:          viper.GetString("mcp.name"),
		Version:       viper.GetString("mcp.version"),
		ServerName:    viper.GetString("server.name"),
		ServerVersion: viper.GetString("server.version"),
	}

	ollamaConfig := OllamaServiceConfig{
		MessageLimit:               viper.GetInt("ollama.message_limit"),
		DefaultSystemMessageLength: viper.GetInt("ollama.default_system_message_length"),
		EnvironmentInfo:            viper.GetString("ollama.environment_info"),
		DefaultToolSize:            viper.GetInt("ollama.default_tool_size"),
		DefaultRequestContentsSize: viper.GetInt("ollama.default_request_contents_size"),
		DefaultToolCallSize:        viper.GetInt("ollama.default_tool_call_size"),
		Url:                        viper.GetString("ollama.url"),
		Model:                      viper.GetString("ollama.model"),
		System:                     viper.GetString("prompt.system"),
		DefaultActiveStreamSize:    viper.GetInt("ollama.default_active_stream_size"),
	}

	eventBusConfig := EventBusConfig{
		PoolSize: viper.GetInt("bus.pool_size"),
	}

	// Set default values if not configured
	if viewConfig.Dot == "" {
		viewConfig.Dot = BackupDot
	}
	if viewConfig.SelectChar == "" {
		viewConfig.SelectChar = BackupSelectChar
	}

	if mcpConfig.Name == "" {
		mcpConfig.Name = BackupName
	}

	if mcpConfig.Version == "" {
		mcpConfig.Version = BackupVersion
	}

	if mcpConfig.ServerName == "" {
		mcpConfig.ServerName = BackupName
	}

	if mcpConfig.ServerVersion == "" {
		mcpConfig.ServerVersion = BackupVersion
	}

	if ollamaConfig.Url == "" {
		ollamaConfig.Url = BackupUrl
	}

	if ollamaConfig.Model == "" {
		ollamaConfig.Model = BackupModel
	}

	if ollamaConfig.MessageLimit == 0 {
		ollamaConfig.MessageLimit = BackupMessageLimit
	}

	if ollamaConfig.DefaultSystemMessageLength == 0 {
		ollamaConfig.DefaultSystemMessageLength = BackupDefaultSystemMessageLength
	}

	if ollamaConfig.EnvironmentInfo == "" {
		ollamaConfig.EnvironmentInfo = BackupEnviromentInfo
	}

	if ollamaConfig.DefaultToolSize == 0 {
		ollamaConfig.DefaultToolSize = BackupDefaultToolSize
	}

	if ollamaConfig.DefaultRequestContentsSize == 0 {
		ollamaConfig.DefaultRequestContentsSize = BackupDefaultRequestContentsSize
	}

	if ollamaConfig.DefaultToolCallSize == 0 {
		ollamaConfig.DefaultToolCallSize = BackupToolCallSize
	}

	if ollamaConfig.DefaultActiveStreamSize == 0 {
		ollamaConfig.DefaultActiveStreamSize = BackupDefaultActiveStreamSzie
	}

	if eventBusConfig.PoolSize == 0 {
		eventBusConfig.PoolSize = BackupPoolSize
	}

	config := &Config{
		ViewConfig:          viewConfig,
		McpServiceConfig:    mcpConfig,
		OllamaServiceConfig: ollamaConfig,
		EventBusConfig:      eventBusConfig,
	}

	return config, nil
}
