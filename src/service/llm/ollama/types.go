package ollama

import (
	"DevCode/src/events"
	"github.com/google/uuid"
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
	RegisterToolCall(requestUUID uuid.UUID, toolCallUUID uuid.UUID, toolName string)
	HasToolCall(requestUUID uuid.UUID, toolCallUUID uuid.UUID) bool
	CompleteToolCall(requestUUID uuid.UUID, toolCallUUID uuid.UUID)
	HasPendingCalls(requestUUID uuid.UUID) bool
	ClearRequest(requestUUID uuid.UUID)
}

type IStreamManager interface {
	StartStream(
		ollama *api.Client,
		bus *events.EventBus,
		requestUUID uuid.UUID,
		model string,
		tools []api.Tool,
		message []api.Message,
		callBack func(requestUUID uuid.UUID, response api.ChatResponse) error,
	)
	Response(
		requestUUID uuid.UUID,
		response api.ChatResponse,
		bus *events.EventBus,
		doneCallBack func(string),
		checkDone func(uuid.UUID) bool,
		toolsCallBack func(uuid.UUID, []api.ToolCall),
	) error
	CancelStream(requestUUID uuid.UUID)
}
