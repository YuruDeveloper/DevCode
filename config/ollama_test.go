package config

import (
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOllamaServiceConfig_Default(t *testing.T) {
	tests := []struct {
		name     string
		initial  OllamaServiceConfig
		expected func(*testing.T, OllamaServiceConfig)
	}{
		{
			name:    "Empty config should use backup values",
			initial: OllamaServiceConfig{},
			expected: func(t *testing.T, config OllamaServiceConfig) {
				assert.Equal(t, BackupMessageLimit, config.MessageLimit)
				assert.Equal(t, BackupDefaultActiveStreamSzie, config.DefaultActiveStreamSize) // Note: There's a bug in the original code
				assert.Equal(t, BackupEnviromentInfo, config.EnvironmentInfo)
				assert.Equal(t, BackupDefaultToolSize, config.DefaultToolSize)
				assert.Equal(t, BackupDefaultRequestContentsSize, config.DefaultRequestContentsSize)
				assert.Equal(t, BackupToolCallSize, config.DefaultToolCallSize)
				assert.Equal(t, BackupModel, config.Model)
				assert.Equal(t, BackupPrompt, config.Prompt)
				assert.NotNil(t, config.Url)
				expectedUrl, _ := url.Parse(BackupUrl)
				assert.Equal(t, expectedUrl, config.Url)
			},
		},
		{
			name: "Config with custom values should keep them",
			initial: OllamaServiceConfig{
				MessageLimit:               200,
				DefaultSystemMessageLength: 20,
				EnvironmentInfo:            "Custom env info",
				DefaultToolSize:            20,
				DefaultRequestContentsSize: 20,
				DefaultToolCallSize:        10,
				urlText:                    "http://localhost:8080",
				Model:                      "custom-model",
				DefaultActiveStreamSize:    20,
			},
			expected: func(t *testing.T, config OllamaServiceConfig) {
				assert.Equal(t, 200, config.MessageLimit)
				assert.Equal(t, 20, config.DefaultActiveStreamSize)
				assert.Equal(t, "Custom env info", config.EnvironmentInfo)
				assert.Equal(t, 20, config.DefaultToolSize)
				assert.Equal(t, 20, config.DefaultRequestContentsSize)
				assert.Equal(t, 10, config.DefaultToolCallSize)
				assert.Equal(t, "custom-model", config.Model)
				assert.NotNil(t, config.Url)
				expectedUrl, _ := url.Parse("http://localhost:8080")
				assert.Equal(t, expectedUrl, config.Url)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := tt.initial
			config.Default()
			tt.expected(t, config)
		})
	}
}

func TestOllamaServiceConfig_Default_WithValidSystemFile(t *testing.T) {
	// Create a temporary system prompt file
	tmpDir := t.TempDir()
	systemFile := filepath.Join(tmpDir, "system.txt")
	systemContent := "This is a custom system prompt"

	err := os.WriteFile(systemFile, []byte(systemContent), 0644)
	require.NoError(t, err)

	config := OllamaServiceConfig{
		system: systemFile,
	}
	config.Default()

	assert.Equal(t, systemContent, config.Prompt)
}

func TestOllamaServiceConfig_Default_WithInvalidSystemFile(t *testing.T) {
	config := OllamaServiceConfig{
		system: "/nonexistent/file.txt",
	}
	config.Default()

	assert.Equal(t, BackupPrompt, config.Prompt)
}

func TestOllamaServiceConfig_Default_InvalidUrl(t *testing.T) {
	config := OllamaServiceConfig{
		urlText: "invalid-url",
	}
	config.Default()

	expectedUrl, _ := url.Parse(BackupUrl)
	assert.Equal(t, expectedUrl, config.Url)
}

func TestOllamaServiceConfigConstants(t *testing.T) {
	assert.Equal(t, 100, BackupMessageLimit)
	assert.Equal(t, 10, BackupDefaultSystemMessageLength)
	assert.Equal(t, "Here is useful information about the environment you are running in:\n", BackupEnviromentInfo)
	assert.Equal(t, 10, BackupDefaultToolSize)
	assert.Equal(t, 10, BackupDefaultRequestContentsSize)
	assert.Equal(t, 5, BackupToolCallSize)
	assert.Equal(t, 10, BackupDefaultActiveStreamSzie)
	assert.Equal(t, "http://127.0.0.1:11434", BackupUrl)
	assert.Equal(t, "llama3.1:8b", BackupModel)
	assert.Equal(t, "You are DevCode : Code Asstance", BackupPrompt)
}
