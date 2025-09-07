package ollama

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/types"
	"context"
	"sync"
	"time"
	"github.com/ollama/ollama/api"
)

func NewStreamManager(config config.OllamaServiceConfig) *StreamManager {
	return &StreamManager{
		ctxs:          make(map[types.RequestID]context.Context, config.DefaultActiveStreamSize),
		activeStreams: make(map[types.RequestID]context.CancelFunc, config.DefaultActiveStreamSize),
		buffer:        "",
	}
}

type StreamManager struct {
	ctxs          map[types.RequestID]context.Context
	activeStreams map[types.RequestID]context.CancelFunc
	streamMutex   sync.RWMutex
	buffer        string
	config        config.OllamaServiceConfig
}

func (instance *StreamManager) StartStream(ollama *api.Client, bus *events.EventBus, requestID types.RequestID, model string, tools []api.Tool, message []api.Message, CallBack func(requestID types.RequestID, response api.ChatResponse) error) {
	instance.streamMutex.Lock()
	instance.ctxs[requestID] = context.Background()
	ctx, cancel := context.WithCancel(instance.ctxs[requestID])
	instance.activeStreams[requestID] = cancel
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

			delete(instance.activeStreams, requestID)
			delete(instance.ctxs, requestID)
		}()

		err := ollama.Chat(ctx, &request, func(cr api.ChatResponse) error {
			return CallBack(requestID, cr)
		})

		if err != nil {
			bus.StreamErrorEvent.Publish(events.Event[dto.StreamErrorData]{
				Data: dto.StreamErrorData{
					RequestID: requestID,
					Error:       err,
				},
				TimeStamp: time.Now(),
				Source:    constants.LLMService,
			})
		}
	}()
}

func (instance *StreamManager) Response(requestID types.RequestID, response api.ChatResponse, bus *events.EventBus, doneCallBack func(string), CheckDone func(types.RequestID) bool, toolsCallBack func(types.RequestID, []api.ToolCall)) error {
	if response.Message.Content != "" {
		bus.StreamChunkEvent.Publish(events.Event[dto.StreamChunkData]{
			Data: dto.StreamChunkData{
				RequestID: requestID,
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
				RequestID:  requestID,
				FinalMessage: response.Message.Content,
				IsComplete:   !CheckDone(requestID),
			},
			TimeStamp: time.Now(),
			Source:    constants.LLMService,
		})
		doneCallBack(instance.buffer)
		instance.buffer = ""
	}
	if len(response.Message.ToolCalls) > 0 {
		toolsCallBack(requestID, response.Message.ToolCalls)
	}
	return nil
}

func (instance *StreamManager) CancelStream(requestID types.RequestID) {
	instance.streamMutex.Lock()
	defer instance.streamMutex.Unlock()
	cancel, exists := instance.activeStreams[requestID]
	if exists {
		cancel()
	}
	delete(instance.activeStreams,requestID)
	delete(instance.ctxs, requestID)
}
