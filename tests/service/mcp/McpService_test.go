package mcp_test

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/types"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type MockToolRawResultHandler struct {
	ReceivedEvents []events.Event[dto.ToolRawResultData]
}

func NewMockToolRawResultHandler() *MockToolRawResultHandler {
	return &MockToolRawResultHandler{
		ReceivedEvents: make([]events.Event[dto.ToolRawResultData], 0),
	}
}

func (m *MockToolRawResultHandler) HandleEvent(event events.Event[dto.ToolRawResultData]) {
	m.ReceivedEvents = append(m.ReceivedEvents, event)
}

func TestMcpService_ToolHandling(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	logger := zap.NewNop()
	bus, err := events.NewEventBus(eventBusConfig, logger)
	require.NoError(t, err, "EventBus 생성 실패")
	defer bus.Close()

	resultHandler := NewMockToolRawResultHandler()
	bus.ToolRawResultEvent.Subscribe(constants.McpService, resultHandler.HandleEvent)

	mockService := &MockMcpService{bus: bus}

	t.Run("Tool 성공 시 적절한 이벤트 발행", func(t *testing.T) {
		requestID := types.NewRequestID()
		toolCallID := types.NewToolCallID()
		toolCallData := dto.ToolCallData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolName:     "ReadTool",
			Parameters: map[string]interface{}{
				"file_path": "/test/file.txt",
			},
		}

		mockResult := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "{\"success\":true,\"content\":\"파일 내용이 성공적으로 읽어졌습니다\"}",
				},
			},
		}

		mockService.SimulateToolCallWithSuccess(toolCallData, mockResult)

		// 이벤트 수신 대기
		time.Sleep(100 * time.Millisecond)

		assert.Len(t, resultHandler.ReceivedEvents, 1)
		event := resultHandler.ReceivedEvents[0]

		assert.Equal(t, requestID, event.Data.RequestID)
		assert.Equal(t, toolCallID, event.Data.ToolCallID)
		assert.NotNil(t, event.Data.Result)
		assert.False(t, event.Data.Result.IsError)
	})

	t.Run("Tool 에러 시 적절한 이벤트 발행", func(t *testing.T) {
		requestID := types.NewRequestID()
		toolCallID := types.NewToolCallID()
		toolCallData := dto.ToolCallData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolName:     "ErrorTool",
			Parameters:   map[string]interface{}{},
		}

		mockResult := &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "Tool Call Error : 파일을 찾을 수 없습니다",
				},
			},
		}

		// 기존 이벤트 클리어
		resultHandler.ReceivedEvents = make([]events.Event[dto.ToolRawResultData], 0)

		mockService.SimulateToolCallWithError(toolCallData, mockResult)

		// 이벤트 수신 대기
		time.Sleep(100 * time.Millisecond)

		assert.Len(t, resultHandler.ReceivedEvents, 1)
		event := resultHandler.ReceivedEvents[0]

		assert.Equal(t, requestID, event.Data.RequestID)
		assert.Equal(t, toolCallID, event.Data.ToolCallID)
		assert.NotNil(t, event.Data.Result)
		assert.True(t, event.Data.Result.IsError)
	})
}

type MockAcceptToolHandler struct {
	ReceivedEvents []events.Event[dto.ToolCallData]
}

func NewMockAcceptToolHandler() *MockAcceptToolHandler {
	return &MockAcceptToolHandler{
		ReceivedEvents: make([]events.Event[dto.ToolCallData], 0),
	}
}

func (m *MockAcceptToolHandler) HandleEvent(event events.Event[dto.ToolCallData]) {
	m.ReceivedEvents = append(m.ReceivedEvents, event)
}

func TestMcpService_EventHandling(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	logger := zap.NewNop()
	bus, err := events.NewEventBus(eventBusConfig, logger)
	require.NoError(t, err, "EventBus 생성 실패")
	defer bus.Close()

	acceptToolHandler := NewMockAcceptToolHandler()
	bus.AcceptToolEvent.Subscribe(constants.McpService, acceptToolHandler.HandleEvent)

	t.Run("AcceptToolEvent 이벤트 발행 테스트", func(t *testing.T) {
		requestID := types.NewRequestID()
		toolCallID := types.NewToolCallID()
		toolCallData := dto.ToolCallData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolName:     "TestTool",
			Parameters:   map[string]interface{}{"key": "value"},
		}

		event := events.Event[dto.ToolCallData]{
			Data:      toolCallData,
			TimeStamp: time.Now(),
			Source:    constants.Model,
		}

		assert.NotPanics(t, func() {
			bus.AcceptToolEvent.Publish(event)
		})

		// 이벤트 수신 대기
		time.Sleep(100 * time.Millisecond)

		assert.Len(t, acceptToolHandler.ReceivedEvents, 1)
		receivedEvent := acceptToolHandler.ReceivedEvents[0]

		assert.Equal(t, requestID, receivedEvent.Data.RequestID)
		assert.Equal(t, toolCallID, receivedEvent.Data.ToolCallID)
		assert.Equal(t, "TestTool", receivedEvent.Data.ToolName)
	})
}

type MockMcpService struct {
	bus *events.EventBus
}

func (m *MockMcpService) SimulateToolCallWithError(data dto.ToolCallData, result *mcp.CallToolResult) {
	m.publishToolResult(data, result)
}

func (m *MockMcpService) SimulateToolCallWithSuccess(data dto.ToolCallData, result *mcp.CallToolResult) {
	m.publishToolResult(data, result)
}

func (m *MockMcpService) publishToolResult(data dto.ToolCallData, result *mcp.CallToolResult) {
	event := events.Event[dto.ToolRawResultData]{
		Data: dto.ToolRawResultData{
			RequestID:  data.RequestID,
			ToolCallID: data.ToolCallID,
			Result:       result,
		},
		TimeStamp: time.Now(),
		Source:    constants.McpService,
	}

	m.bus.ToolRawResultEvent.Publish(event)
}