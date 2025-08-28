package mcp_test

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMcpService_ToolErrorHandling(t *testing.T) {
	// Given: EventBus 초기화
	logger := zap.NewNop()
	bus, err := events.NewEventBus(logger)
	require.NoError(t, err, "EventBus 생성 실패")

	// 이벤트 수신을 위한 채널 설정
	resultEvents := make(chan events.Event, 10)

	// 이벤트 수신기 Mock
	mockSubscriber := &MockEventSubscriber{
		events: resultEvents,
	}

	bus.Subscribe(events.ToolRawResultEvent, mockSubscriber)

	// Mock MCP Service
	mockService := &MockMcpService{
		bus:    bus,
		logger: logger,
	}

	t.Run("Tool Error 발생 시 적절한 에러 이벤트 발행", func(t *testing.T) {
		// When: Tool 호출 시 에러 발생
		toolCallData := dto.ToolCallData{
			RequestUUID:  uuid.New(),
			ToolCallUUID: uuid.New(),
			ToolName:     "ErrorTool",
			Parameters:   map[string]interface{}{},
		}

		expectedError := errors.New("파일을 찾을 수 없습니다: /nonexistent/file.txt")
		mockService.SimulateToolCallWithError(toolCallData, expectedError)

		// Then: 에러 이벤트가 발생해야 함
		select {
		case event := <-resultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCall)
			assert.Nil(t, data.Result)
			assert.NotNil(t, data.Error)
			assert.Contains(t, data.Error.Error(), "파일을 찾을 수 없습니다")

		case <-time.After(100 * time.Millisecond):
			t.Fatal("에러 이벤트가 시간 내에 수신되지 않음")
		}
	})

	t.Run("Tool 성공 시 적절한 성공 이벤트 발행", func(t *testing.T) {
		// When: Tool 호출 성공
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
					Text: "{\"success\":true,\"content\":\"파일 내용이 성공적으로 읽어졌습니다\",\"total_lines\":5,\"lines_read\":5}",
				},
			},
		}

		mockService.SimulateToolCallWithSuccess(toolCallData, mockResult)

		// Then: 성공 이벤트가 발생해야 함
		select {
		case event := <-resultEvents:
			assert.Equal(t, events.ToolRawResultEvent, event.Type)

			data, ok := event.Data.(dto.ToolRawResultData)
			require.True(t, ok, "이벤트 데이터 타입이 올바르지 않음")

			assert.Equal(t, toolCallData.RequestUUID, data.RequestUUID)
			assert.Equal(t, toolCallData.ToolCallUUID, data.ToolCall)
			assert.NotNil(t, data.Result)
			assert.Nil(t, data.Error)

		case <-time.After(100 * time.Millisecond):
			t.Fatal("성공 이벤트가 시간 내에 수신되지 않음")
		}
	})
}

func TestMcpService_EventHandling(t *testing.T) {
	// Given: EventBus와 Mock Service 초기화
	logger := zap.NewNop()
	bus, err := events.NewEventBus(logger)
	require.NoError(t, err, "EventBus 생성 실패")

	mockService := &MockMcpService{
		bus:    bus,
		logger: logger,
	}

	t.Run("AcceptToolEvent 이벤트 처리", func(t *testing.T) {
		// When: AcceptToolEvent 이벤트 처리
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

		// Then: 이벤트 처리가 정상적으로 수행되어야 함
		assert.NotPanics(t, func() {
			mockService.HandleEvent(event)
		})
	})

	t.Run("RequestToolListEvent 이벤트 처리", func(t *testing.T) {
		// When: RequestToolListEvent 이벤트 처리
		event := events.Event{
			Type:      events.RequestToolListEvent,
			Data:      nil,
			Timestamp: time.Now(),
			Source:    constants.Model,
		}

		// Then: 이벤트 처리가 정상적으로 수행되어야 함
		assert.NotPanics(t, func() {
			mockService.HandleEvent(event)
		})
	})
}

func TestMcpService_ErrorTypes(t *testing.T) {
	// Given: EventBus 초기화
	logger := zap.NewNop()
	bus, err := events.NewEventBus(logger)
	require.NoError(t, err, "EventBus 생성 실패")

	resultEvents := make(chan events.Event, 10)
	mockSubscriber := &MockEventSubscriber{events: resultEvents}
	bus.Subscribe(events.ToolRawResultEvent, mockSubscriber)

	mockService := &MockMcpService{bus: bus, logger: logger}

	testCases := []struct {
		name          string
		toolName      string
		expectedError error
		errorContains string
	}{
		{
			name:          "파일을 찾을 수 없음 에러",
			toolName:      "Read",
			expectedError: errors.New("file not found: /nonexistent/file.txt"),
			errorContains: "file not found",
		},
		{
			name:          "권한 거부 에러",
			toolName:      "Read",
			expectedError: errors.New("permission denied: /root/restricted.txt"),
			errorContains: "permission denied",
		},
		{
			name:          "잘못된 경로 형식 에러",
			toolName:      "Read",
			expectedError: errors.New("invalid path format: "),
			errorContains: "invalid path format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			toolCallData := dto.ToolCallData{
				RequestUUID:  uuid.New(),
				ToolCallUUID: uuid.New(),
				ToolName:     tc.toolName,
				Parameters:   map[string]interface{}{},
			}

			mockService.SimulateToolCallWithError(toolCallData, tc.expectedError)

			select {
			case event := <-resultEvents:
				data := event.Data.(dto.ToolRawResultData)
				assert.NotNil(t, data.Error)
				assert.Contains(t, data.Error.Error(), tc.errorContains)

			case <-time.After(100 * time.Millisecond):
				t.Fatalf("%s: 에러 이벤트가 시간 내에 수신되지 않음", tc.name)
			}
		})
	}
}

// Mock 구조체들
type MockEventSubscriber struct {
	events chan events.Event
}

func (m *MockEventSubscriber) HandleEvent(event events.Event) {
	select {
	case m.events <- event:
	default:
		// 채널이 가득 찬 경우 무시
	}
}

func (m *MockEventSubscriber) GetID() constants.Source {
	return constants.Model
}

type MockMcpService struct {
	bus    *events.EventBus
	logger *zap.Logger
}

func (m *MockMcpService) SimulateToolCallWithError(data dto.ToolCallData, err error) {
	m.logger.Error("Mock tool call error",
		zap.String("tool_name", data.ToolName),
		zap.Error(err))

	m.publishToolResult(data, nil, err)
}

func (m *MockMcpService) SimulateToolCallWithSuccess(data dto.ToolCallData, result *mcp.CallToolResult) {
	m.logger.Info("Mock tool call success",
		zap.String("tool_name", data.ToolName))

	m.publishToolResult(data, result, nil)
}

func (m *MockMcpService) publishToolResult(data dto.ToolCallData, result *mcp.CallToolResult, err error) {
	event := events.Event{
		Type: events.ToolRawResultEvent,
		Data: dto.ToolRawResultData{
			RequestUUID: data.RequestUUID,
			ToolCall:    data.ToolCallUUID,
			Result:      result,
			Error:       err,
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
		// Mock tool call - 테스트용으로 성공 시뮬레이션
		mockResult := &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					Text: "{\"success\":true,\"content\":\"Mock success\"}",
				},
			},
		}
		m.SimulateToolCallWithSuccess(data, mockResult)

	case events.RequestToolListEvent:
		m.logger.Info("Mock tool list request handled")
		// 실제로는 도구 목록을 발행하지만 테스트에서는 생략
	}
}

func (m *MockMcpService) GetID() constants.Source {
	return constants.McpService
}