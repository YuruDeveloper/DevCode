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
	EnviromentInfo = "  Here is useful information about the environment you are running in:\n"
)

type OllamaService struct {
	Client       *api.Client
	Model        string
	Ctx          context.Context
	SystemPrompt string
	Bus *events.EventBus
	SystemMessages []api.Message
	Messages    []api.Message
	Tools []api.Tool
	Environment string
	ActiveStreams map[uuid.UUID]context.CancelFunc
	StreamMutex sync.RWMutex
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

	systemPrompt, err := os.ReadFile(ollamaPath)
	if err != nil {
		panic(err)
	}
	service := &OllamaService{
		Client:       &ollama,
		Model:        ollamaModel,
		Ctx:          ctx,
		Bus: bus,
		SystemPrompt: string(systemPrompt),
		SystemMessages: make([]api.Message, 10),
		Messages: make([]api.Message, 100),
	}
	bus.Subscribe(events.UserInputEvent,service)
	bus.Subscribe(events.UpdateEnvionmentEvent,service)
	bus.Subscribe(events.UpdateToolListEvent,service)
	bus.Subscribe(events.StreamCancelEvent,service)
	return service
}

func (instance *OllamaService) HandleEvent(event events.Event) {
	switch event.Type {
		case events.UserInputEvent:
			instance.UpdateUserInput(event.Data.(types.RequestData).Message)
			instance.UpdateEnviromentToolList()
			instance.CallApi(event.Data.(types.RequestData).RequestUUID)
		case events.UpdateEnvionmentEvent:
			instance.Environment = utils.EnviromentUpdateDataToString(event.Data.(types.EnviromentUpdateData))
		case events.UpdateToolListEvent:
			instance.UpdateToolList(event.Data.(types.ToolListUpdateData).List)
		case events.StreamCancelEvent:
			instance.CancelStream(event.Data.(types.StreamCancelData).RequestUUID)
	}
}

func (instance *OllamaService) EnviromentMessage() *api.Message {
	return &api.Message{
		Role: "system",
		Content: EnviromentInfo + instance.Environment,
	}
}

func (instance *OllamaService) UpdateToolList(data []*mcp.Tool) {
	for _ , tool := range data {
		instance.Tools = append(instance.Tools,utils.ConvertTool(tool))
	}
}

func (instance *OllamaService) UpdateUserInput(message string) {
	instance.Messages = append(instance.Messages, api.Message{
		Role: "user",
		Content: message,
	})
}

func (instance *OllamaService) GetID() types.Source {
	return types.LLMService
}

func (instance *OllamaService) UpdateEnviromentToolList() {
	instance.Bus.Publish(
		events.Event{
			Type: events.RequestEnvionmentvent,
			Data: types.EnviromentRequestData {
				CreateUUID: uuid.New(),
			},
			Timestamp: time.Now(),
			Source: types.LLMService,
		},
	)
	instance.Bus.Publish(
		events.Event{
			Type: events.RequestToolListEvent,
			Data: types.RequestToolListData {
				CreateUUID: uuid.New(),
			},
			Timestamp: time.Now(),
			Source: types.LLMService,
		},
	)
}

func (instance *OllamaService) CallApi(requestUUID uuid.UUID) {
	instance.Bus.Publish(
		events.Event{
			Type: events.StreamStartEvent,
			Data: types.StreamStartData {
				RequestUUID: requestUUID,
			},
			Timestamp: time.Now(),
			Source: types.LLMService,
		},
	)
	ctx , cancel := context.WithCancel(instance.Ctx)

	instance.StreamMutex.Lock()
	if instance.ActiveStreams == nil {
		instance.ActiveStreams = make(map[uuid.UUID]context.CancelFunc)
	}
	instance.ActiveStreams[requestUUID] = cancel
	instance.StreamMutex.Unlock()

	request := api.ChatRequest {
		Model: instance.Model,
		Messages: append(append(instance.SystemMessages,*instance.EnviromentMessage()),instance.Messages...),
		Tools: instance.Tools,
		Stream: &[]bool{true}[0],
	}

	go func()  {
		defer func()  {
			instance.StreamMutex.Lock()
			delete(instance.ActiveStreams,requestUUID)
			instance.StreamMutex.Unlock()
		}()

		err := instance.Client.Chat(ctx,&request,func(response api.ChatResponse) error {
			return instance.Response(requestUUID,response)
		})

		if err != nil {
			instance.Bus.Publish(
				events.Event{
					Type: events.StreamErrorEvent,
					Data: types.SteramErrorData {
						RequestUUID: requestUUID,
						Error: err,
					},
					Timestamp: time.Now(),
					Source: types.LLMService,
				},
			)
		}
	}()
}

func (instance *OllamaService) Response(requestUUID uuid.UUID,response api.ChatResponse) error{
	
	if response.Message.Content != "" {
		instance.Bus.Publish(
			events.Event{
				Type: events.StreramChunkEvnet,
				Data: types.StreamChunkData {
				RequestUUID: requestUUID,
				Chunk: types.StreamChunk{
					Content: response.Message.Content,
					IsComplete: response.Done,
				},
			},
			Timestamp: time.Now(),
			Source: types.LLMService,
		},
	)
	}
	if response.Done {
		instance.Bus.Publish(
			events.Event{
				Type: events.StreamCompleteEvent,
				Data: types.StreamCompleteData {
					RequestUUID: requestUUID,
					FinalMessage: response.Message,
				},
				Timestamp: time.Now(),
				Source: types.LLMService,
			},
		)
		if len(response.Message.ToolCalls) > 0 {
			for _ , call := range response.Message.ToolCalls {
				instance.Bus.Publish(
					events.Event{
						Type: events.ToolCallEvent,
						Data: types.ToolCallData {
							RequestUUID: uuid.New(),
							ToolName: call.Function.Name,
							Paramters: call.Function.Arguments,
						},
						Timestamp: time.Now(),
						Source: types.LLMService,
					},
				)
			}
		}
	}
	return nil
}


func (instance *OllamaService) CancelStream(requestUUID uuid.UUID) {
	instance.StreamMutex.RLock()
	cancel , exists := instance.ActiveStreams[requestUUID]
	instance.StreamMutex.RUnlock()
	if exists {
		cancel()
	}
}

