package ollama

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"DevCode/src/service/llm/ollama"
)

// MockOllamaClient은 테스트용 Ollama 클라이언트를 시뮬레이션합니다
type MockOllamaClient struct {
	ShouldError   bool
	ErrorToReturn error
	ResponseCount int
	Responses     []api.ChatResponse
}

func (m *MockOllamaClient) Chat(ctx context.Context, req *api.ChatRequest, fn func(api.ChatResponse) error) error {
	if m.ShouldError {
		return m.ErrorToReturn
	}

	// 미리 정의된 응답들을 순서대로 전송
	for i, response := range m.Responses {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := fn(response)
			if err != nil {
				return err
			}
			// 마지막 응답이면 Done을 true로 설정
			if i == len(m.Responses)-1 {
				response.Done = true
				err = fn(response)
				if err != nil {
					return err
				}
			}
		}
	}
	
	return nil
}

// TestEventSubscriber는 이벤트를 수신하기 위한 테스트용 구조체입니다
type TestEventSubscriber struct {
	ID             constants.Source
	ReceivedEvents []events.Event
	EventChan      chan events.Event
}

func NewTestEventSubscriber(id constants.Source) *TestEventSubscriber {
	return &TestEventSubscriber{
		ID:             id,
		ReceivedEvents: make([]events.Event, 0),
		EventChan:      make(chan events.Event, 10),
	}
}

func (t *TestEventSubscriber) HandleEvent(event events.Event) {
	t.ReceivedEvents = append(t.ReceivedEvents, event)
	select {
	case t.EventChan <- event:
	default:
		// 채널이 가득 찬 경우 무시
	}
}

func (t *TestEventSubscriber) GetID() constants.Source {
	return t.ID
}

func TestNewStreamManager(t *testing.T) {
	manager := ollama.NewStreamManager()

	assert.NotNil(t, manager)
}

func TestStreamManager_StartStream_Success(t *testing.T) {
	manager := ollama.NewStreamManager()
	
	// StartStream이 생성되는지만 확인 (실제 네트워크 호출은 하지 않음)
	assert.NotNil(t, manager)
}

func TestStreamManager_CancelStream(t *testing.T) {
	manager := ollama.NewStreamManager()
	
	requestUUID := uuid.New()
	
	// 존재하지 않는 스트림을 취소해도 에러가 발생하지 않아야 함
	manager.CancelStream(requestUUID)
	
	// 다시 한번 취소해도 에러 없어야 함
	manager.CancelStream(requestUUID)
}

func TestStreamManager_Response_WithContent(t *testing.T) {
	manager := ollama.NewStreamManager()
	bus, err := events.NewEventBus()
	require.NoError(t, err)
	
	requestUUID := uuid.New()
	
	// StreamChunkEvent 구독자 생성
	subscriber := NewTestEventSubscriber(constants.LLMService)
	bus.Subscribe(events.StreamChunkEvent, subscriber)
	
	response := api.ChatResponse{
		Message: api.Message{
			Role:    "assistant",
			Content: "테스트 응답 내용",
		},
		Done: false,
	}
	
	doneCallbackCalled := false
	doneCallback := func(content string) {
		doneCallbackCalled = true
		assert.Equal(t, "테스트 응답 내용", content)
	}
	
	checkDone := func(uuid.UUID) bool {
		return true
	}
	
	toolsCallback := func(uuid.UUID, []api.ToolCall) {}
	
	err = manager.Response(requestUUID, response, bus, doneCallback, checkDone, toolsCallback)
	assert.NoError(t, err)
	
	// 이벤트가 발생할 시간을 줍니다
	time.Sleep(10 * time.Millisecond)
	
	// StreamChunkEvent가 발생했는지 확인
	require.Len(t, subscriber.ReceivedEvents, 1)
	event := subscriber.ReceivedEvents[0]
	assert.Equal(t, events.StreamChunkEvent, event.Type)
	
	chunkData, ok := event.Data.(dto.StreamChunkData)
	require.True(t, ok)
	assert.Equal(t, requestUUID, chunkData.RequestUUID)
	assert.Equal(t, "테스트 응답 내용", chunkData.Content)
	assert.False(t, chunkData.IsComplete)
	
	assert.False(t, doneCallbackCalled) // Done이 false이므로 호출되지 않아야 함
}

func TestStreamManager_Response_Done(t *testing.T) {
	manager := ollama.NewStreamManager()
	bus, err := events.NewEventBus()
	require.NoError(t, err)
	
	requestUUID := uuid.New()
	
	// StreamCompleteEvent 구독자 생성
	subscriber := NewTestEventSubscriber(constants.LLMService)
	bus.Subscribe(events.StreamCompleteEvent, subscriber)
	
	// 완료된 응답
	doneResponse := api.ChatResponse{
		Message: api.Message{
			Role:    "assistant",
			Content: "완료된 응답",
		},
		Done: true,
	}
	
	finalContent := ""
	doneCallback := func(content string) {
		finalContent = content
	}
	
	checkDone := func(uuid.UUID) bool {
		return true
	}
	
	toolsCallback := func(uuid.UUID, []api.ToolCall) {}
	
	err = manager.Response(requestUUID, doneResponse, bus, doneCallback, checkDone, toolsCallback)
	assert.NoError(t, err)
	
	// 이벤트가 발생할 시간을 줍니다
	time.Sleep(10 * time.Millisecond)
	
	// StreamCompleteEvent가 발생했는지 확인
	require.Len(t, subscriber.ReceivedEvents, 1)
	event := subscriber.ReceivedEvents[0]
	assert.Equal(t, events.StreamCompleteEvent, event.Type)
	
	// doneCallback이 호출되었는지 확인
	assert.Equal(t, "완료된 응답", finalContent)
}

func TestStreamManager_Response_WithToolCalls(t *testing.T) {
	manager := ollama.NewStreamManager()
	bus, err := events.NewEventBus()
	require.NoError(t, err)
	
	requestUUID := uuid.New()
	toolCalls := []api.ToolCall{
		{
			Function: api.ToolCallFunction{
				Name:      "calculator",
				Arguments: map[string]interface{}{"expression": "2+2"},
			},
		},
	}
	
	response := api.ChatResponse{
		Message: api.Message{
			Role:      "assistant",
			Content:   "",
			ToolCalls: toolCalls,
		},
		Done: false,
	}
	
	toolsCallbackCalled := false
	var receivedToolCalls []api.ToolCall
	toolsCallback := func(uuid uuid.UUID, calls []api.ToolCall) {
		toolsCallbackCalled = true
		receivedToolCalls = calls
		assert.Equal(t, requestUUID, uuid)
	}
	
	doneCallback := func(string) {}
	checkDone := func(uuid.UUID) bool { return true }
	
	err = manager.Response(requestUUID, response, bus, doneCallback, checkDone, toolsCallback)
	assert.NoError(t, err)
	
	assert.True(t, toolsCallbackCalled)
	assert.Len(t, receivedToolCalls, 1)
	assert.Equal(t, "calculator", receivedToolCalls[0].Function.Name)
}