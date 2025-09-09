package devcodeerror

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorCode_Values(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected uint
	}{
		{"FailLoggerSetup", FailLoggerSetup, 100},
		{"FailReadConfig", FailReadConfig, 101},
		{"FailCreateEventBus", FailCreateEventBus, 102},
		{"FailRunApp", FailRunApp, 200},
		{"FailHandleEvent", FailHandleEvent, 201},
		{"FailReadEnvironment", FailReadEnvironment, 300},
		{"FailOllaConnect", FailOllaConnect, 400},
		{"FailRunMcpServer", FailRunMcpServer, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, uint(tt.code))
		})
	}
}

func TestDevCodeError_Error_WithoutCause(t *testing.T) {
	devError := &DevCodeError{
		ErrorCode: FailLoggerSetup,
		Message:   "Logger setup failed",
		Cause:     nil,
		Timestap:  time.Now(),
	}

	expected := "[100] Logger setup failed"
	assert.Equal(t, expected, devError.Error())
}

func TestDevCodeError_Error_WithCause(t *testing.T) {
	originalError := errors.New("original error")
	devError := &DevCodeError{
		ErrorCode: FailReadConfig,
		Message:   "Config read failed",
		Cause:     originalError,
		Timestap:  time.Now(),
	}

	expected := "[101] Config read failed : original error"
	assert.Equal(t, expected, devError.Error())
}

func TestWrap_WithNilError(t *testing.T) {
	result := Wrap(nil, FailLoggerSetup, "test message")

	require.NotNil(t, result)
	assert.Equal(t, FailLoggerSetup, result.ErrorCode)
	assert.Equal(t, "test message", result.Message)
	assert.Nil(t, result.Cause)
	assert.True(t, time.Since(result.Timestap) < time.Second)
}

func TestWrap_WithError(t *testing.T) {
	originalError := errors.New("original error")
	result := Wrap(originalError, FailReadConfig, "wrapped message")

	require.NotNil(t, result)
	assert.Equal(t, FailReadConfig, result.ErrorCode)
	assert.Equal(t, "wrapped message", result.Message)
	assert.Equal(t, originalError, result.Cause)
	assert.True(t, time.Since(result.Timestap) < time.Second)
}

func TestWrap_Timestamp(t *testing.T) {
	before := time.Now()
	result := Wrap(nil, FailRunApp, "test")
	after := time.Now()

	assert.True(t, result.Timestap.After(before) || result.Timestap.Equal(before))
	assert.True(t, result.Timestap.Before(after) || result.Timestap.Equal(after))
}

func TestDevCodeError_ErrorInterface(t *testing.T) {
	var err error = &DevCodeError{
		ErrorCode: FailHandleEvent,
		Message:   "Event handling failed",
		Cause:     nil,
		Timestap:  time.Now(),
	}

	assert.Equal(t, "[201] Event handling failed", err.Error())
}

func TestDevCodeError_Fields(t *testing.T) {
	originalError := errors.New("root cause")
	timestamp := time.Now()

	devError := &DevCodeError{
		ErrorCode: FailReadEnvironment,
		Message:   "Environment read failed",
		Cause:     originalError,
		Timestap:  timestamp,
	}

	assert.Equal(t, FailReadEnvironment, devError.ErrorCode)
	assert.Equal(t, "Environment read failed", devError.Message)
	assert.Equal(t, originalError, devError.Cause)
	assert.Equal(t, timestamp, devError.Timestap)
}

func TestWrap_DifferentErrorCodes(t *testing.T) {
	testCases := []struct {
		name      string
		errorCode ErrorCode
		expected  uint
	}{
		{"Logger error", FailLoggerSetup, 100},
		{"Config error", FailReadConfig, 101},
		{"EventBus error", FailCreateEventBus, 102},
		{"App error", FailRunApp, 200},
		{"Event error", FailHandleEvent, 201},
		{"Environment error", FailReadEnvironment, 300},
		{"Ollama error", FailOllaConnect, 400},
		{"MCP error", FailRunMcpServer, 500},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := Wrap(nil, tc.errorCode, "test message")
			assert.Equal(t, tc.expected, uint(result.ErrorCode))
		})
	}
}

func TestDevCodeError_ErrorMessage_Formatting(t *testing.T) {
	tests := []struct {
		name          string
		errorCode     ErrorCode
		message       string
		cause         error
		expectedError string
	}{
		{
			name:          "Simple error without cause",
			errorCode:     FailLoggerSetup,
			message:       "Simple failure",
			cause:         nil,
			expectedError: "[100] Simple failure",
		},
		{
			name:          "Error with cause",
			errorCode:     FailReadConfig,
			message:       "Config failure",
			cause:         errors.New("file not found"),
			expectedError: "[101] Config failure : file not found",
		},
		{
			name:          "Error with complex cause",
			errorCode:     FailRunApp,
			message:       "Application startup failed",
			cause:         errors.New("port already in use: 8080"),
			expectedError: "[200] Application startup failed : port already in use: 8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devError := &DevCodeError{
				ErrorCode: tt.errorCode,
				Message:   tt.message,
				Cause:     tt.cause,
				Timestap:  time.Now(),
			}
			assert.Equal(t, tt.expectedError, devError.Error())
		})
	}
}
