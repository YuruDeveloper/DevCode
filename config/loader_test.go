package config

import (
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfig(t *testing.T) {
	// Setup viper with test values
	viper.Reset()

	// Set test values
	viper.Set("view.dot", "★")
	viper.Set("view.select", "→")
	viper.Set("mcp.name", "TestApp")
	viper.Set("mcp.version", "1.0.0")
	viper.Set("server.name", "TestServer")
	viper.Set("server.version", "2.0.0")
	viper.Set("ollama.message_limit", 50)
	viper.Set("ollama.default_system_message_length", 5)
	viper.Set("ollama.environment_info", "Test environment")
	viper.Set("ollama.default_tool_size", 5)
	viper.Set("ollama.default_request_contents_size", 5)
	viper.Set("ollama.default_tool_call_size", 3)
	viper.Set("ollama.url", "http://test:8080")
	viper.Set("ollama.model", "test-model")
	viper.Set("prompt.system", "/nonexistent/prompt.txt")
	viper.Set("ollama.default_active_stream_size", 5)
	viper.Set("bus.pool_size", 5000)
	viper.Set("tool.allowed", []string{"Read", "Write", "List"})

	config := LoadConfig()
	require.NotNil(t, config)

	// Test ViewConfig
	assert.Equal(t, "★", config.ViewConfig.Dot)
	assert.Equal(t, "→", config.ViewConfig.SelectChar)

	// Test McpServiceConfig
	assert.Equal(t, "TestApp", config.McpServiceConfig.Name)
	assert.Equal(t, "1.0.0", config.McpServiceConfig.Version)
	assert.Equal(t, "TestServer", config.McpServiceConfig.ServerName)
	assert.Equal(t, "2.0.0", config.McpServiceConfig.ServerVersion)

	// Test OllamaServiceConfig
	assert.Equal(t, 50, config.OllamaServiceConfig.MessageLimit)
	assert.Equal(t, 5, config.OllamaServiceConfig.DefaultSystemMessageLength)
	assert.Equal(t, "Test environment", config.OllamaServiceConfig.EnvironmentInfo)
	assert.Equal(t, 5, config.OllamaServiceConfig.DefaultToolSize)
	assert.Equal(t, 5, config.OllamaServiceConfig.DefaultRequestContentsSize)
	assert.Equal(t, 3, config.OllamaServiceConfig.DefaultToolCallSize)
	assert.Equal(t, "test-model", config.OllamaServiceConfig.Model)
	assert.Equal(t, 5, config.OllamaServiceConfig.DefaultActiveStreamSize)
	assert.NotNil(t, config.OllamaServiceConfig.Url)
	assert.Equal(t, "http://test:8080", config.OllamaServiceConfig.Url.String())

	// Test EventBusConfig
	assert.Equal(t, 5000, config.EventBusConfig.PoolSize)

	// Test ToolServiceConfig
	assert.Equal(t, []string{"Read", "Write", "List"}, config.ToolServiceConfig.Allowed)
}

func TestLoadConfig_WithDefaults(t *testing.T) {
	// Reset viper to empty state
	viper.Reset()

	config := LoadConfig()
	require.NotNil(t, config)

	// Test that Default() methods were called and backup values are used
	assert.Equal(t, BackupDot, config.ViewConfig.Dot)
	assert.Equal(t, BackupSelectChar, config.ViewConfig.SelectChar)
	assert.Equal(t, BackupName, config.McpServiceConfig.Name)
	assert.Equal(t, BackupVersion, config.McpServiceConfig.Version)
	assert.Equal(t, BackupMessageLimit, config.OllamaServiceConfig.MessageLimit)
	assert.Equal(t, BackupPoolSize, config.EventBusConfig.PoolSize)
}

func TestLoadConfig_PartialConfiguration(t *testing.T) {
	// Reset viper and set only some values
	viper.Reset()
	viper.Set("view.dot", "♦")
	viper.Set("ollama.message_limit", 200)
	viper.Set("bus.pool_size", 15000)

	config := LoadConfig()
	require.NotNil(t, config)

	// Test that set values are used
	assert.Equal(t, "♦", config.ViewConfig.Dot)
	assert.Equal(t, 200, config.OllamaServiceConfig.MessageLimit)
	assert.Equal(t, 15000, config.EventBusConfig.PoolSize)

	// Test that unset values use defaults
	assert.Equal(t, BackupSelectChar, config.ViewConfig.SelectChar)
	assert.Equal(t, BackupName, config.McpServiceConfig.Name)
}
