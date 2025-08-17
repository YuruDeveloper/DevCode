package service

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"UniCode/src/utils"
	"context"
	"net/http"
	"net/url"
	"os"
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
}

func NewOllamaService(bus *events.EventBus) *OllamaService {
	ollamaUrl := viper.GetString("ollama.url")
	url := url.URL{Host: ollamaUrl}
	ollama := *api.NewClient(&url, http.DefaultClient)

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
	return service
}

func (instance *OllamaService) HandleEvent(event events.Event) {
	switch event.Type {
		case events.UserInputEvent:
			instance.UpdateUserInput(event.Data.(types.RequestData).Message)
			instance.UpdateEnviromentToolList()
			instance.CallApi()
		case events.UpdateEnvionmentEvent:
			instance.Environment = utils.EnviromentUpdateDataToString(event.Data.(types.EnviromentUpdateData))
		case events.UpdateToolListEvent:
			instance.UpdateToolList(event.Data.(types.ToolListUpdate).List)
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

func (instance *OllamaService) CallApi() {
	request := api.ChatRequest {
		Model: instance.Model,
		Messages: append(append(instance.SystemMessages,*instance.EnviromentMessage()),instance.Messages...),
		Tools: instance.Tools,
		Stream: &[]bool{false}[0],
	}
	instance.Client.Chat(instance.Ctx,&request,instance.ResponseApi)
}

func (instance *OllamaService)ResponseApi(response api.ChatResponse) error {
  	instance.Bus.Publish(
		events.Event{
			Type: events.LLMResponseEvent,
			Data: types.ResponseData {
				Message: response.Message,
			},
			Timestamp: time.Now(),
			Source: types.LLMService,
		},
	)
	return  nil
}

