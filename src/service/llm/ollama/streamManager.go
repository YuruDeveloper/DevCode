package ollama

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service"
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
)

const (
	DefaultActiveStreamSize = 5
)

func NewStreamManager() *StreamManager {
	return &StreamManager{
		ctxs:          make(map[uuid.UUID]context.Context, DefaultActiveStreamSize),
		activeStreams: make(map[uuid.UUID]context.CancelFunc, DefaultActiveStreamSize),
		buffer:        "",
	}
}

type StreamManager struct {
	ctxs          map[uuid.UUID]context.Context
	activeStreams map[uuid.UUID]context.CancelFunc
	streamMutex   sync.RWMutex
	buffer        string
}

func (instance *StreamManager) StartStream(ollama *api.Client, bus events.Bus, requestUUID uuid.UUID, model string, tools []api.Tool, message []api.Message, CallBack func(requestUUID uuid.UUID, response api.ChatResponse) error) {
	instance.streamMutex.Lock()
	if instance.ctxs == nil {
		instance.ctxs = make(map[uuid.UUID]context.Context, DefaultActiveStreamSize)
	}
	instance.ctxs[requestUUID] = context.Background()
	ctx, cancel := context.WithCancel(instance.ctxs[requestUUID])
	if instance.activeStreams == nil {
		instance.activeStreams = make(map[uuid.UUID]context.CancelFunc, DefaultActiveStreamSize)
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
			bus.Publish(
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

func (instance *StreamManager) Response(requestUUID uuid.UUID, response api.ChatResponse, bus events.Bus, doneCallBack func(string), CheckDone func(uuid.UUID) bool, toolsCallBack func(uuid.UUID, []api.ToolCall)) error {
	if response.Message.Content != "" {
		service.PublishEvent(bus, events.StreamChunkEvent, dto.StreamChunkData{
			RequestUUID: requestUUID,
			Content:     response.Message.Content,
			IsComplete:  response.Done,
		}, constants.LLMService)
		instance.buffer += response.Message.Content
	}
	if response.Done {
		service.PublishEvent(bus, events.StreamCompleteEvent, dto.StreamCompleteData{
			RequestUUID:  requestUUID,
			FinalMessage: response.Message.Content,
			IsComplete:   !CheckDone(requestUUID),
		}, constants.LLMService)
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
