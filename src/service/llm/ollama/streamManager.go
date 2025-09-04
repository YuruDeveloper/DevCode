package ollama

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

func NewStreamManager(config config.OllamaServiceConfig) *StreamManager {
	return &StreamManager{
		ctxs:          make(map[uuid.UUID]context.Context, config.DefaultActiveStreamSize),
		activeStreams: make(map[uuid.UUID]context.CancelFunc, config.DefaultActiveStreamSize),
		buffer:        "",
	}
}

type StreamManager struct {
	ctxs          map[uuid.UUID]context.Context
	activeStreams map[uuid.UUID]context.CancelFunc
	streamMutex   sync.RWMutex
	buffer        string
	config        config.OllamaServiceConfig
}

func (instance *StreamManager) StartStream(ollama *api.Client, bus *events.EventBus, requestUUID uuid.UUID, model string, tools []api.Tool, message []api.Message, CallBack func(requestUUID uuid.UUID, response api.ChatResponse) error) {
	instance.streamMutex.Lock()
	if instance.ctxs == nil {
		instance.ctxs = make(map[uuid.UUID]context.Context, instance.config.DefaultActiveStreamSize)
	}
	instance.ctxs[requestUUID] = context.Background()
	ctx, cancel := context.WithCancel(instance.ctxs[requestUUID])
	if instance.activeStreams == nil {
		instance.activeStreams = make(map[uuid.UUID]context.CancelFunc, instance.config.DefaultActiveStreamSize)
	}
	instance.activeStreams[requestUUID] = cancel
	instance.streamMutex.Unlock()

	request := api.ChatRequest{
		Model:    model,
		Messages: message,
		Tools:    tools,
		Stream:   &[]bool{true}[0],
	}
	go func() {
		defer func() {
			instance.streamMutex.Lock()
			defer instance.streamMutex.Unlock()

			delete(instance.activeStreams, requestUUID)
			delete(instance.ctxs, requestUUID)
		}()

		err := ollama.Chat(ctx, &request, func(cr api.ChatResponse) error {
			return CallBack(requestUUID, cr)
		})

		if err != nil {
			bus.StreamErrorEvent.Publish(events.Event[dto.StreamErrorData]{
				Data: dto.StreamErrorData{
					RequestUUID: requestUUID,
					Error:       err,
				},
				TimeStamp: time.Now(),
				Source:    constants.LLMService,
			})
		}
	}()
}

func (instance *StreamManager) Response(requestUUID uuid.UUID, response api.ChatResponse, bus *events.EventBus, doneCallBack func(string), CheckDone func(uuid.UUID) bool, toolsCallBack func(uuid.UUID, []api.ToolCall)) error {
	if response.Message.Content != "" {
		bus.StreamChunkEvent.Publish(events.Event[dto.StreamChunkData]{
			Data: dto.StreamChunkData{
				RequestUUID: requestUUID,
				Content:     response.Message.Content,
				IsComplete:  response.Done,
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMService,
		})
		instance.buffer += response.Message.Content
	}
	if response.Done {
		bus.StreamCompleteEvent.Publish(events.Event[dto.StreamCompleteData]{
			Data: dto.StreamCompleteData{
				RequestUUID:  requestUUID,
				FinalMessage: response.Message.Content,
				IsComplete:   !CheckDone(requestUUID),
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMService,
		})
		doneCallBack(instance.buffer)
		instance.buffer = ""
	}
	if len(response.Message.ToolCalls) > 0 {
		toolsCallBack(requestUUID, response.Message.ToolCalls)
	}
	return nil
}

func (instance *StreamManager) CancelStream(requestUUID uuid.UUID) {
	instance.streamMutex.Lock()
	defer instance.streamMutex.Unlock()
	if instance.activeStreams == nil {
		return
	}
	cancel, exists := instance.activeStreams[requestUUID]
	if exists {
		cancel()
	}
	delete(instance.activeStreams, requestUUID)
	delete(instance.ctxs, requestUUID)
}
