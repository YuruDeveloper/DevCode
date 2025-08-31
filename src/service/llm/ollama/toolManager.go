package ollama

import (
	"sync"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
)

const (
	DefaultToolSize            = 10
	DefaultRequestContentsSize = 10
	DefaultToolCallSize        = 20
)

type RequestContext struct {
	ToolCalls map[uuid.UUID]string
}

func NewToolManager() *ToolManager {
	return &ToolManager{
		tools:           make([]api.Tool, 0, DefaultToolSize),
		requestContents: make(map[uuid.UUID]*RequestContext, DefaultRequestContentsSize),
	}
}

type ToolManager struct {
	tools           []api.Tool
	requestContents map[uuid.UUID]*RequestContext
	requestMutex    sync.RWMutex
}

func (instance *ToolManager) RegisterToolList(tools []*mcp.Tool) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	if instance.tools == nil {
		instance.tools = make([]api.Tool, 0, DefaultToolSize)
	}
	instance.tools = instance.tools[:0]
	for _, tool := range tools {
		if tool == nil {
			continue
		}
		instance.tools = append(instance.tools, ConvertTool(tool))
	}
}

func (instance *ToolManager) GetToolList() []api.Tool {
	instance.requestMutex.RLock()
	defer instance.requestMutex.RUnlock()
	return instance.tools
}

func (instance *ToolManager) RegisterToolCall(requestUUID uuid.UUID, toolCallUUID uuid.UUID, toolName string) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	if instance.requestContents == nil {
		instance.requestContents = make(map[uuid.UUID]*RequestContext, DefaultRequestContentsSize)
	}
	if content, exists := instance.requestContents[requestUUID]; exists {
		if content.ToolCalls == nil {
			content.ToolCalls = make(map[uuid.UUID]string, DefaultToolCallSize)
		}
		content.ToolCalls[toolCallUUID] = toolName
	} else {
		instance.requestContents[requestUUID] = &RequestContext{
			ToolCalls: make(map[uuid.UUID]string, DefaultToolCallSize),
		}
		instance.requestContents[requestUUID].ToolCalls[toolCallUUID] = toolName
	}
}

func (instance *ToolManager) HasToolCall(requestUUID uuid.UUID, toolCallUUID uuid.UUID) bool {
	instance.requestMutex.RLock()
	defer instance.requestMutex.RUnlock()
	if content, exists := instance.requestContents[requestUUID]; exists {
		if content.ToolCalls == nil {
			return false
		}
		if _, exists := content.ToolCalls[toolCallUUID]; exists {
			return true
		}
	}
	return false
}

func (instance *ToolManager) CompleteToolCall(requestUUID uuid.UUID, toolCallUUID uuid.UUID) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	if content, exists := instance.requestContents[requestUUID]; exists {
		if content.ToolCalls == nil {
			return
		}
		delete(content.ToolCalls, toolCallUUID)
	}
}

func (instance *ToolManager) HasPendingCalls(requestUUID uuid.UUID) bool {
	instance.requestMutex.RLock()
	defer instance.requestMutex.RUnlock()
	if content, exists := instance.requestContents[requestUUID]; exists {
		if content.ToolCalls == nil {
			return false
		}
		return len(content.ToolCalls) > 0
	}
	return false
}

func (instance *ToolManager) ClearRequest(requestUUID uuid.UUID) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	delete(instance.requestContents, requestUUID)
}
