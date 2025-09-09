package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMcpServiceConfig_Default(t *testing.T) {
	tests := []struct {
		name     string
		initial  McpServiceConfig
		expected McpServiceConfig
	}{
		{
			name:    "Empty config should use backup values",
			initial: McpServiceConfig{},
			expected: McpServiceConfig{
				Name:          BackupName,
				Version:       BackupVersion,
				ServerName:    BackupName,
				ServerVersion: BackupVersion,
			},
		},
		{
			name: "Config with Name set should keep Name, use backup for others",
			initial: McpServiceConfig{
				Name: "CustomApp",
			},
			expected: McpServiceConfig{
				Name:          "CustomApp",
				Version:       BackupVersion,
				ServerName:    BackupName,
				ServerVersion: BackupVersion,
			},
		},
		{
			name: "Config with all values set should keep all",
			initial: McpServiceConfig{
				Name:          "CustomApp",
				Version:       "1.0.0",
				ServerName:    "CustomServer",
				ServerVersion: "2.0.0",
			},
			expected: McpServiceConfig{
				Name:          "CustomApp",
				Version:       "1.0.0",
				ServerName:    "CustomServer",
				ServerVersion: "2.0.0",
			},
		},
		{
			name: "Config with partial values should fill missing with backup",
			initial: McpServiceConfig{
				Name:    "CustomApp",
				Version: "1.0.0",
			},
			expected: McpServiceConfig{
				Name:          "CustomApp",
				Version:       "1.0.0",
				ServerName:    BackupName,
				ServerVersion: BackupVersion,
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

func TestMcpServiceConfigConstants(t *testing.T) {
	assert.Equal(t, "DevCode", BackupName)
	assert.Equal(t, "0.0.1", BackupVersion)
}
