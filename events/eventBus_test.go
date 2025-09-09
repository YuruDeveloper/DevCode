package events

import (
	"DevCode/config"
	"DevCode/constants"
	"DevCode/dto"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestNewEventBus(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)

	require.NoError(t, err)
	assert.NotNil(t, bus)
	assert.NotNil(t, bus.UserInputEvent)
	assert.NotNil(t, bus.UserDecisionEvent)
	assert.NotNil(t, bus.ToolCallEvent)
	assert.NotNil(t, bus.StreamStartEvent)
	assert.NotNil(t, bus.RagnarokEvent)
	assert.Equal(t, logger, bus.logger)
	assert.NotNil(t, bus.pool)

	bus.Close()
}

func TestNewEventBusWithInvalidPoolSize(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: -1,
	}

	bus, err := NewEventBus(config, logger)

	assert.Error(t, err)
	assert.Nil(t, bus)
}

func TestEventBusSubscribe(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	var receivedEvent Event[dto.UserRequestData]
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(event Event[dto.UserRequestData]) {
		receivedEvent = event
		wg.Done()
	}

	Subscribe(bus, bus.UserInputEvent, constants.Model, handler)

	testEvent := Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			Message: "test message",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	Publish(bus, bus.UserInputEvent, testEvent)

	wg.Wait()

	assert.Equal(t, testEvent.Data.Message, receivedEvent.Data.Message)
	assert.Equal(t, testEvent.Source, receivedEvent.Source)
}

func TestEventBusUnsubscribe(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	handlerCalled := false
	handler := func(event Event[dto.UserRequestData]) {
		handlerCalled = true
	}

	Subscribe(bus, bus.UserInputEvent, constants.Model, handler)
	UnSubscribe(bus, bus.UserInputEvent, constants.Model)

	testEvent := Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			Message: "test message",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	Publish(bus, bus.UserInputEvent, testEvent)

	time.Sleep(100 * time.Millisecond)

	assert.False(t, handlerCalled)
}

func TestEventBusMultipleHandlers(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	handler1Called := false
	handler2Called := false

	handler1 := func(event Event[dto.UserRequestData]) {
		handler1Called = true
		wg.Done()
	}

	handler2 := func(event Event[dto.UserRequestData]) {
		handler2Called = true
		wg.Done()
	}

	Subscribe(bus, bus.UserInputEvent, constants.Model, handler1)
	Subscribe(bus, bus.UserInputEvent, constants.LLMModule, handler2)

	testEvent := Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			Message: "test message",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	Publish(bus, bus.UserInputEvent, testEvent)

	wg.Wait()

	assert.True(t, handler1Called)
	assert.True(t, handler2Called)
}

func TestEventBusRagnarok(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	var receivedEvent Event[dto.RagnarokData]
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(event Event[dto.RagnarokData]) {
		receivedEvent = event
		wg.Done()
	}

	Subscribe(bus, bus.RagnarokEvent, constants.Model, handler)

	bus.Ragnarok()

	wg.Wait()

	assert.NotZero(t, receivedEvent.TimeStamp)
}

func TestEventBusPublishWithPanic(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)
	defer bus.Close()

	var ragnarokCalled bool
	var wg sync.WaitGroup
	wg.Add(1)

	ragnarokHandler := func(event Event[dto.RagnarokData]) {
		ragnarokCalled = true
		wg.Done()
	}

	panicHandler := func(event Event[dto.UserRequestData]) {
		panic("test panic")
	}

	Subscribe(bus, bus.RagnarokEvent, constants.Model, ragnarokHandler)
	Subscribe(bus, bus.UserInputEvent, constants.Model, panicHandler)

	testEvent := Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			Message: "test message",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	Publish(bus, bus.UserInputEvent, testEvent)

	wg.Wait()

	assert.True(t, ragnarokCalled)
}

func TestEventBusClose(t *testing.T) {
	logger := zap.NewNop()
	config := config.EventBusConfig{
		PoolSize: 100,
	}

	bus, err := NewEventBus(config, logger)
	require.NoError(t, err)

	assert.NotNil(t, bus.pool)

	bus.Close()

	assert.True(t, bus.pool.IsClosed())
}
