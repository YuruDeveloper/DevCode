package tool

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewToolModule(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{"Read", "List"},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	assert.NotNil(t, module)
	assert.NotNil(t, module.bus)
	assert.NotNil(t, module.logger)
	assert.NotNil(t, module.toolCallBuffer)
	assert.Equal(t, []string{"Read", "List"}, module.allowed)
}

func TestToolModuleSubscribe(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{"Read", "List"},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// Subscribe 메서드가 정상적으로 실행되는지 확인
	assert.NotPanics(t, func() {
		module.Subscribe()
	})
}

func TestToolModuleProcessUserDecisionAccept(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 이벤트 구독
	acceptToolReceived := make(chan dto.ToolCallData, 1)
	events.Subscribe(bus, bus.AcceptToolEvent, constants.ToolModule, func(event events.Event[dto.ToolCallData]) {
		acceptToolReceived <- event.Data
	})

	// 버퍼에 tool call 데이터 추가
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolCallData := dto.ToolCallData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolName:   "TestTool",
		Parameters: map[string]interface{}{},
	}
	module.toolCallBuffer[toolCallID] = toolCallData

	// 사용자 승인 결정 처리
	userDecisionData := dto.UserDecisionData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		Accept:     true,
	}

	module.ProcessUserDecision(userDecisionData)

	// AcceptToolEvent가 발행되는지 확인
	select {
	case data := <-acceptToolReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, "TestTool", data.ToolName)
	case <-time.After(2 * time.Second):
		t.Fatal("Expected AcceptToolEvent was not received within timeout")
	}

	// 버퍼에서 제거되었는지 확인
	_, exists := module.toolCallBuffer[toolCallID]
	assert.False(t, exists, "Tool call should be removed from buffer after processing")
}

func TestToolModuleProcessUserDecisionReject(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 이벤트 구독
	toolResultReceived := make(chan dto.ToolResultData, 1)
	events.Subscribe(bus, bus.ToolResultEvent, constants.ToolModule, func(event events.Event[dto.ToolResultData]) {
		toolResultReceived <- event.Data
	})

	toolUseReportReceived := make(chan dto.ToolUseReportData, 1)
	events.Subscribe(bus, bus.ToolUseReportEvent, constants.ToolModule, func(event events.Event[dto.ToolUseReportData]) {
		toolUseReportReceived <- event.Data
	})

	// 버퍼에 tool call 데이터 추가
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolCallData := dto.ToolCallData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolName:   "TestTool",
		Parameters: map[string]interface{}{},
	}
	module.toolCallBuffer[toolCallID] = toolCallData

	// 사용자 거부 결정 처리
	userDecisionData := dto.UserDecisionData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		Accept:     false,
	}

	module.ProcessUserDecision(userDecisionData)

	// ToolResultEvent가 발행되는지 확인 (에러 결과)
	select {
	case data := <-toolResultReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Contains(t, data.ToolResult, "User Reject Tool Use")
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolResultEvent was not received within timeout")
	}

	// ToolUseReportEvent가 발행되는지 확인 (에러 상태)
	select {
	case data := <-toolUseReportReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, constants.Error, data.ToolStatus)
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolUseReportEvent was not received within timeout")
	}

	// 버퍼에서 제거되었는지 확인
	_, exists := module.toolCallBuffer[toolCallID]
	assert.False(t, exists, "Tool call should be removed from buffer after processing")
}

func TestToolModuleProcessUserDecisionNotFound(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 존재하지 않는 tool call ID로 사용자 결정 처리
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	userDecisionData := dto.UserDecisionData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		Accept:     true,
	}

	// 에러 로그가 발생하더라도 패닉이 발생하지 않아야 함
	assert.NotPanics(t, func() {
		module.ProcessUserDecision(userDecisionData)
	})
}

func TestToolModuleProcessToolResultSuccess(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 이벤트 구독
	toolResultReceived := make(chan dto.ToolResultData, 1)
	events.Subscribe(bus, bus.ToolResultEvent, constants.ToolModule, func(event events.Event[dto.ToolResultData]) {
		toolResultReceived <- event.Data
	})

	toolUseReportReceived := make(chan dto.ToolUseReportData, 1)
	events.Subscribe(bus, bus.ToolUseReportEvent, constants.ToolModule, func(event events.Event[dto.ToolUseReportData]) {
		toolUseReportReceived <- event.Data
	})

	// 성공적인 tool result 데이터
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	textContent := &mcp.TextContent{Text: "Tool execution successful"}
	toolRawResultData := dto.ToolRawResultData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		Result: &mcp.CallToolResult{
			Content: []mcp.Content{textContent},
			IsError: false,
		},
	}

	module.ProcessToolResult(toolRawResultData)

	// ToolResultEvent가 발행되는지 확인 (성공 결과)
	select {
	case data := <-toolResultReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Contains(t, data.ToolResult, "Tool execution successful")
		assert.Contains(t, data.ToolResult, "<result>")
		assert.Contains(t, data.ToolResult, "</result>")
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolResultEvent was not received within timeout")
	}

	// ToolUseReportEvent가 발행되는지 확인 (성공 상태)
	select {
	case data := <-toolUseReportReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, constants.Success, data.ToolStatus)
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolUseReportEvent was not received within timeout")
	}
}

func TestToolModuleProcessToolResultError(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 이벤트 구독
	toolResultReceived := make(chan dto.ToolResultData, 1)
	events.Subscribe(bus, bus.ToolResultEvent, constants.ToolModule, func(event events.Event[dto.ToolResultData]) {
		toolResultReceived <- event.Data
	})

	toolUseReportReceived := make(chan dto.ToolUseReportData, 1)
	events.Subscribe(bus, bus.ToolUseReportEvent, constants.ToolModule, func(event events.Event[dto.ToolUseReportData]) {
		toolUseReportReceived <- event.Data
	})

	// 에러가 발생한 tool result 데이터
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	textContent := &mcp.TextContent{Text: "Tool execution failed"}
	toolRawResultData := dto.ToolRawResultData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		Result: &mcp.CallToolResult{
			Content: []mcp.Content{textContent},
			IsError: true,
		},
	}

	module.ProcessToolResult(toolRawResultData)

	// ToolResultEvent가 발행되는지 확인 (에러 결과)
	select {
	case data := <-toolResultReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Contains(t, data.ToolResult, "Tool execution failed")
		assert.Contains(t, data.ToolResult, "<tool_use_error>")
		assert.Contains(t, data.ToolResult, "</tool_use_error>")
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolResultEvent was not received within timeout")
	}

	// ToolUseReportEvent가 발행되는지 확인 (에러 상태)
	select {
	case data := <-toolUseReportReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, constants.Error, data.ToolStatus)
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolUseReportEvent was not received within timeout")
	}
}

func TestToolModuleProcessToolCallAllowed(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{"Read", "List"},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 이벤트 구독
	acceptToolReceived := make(chan dto.ToolCallData, 1)
	events.Subscribe(bus, bus.AcceptToolEvent, constants.ToolModule, func(event events.Event[dto.ToolCallData]) {
		acceptToolReceived <- event.Data
	})

	toolUseReportReceived := make(chan dto.ToolUseReportData, 1)
	events.Subscribe(bus, bus.ToolUseReportEvent, constants.ToolModule, func(event events.Event[dto.ToolUseReportData]) {
		toolUseReportReceived <- event.Data
	})

	// 허용된 도구 호출 데이터
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolCallData := dto.ToolCallData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolName:   "Read",
		Parameters: map[string]interface{}{
			"file_path": "/test/path",
		},
	}

	module.ProcessToolCall(toolCallData)

	// ToolUseReportEvent가 먼저 발행되는지 확인
	select {
	case data := <-toolUseReportReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, constants.Call, data.ToolStatus)
		assert.Contains(t, data.ToolInfo, "Read")
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolUseReportEvent was not received within timeout")
	}

	// AcceptToolEvent가 발행되는지 확인 (허용된 도구이므로 자동 승인)
	select {
	case data := <-acceptToolReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, "Read", data.ToolName)
	case <-time.After(2 * time.Second):
		t.Fatal("Expected AcceptToolEvent was not received within timeout")
	}

	// 버퍼에 추가되지 않았는지 확인 (자동 승인되었으므로)
	_, exists := module.toolCallBuffer[toolCallID]
	assert.False(t, exists, "Tool call should not be in buffer for allowed tools")
}

func TestToolModuleProcessToolCallNotAllowed(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{"Read"},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 이벤트 구독
	requestToolUseReceived := make(chan dto.ToolUseReportData, 1)
	events.Subscribe(bus, bus.RequestToolUseEvent, constants.ToolModule, func(event events.Event[dto.ToolUseReportData]) {
		requestToolUseReceived <- event.Data
	})

	toolUseReportReceived := make(chan dto.ToolUseReportData, 1)
	events.Subscribe(bus, bus.ToolUseReportEvent, constants.ToolModule, func(event events.Event[dto.ToolUseReportData]) {
		toolUseReportReceived <- event.Data
	})

	// 허용되지 않은 도구 호출 데이터
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolCallData := dto.ToolCallData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolName:   "List",
		Parameters: map[string]interface{}{
			"path": "/test/path",
		},
	}

	module.ProcessToolCall(toolCallData)

	// ToolUseReportEvent가 먼저 발행되는지 확인
	select {
	case data := <-toolUseReportReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, constants.Call, data.ToolStatus)
		assert.Contains(t, data.ToolInfo, "List")
	case <-time.After(2 * time.Second):
		t.Fatal("Expected ToolUseReportEvent was not received within timeout")
	}

	// RequestToolUseEvent가 발행되는지 확인 (허용되지 않은 도구이므로 승인 요청)
	select {
	case data := <-requestToolUseReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, constants.Call, data.ToolStatus)
		assert.Contains(t, data.ToolInfo, "List")
	case <-time.After(2 * time.Second):
		t.Fatal("Expected RequestToolUseEvent was not received within timeout")
	}

	// 버퍼에 추가되었는지 확인 (승인 대기 중이므로)
	bufferedData, exists := module.toolCallBuffer[toolCallID]
	assert.True(t, exists, "Tool call should be in buffer for non-allowed tools")
	assert.Equal(t, "List", bufferedData.ToolName)
}

func TestToolModuleToolInfoRead(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// Read 도구 정보 테스트
	parameters := map[string]any{
		"file_path": "/test/file.txt",
	}
	result := module.ToolInfo("Read", parameters)
	assert.Equal(t, "Read (/test/file.txt)", result)

	// file_path가 없는 경우
	parametersEmpty := map[string]any{}
	result = module.ToolInfo("Read", parametersEmpty)
	assert.Equal(t, "Read", result)

	// file_path가 string이 아닌 경우
	parametersWrongType := map[string]any{
		"file_path": 123,
	}
	result = module.ToolInfo("Read", parametersWrongType)
	assert.Equal(t, "Read", result)
}

func TestToolModuleToolInfoList(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// List 도구 정보 테스트
	parameters := map[string]any{
		"path": "/test/directory",
	}
	result := module.ToolInfo("List", parameters)
	assert.Equal(t, "List (/test/directory)", result)

	// path가 없는 경우
	parametersEmpty := map[string]any{}
	result = module.ToolInfo("List", parametersEmpty)
	assert.Equal(t, "List", result)

	// path가 string이 아닌 경우
	parametersWrongType := map[string]any{
		"path": 456,
	}
	result = module.ToolInfo("List", parametersWrongType)
	assert.Equal(t, "List", result)
}

func TestToolModuleToolInfoUnknown(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)

	// 알려지지 않은 도구 정보 테스트
	parameters := map[string]any{
		"param": "value",
	}
	result := module.ToolInfo("UnknownTool", parameters)
	assert.Equal(t, "UnknownTool", result)
}

// 통합 테스트: 전체 워크플로우 테스트
func TestToolModuleIntegration(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	toolConfig := config.ToolServiceConfig{
		Allowed: []string{"Read"},
	}
	logger := zap.NewNop()

	module := NewToolModule(bus, toolConfig, logger)
	require.NotNil(t, module)

	// 1. 허용된 도구 호출 테스트 (자동 승인)
	acceptToolReceived := make(chan dto.ToolCallData, 1)
	events.Subscribe(bus, bus.AcceptToolEvent, constants.ToolModule, func(event events.Event[dto.ToolCallData]) {
		acceptToolReceived <- event.Data
	})

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	events.Publish(bus, bus.ToolCallEvent, events.Event[dto.ToolCallData]{
		Data: dto.ToolCallData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolName:   "Read",
			Parameters: map[string]interface{}{
				"file_path": "/test/file.txt",
			},
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	})

	// AcceptToolEvent 수신 확인
	select {
	case data := <-acceptToolReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.Equal(t, "Read", data.ToolName)
		t.Logf("Received AcceptToolEvent for allowed tool: %s", data.ToolName)
	case <-time.After(3 * time.Second):
		t.Fatal("Expected AcceptToolEvent was not received")
	}

	// 2. 허용되지 않은 도구 호출 테스트 (승인 요청)
	requestToolUseReceived := make(chan dto.ToolUseReportData, 1)
	events.Subscribe(bus, bus.RequestToolUseEvent, constants.ToolModule, func(event events.Event[dto.ToolUseReportData]) {
		requestToolUseReceived <- event.Data
	})

	requestID2 := types.NewRequestID()
	toolCallID2 := types.NewToolCallID()
	events.Publish(bus, bus.ToolCallEvent, events.Event[dto.ToolCallData]{
		Data: dto.ToolCallData{
			RequestID:  requestID2,
			ToolCallID: toolCallID2,
			ToolName:   "List",
			Parameters: map[string]interface{}{
				"path": "/test/directory",
			},
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	})

	// RequestToolUseEvent 수신 확인
	select {
	case data := <-requestToolUseReceived:
		assert.Equal(t, requestID2, data.RequestID)
		assert.Equal(t, toolCallID2, data.ToolCallID)
		assert.Equal(t, constants.Call, data.ToolStatus)
		t.Logf("Received RequestToolUseEvent for non-allowed tool: %s", data.ToolInfo)
	case <-time.After(3 * time.Second):
		t.Fatal("Expected RequestToolUseEvent was not received")
	}

	// 3. 사용자 승인 처리 테스트
	toolResultReceived := make(chan dto.ToolResultData, 1)
	events.Subscribe(bus, bus.ToolResultEvent, constants.ToolModule, func(event events.Event[dto.ToolResultData]) {
		toolResultReceived <- event.Data
	})

	events.Publish(bus, bus.UserDecisionEvent, events.Event[dto.UserDecisionData]{
		Data: dto.UserDecisionData{
			RequestID:  requestID2,
			ToolCallID: toolCallID2,
			Accept:     false,
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	})

	// ToolResultEvent 수신 확인 (거부된 경우)
	select {
	case data := <-toolResultReceived:
		assert.Equal(t, requestID2, data.RequestID)
		assert.Equal(t, toolCallID2, data.ToolCallID)
		assert.Contains(t, data.ToolResult, "User Reject Tool Use")
		t.Logf("Received ToolResultEvent for rejected tool")
	case <-time.After(3 * time.Second):
		t.Fatal("Expected ToolResultEvent was not received")
	}

	// 4. Tool 실행 결과 처리 테스트
	requestID3 := types.NewRequestID()
	toolCallID3 := types.NewToolCallID()
	textContent := &mcp.TextContent{Text: "Successful result"}
	events.Publish(bus, bus.ToolRawResultEvent, events.Event[dto.ToolRawResultData]{
		Data: dto.ToolRawResultData{
			RequestID:  requestID3,
			ToolCallID: toolCallID3,
			Result: &mcp.CallToolResult{
				Content: []mcp.Content{textContent},
				IsError: false,
			},
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	})

	// ToolResultEvent 수신 확인 (성공적인 결과)
	select {
	case data := <-toolResultReceived:
		assert.Equal(t, requestID3, data.RequestID)
		assert.Equal(t, toolCallID3, data.ToolCallID)
		assert.Contains(t, data.ToolResult, "Successful result")
		assert.Contains(t, data.ToolResult, "<result>")
		t.Logf("Received ToolResultEvent for successful execution")
	case <-time.After(3 * time.Second):
		t.Fatal("Expected ToolResultEvent for successful result was not received")
	}
}
