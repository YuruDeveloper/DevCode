package ollama

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service"
	"DevCode/src/utils"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
	"github.com/spf13/viper"
)




type OllamaService struct {
	client *api.Client
	model  string
	bus    events.Bus
	messageManager IMessageManager
	toolManager IToolManager
	StreamManager IStreamManager
}

func NewOllamaService(bus events.Bus) (*OllamaService, error) {

	requireds := []string{"ollama.url", "ollama.model", "prompt.system"}
	data := make([]string, 3)
	for index, required := range requireds {
		data[index] = viper.GetString(required)
	}

	parsedUrl, err := url.Parse(data[0])
	if err != nil {
		return nil, fmt.Errorf("invalid Ollama URL: %v", err)
	}

	ollamaClient := api.NewClient(parsedUrl, http.DefaultClient)

	if data[2] == "" {
		return nil, fmt.Errorf("prompt.system not configured in env.toml")
	}

	systemPrompt, err := os.ReadFile(data[2])

	if err != nil {
		return nil, fmt.Errorf("fail to Read SystemPrompt %v", err)
	}

	service := &OllamaService{
		client:          ollamaClient,
		model:           data[1],
		bus:             bus,
		messageManager: NewMessageManager(),
		toolManager: NewToolManager(),
		StreamManager: NewStreamManager(),
	}
	service.messageManager.AddSystemMessage(string(systemPrompt))
	bus.Subscribe(events.UserInputEvent, service)
	bus.Subscribe(events.UpdateEnvironmentEvent, service)
	bus.Subscribe(events.UpdateToolListEvent, service)
	bus.Subscribe(events.StreamCancelEvent, service)
	bus.Subscribe(events.ToolResultEvent, service)
	return service, nil
}

func (instance *OllamaService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.UserInputEvent:
		instance.messageManager.AddUserMessage(event.Data.(dto.UserRequestData).Message)
		instance.UpdateEnvironmentToolList()
		instance.CallApi(event.Data.(dto.UserRequestData).RequestUUID)
	case events.UpdateEnvironmentEvent:
		instance.messageManager.SetEnvironmentMessage(utils.EnvironmentUpdateDataToString(event.Data.(dto.EnvironmentUpdateData)))
	case events.UpdateToolListEvent:
		instance.toolManager.RegisterToolList(event.Data.(dto.ToolListUpdateData).List)
	case events.StreamCancelEvent:
		instance.CancelStream(event.Data.(dto.StreamCancelData).RequestUUID)
	case events.ToolResultEvent:
		instance.ProcessToolResult(event.Data.(dto.ToolResultData))
	}
}

func (instance *OllamaService) ProcessToolResult(data dto.ToolResultData) {
	if instance.toolManager.HasToolCall(data.RequestUUID,data.ToolCallUUID) {
		instance.messageManager.AddToolMessage(data.ToolResult)
		instance.toolManager.CompleteToolCall(data.RequestUUID,data.ToolCallUUID)
		if !instance.toolManager.HasPendingCalls(data.RequestUUID) {
			instance.toolManager.ClearRequest(data.RequestUUID)
			instance.CallApi(data.RequestUUID)
		}
	}
}

func (instance *OllamaService) GetID() constants.Source {
	return constants.LLMService
}

func (instance *OllamaService) UpdateEnvironmentToolList() {
	instance.bus.Publish(
		events.Event{
			Type: events.RequestEnvironmentEvent,
			Data: dto.EnvironmentRequestData{
				CreateUUID: uuid.New(),
			},
			Timestamp: time.Now(),
			Source:    constants.LLMService,
		},
	)
	instance.bus.Publish(
		events.Event{
			Type: events.RequestToolListEvent,
			Data: dto.RequestToolListData{
				CreateUUID: uuid.New(),
			},
			Timestamp: time.Now(),
			Source:    constants.LLMService,
		},
	)
}

func (instance *OllamaService) AddAssistantMessage(message string) {
	instance.messageManager.AddAssistantMessage(message)
}

func (instance *OllamaService) CallApi(requestUUID uuid.UUID) {
	instance.bus.Publish(
		events.Event{
			Type: events.StreamStartEvent,
			Data: dto.StreamStartData{
				RequestUUID: requestUUID,
			},
			Timestamp: time.Now(),
			Source:    constants.LLMService,
		},
	)
	instance.StreamManager.StartStream(instance.client,
		instance.bus,requestUUID,
		instance.model,
		instance.toolManager.GetToolList(),
		instance.messageManager.GetMessages(),
	func(requestUUID uuid.UUID, response api.ChatResponse) error {
		return instance.StreamManager.Response(
			requestUUID,
			response,instance.bus,
			instance.AddAssistantMessage,
			instance.toolManager.HasPendingCalls,
			instance.ProcessToolCalls,
		)
	})
}

func (instance *OllamaService) ProcessToolCalls(requestUUID uuid.UUID,ToolCalls []api.ToolCall) {
	for _ , call := range ToolCalls {
		toolCallUUID := uuid.New()
		service.PublishEvent(instance.bus,events.ToolCallEvent,dto.ToolCallData {
			RequestUUID: requestUUID,
			ToolCallUUID: toolCallUUID,
			ToolName: call.Function.Name,
			Parameters: call.Function.Arguments,
		},constants.LLMService)
		instance.toolManager.RegisterToolCall(requestUUID,toolCallUUID,call.Function.Name)
	}
}

func (instance *OllamaService) CancelStream(requestUUID uuid.UUID) {
	instance.StreamManager.CancelStream(requestUUID)
	instance.toolManager.ClearRequest(requestUUID)
}
