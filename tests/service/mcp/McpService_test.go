package mcp_test

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMcpService_ToolHandling(t *testing.T) {
	bus, err := events.NewEventBus()
	require.NoError(t, err, "EventBus 생성 실패")

	resultEvents := make(chan events.Event, 10)
	mockSubscriber := &MockEventSubscriber{events: resultEvents}
	bus.Subscribe(events.ToolRawResultEvent, mockSubscriber)

	mockService := &MockMcpService{bus: bus}

	t.Run("Tool 성공 시 적절한 이벤트 발행", func(t *testing.T) {
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
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

		select {
		case event := <-resultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCallUUID)
			assert.NotNil(t, data.Result)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("성공 이벤트가 시간 내에 수신되지 않음")
		}
	})

	t.Run("Tool 에러 시 적절한 이벤트 발행", func(t *testing.T) {
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
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

		mockService.SimulateToolCallWithError(toolCallData, mockResult)

		select {
		case event := <-resultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCallUUID)
			assert.NotNil(t, data.Result)
			assert.True(t, data.Result.IsError)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("에러 이벤트가 시간 내에 수신되지 않음")
		}
	})
}

func TestMcpService_EventHandling(t *testing.T) {
	bus, err := events.NewEventBus()
	require.NoError(t, err, "EventBus 생성 실패")

	mockService := &MockMcpService{bus: bus}

	t.Run("AcceptToolEvent 이벤트 처리", func(t *testing.T) {
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "TestTool",
			Parameters:   map[string]interface{}{"key": "value"},
		}

		event := events.Event{
			Type:      events.AcceptToolEvent,
			Data:      toolCallData,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		assert.NotPanics(t, func() {
			mockService.HandleEvent(event)
		})
	})

	t.Run("RequestToolListEvent 이벤트 처리", func(t *testing.T) {
		event := events.Event{
			Type:      events.RequestToolListEvent,
			Data:      nil,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		assert.NotPanics(t, func() {
			mockService.HandleEvent(event)
		})
	})
}

type MockEventSubscriber struct {
	events chan events.Event
}

func (m *MockEventSubscriber) HandleEvent(event events.Event) {
	select {
	case m.events <- event:
	default:
	}
}

func (m *MockEventSubscriber) GetID() constants.Source {
	return constants.Model
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
	event := events.Event{
		Type: events.ToolRawResultEvent,
		Data: dto.ToolRawResultData{
			RequestUUID:  data.RequestUUID,
			ToolCallUUID: data.ToolCallUUID,
			Result:       result,
		},
		Timestamp: time.Now(),
		Source:    constants.McpService,
	}

	m.bus.Publish(event)
}

func (m *MockMcpService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.AcceptToolEvent:
		data := event.Data.(dto.ToolCallData)
		mockResult := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "{\"success\":true,\"content\":\"Mock success\"}",
				},
			},
		}
		m.SimulateToolCallWithSuccess(data, mockResult)

	case events.RequestToolListEvent:
	}
}

func (m *MockMcpService) GetID() constants.Source {
	return constants.McpService
}
