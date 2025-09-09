package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolServiceConfig_Default(t *testing.T) {
	tests := []struct {
		name     string
		initial  ToolServiceConfig
		expected ToolServiceConfig
	}{
		{
			name:    "Empty config should remain empty",
			initial: ToolServiceConfig{},
			expected: ToolServiceConfig{
				Allowed: nil,
			},
		},
		{
			name: "Config with Allowed set should keep Allowed",
			initial: ToolServiceConfig{
				Allowed: []string{"Read", "Write", "List"},
			},
			expected: ToolServiceConfig{
				Allowed: []string{"Read", "Write", "List"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.initial
			config.Default()
			assert.Equal(t, tt.expected, config)
		})
	}
}

func TestToolServiceConfigStruct(t *testing.T) {
	config := ToolServiceConfig{
		Allowed: []string{"Read", "Write"},
	}

	assert.Len(t, config.Allowed, 2)
	assert.Contains(t, config.Allowed, "Read")
	assert.Contains(t, config.Allowed, "Write")
}
