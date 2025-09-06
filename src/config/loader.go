package config

import (
	"github.com/spf13/viper"
)

func LoadConfig() *Config {
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
		urlText:                    viper.GetString("ollama.url"),
		Model:                      viper.GetString("ollama.model"),
		system:                     viper.GetString("prompt.system"),
		DefaultActiveStreamSize:    viper.GetInt("ollama.default_active_stream_size"),
	}

	eventBusConfig := EventBusConfig{
		PoolSize: viper.GetInt("bus.pool_size"),
	}

	toolServiceConfig := ToolServiceConfig{
		Allowed: viper.GetStringSlice("tool.allowed"),
	}

	viewConfig.Default()
	mcpConfig.Default()
	ollamaConfig.Default()

	config := &Config{
		ViewConfig:          viewConfig,
		McpServiceConfig:    mcpConfig,
		OllamaServiceConfig: ollamaConfig,
		EventBusConfig:      eventBusConfig,
		ToolServiceConfig:   toolServiceConfig,
	}

	return config
}
