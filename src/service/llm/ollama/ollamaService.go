package ollama

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/utils"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

type OllamaService struct {
	client         *api.Client
	config         config.OllamaServiceConfig
	bus            *events.EventBus
	messageManager IMessageManager
	toolManager    IToolManager
	StreamManager  IStreamManager
}

func NewOllamaService(bus *events.EventBus, config config.OllamaServiceConfig) *OllamaService {
	ollamaClient := api.NewClient(config.Url, http.DefaultClient)
	service := &OllamaService{
		client:         ollamaClient,
		config:         config,
		bus:            bus,
		messageManager: NewMessageManager(config),
		toolManager:    NewToolManager(config),
		StreamManager:  NewStreamManager(config),
	}
	service.messageManager.AddSystemMessage(config.Prompt)
	service.Subscribe()
	return service
}

func (instance *OllamaService) Subscribe() {
	instance.bus.UserInputEvent.Subscribe(constants.LLMService, func(event events.Event[dto.UserRequestData]) {
		instance.messageManager.AddUserMessage(event.Data.Message)
		instance.UpdateEnvironmentToolList()
		instance.CallApi(event.Data.RequestUUID)
	})
	instance.bus.UpdateEnvironmentEvent.Subscribe(constants.LLMService, func(event events.Event[dto.EnvironmentUpdateData]) {
		instance.messageManager.SetEnvironmentMessage(utils.EnvironmentUpdateDataToString(event.Data))
	})
	instance.bus.UpdateToolListEvent.Subscribe(constants.LLMService, func(event events.Event[dto.ToolListUpdateData]) {
		instance.toolManager.RegisterToolList(event.Data.List)
	})
	instance.bus.StreamCancelEvent.Subscribe(constants.LLMService, func(event events.Event[dto.StreamCancelData]) {
		instance.CancelStream(event.Data.RequestUUID)
	})
	instance.bus.ToolResultEvent.Subscribe(constants.LLMService, func(event events.Event[dto.ToolResultData]) {
		instance.ProcessToolResult(event.Data)
	})
}

func (instance *OllamaService) ProcessToolResult(data dto.ToolResultData) {
	if instance.toolManager.HasToolCall(data.RequestUUID, data.ToolCallUUID) {
		instance.messageManager.AddToolMessage(data.ToolResult)
		instance.toolManager.CompleteToolCall(data.RequestUUID, data.ToolCallUUID)
		if !instance.toolManager.HasPendingCalls(data.RequestUUID) {
			instance.toolManager.ClearRequest(data.RequestUUID)
			instance.CallApi(data.RequestUUID)
		}
	}
}

func (instance *OllamaService) UpdateEnvironmentToolList() {
	instance.bus.RequestEnvironmentEvent.Publish(
		events.Event[dto.EnvironmentRequestData]{
			Data: dto.EnvironmentRequestData{
				CreateUUID: uuid.New(),
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMService,
		})
	instance.bus.RequestToolListEvent.Publish(events.Event[dto.RequestToolListData]{
		Data: dto.RequestToolListData{
			CreateUUID: uuid.New(),
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMService,
	})
}

func (instance *OllamaService) AddAssistantMessage(message string) {
	instance.messageManager.AddAssistantMessage(message)
}

func (instance *OllamaService) CallApi(requestUUID uuid.UUID) {
	instance.bus.StreamStartEvent.Publish(events.Event[dto.StreamStartData]{
		Data: dto.StreamStartData{
			RequestUUID: requestUUID,
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMService,
	})
	instance.StreamManager.StartStream(instance.client,
		instance.bus, requestUUID,
		instance.config.Model,
		instance.toolManager.GetToolList(),
		instance.messageManager.GetMessages(),
		func(requestUUID uuid.UUID, response api.ChatResponse) error {
			return instance.StreamManager.Response(
				requestUUID,
				response, instance.bus,
				instance.AddAssistantMessage,
				instance.toolManager.HasPendingCalls,
				instance.ProcessToolCalls,
			)
		})
}

func (instance *OllamaService) ProcessToolCalls(requestUUID uuid.UUID, ToolCalls []api.ToolCall) {
	for _, call := range ToolCalls {
		toolCallUUID := uuid.New()
		instance.bus.ToolCallEvent.Publish(events.Event[dto.ToolCallData]{
			Data: dto.ToolCallData{
				RequestUUID:  requestUUID,
				ToolCallUUID: toolCallUUID,
				ToolName:     call.Function.Name,
				Parameters:   call.Function.Arguments,
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMService,
		})
		instance.toolManager.RegisterToolCall(requestUUID, toolCallUUID, call.Function.Name)
	}
}

func (instance *OllamaService) CancelStream(requestUUID uuid.UUID) {
	instance.StreamManager.CancelStream(requestUUID)
	instance.toolManager.ClearRequest(requestUUID)
}
