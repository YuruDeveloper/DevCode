package ollama

import (
	"DevCode/src/config"
	"DevCode/src/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
	"sync"
)

type RequestContext struct {
	ToolCalls map[types.ToolCallID]string
}

func NewToolManager(config config.OllamaServiceConfig) *ToolManager {
	return &ToolManager{
		tools:           make([]api.Tool, 0, config.DefaultToolSize),
		requestContents: make(map[types.RequestID]*RequestContext, config.DefaultRequestContentsSize),
		config:          config,
	}
}

type ToolManager struct {
	config          config.OllamaServiceConfig
	tools           []api.Tool
	requestContents map[types.RequestID]*RequestContext
	requestMutex    sync.RWMutex
}

func (instance *ToolManager) RegisterToolList(tools []*mcp.Tool) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	if instance.tools == nil {
		instance.tools = make([]api.Tool, 0, instance.config.DefaultToolSize)
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

func (instance *ToolManager) RegisterToolCall(requestID types.RequestID, toolCallID types.ToolCallID, toolName string) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	if content, exists := instance.requestContents[requestID]; exists {
		content.ToolCalls[toolCallID] = toolName
	} else {
		instance.requestContents[requestID] = &RequestContext{
			ToolCalls: make(map[types.ToolCallID]string, instance.config.DefaultToolCallSize),
		}
		instance.requestContents[requestID].ToolCalls[toolCallID] = toolName
	}
}

func (instance *ToolManager) HasToolCall(requestID types.RequestID, toolCallID types.ToolCallID) bool {
	instance.requestMutex.RLock()
	defer instance.requestMutex.RUnlock()
	if content, exists := instance.requestContents[requestID]; exists {
		if content.ToolCalls == nil {
			return false
		}
		if _, exists := content.ToolCalls[toolCallID]; exists {
			return true
		}
	}
	return false
}

func (instance *ToolManager) CompleteToolCall(requestID types.RequestID, toolCallID types.ToolCallID) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	if content, exists := instance.requestContents[requestID]; exists {
		if content.ToolCalls == nil {
			return
		}
		delete(content.ToolCalls, toolCallID)
	}
}

func (instance *ToolManager) HasPendingCalls(requestID types.RequestID) bool {
	instance.requestMutex.RLock()
	defer instance.requestMutex.RUnlock()
	if content, exists := instance.requestContents[requestID]; exists {
		if content.ToolCalls == nil {
			return false
		}
		return len(content.ToolCalls) > 0
	}
	return false
}

func (instance *ToolManager) ClearRequest(requestID types.RequestID) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()
	delete(instance.requestContents, requestID)
}
