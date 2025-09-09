package ollama

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"net/url"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const TestModule = constants.Source(999)

// Mock implementations
type MockMessageManager struct {
	mock.Mock
}

func (m *MockMessageManager) AddSystemMessage(content string) {
	m.Called(content)
}

func (m *MockMessageManager) SetEnvironmentMessage(content string) {
	m.Called(content)
}

func (m *MockMessageManager) AddUserMessage(content string) {
	m.Called(content)
}

func (m *MockMessageManager) AddAssistantMessage(content string) {
	m.Called(content)
}

func (m *MockMessageManager) AddToolMessage(content string) {
	m.Called(content)
}

func (m *MockMessageManager) Clear() {
	m.Called()
}

func (m *MockMessageManager) GetMessages() []api.Message {
	args := m.Called()
	return args.Get(0).([]api.Message)
}

type MockToolManager struct {
	mock.Mock
}

func (m *MockToolManager) RegisterToolList(tools []*mcp.Tool) {
	m.Called(tools)
}

func (m *MockToolManager) GetToolList() []api.Tool {
	args := m.Called()
	return args.Get(0).([]api.Tool)
}

func (m *MockToolManager) RegisterToolCall(requestID types.RequestID, toolCallID types.ToolCallID, toolName string) {
	m.Called(requestID, toolCallID, toolName)
}

func (m *MockToolManager) HasToolCall(requestID types.RequestID, toolCallID types.ToolCallID) bool {
	args := m.Called(requestID, toolCallID)
	return args.Bool(0)
}

func (m *MockToolManager) CompleteToolCall(requestID types.RequestID, toolCallID types.ToolCallID) {
	m.Called(requestID, toolCallID)
}

func (m *MockToolManager) HasPendingCalls(requestID types.RequestID) bool {
	args := m.Called(requestID)
	return args.Bool(0)
}

func (m *MockToolManager) ClearRequest(requestID types.RequestID) {
	m.Called(requestID)
}

type MockStreamManager struct {
	mock.Mock
}

func (m *MockStreamManager) StartStream(
	ollama *api.Client,
	bus *events.EventBus,
	requestID types.RequestID,
	model string,
	tools []api.Tool,
	message []api.Message,
	callBack func(requestID types.RequestID, response api.ChatResponse) error,
) {
	m.Called(ollama, bus, requestID, model, tools, message, callBack)
}

func (m *MockStreamManager) Response(
	requestID types.RequestID,
	response api.ChatResponse,
	bus *events.EventBus,
	doneCallBack func(string),
	checkDone func(types.RequestID) bool,
	toolsCallBack func(types.RequestID, []api.ToolCall),
) error {
	args := m.Called(requestID, response, bus, doneCallBack, checkDone, toolsCallBack)
	return args.Error(0)
}

func (m *MockStreamManager) CancelStream(requestUUID types.RequestID) {
	m.Called(requestUUID)
}

func TestNewOllamaModule(t *testing.T) {
	logger := zap.NewNop()
	testUrl, _ := url.Parse("http://localhost:11434")
	ollamaConfig := config.OllamaServiceConfig{
		Url:    testUrl,
		Model:  "test-model",
		Prompt: "Test prompt",
	}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := NewOllamaModule(bus, ollamaConfig, logger)

	assert.NotNil(t, module)
	assert.NotNil(t, module.client)
	assert.Equal(t, ollamaConfig, module.config)
	assert.Equal(t, bus, module.bus)
	assert.NotNil(t, module.messageManager)
	assert.NotNil(t, module.toolManager)
	assert.NotNil(t, module.StreamManager)
	assert.Equal(t, logger, module.logger)
}

func TestOllamaModule_ProcessToolResult_ValidToolCall(t *testing.T) {
	logger := zap.NewNop()
	ollamaConfig := config.OllamaServiceConfig{}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := &OllamaModule{
		config: ollamaConfig,
		bus:    bus,
		logger: logger,
	}

	// Mock managers
	mockMessageManager := &MockMessageManager{}
	mockToolManager := &MockToolManager{}
	module.messageManager = mockMessageManager
	module.toolManager = mockToolManager

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolResult := "test result"

	// Set up mock expectations
	mockToolManager.On("HasToolCall", requestID, toolCallID).Return(true)
	mockMessageManager.On("AddToolMessage", toolResult).Return()
	mockToolManager.On("CompleteToolCall", requestID, toolCallID).Return()
	mockToolManager.On("HasPendingCalls", requestID).Return(false)
	mockToolManager.On("ClearRequest", requestID).Return()
	mockToolManager.On("GetToolList").Return([]api.Tool{})
	mockMessageManager.On("GetMessages").Return([]api.Message{})

	// Mock StreamManager for CallApi call
	mockStreamManager := &MockStreamManager{}
	module.StreamManager = mockStreamManager
	mockStreamManager.On("StartStream", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return()

	data := dto.ToolResultData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolResult: toolResult,
	}

	module.ProcessToolResult(data)

	mockMessageManager.AssertExpectations(t)
	mockToolManager.AssertExpectations(t)
}

func TestOllamaModule_ProcessToolResult_InvalidToolCall(t *testing.T) {
	logger := zap.NewNop()
	ollamaConfig := config.OllamaServiceConfig{}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := &OllamaModule{
		config: ollamaConfig,
		bus:    bus,
		logger: logger,
	}

	mockToolManager := &MockToolManager{}
	module.toolManager = mockToolManager

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Set up mock expectations
	mockToolManager.On("HasToolCall", requestID, toolCallID).Return(false)

	data := dto.ToolResultData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolResult: "test result",
	}

	module.ProcessToolResult(data)

	mockToolManager.AssertExpectations(t)
}

func TestOllamaModule_ProcessToolResult_WithPendingCalls(t *testing.T) {
	logger := zap.NewNop()
	ollamaConfig := config.OllamaServiceConfig{}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := &OllamaModule{
		config: ollamaConfig,
		bus:    bus,
		logger: logger,
	}

	mockMessageManager := &MockMessageManager{}
	mockToolManager := &MockToolManager{}
	module.messageManager = mockMessageManager
	module.toolManager = mockToolManager

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolResult := "test result"

	// Set up mock expectations
	mockToolManager.On("HasToolCall", requestID, toolCallID).Return(true)
	mockMessageManager.On("AddToolMessage", toolResult).Return()
	mockToolManager.On("CompleteToolCall", requestID, toolCallID).Return()
	mockToolManager.On("HasPendingCalls", requestID).Return(true)

	data := dto.ToolResultData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolResult: toolResult,
	}

	module.ProcessToolResult(data)

	mockMessageManager.AssertExpectations(t)
	mockToolManager.AssertExpectations(t)
}

func TestOllamaModule_AddAssistantMessage(t *testing.T) {
	logger := zap.NewNop()
	ollamaConfig := config.OllamaServiceConfig{}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := &OllamaModule{
		config: ollamaConfig,
		bus:    bus,
		logger: logger,
	}

	mockMessageManager := &MockMessageManager{}
	module.messageManager = mockMessageManager

	message := "Test message"
	mockMessageManager.On("AddAssistantMessage", message).Return()

	module.AddAssistantMessage(message)

	mockMessageManager.AssertExpectations(t)
}

func TestOllamaModule_ProcessToolCalls(t *testing.T) {
	logger := zap.NewNop()
	ollamaConfig := config.OllamaServiceConfig{}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := &OllamaModule{
		config: ollamaConfig,
		bus:    bus,
		logger: logger,
	}

	mockToolManager := &MockToolManager{}
	module.toolManager = mockToolManager

	requestID := types.NewRequestID()
	toolCalls := []api.ToolCall{
		{
			Function: api.ToolCallFunction{
				Name:      "test-tool",
				Arguments: map[string]interface{}{"param": "value"},
			},
		},
	}

	// Set up expectations
	mockToolManager.On("RegisterToolCall", requestID, mock.AnythingOfType("types.ToolCallID"), "test-tool").Return()

	// Create a channel to capture published events
	eventsChan := make(chan events.Event[dto.ToolCallData], 1)
	events.Subscribe(bus, bus.ToolCallEvent, TestModule, func(event events.Event[dto.ToolCallData]) {
		eventsChan <- event
	})

	module.ProcessToolCalls(requestID, toolCalls)

	// Wait for event to be published
	select {
	case event := <-eventsChan:
		assert.Equal(t, requestID, event.Data.RequestID)
		assert.Equal(t, "test-tool", event.Data.ToolName)
		// Parameters 타입이 다를 수 있으므로 내용만 확인
		if params, ok := event.Data.Parameters["param"]; ok {
			assert.Equal(t, "value", params)
		}
		assert.Equal(t, constants.LLMModule, event.Source)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected ToolCallEvent was not published")
	}

	mockToolManager.AssertExpectations(t)
}

func TestOllamaModule_CancelStream(t *testing.T) {
	logger := zap.NewNop()
	ollamaConfig := config.OllamaServiceConfig{}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := &OllamaModule{
		config: ollamaConfig,
		bus:    bus,
		logger: logger,
	}

	mockStreamManager := &MockStreamManager{}
	mockToolManager := &MockToolManager{}
	module.StreamManager = mockStreamManager
	module.toolManager = mockToolManager

	requestID := types.NewRequestID()

	mockStreamManager.On("CancelStream", requestID).Return()
	mockToolManager.On("ClearRequest", requestID).Return()

	module.CancelStream(requestID)

	mockStreamManager.AssertExpectations(t)
	mockToolManager.AssertExpectations(t)
}

func TestOllamaModule_UpdateEnvironmentToolList(t *testing.T) {
	logger := zap.NewNop()
	ollamaConfig := config.OllamaServiceConfig{}
	busConfig := config.EventBusConfig{PoolSize: 100}
	bus, err := events.NewEventBus(busConfig, logger)
	require.NoError(t, err)
	defer bus.Close()

	module := &OllamaModule{
		config: ollamaConfig,
		bus:    bus,
		logger: logger,
	}

	// Create channels to capture published events
	envEventsChan := make(chan events.Event[dto.EnvironmentRequestData], 1)
	toolListEventsChan := make(chan events.Event[dto.RequestToolListData], 1)

	events.Subscribe(bus, bus.RequestEnvironmentEvent, TestModule, func(event events.Event[dto.EnvironmentRequestData]) {
		envEventsChan <- event
	})

	events.Subscribe(bus, bus.RequestToolListEvent, TestModule, func(event events.Event[dto.RequestToolListData]) {
		toolListEventsChan <- event
	})

	module.UpdateEnvironmentToolList()

	// Verify environment request event
	select {
	case event := <-envEventsChan:
		assert.NotNil(t, event.Data.CreateID)
		assert.Equal(t, constants.LLMModule, event.Source)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected RequestEnvironmentEvent was not published")
	}

	// Verify tool list request event
	select {
	case event := <-toolListEventsChan:
		assert.NotNil(t, event.Data.CreateID)
		assert.Equal(t, constants.LLMModule, event.Source)
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Expected RequestToolListEvent was not published")
	}
}
