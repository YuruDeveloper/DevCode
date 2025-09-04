package config

import (
	"github.com/spf13/viper"
)

// LoadConfig loads configuration from env.toml file
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
		viewConfig.Dot = "●"
	}
	if viewConfig.SelectChar == "" {
		viewConfig.SelectChar = ">"
	}
	if ollamaConfig.MessageLimit == 0 {
		ollamaConfig.MessageLimit = 100
	}
	if ollamaConfig.DefaultSystemMessageLength == 0 {
		ollamaConfig.DefaultSystemMessageLength = 10
	}
	if ollamaConfig.EnvironmentInfo == "" {
		ollamaConfig.EnvironmentInfo = "Here is useful information about the environment you are running in:\n"
	}
	if ollamaConfig.DefaultToolSize == 0 {
		ollamaConfig.DefaultToolSize = 10
	}
	if ollamaConfig.DefaultRequestContentsSize == 0 {
		ollamaConfig.DefaultRequestContentsSize = 10
	}
	if ollamaConfig.DefaultToolCallSize == 0 {
		ollamaConfig.DefaultToolCallSize = 5
	}
	if ollamaConfig.DefaultActiveStreamSize == 0 {
		ollamaConfig.DefaultActiveStreamSize = 10
	}

	config := &Config{
		ViewConfig:          viewConfig,
		McpServiceConfig:    mcpConfig,
		OllamaServiceConfig: ollamaConfig,
		EventBusConfig:      eventBusConfig,
	}

	return config, nil
}
