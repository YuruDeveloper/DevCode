package mcp

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewMcpModule(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)

	assert.NotNil(t, module)
	assert.NotNil(t, module.client)
	assert.NotNil(t, module.clientSession)
	assert.NotNil(t, module.toolServer)
	assert.NotNil(t, module.bus)
	assert.NotNil(t, module.ctx)
	assert.NotNil(t, module.logger)
}

func TestMcpModuleSubscribe(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)

	// Subscribe 메서드가 정상적으로 실행되는지 확인
	assert.NotPanics(t, func() {
		module.Subscribe()
	})
}

func TestMcpModuleInitTools(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)

	// InitTools가 정상적으로 실행되는지 확인
	assert.NotPanics(t, func() {
		module.InitTools()
	})

	// 도구 목록을 가져와서 read와 list 도구가 있는지 확인
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	toolNames := make(map[string]bool)
	for tool := range module.clientSession.Tools(ctx, nil) {
		toolNames[tool.Name] = true
	}

	assert.True(t, toolNames["Read"], "Read tool should be registered")
	assert.True(t, toolNames["List"], "List tool should be registered")
}

func TestMcpModulePublishToolList(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)

	// 이벤트 구독
	received := make(chan dto.ToolListUpdateData, 1)
	events.Subscribe(bus, bus.UpdateToolListEvent, constants.McpModule, func(event events.Event[dto.ToolListUpdateData]) {
		received <- event.Data
	})

	// PublishToolList 호출
	module.PublishToolList()

	// 이벤트가 발행되는지 확인
	select {
	case data := <-received:
		assert.NotEmpty(t, data.List)
	case <-time.After(2 * time.Second):
		t.Fatal("Expected UpdateToolListEvent was not received within timeout")
	}
}

func TestMcpModuleToolCall(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)

	// 이벤트 구독
	received := make(chan dto.ToolRawResultData, 1)
	events.Subscribe(bus, bus.ToolRawResultEvent, constants.McpModule, func(event events.Event[dto.ToolRawResultData]) {
		received <- event.Data
	})

	// 테스트용 도구 호출 데이터
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolCallData := dto.ToolCallData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolName:   "Read",
		Parameters: map[string]interface{}{
			"filePath": "/home/cecil/DevCode/go.mod",
		},
	}

	// 도구 호출 실행
	module.ToolCall(toolCallData)

	// 결과 이벤트가 발행되는지 확인
	select {
	case data := <-received:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.NotNil(t, data.Result)
	case <-time.After(5 * time.Second):
		t.Fatal("Expected ToolRawResultEvent was not received within timeout")
	}
}

func TestMcpModuleToolCallWithInvalidTool(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)

	// 이벤트 구독
	received := make(chan dto.ToolRawResultData, 1)
	events.Subscribe(bus, bus.ToolRawResultEvent, constants.McpModule, func(event events.Event[dto.ToolRawResultData]) {
		received <- event.Data
	})

	// 존재하지 않는 도구 호출 데이터
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolCallData := dto.ToolCallData{
		RequestID:  requestID,
		ToolCallID: toolCallID,
		ToolName:   "NonExistentTool",
		Parameters: map[string]interface{}{},
	}

	// 도구 호출 실행
	module.ToolCall(toolCallData)

	// 에러 결과 이벤트가 발행되는지 확인
	select {
	case data := <-received:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.NotNil(t, data.Result)
		assert.True(t, data.Result.IsError, "Result should indicate error")
	case <-time.After(5 * time.Second):
		t.Fatal("Expected ToolRawResultEvent was not received within timeout")
	}
}

// 모의 도구를 생성하여 InsertTool 함수 테스트
type MockToolParams struct {
	Message string `json:"message"`
}

type MockTool struct {
	name        string
	description string
}

func (t *MockTool) Name() string {
	return t.name
}

func (t *MockTool) Description() string {
	return t.description
}

func (t *MockTool) Handler() mcp.ToolHandlerFor[MockToolParams, any] {
	return func(ctx context.Context, session *mcp.ServerSession, params *mcp.CallToolParamsFor[MockToolParams]) (*mcp.CallToolResultFor[any], error) {
		content := mcp.TextContent{Text: "Mock tool result"}
		return &mcp.CallToolResultFor[any]{
			Content: []mcp.Content{
				&content,
			},
		}, nil
	}
}

func TestInsertTool(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)

	mockTool := &MockTool{
		name:        "MockTool",
		description: "A mock tool for testing",
	}

	// InsertTool이 정상적으로 실행되는지 확인
	assert.NotPanics(t, func() {
		InsertTool(module, mockTool)
	})

	// 도구가 정상적으로 등록되었는지 확인
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	toolFound := false
	for tool := range module.clientSession.Tools(ctx, nil) {
		if tool.Name == "MockTool" {
			toolFound = true
			assert.Equal(t, "A mock tool for testing", tool.Description)
			break
		}
	}

	assert.True(t, toolFound, "MockTool should be found in the tool list")
}

// 통합 테스트: 전체 워크플로우 테스트
func TestMcpModuleIntegration(t *testing.T) {
	busConfig := config.EventBusConfig{PoolSize: 10}
	bus, err := events.NewEventBus(busConfig, zap.NewNop())
	require.NoError(t, err)

	mcpConfig := config.McpServiceConfig{
		Name:          "test-client",
		Version:       "1.0.0",
		ServerName:    "test-server",
		ServerVersion: "1.0.0",
	}
	logger := zap.NewNop()

	module := NewMcpModule(bus, mcpConfig, logger)
	require.NotNil(t, module)

	// 1. 도구 목록 요청 이벤트 발행
	toolListReceived := make(chan dto.ToolListUpdateData, 1)
	events.Subscribe(bus, bus.UpdateToolListEvent, constants.McpModule, func(event events.Event[dto.ToolListUpdateData]) {
		toolListReceived <- event.Data
	})

	// RequestToolListEvent 발행
	events.Publish(bus, bus.RequestToolListEvent, events.Event[dto.RequestToolListData]{
		Data:      dto.RequestToolListData{},
		TimeStamp: time.Now(),
		Source:    constants.McpModule,
	})

	// 도구 목록 업데이트 이벤트 수신 확인
	select {
	case data := <-toolListReceived:
		assert.NotEmpty(t, data.List)
		t.Logf("Received tool list with %d tools", len(data.List))
	case <-time.After(3 * time.Second):
		t.Fatal("Expected UpdateToolListEvent was not received")
	}

	// 2. 도구 실행 테스트
	toolResultReceived := make(chan dto.ToolRawResultData, 1)
	events.Subscribe(bus, bus.ToolRawResultEvent, constants.McpModule, func(event events.Event[dto.ToolRawResultData]) {
		toolResultReceived <- event.Data
	})

	// AcceptToolEvent 발행
	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	events.Publish(bus, bus.AcceptToolEvent, events.Event[dto.ToolCallData]{
		Data: dto.ToolCallData{
			RequestID:  requestID,
			ToolCallID: toolCallID,
			ToolName:   "Read",
			Parameters: map[string]interface{}{
				"filePath": "/home/cecil/DevCode/go.mod",
			},
		},
		TimeStamp: time.Now(),
		Source:    constants.McpModule,
	})

	// 도구 실행 결과 이벤트 수신 확인
	select {
	case data := <-toolResultReceived:
		assert.Equal(t, requestID, data.RequestID)
		assert.Equal(t, toolCallID, data.ToolCallID)
		assert.NotNil(t, data.Result)
		t.Logf("Received tool result: %+v", data.Result)
	case <-time.After(5 * time.Second):
		t.Fatal("Expected ToolRawResultEvent was not received")
	}
}
