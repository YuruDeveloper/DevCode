package ollama

import (
	"DevCode/config"
	"DevCode/types"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
)

func TestNewToolManager(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultToolSize:            5,
		DefaultRequestContentsSize: 10,
	}

	manager := NewToolManager(ollamaConfig)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.tools)
	assert.NotNil(t, manager.requestContents)
	assert.Equal(t, ollamaConfig, manager.config)
	assert.Equal(t, 0, len(manager.tools))
	assert.Equal(t, 0, len(manager.requestContents))
}

func TestToolManager_RegisterToolList(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	// Create test MCP tools
	mcpTools := []*mcp.Tool{
		{
			Name:        "test-tool-1",
			Description: "Test tool 1",
		},
		{
			Name:        "test-tool-2",
			Description: "Test tool 2",
		},
	}

	manager.RegisterToolList(mcpTools)

	tools := manager.GetToolList()
	assert.Equal(t, 2, len(tools))

	// Verify first tool conversion
	assert.Equal(t, "function", tools[0].Type)
	assert.Equal(t, "test-tool-1", tools[0].Function.Name)
	assert.Equal(t, "Test tool 1", tools[0].Function.Description)

	// Verify second tool conversion
	assert.Equal(t, "function", tools[1].Type)
	assert.Equal(t, "test-tool-2", tools[1].Function.Name)
	assert.Equal(t, "Test tool 2", tools[1].Function.Description)
}

func TestToolManager_RegisterToolList_WithNilTool(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	mcpTools := []*mcp.Tool{
		{
			Name:        "valid-tool",
			Description: "Valid tool",
		},
		nil, // Nil tool should be skipped
	}

	manager.RegisterToolList(mcpTools)

	tools := manager.GetToolList()
	assert.Equal(t, 1, len(tools))
	assert.Equal(t, "valid-tool", tools[0].Function.Name)
}

func TestToolManager_RegisterToolCall(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultToolCallSize: 5,
	}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()
	toolName := "test-tool"

	// Register tool call
	manager.RegisterToolCall(requestID, toolCallID, toolName)

	// Verify tool call is registered
	assert.True(t, manager.HasToolCall(requestID, toolCallID))
	assert.True(t, manager.HasPendingCalls(requestID))
}

func TestToolManager_RegisterToolCall_ExistingRequest(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultToolCallSize: 5,
	}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()
	toolCallID1 := types.NewToolCallID()
	toolCallID2 := types.NewToolCallID()

	// Register first tool call
	manager.RegisterToolCall(requestID, toolCallID1, "tool-1")

	// Register second tool call for same request
	manager.RegisterToolCall(requestID, toolCallID2, "tool-2")

	// Both tool calls should exist
	assert.True(t, manager.HasToolCall(requestID, toolCallID1))
	assert.True(t, manager.HasToolCall(requestID, toolCallID2))
	assert.True(t, manager.HasPendingCalls(requestID))
}

func TestToolManager_HasToolCall_NonExistent(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Should return false for non-existent tool call
	assert.False(t, manager.HasToolCall(requestID, toolCallID))
}

func TestToolManager_CompleteToolCall(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultToolCallSize: 5,
	}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Register and then complete tool call
	manager.RegisterToolCall(requestID, toolCallID, "test-tool")
	assert.True(t, manager.HasToolCall(requestID, toolCallID))

	manager.CompleteToolCall(requestID, toolCallID)
	assert.False(t, manager.HasToolCall(requestID, toolCallID))
	assert.False(t, manager.HasPendingCalls(requestID))
}

func TestToolManager_CompleteToolCall_NonExistent(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Should not panic when completing non-existent tool call
	manager.CompleteToolCall(requestID, toolCallID)
	assert.False(t, manager.HasToolCall(requestID, toolCallID))
}

func TestToolManager_HasPendingCalls(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultToolCallSize: 5,
	}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()

	// No pending calls initially
	assert.False(t, manager.HasPendingCalls(requestID))

	// Add tool call
	toolCallID := types.NewToolCallID()
	manager.RegisterToolCall(requestID, toolCallID, "test-tool")
	assert.True(t, manager.HasPendingCalls(requestID))

	// Complete tool call
	manager.CompleteToolCall(requestID, toolCallID)
	assert.False(t, manager.HasPendingCalls(requestID))
}

func TestToolManager_ClearRequest(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultToolCallSize: 5,
	}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()
	toolCallID := types.NewToolCallID()

	// Register tool call
	manager.RegisterToolCall(requestID, toolCallID, "test-tool")
	assert.True(t, manager.HasToolCall(requestID, toolCallID))

	// Clear request
	manager.ClearRequest(requestID)
	assert.False(t, manager.HasToolCall(requestID, toolCallID))
	assert.False(t, manager.HasPendingCalls(requestID))
}

func TestToolManager_ClearRequest_NonExistent(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()

	// Should not panic when clearing non-existent request
	manager.ClearRequest(requestID)
	assert.False(t, manager.HasPendingCalls(requestID))
}

func TestToolManager_GetToolList_Empty(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	tools := manager.GetToolList()
	assert.NotNil(t, tools)
	assert.Equal(t, 0, len(tools))
}

func TestToolManager_RegisterToolList_EmptyList(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	// Register empty tool list
	manager.RegisterToolList([]*mcp.Tool{})

	tools := manager.GetToolList()
	assert.Equal(t, 0, len(tools))
}

func TestToolManager_RegisterToolList_OverwriteExisting(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{}
	manager := NewToolManager(ollamaConfig)

	// Register initial tools
	initialTools := []*mcp.Tool{
		{
			Name:        "initial-tool",
			Description: "Initial tool",
		},
	}
	manager.RegisterToolList(initialTools)
	assert.Equal(t, 1, len(manager.GetToolList()))

	// Register new tools (should overwrite)
	newTools := []*mcp.Tool{
		{
			Name:        "new-tool-1",
			Description: "New tool 1",
		},
		{
			Name:        "new-tool-2",
			Description: "New tool 2",
		},
	}
	manager.RegisterToolList(newTools)

	tools := manager.GetToolList()
	assert.Equal(t, 2, len(tools))
	assert.Equal(t, "new-tool-1", tools[0].Function.Name)
	assert.Equal(t, "new-tool-2", tools[1].Function.Name)
}

func TestToolManager_ThreadSafety_RegisterAndComplete(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultToolCallSize: 10,
	}
	manager := NewToolManager(ollamaConfig)

	requestID := types.NewRequestID()
	toolCallIDs := make([]types.ToolCallID, 5)

	// Register multiple tool calls
	for i := 0; i < 5; i++ {
		toolCallIDs[i] = types.NewToolCallID()
		manager.RegisterToolCall(requestID, toolCallIDs[i], "test-tool")
	}

	// Verify all are registered
	assert.True(t, manager.HasPendingCalls(requestID))
	for _, id := range toolCallIDs {
		assert.True(t, manager.HasToolCall(requestID, id))
	}

	// Complete all tool calls
	for _, id := range toolCallIDs {
		manager.CompleteToolCall(requestID, id)
	}

	// Verify all are completed
	assert.False(t, manager.HasPendingCalls(requestID))
	for _, id := range toolCallIDs {
		assert.False(t, manager.HasToolCall(requestID, id))
	}
}
