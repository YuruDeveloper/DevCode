package ollama

import (
	"UniCode/src/constants"
	"UniCode/src/dto"
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/utils"
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/ollama/ollama/api"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

const (
	EnvironmentInfo = "Here is useful information about the environment you are running in:\n"
)

type RequestContext struct {
	RequestUUID uuid.UUID
	ToolCalls   map[uuid.UUID]string
}

type OllamaService struct {
	client *api.Client
	model  string
	ctx    context.Context
	bus    events.Bus
	logger *zap.Logger

	systemMessages []api.Message
	messages       []api.Message

	tools []api.Tool

	environment string

	activeStreams map[uuid.UUID]context.CancelFunc
	streamMutex   sync.RWMutex
	buffer        string

	requestContents map[uuid.UUID]RequestContext
	requestMutex    sync.RWMutex
}

func NewOllamaService(bus events.Bus, logger *zap.Logger) (*OllamaService, error) {

	requireds := []string{"ollama.url", "ollama.model", "prompt.system"}
	data := make([]string, 0, 3)
	for index, required := range requireds {
		data[index] = viper.GetString(required)
	}

	parsedUrl, err := url.Parse(data[0])
	if err != nil {
		logger.Error("Failed to parse Ollama URL",
			zap.String("url", data[0]),
			zap.Error(err),
		)
		return nil, fmt.Errorf("invalid Ollama URL: %v", err)
	}

	ollamaClient := api.NewClient(parsedUrl, http.DefaultClient)

	if data[2] == "" {
		logger.Error("System prompt not configured")
		return nil, fmt.Errorf("prompt.system not configured in env.toml")
	}

	systemPrompt, err := os.ReadFile(data[2])

	if err != nil {
		logger.Error("Failed to read system prompt file",
			zap.String("file", data[2]),
			zap.Error(err),
		)
		return nil, fmt.Errorf("fail to Read SystemPrompt %v", err)
	}

	systemMessages := make([]api.Message, 0, 10)

	systemMessages = append(systemMessages, api.Message{
		Role:    "system",
		Content: string(systemPrompt),
	})

	ctx := context.Background()

	service := &OllamaService{
		client:          ollamaClient,
		model:           data[1],
		ctx:             ctx,
		bus:             bus,
		systemMessages:  systemMessages,
		messages:        make([]api.Message, 0, 100),
		tools:           make([]api.Tool, 0, 10),
		requestContents: make(map[uuid.UUID]RequestContext, 10),
	}
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
		instance.UpdateUserInput(event.Data.(dto.UserRequestData).Message)
		instance.UpdateEnvironmentToolList()
		instance.CallApi(event.Data.(dto.UserRequestData).RequestUUID)
	case events.UpdateEnvironmentEvent:
		instance.environment = utils.EnvironmentUpdateDataToString(event.Data.(dto.EnvironmentUpdateData))
	case events.UpdateToolListEvent:
		instance.UpdateToolList(event.Data.(dto.ToolListUpdateData).List)
	case events.StreamCancelEvent:
		instance.CancelStream(event.Data.(dto.StreamCancelData).RequestUUID)
	case events.ToolResultEvent:
		instance.ProcessToolResult(event.Data.(dto.ToolResultData))
	}
}

func (instance *OllamaService) ProcessToolResult(data dto.ToolResultData) {
	instance.requestMutex.Lock()
	defer instance.requestMutex.Unlock()

	instance.logger.Debug("Processing tool result",
		zap.String("requestUUID", data.RequestUUID.String()),
		zap.String("toolCall", data.ToolCall.String()))
	if _, exists := instance.requestContents[data.RequestUUID].ToolCalls[data.ToolCall]; exists {
		msg := api.Message{
			Role:    "tool",
			Content: data.ToolResult,
		}
		instance.messages = append(instance.messages, msg)
		delete(instance.requestContents[data.RequestUUID].ToolCalls, data.ToolCall)
		if len(instance.requestContents[data.RequestUUID].ToolCalls) == 0 {
			delete(instance.requestContents, data.RequestUUID)
			instance.CallApi(data.RequestUUID)
		}
	}
}

func (instance *OllamaService) HasActiveTollCalls(requestUUID uuid.UUID) bool {
	instance.requestMutex.RLock()
	defer instance.requestMutex.RUnlock()

	if content, exists := instance.requestContents[requestUUID]; exists {
		return len(content.ToolCalls) > 0
	}
	return false
}

func (instance *OllamaService) EnvironmentMessage() *api.Message {
	return &api.Message{
		Role:    "system",
		Content: EnvironmentInfo + instance.environment,
	}
}

func (instance *OllamaService) UpdateToolList(data []*mcp.Tool) {
	instance.tools = make([]api.Tool, 0, len(data))
	instance.logger.Info("Updating tool list", zap.Int("toolCount", len(data)))
	for _, tool := range data {
		if tool == nil {
			continue
		}
		instance.tools = append(instance.tools, ConvertTool(tool))
	}
}

func (instance *OllamaService) UpdateUserInput(message string) {
	instance.messages = append(instance.messages, api.Message{
		Role:    "user",
		Content: message,
	})
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
	ctx, cancel := context.WithCancel(instance.ctx)

	instance.streamMutex.Lock()
	if instance.activeStreams == nil {
		instance.activeStreams = make(map[uuid.UUID]context.CancelFunc)
	}
	instance.activeStreams[requestUUID] = cancel
	instance.streamMutex.Unlock()

	request := api.ChatRequest{
		Model:    instance.model,
		Messages: append(append(instance.systemMessages, *instance.EnvironmentMessage()), instance.messages...),
		Tools:    instance.tools,
		Stream:   &[]bool{true}[0],
	}

	go func() {
		defer func() {
			instance.streamMutex.Lock()
			delete(instance.activeStreams, requestUUID)
			instance.streamMutex.Unlock()
		}()

		err := instance.client.Chat(ctx, &request, func(response api.ChatResponse) error {
			return instance.Response(requestUUID, response)
		})

		if err != nil {
			instance.logger.Error("Chat API call failed",
				zap.String("requestUUID",
					requestUUID.String()),
				zap.Error(err),
			)
			instance.bus.Publish(
				events.Event{
					Type: events.StreamErrorEvent,
					Data: dto.StreamErrorData{
						RequestUUID: requestUUID,
						Error:       err,
					},
					Timestamp: time.Now(),
					Source:    constants.LLMService,
				},
			)
		}
	}()
}

func (instance *OllamaService) Response(requestUUID uuid.UUID, response api.ChatResponse) error {

	if response.Message.Content != "" {
		service.PublishEvent(instance.bus, events.StreamChunkEvent, dto.StreamChunkData{
			RequestUUID: requestUUID,
			Content:     response.Message.Content,
			IsComplete:  response.Done}, constants.LLMService)
		instance.buffer += response.Message.Content
	}
	if response.Done {
		service.PublishEvent(instance.bus, events.StreamCompleteEvent, dto.StreamCompleteData{
			RequestUUID:  requestUUID,
			FinalMessage: response.Message.Content,
			IsComplete:   !instance.HasActiveTollCalls(requestUUID),
		}, constants.LLMService)

		instance.messages = append(instance.messages, api.Message{
			Role:    "assistant",
			Content: instance.buffer,
		})
		instance.buffer = ""
	}

	if len(response.Message.ToolCalls) > 0 {
		instance.logger.Debug("Processing tool calls",
			zap.String("requestUUID", requestUUID.String()),
			zap.Int("toolCallCount", len(response.Message.ToolCalls)))
		for _, call := range response.Message.ToolCalls {
			toolCall := uuid.New()
			instance.logger.Info("Tool call initiated",
				zap.String("toolName", call.Function.Name),
				zap.String("toolCallUUID", toolCall.String()))
			service.PublishEvent(instance.bus, events.ToolCallEvent, dto.ToolCallData{
				RequestUUID:  requestUUID,
				ToolCallUUID: toolCall,
				ToolName:     call.Function.Name,
				Parameters:   call.Function.Arguments,
			}, constants.LLMService)
			instance.requestMutex.Lock()
			if content, exist := instance.requestContents[requestUUID]; exist {
				content.ToolCalls[toolCall] = call.Function.Name
			} else {

				instance.requestContents[requestUUID] = RequestContext{
					RequestUUID: requestUUID,
					ToolCalls:   make(map[uuid.UUID]string),
				}
				instance.requestContents[requestUUID].ToolCalls[toolCall] = call.Function.Name
			}
			instance.requestMutex.Unlock()
		}
	}
	return nil
}

func (instance *OllamaService) CancelStream(requestUUID uuid.UUID) {
	instance.logger.Info("Cancelling stream",
		zap.String("requestUUID",
			requestUUID.String()),
	)
	instance.streamMutex.RLock()
	cancel, exists := instance.activeStreams[requestUUID]
	instance.streamMutex.RUnlock()
	if exists {
		cancel()
	}
	instance.requestMutex.Lock()
	_, exists = instance.requestContents[requestUUID]
	if exists {
		delete(instance.requestContents, requestUUID)
	}
	instance.requestMutex.Unlock()
}
