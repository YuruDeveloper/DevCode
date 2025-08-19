package service

import (
	"UniCode/src/events"
	"UniCode/src/types"
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
)

const (
	EnvironmentInfo = "  Here is useful information about the environment you are running in:\n"
)

type RequestContext struct {
	RequestUUID uuid.UUID
	ToolCalls   map[uuid.UUID]string
}

type OllamaService struct {
	Client          *api.Client
	Model           string
	Ctx             context.Context
	Bus             *events.EventBus
	SystemMessages  []api.Message
	Messages        []api.Message
	Tools           []api.Tool
	Environment     string
	ActiveStreams   map[uuid.UUID]context.CancelFunc
	StreamMutex     sync.RWMutex
	Buffer          string
	RequestContents map[uuid.UUID]RequestContext
	RequestMutex    sync.RWMutex
	ToolCall        bool
}

func NewOllamaService(bus *events.EventBus) *OllamaService {
	ollamaUrl := viper.GetString("ollama.url")
	parsedUrl, err := url.Parse(ollamaUrl)
	if err != nil {
		panic(fmt.Sprintf("Invalid Ollama URL: %v", err))
	}
	ollama := *api.NewClient(parsedUrl, http.DefaultClient)

	ctx := context.Background()
	ollamaModel := viper.GetString("ollama.model")
	ollamaPath := viper.GetString("prompt.system")
	if ollamaPath == "" {
		panic("prompt.system not configured in env.toml")
	}

	systemPrompt, err := os.ReadFile(ollamaPath)
	systemMessages := make([]api.Message, 0, 10)
	systemMessages = append(systemMessages, api.Message{
		Role:    "system",
		Content: string(systemPrompt),
	})
	if err != nil {
		panic(err)
	}
	service := &OllamaService{
		Client:          &ollama,
		Model:           ollamaModel,
		Ctx:             ctx,
		Bus:             bus,
		SystemMessages:  systemMessages,
		Messages:        make([]api.Message, 0, 100),
		Tools:           make([]api.Tool, 0, 10),
		RequestContents: make(map[uuid.UUID]RequestContext, 10),
	}
	bus.Subscribe(events.UserInputEvent, service)
	bus.Subscribe(events.UpdateEnvironmentEvent, service)
	bus.Subscribe(events.UpdateToolListEvent, service)
	bus.Subscribe(events.StreamCancelEvent, service)
	bus.Subscribe(events.ToolResultEvent, service)
	return service
}

func (instance *OllamaService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.UserInputEvent:
		instance.UpdateUserInput(event.Data.(types.RequestData).Message)
		instance.UpdateEnvironmentToolList()
		instance.CallApi(event.Data.(types.RequestData).RequestUUID)
	case events.UpdateEnvironmentEvent:
		instance.Environment = utils.EnvironmentUpdateDataToString(event.Data.(types.EnvironmentUpdateData))
	case events.UpdateToolListEvent:
		instance.UpdateToolList(event.Data.(types.ToolListUpdateData).List)
	case events.StreamCancelEvent:
		instance.CancelStream(event.Data.(types.StreamCancelData).RequestUUID)
	case events.ToolResultEvent:
		instance.ProcessToolResult(event.Data.(types.ToolResultData))
	}
}

func (instance *OllamaService) ProcessToolResult(data types.ToolResultData) {
	instance.RequestMutex.Lock()
	defer instance.RequestMutex.Unlock()

	if _, exists := instance.RequestContents[data.RequestUUID].ToolCalls[data.ToolCall]; exists {
		msg := api.Message{
			Role:    "tool",
			Content: data.ToolResult,
		}
		instance.Messages = append(instance.Messages, msg)
		delete(instance.RequestContents[data.RequestUUID].ToolCalls, data.ToolCall)
		if len(instance.RequestContents[data.RequestUUID].ToolCalls) == 0 {
			delete(instance.RequestContents, data.RequestUUID)
			instance.CallApi(data.RequestUUID)
		}
	}
}

func (instance *OllamaService) EnvironmentMessage() *api.Message {
	return &api.Message{
		Role:    "system",
		Content: EnvironmentInfo + instance.Environment,
	}
}

func (instance *OllamaService) UpdateToolList(data []*mcp.Tool) {
	// 기존 도구 목록 초기화
	instance.Tools = make([]api.Tool, 0, len(data))

	for _, tool := range data {
		if tool == nil {
			continue
		}
		instance.Tools = append(instance.Tools, utils.ConvertTool(tool))
	}
}

func (instance *OllamaService) UpdateUserInput(message string) {
	instance.Messages = append(instance.Messages, api.Message{
		Role:    "user",
		Content: message,
	})
}

func (instance *OllamaService) GetID() types.Source {
	return types.LLMService
}

func (instance *OllamaService) UpdateEnvironmentToolList() {
	instance.Bus.Publish(
		events.Event{
			Type: events.RequestEnvironmentEvent,
			Data: types.EnvironmentRequestData{
				CreateUUID: uuid.New(),
			},
			Timestamp: time.Now(),
			Source:    types.LLMService,
		},
	)
	instance.Bus.Publish(
		events.Event{
			Type: events.RequestToolListEvent,
			Data: types.RequestToolListData{
				CreateUUID: uuid.New(),
			},
			Timestamp: time.Now(),
			Source:    types.LLMService,
		},
	)
}

func (instance *OllamaService) CallApi(requestUUID uuid.UUID) {
	instance.Bus.Publish(
		events.Event{
			Type: events.StreamStartEvent,
			Data: types.StreamStartData{
				RequestUUID: requestUUID,
			},
			Timestamp: time.Now(),
			Source:    types.LLMService,
		},
	)
	ctx, cancel := context.WithCancel(instance.Ctx)

	instance.StreamMutex.Lock()
	if instance.ActiveStreams == nil {
		instance.ActiveStreams = make(map[uuid.UUID]context.CancelFunc)
	}
	instance.ActiveStreams[requestUUID] = cancel
	instance.StreamMutex.Unlock()

	request := api.ChatRequest{
		Model:    instance.Model,
		Messages: append(append(instance.SystemMessages, *instance.EnvironmentMessage()), instance.Messages...),
		Tools:    instance.Tools,
		Stream:   &[]bool{true}[0],
	}

	go func() {
		defer func() {
			instance.StreamMutex.Lock()
			delete(instance.ActiveStreams, requestUUID)
			instance.StreamMutex.Unlock()
		}()

		err := instance.Client.Chat(ctx, &request, func(response api.ChatResponse) error {
			return instance.Response(requestUUID, response)
		})

		if err != nil {
			instance.Bus.Publish(
				events.Event{
					Type: events.StreamErrorEvent,
					Data: types.StreamErrorData{
						RequestUUID: requestUUID,
						Error:       err,
					},
					Timestamp: time.Now(),
					Source:    types.LLMService,
				},
			)
		}
	}()
}

func (instance *OllamaService) Response(requestUUID uuid.UUID, response api.ChatResponse) error {

	if response.Message.Content != "" {
		PublishEvent(instance.Bus, events.StreamChunkEvent, types.StreamChunkData{
			RequestUUID: requestUUID,
			Content:     response.Message.Content,
			IsComplete:  response.Done}, types.LLMService)
		instance.Buffer += response.Message.Content
	}
	if response.Done {

		PublishEvent(instance.Bus, events.StreamCompleteEvent, types.StreamCompleteData{
			RequestUUID:  requestUUID,
			FinalMessage: response.Message,
			IsComplete:   !instance.ToolCall,
		}, types.LLMService)

		instance.Messages = append(instance.Messages, api.Message{
			Role:    "assistant",
			Content: instance.Buffer,
		})
		instance.Buffer = ""
		if instance.ToolCall {
			instance.RequestMutex.RLock()
			instance.ToolCall = len(instance.RequestContents[requestUUID].ToolCalls) != 0
			instance.RequestMutex.RUnlock()
		}
	}
	if len(response.Message.ToolCalls) > 0 {
		for _, call := range response.Message.ToolCalls {
			toolCall := uuid.New()
			PublishEvent(instance.Bus, events.ToolCallEvent, types.ToolCallData{
				RequestUUID: requestUUID,
				ToolCall:    toolCall,
				ToolName:    call.Function.Name,
				Parameters:  call.Function.Arguments,
			}, types.LLMService)
			instance.RequestMutex.Lock()
			if content, exist := instance.RequestContents[requestUUID]; exist {
				content.ToolCalls[toolCall] = call.Function.Name
			} else {
				instance.RequestContents[requestUUID] = RequestContext{
					RequestUUID: requestUUID,
					ToolCalls:   make(map[uuid.UUID]string),
				}
				instance.RequestContents[requestUUID].ToolCalls[toolCall] = call.Function.Name
			}
			instance.RequestMutex.Unlock()
			instance.ToolCall = true
		}
	}
	return nil
}

func (instance *OllamaService) CancelStream(requestUUID uuid.UUID) {
	instance.StreamMutex.RLock()
	cancel, exists := instance.ActiveStreams[requestUUID]
	instance.StreamMutex.RUnlock()
	if exists {
		cancel()
	}
	instance.RequestMutex.Lock()
	_, exists = instance.RequestContents[requestUUID]
	if exists {
		delete(instance.RequestContents, requestUUID)
	}
	instance.RequestMutex.Unlock()
	instance.ToolCall = false
}
