package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigStruct(t *testing.T) {
	// Test that Config struct can be instantiated
	config := &Config{
		ViewConfig:          ViewConfig{},
		McpServiceConfig:    McpServiceConfig{},
		OllamaServiceConfig: OllamaServiceConfig{},
		EventBusConfig:      EventBusConfig{},
		ToolServiceConfig:   ToolServiceConfig{},
	}

	assert.NotNil(t, config)
	assert.IsType(t, ViewConfig{}, config.ViewConfig)
	assert.IsType(t, McpServiceConfig{}, config.McpServiceConfig)
	assert.IsType(t, OllamaServiceConfig{}, config.OllamaServiceConfig)
	assert.IsType(t, EventBusConfig{}, config.EventBusConfig)
	assert.IsType(t, ToolServiceConfig{}, config.ToolServiceConfig)
}
