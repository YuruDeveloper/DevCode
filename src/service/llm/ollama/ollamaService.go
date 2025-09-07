package ollama

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/types"
	"DevCode/src/utils"
	"net/http"
	"time"

	"github.com/ollama/ollama/api"
	"go.uber.org/zap"
)

type OllamaService struct {
	client         *api.Client
	config         config.OllamaServiceConfig
	bus            *events.EventBus
	messageManager IMessageManager
	toolManager    IToolManager
	StreamManager  IStreamManager
	logger         *zap.Logger
}

func NewOllamaService(bus *events.EventBus, config config.OllamaServiceConfig, logger *zap.Logger) *OllamaService {
	ollamaClient := api.NewClient(config.Url, http.DefaultClient)
	service := &OllamaService{
		client:         ollamaClient,
		config:         config,
		bus:            bus,
		messageManager: NewMessageManager(config),
		toolManager:    NewToolManager(config),
		StreamManager:  NewStreamManager(config),
		logger:         logger,
	}
	service.messageManager.AddSystemMessage(config.Prompt)
	service.Subscribe()
	return service
}

func (instance *OllamaService) Subscribe() {
	instance.bus.UserInputEvent.Subscribe(constants.LLMService, func(event events.Event[dto.UserRequestData]) {
		instance.messageManager.AddUserMessage(event.Data.Message)
		instance.UpdateEnvironmentToolList()
		instance.CallApi(event.Data.RequestID)
	})
	instance.bus.UpdateEnvironmentEvent.Subscribe(constants.LLMService, func(event events.Event[dto.EnvironmentUpdateData]) {
		instance.messageManager.SetEnvironmentMessage(utils.EnvironmentUpdateDataToString(event.Data))
	})
	instance.bus.UpdateToolListEvent.Subscribe(constants.LLMService, func(event events.Event[dto.ToolListUpdateData]) {
		instance.toolManager.RegisterToolList(event.Data.List)
	})
	instance.bus.StreamCancelEvent.Subscribe(constants.LLMService, func(event events.Event[dto.StreamCancelData]) {
		instance.CancelStream(event.Data.RequestID)
	})
	instance.bus.ToolResultEvent.Subscribe(constants.LLMService, func(event events.Event[dto.ToolResultData]) {
		instance.ProcessToolResult(event.Data)
	})
}

func (instance *OllamaService) ProcessToolResult(data dto.ToolResultData) {
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
	}
}

func (instance *OllamaService) UpdateEnvironmentToolList() {
	instance.bus.RequestEnvironmentEvent.Publish(
		events.Event[dto.EnvironmentRequestData]{
			Data: dto.EnvironmentRequestData{
				CreateID: types.NewCreateID(),
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMService,
		})
	instance.bus.RequestToolListEvent.Publish(events.Event[dto.RequestToolListData]{
		Data: dto.RequestToolListData{
			CreateID: types.NewCreateID(),
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMService,
	})
}

func (instance *OllamaService) AddAssistantMessage(message string) {
	instance.messageManager.AddAssistantMessage(message)
}

func (instance *OllamaService) CallApi(requestID types.RequestID) {
	instance.bus.StreamStartEvent.Publish(events.Event[dto.StreamStartData]{
		Data: dto.StreamStartData{
			RequestID: requestID,
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMService,
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

func (instance *OllamaService) ProcessToolCalls(requestID types.RequestID, ToolCalls []api.ToolCall) {
	for _, call := range ToolCalls {
		toolCallID := types.NewToolCallID()
		instance.bus.ToolCallEvent.Publish(events.Event[dto.ToolCallData]{
			Data: dto.ToolCallData{
				RequestID:  requestID,
				ToolCallID: toolCallID,
				ToolName:     call.Function.Name,
				Parameters:   call.Function.Arguments,
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMService,
		})
		instance.toolManager.RegisterToolCall(requestID, toolCallID, call.Function.Name)
	}
}

func (instance *OllamaService) CancelStream(requestID types.RequestID) {
	instance.StreamManager.CancelStream(requestID)
	instance.toolManager.ClearRequest(requestID)
}
