package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventBusConfig_Default(t *testing.T) {
	tests := []struct {
		name     string
		initial  EventBusConfig
		expected EventBusConfig
	}{
		{
			name:    "Empty config should use backup PoolSize",
			initial: EventBusConfig{},
			expected: EventBusConfig{
				PoolSize: BackupPoolSize,
			},
		},
		{
			name: "Config with PoolSize set should keep PoolSize",
			initial: EventBusConfig{
				PoolSize: 5000,
			},
			expected: EventBusConfig{
				PoolSize: 5000,
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

func TestEventBusConfigConstants(t *testing.T) {
	assert.Equal(t, 10000, BackupPoolSize)
}
