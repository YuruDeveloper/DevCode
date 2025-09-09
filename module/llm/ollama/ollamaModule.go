package ollama

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"DevCode/types"
	"DevCode/utils"
	"net/http"
	"time"

	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

type OllamaModule struct {
	client         *api.Client
	config         config.OllamaServiceConfig
	bus            *events.EventBus
	messageManager IMessageManager
	toolManager    IToolManager

	StreamManager IStreamManager
	logger        *zap.Logger
}

func NewOllamaModule(bus *events.EventBus, config config.OllamaServiceConfig, logger *zap.Logger) *OllamaModule {
	ollamaClient := api.NewClient(config.Url, http.DefaultClient)
	module := &OllamaModule{
		client:         ollamaClient,
		config:         config,
		bus:            bus,
		messageManager: NewMessageManager(config),
		toolManager:    NewToolManager(config),
		StreamManager:  NewStreamManager(config),
		logger:         logger,
	}
	module.messageManager.AddSystemMessage(config.Prompt)
	module.Subscribe()
	return module
}

func (instance *OllamaModule) Subscribe() {
	events.Subscribe(instance.bus, instance.bus.UserInputEvent, constants.LLMModule, func(event events.Event[dto.UserRequestData]) {
		instance.messageManager.AddUserMessage(event.Data.Message)
		instance.UpdateEnvironmentToolList()
		instance.CallApi(event.Data.RequestID)
	})
	events.Subscribe(instance.bus, instance.bus.UpdateEnvironmentEvent, constants.LLMModule, func(event events.Event[dto.EnvironmentUpdateData]) {
		instance.messageManager.SetEnvironmentMessage(utils.EnvironmentUpdateDataToString(event.Data))
	})
	events.Subscribe(instance.bus, instance.bus.UpdateToolListEvent, constants.LLMModule, func(event events.Event[dto.ToolListUpdateData]) {
		instance.toolManager.RegisterToolList(event.Data.List)
	})
	events.Subscribe(instance.bus, instance.bus.StreamCancelEvent, constants.LLMModule, func(event events.Event[dto.StreamCancelData]) {
		instance.CancelStream(event.Data.RequestID)
	})
	events.Subscribe(instance.bus, instance.bus.ToolResultEvent, constants.LLMModule, func(event events.Event[dto.ToolResultData]) {
		instance.ProcessToolResult(event.Data)
	})
}

func (instance *OllamaModule) ProcessToolResult(data dto.ToolResultData) {
	if instance.toolManager.HasToolCall(data.RequestID, data.ToolCallID) {
		instance.messageManager.AddToolMessage(data.ToolResult)
		instance.toolManager.CompleteToolCall(data.RequestID, data.ToolCallID)
		if !instance.toolManager.HasPendingCalls(data.RequestID) {
			instance.toolManager.ClearRequest(data.RequestID)
			instance.CallApi(data.RequestID)
		}
	} else {
		instance.logger.Warn("Tool call not found",
			zap.String("requestUUID", data.RequestID.String()),
			zap.String("toolCallUUID", data.ToolCallID.String()))
		return
	}
}

func (instance *OllamaModule) UpdateEnvironmentToolList() {
	events.Publish(instance.bus, instance.bus.RequestEnvironmentEvent, events.Event[dto.EnvironmentRequestData]{
		Data: dto.EnvironmentRequestData{
			CreateID: types.NewCreateID(),
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMModule,
	})
	events.Publish(instance.bus, instance.bus.RequestToolListEvent, events.Event[dto.RequestToolListData]{
		Data: dto.RequestToolListData{
			CreateID: types.NewCreateID(),
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMModule,
	})
}

func (instance *OllamaModule) AddAssistantMessage(message string) {
	instance.messageManager.AddAssistantMessage(message)
}

func (instance *OllamaModule) CallApi(requestID types.RequestID) {
	events.Publish(instance.bus, instance.bus.StreamStartEvent, events.Event[dto.StreamStartData]{
		Data: dto.StreamStartData{
			RequestID: requestID,
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMModule,
	})
	instance.StreamManager.StartStream(instance.client,
		instance.bus, requestID,
		instance.config.Model,
		instance.toolManager.GetToolList(),
		instance.messageManager.GetMessages(),
		func(requestID types.RequestID, response api.ChatResponse) error {
			return instance.StreamManager.Response(
				requestID,
				response, instance.bus,
				instance.AddAssistantMessage,
				instance.toolManager.HasPendingCalls,
				instance.ProcessToolCalls,
			)
		})
}

func (instance *OllamaModule) ProcessToolCalls(requestID types.RequestID, ToolCalls []api.ToolCall) {
	for _, call := range ToolCalls {
		toolCallID := types.NewToolCallID()
		events.Publish(instance.bus, instance.bus.ToolCallEvent, events.Event[dto.ToolCallData]{
			Data: dto.ToolCallData{
				RequestID:  requestID,
				ToolCallID: toolCallID,
				ToolName:   call.Function.Name,
				Parameters: call.Function.Arguments,
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMModule,
		})
		instance.toolManager.RegisterToolCall(requestID, toolCallID, call.Function.Name)
	}
}

func (instance *OllamaModule) CancelStream(requestID types.RequestID) {
	instance.StreamManager.CancelStream(requestID)
	instance.toolManager.ClearRequest(requestID)
}
