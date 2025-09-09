package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestViewConfig_Default(t *testing.T) {
	tests := []struct {
		name     string
		initial  ViewConfig
		expected ViewConfig
	}{
		{
			name:    "Empty config should use backup values",
			initial: ViewConfig{},
			expected: ViewConfig{
				Dot:        BackupDot,
				SelectChar: BackupSelectChar,
			},
		},
		{
			name: "Config with Dot set should keep Dot, use backup for SelectChar",
			initial: ViewConfig{
				Dot: "♦",
			},
			expected: ViewConfig{
				Dot:        "♦",
				SelectChar: BackupSelectChar,
			},
		},
		{
			name: "Config with SelectChar set should keep SelectChar, use backup for Dot",
			initial: ViewConfig{
				SelectChar: "→",
			},
			expected: ViewConfig{
				Dot:        BackupDot,
				SelectChar: "→",
			},
		},
		{
			name: "Config with both values set should keep both",
			initial: ViewConfig{
				Dot:        "★",
				SelectChar: "→",
			},
			expected: ViewConfig{
				Dot:        "★",
				SelectChar: "→",
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

func TestViewConfigConstants(t *testing.T) {
	assert.Equal(t, "●", BackupDot)
	assert.Equal(t, ">", BackupSelectChar)
}
