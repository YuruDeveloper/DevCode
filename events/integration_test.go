package events

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/types"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestEventBusIntegration(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	var receivedEvents []Event[dto.UserRequestData]
	var mu sync.Mutex
	var wg sync.WaitGroup

	wg.Add(3)
	handler := func(event Event[dto.UserRequestData]) {
		mu.Lock()
		receivedEvents = append(receivedEvents, event)
		mu.Unlock()
		wg.Done()
	}

	Subscribe(bus, bus.UserInputEvent, constants.Model, handler)

	for i := 0; i < 3; i++ {
		event := Event[dto.UserRequestData]{
			Data: dto.UserRequestData{
				SessionID: types.NewSessionID(),
				RequestID: types.NewRequestID(),
				Message:   "Test message " + string(rune(i+'1')),
			},
			TimeStamp: time.Now(),
			Source:    constants.Model,
		}
		Publish(bus, bus.UserInputEvent, event)
	}

	wg.Wait()

	mu.Lock()
	assert.Equal(t, 3, len(receivedEvents))
	mu.Unlock()
}

func TestMultipleEventTypeIntegration(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	userEventReceived := false
	toolEventReceived := false
	streamEventReceived := false

	var wg sync.WaitGroup
	wg.Add(3)

	userHandler := func(event Event[dto.UserRequestData]) {
		userEventReceived = true
		wg.Done()
	}

	toolHandler := func(event Event[dto.ToolCallData]) {
		toolEventReceived = true
		wg.Done()
	}

	streamHandler := func(event Event[dto.StreamStartData]) {
		streamEventReceived = true
		wg.Done()
	}

	Subscribe(bus, bus.UserInputEvent, constants.Model, userHandler)
	Subscribe(bus, bus.ToolCallEvent, constants.ToolModule, toolHandler)
	Subscribe(bus, bus.StreamStartEvent, constants.LLMModule, streamHandler)

	userEvent := Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			SessionID: types.NewSessionID(),
			RequestID: types.NewRequestID(),
			Message:   "User input",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	toolEvent := Event[dto.ToolCallData]{
		Data: dto.ToolCallData{
			RequestID:  types.NewRequestID(),
			ToolCallID: types.NewToolCallID(),
			ToolName:   "TestTool",
			Parameters: map[string]any{"test": "value"},
		},
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}

	streamEvent := Event[dto.StreamStartData]{
		Data: dto.StreamStartData{
			RequestID: types.NewRequestID(),
		},
		TimeStamp: time.Now(),
		Source:    constants.LLMModule,
	}

	Publish(bus, bus.UserInputEvent, userEvent)
	Publish(bus, bus.ToolCallEvent, toolEvent)
	Publish(bus, bus.StreamStartEvent, streamEvent)

	wg.Wait()

	assert.True(t, userEventReceived)
	assert.True(t, toolEventReceived)
	assert.True(t, streamEventReceived)
}

func TestEventOrderingAndTiming(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	var eventTimes []time.Time
	var mu sync.Mutex
	var wg sync.WaitGroup

	numEvents := 5
	wg.Add(numEvents)

	handler := func(event Event[dto.UserRequestData]) {
		mu.Lock()
		eventTimes = append(eventTimes, event.TimeStamp)
		mu.Unlock()
		wg.Done()
	}

	Subscribe(bus, bus.UserInputEvent, constants.Model, handler)

	baseTime := time.Now()
	for i := 0; i < numEvents; i++ {
		event := Event[dto.UserRequestData]{
			Data: dto.UserRequestData{
				SessionID: types.NewSessionID(),
				RequestID: types.NewRequestID(),
				Message:   "Timed message",
			},
			TimeStamp: baseTime.Add(time.Duration(i) * time.Millisecond),
			Source:    constants.Model,
		}
		Publish(bus, bus.UserInputEvent, event)
	}

	wg.Wait()

	mu.Lock()
	assert.Equal(t, numEvents, len(eventTimes))

	for _, eventTime := range eventTimes {
		assert.True(t, eventTime.After(baseTime) || eventTime.Equal(baseTime))
	}
	mu.Unlock()
}

func TestEventBusRobustnessUnderLoad(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 1000,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	var eventCount int32
	var mu sync.Mutex
	var wg sync.WaitGroup

	numEvents := 100
	numHandlers := 5

	for h := 0; h < numHandlers; h++ {
		source := constants.Source(h%6 + 1)
		handler := func(event Event[dto.UserRequestData]) {
			mu.Lock()
			eventCount++
			mu.Unlock()
			wg.Done()
		}
		Subscribe(bus, bus.UserInputEvent, source, handler)
	}

	wg.Add(numEvents * numHandlers)

	for i := 0; i < numEvents; i++ {
		event := Event[dto.UserRequestData]{
			Data: dto.UserRequestData{
				SessionID: types.NewSessionID(),
				RequestID: types.NewRequestID(),
				Message:   "Load test message",
			},
			TimeStamp: time.Now(),
			Source:    constants.Model,
		}
		Publish(bus, bus.UserInputEvent, event)
	}

	wg.Wait()

	mu.Lock()
	assert.Equal(t, int32(numEvents*numHandlers), eventCount)
	mu.Unlock()
}

func TestEventBusUnsubscribeIntegration(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	handler1Called := false
	handler2Called := false

	var wg sync.WaitGroup
	wg.Add(1)

	handler1 := func(event Event[dto.UserRequestData]) {
		handler1Called = true
	}

	handler2 := func(event Event[dto.UserRequestData]) {
		handler2Called = true
		wg.Done()
	}

	Subscribe(bus, bus.UserInputEvent, constants.Model, handler1)
	Subscribe(bus, bus.UserInputEvent, constants.LLMModule, handler2)

	UnSubscribe(bus, bus.UserInputEvent, constants.Model)

	event := Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			SessionID: types.NewSessionID(),
			RequestID: types.NewRequestID(),
			Message:   "Test message",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	Publish(bus, bus.UserInputEvent, event)

	wg.Wait()

	assert.False(t, handler1Called)
	assert.True(t, handler2Called)
}
