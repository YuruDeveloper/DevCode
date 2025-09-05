package events_test

import (
	"DevCode/src/config"
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockUserInputHandler struct {
	ReceivedEvents []events.Event[dto.UserRequestData]
	Mutex          sync.Mutex
}

func NewMockUserInputHandler() *MockUserInputHandler {
	return &MockUserInputHandler{
		ReceivedEvents: make([]events.Event[dto.UserRequestData], 0),
	}
}

func (m *MockUserInputHandler) HandleEvent(event events.Event[dto.UserRequestData]) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	m.ReceivedEvents = append(m.ReceivedEvents, event)
}

func (m *MockUserInputHandler) GetReceivedEvents() []events.Event[dto.UserRequestData] {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	return append([]events.Event[dto.UserRequestData]{}, m.ReceivedEvents...)
}

func TestNewEventBus(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	eventBus, err := events.NewEventBus(eventBusConfig)

	require.NoError(t, err)
	assert.NotNil(t, eventBus)
	eventBus.Close()
}

func TestEventBus_UserInputEvent_Subscribe_And_Publish(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	eventBus, err := events.NewEventBus(eventBusConfig)
	require.NoError(t, err)
	defer eventBus.Close()

	handler := NewMockUserInputHandler()
	eventBus.UserInputEvent.Subscribe(constants.Model, handler.HandleEvent)

	testUUID := uuid.New()
	sessionUUID := uuid.New()
	testEvent := events.Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			SessionUUID: sessionUUID,
			RequestUUID: testUUID,
			Message:     "test message",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	eventBus.UserInputEvent.Publish(testEvent)
	time.Sleep(50 * time.Millisecond)

	receivedEvents := handler.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	assert.Equal(t, "test message", receivedEvents[0].Data.Message)
	assert.Equal(t, testUUID, receivedEvents[0].Data.RequestUUID)
	assert.Equal(t, sessionUUID, receivedEvents[0].Data.SessionUUID)
}

func TestEventBus_UserInputEvent_UnSubscribe(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	eventBus, err := events.NewEventBus(eventBusConfig)
	require.NoError(t, err)
	defer eventBus.Close()

	handler := NewMockUserInputHandler()
	eventBus.UserInputEvent.Subscribe(constants.Model, handler.HandleEvent)

	testUUID := uuid.New()
	sessionUUID := uuid.New()
	testEvent := events.Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			SessionUUID: sessionUUID,
			RequestUUID: testUUID,
			Message:     "test message",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	eventBus.UserInputEvent.Publish(testEvent)
	time.Sleep(50 * time.Millisecond)
	require.Len(t, handler.GetReceivedEvents(), 1)

	eventBus.UserInputEvent.UnSubscribe(constants.Model)
	eventBus.UserInputEvent.Publish(testEvent)
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, handler.GetReceivedEvents(), 1)
}

func TestEventBus_UserInputEvent_MultipleSubscribers(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	eventBus, err := events.NewEventBus(eventBusConfig)
	require.NoError(t, err)
	defer eventBus.Close()

	handler1 := NewMockUserInputHandler()
	handler2 := NewMockUserInputHandler()

	eventBus.UserInputEvent.Subscribe(constants.Model, handler1.HandleEvent)
	eventBus.UserInputEvent.Subscribe(constants.LLMService, handler2.HandleEvent)

	testUUID := uuid.New()
	sessionUUID := uuid.New()
	testEvent := events.Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			SessionUUID: sessionUUID,
			RequestUUID: testUUID,
			Message:     "broadcast test",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	eventBus.UserInputEvent.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)

	assert.Len(t, handler1.GetReceivedEvents(), 1)
	assert.Len(t, handler2.GetReceivedEvents(), 1)

	for _, handler := range []*MockUserInputHandler{handler1, handler2} {
		receivedEvents := handler.GetReceivedEvents()
		assert.Equal(t, "broadcast test", receivedEvents[0].Data.Message)
		assert.Equal(t, testUUID, receivedEvents[0].Data.RequestUUID)
		assert.Equal(t, sessionUUID, receivedEvents[0].Data.SessionUUID)
	}
}

func TestEventBus_UserInputEvent_HandlerPanic(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	eventBus, err := events.NewEventBus(eventBusConfig)
	require.NoError(t, err)
	defer eventBus.Close()

	panicHandler := func(event events.Event[dto.UserRequestData]) {
		panic("test panic")
	}

	normalHandler := NewMockUserInputHandler()

	eventBus.UserInputEvent.Subscribe(constants.Model, panicHandler)
	eventBus.UserInputEvent.Subscribe(constants.LLMService, normalHandler.HandleEvent)

	testUUID := uuid.New()
	sessionUUID := uuid.New()
	testEvent := events.Event[dto.UserRequestData]{
		Data: dto.UserRequestData{
			SessionUUID: sessionUUID,
			RequestUUID: testUUID,
			Message:     "panic test",
		},
		TimeStamp: time.Now(),
		Source:    constants.Model,
	}

	eventBus.UserInputEvent.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)

	receivedEvents := normalHandler.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	assert.Equal(t, "panic test", receivedEvents[0].Data.Message)
}

func TestEventBus_Close(t *testing.T) {
	eventBusConfig := config.EventBusConfig{PoolSize: 100}
	eventBus, err := events.NewEventBus(eventBusConfig)
	require.NoError(t, err)

	handler := NewMockUserInputHandler()
	eventBus.UserInputEvent.Subscribe(constants.Model, handler.HandleEvent)

	eventBus.Close()
	assert.NotPanics(t, func() {
		eventBus.Close()
	})
}