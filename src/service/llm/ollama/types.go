package ollama

import (
	"DevCode/src/events"
	"DevCode/src/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
)

type IMessageManager interface {
	AddSystemMessage(content string)
	SetEnvironmentMessage(content string)
	AddUserMessage(content string)
	AddAssistantMessage(content string)
	AddToolMessage(content string)
	Clear()
	GetMessages() []api.Message
}

type IToolManager interface {
	RegisterToolList(tools []*mcp.Tool)
	GetToolList() []api.Tool
	RegisterToolCall(requestID types.RequestID, toolCallID types.ToolCallID, toolName string)
	HasToolCall(requestID types.RequestID, toolCallID types.ToolCallID) bool
	CompleteToolCall(requestID types.RequestID, toolCallID types.ToolCallID)
	HasPendingCalls(requestID types.RequestID) bool
	ClearRequest(requestID types.RequestID)
}

type IStreamManager interface {
	StartStream(
		ollama *api.Client,
		bus *events.EventBus,
		requestID types.RequestID,
		model string,
		tools []api.Tool,
		message []api.Message,
		callBack func(requestID types.RequestID, response api.ChatResponse) error,
	)
	Response(
		requestID types.RequestID,
		response api.ChatResponse,
		bus *events.EventBus,
		doneCallBack func(string),
		checkDone func(types.RequestID) bool,
		toolsCallBack func(types.RequestID, []api.ToolCall),
	) error
	CancelStream(requestUUID types.RequestID)
}
