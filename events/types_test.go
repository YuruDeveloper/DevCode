package events

import (
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/types"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewTypedBus(t *testing.T) {
	bus := NewTypedBus[dto.UserRequestData]()

	assert.NotNil(t, bus)
	assert.NotNil(t, bus.handlers)
	assert.Equal(t, 0, len(bus.handlers))
}

func TestEvent(t *testing.T) {
	testData := dto.UserRequestData{
		Message: "test message",
	}
	timestamp := time.Now()
	source := constants.Model

	event := Event[dto.UserRequestData]{
		Data:      testData,
		TimeStamp: timestamp,
		Source:    source,
	}

	assert.Equal(t, testData, event.Data)
	assert.Equal(t, timestamp, event.TimeStamp)
	assert.Equal(t, source, event.Source)
}

func TestTypedBusHandlerManagement(t *testing.T) {
	bus := NewTypedBus[dto.UserRequestData]()

	assert.Equal(t, 0, len(bus.handlers))

	handler1 := func(event Event[dto.UserRequestData]) {}
	handler2 := func(event Event[dto.UserRequestData]) {}

	bus.handlers[constants.Model] = handler1
	assert.Equal(t, 1, len(bus.handlers))

	bus.handlers[constants.LLMModule] = handler2
	assert.Equal(t, 2, len(bus.handlers))

	delete(bus.handlers, constants.Model)
	assert.Equal(t, 1, len(bus.handlers))

	delete(bus.handlers, constants.LLMModule)
	assert.Equal(t, 0, len(bus.handlers))
}

func TestTypedBusConcurrentAccess(t *testing.T) {
	bus := NewTypedBus[dto.UserRequestData]()

	var wg sync.WaitGroup
	numGoroutines := 100

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			source := constants.Source(id%6 + 1)
			handler := func(event Event[dto.UserRequestData]) {}

			bus.handlerMutex.Lock()
			bus.handlers[source] = handler
			bus.handlerMutex.Unlock()

			bus.handlerMutex.RLock()
			_ = bus.handlers[source]
			bus.handlerMutex.RUnlock()

			bus.handlerMutex.Lock()
			delete(bus.handlers, source)
			bus.handlerMutex.Unlock()
		}(i)
	}

	wg.Wait()

	assert.Equal(t, 0, len(bus.handlers))
}

func TestTypedBusWithDifferentTypes(t *testing.T) {
	userBus := NewTypedBus[dto.UserRequestData]()
	toolBus := NewTypedBus[dto.ToolCallData]()
	streamBus := NewTypedBus[dto.StreamStartData]()

	assert.NotNil(t, userBus)
	assert.NotNil(t, toolBus)
	assert.NotNil(t, streamBus)

	userHandler := func(event Event[dto.UserRequestData]) {}
	toolHandler := func(event Event[dto.ToolCallData]) {}
	streamHandler := func(event Event[dto.StreamStartData]) {}

	userBus.handlers[constants.Model] = userHandler
	toolBus.handlers[constants.ToolModule] = toolHandler
	streamBus.handlers[constants.LLMModule] = streamHandler

	assert.Equal(t, 1, len(userBus.handlers))
	assert.Equal(t, 1, len(toolBus.handlers))
	assert.Equal(t, 1, len(streamBus.handlers))
}

func TestTypedBusReadWriteMutex(t *testing.T) {
	bus := NewTypedBus[dto.UserRequestData]()

	handler := func(event Event[dto.UserRequestData]) {}

	bus.handlerMutex.Lock()
	bus.handlers[constants.Model] = handler
	bus.handlerMutex.Unlock()

	bus.handlerMutex.RLock()
	retrievedHandler, exists := bus.handlers[constants.Model]
	bus.handlerMutex.RUnlock()

	assert.True(t, exists)
	assert.NotNil(t, retrievedHandler)

	bus.handlerMutex.RLock()
	_, notExists := bus.handlers[constants.LLMModule]
	bus.handlerMutex.RUnlock()

	assert.False(t, notExists)
}

func TestEventWithComplexData(t *testing.T) {
	complexData := dto.ToolCallData{
		RequestID:  types.NewRequestID(),
		ToolCallID: types.NewToolCallID(),
		ToolName:   "TestTool",
		Parameters: map[string]any{
			"param1": "value1",
			"param2": 42,
			"param3": []string{"item1", "item2"},
		},
	}

	event := Event[dto.ToolCallData]{
		Data:      complexData,
		TimeStamp: time.Now(),
		Source:    constants.ToolModule,
	}

	assert.Equal(t, complexData.RequestID, event.Data.RequestID)
	assert.Equal(t, complexData.ToolName, event.Data.ToolName)
	assert.Equal(t, complexData.Parameters, event.Data.Parameters)
	assert.Equal(t, constants.ToolModule, event.Source)
}

func TestTypedBusZeroValue(t *testing.T) {
	var bus TypedBus[dto.UserRequestData]

	assert.Nil(t, bus.handlers)

	bus.handlers = make(map[constants.Source]func(Event[dto.UserRequestData]))

	handler := func(event Event[dto.UserRequestData]) {}
	bus.handlers[constants.Model] = handler

	assert.Equal(t, 1, len(bus.handlers))
}
