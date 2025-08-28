package events_test

import (
	"DevCode/src/constants"
	"DevCode/src/events"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockSubscriber struct {
	ID             constants.Source
	ReceivedEvents []events.Event
	Mutex          sync.Mutex
}

func NewMockSubscriber(id constants.Source) *MockSubscriber {
	return &MockSubscriber{
		ID:             id,
		ReceivedEvents: make([]events.Event, 0),
	}
}

func (m *MockSubscriber) HandleEvent(event events.Event) {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	m.ReceivedEvents = append(m.ReceivedEvents, event)
}

func (m *MockSubscriber) GetID() constants.Source {
	return m.ID
}

func (m *MockSubscriber) GetReceivedEvents() []events.Event {
	m.Mutex.Lock()
	defer m.Mutex.Unlock()
	return append([]events.Event{}, m.ReceivedEvents...)
}

func TestNewEventBus(t *testing.T) {
	eventBus, err := events.NewEventBus()

	require.NoError(t, err)
	assert.NotNil(t, eventBus)
	eventBus.Close()
}

func TestEventBus_Subscribe_And_Publish(t *testing.T) {
	eventBus, err := events.NewEventBus()
	require.NoError(t, err)
	defer eventBus.Close()

	subscriber := NewMockSubscriber(constants.EnvironmentService)
	eventBus.Subscribe(events.UserInputEvent, subscriber)

	testEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "test data",
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}

	eventBus.Publish(testEvent)
	time.Sleep(50 * time.Millisecond)

	receivedEvents := subscriber.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	assert.Equal(t, events.UserInputEvent, receivedEvents[0].Type)
	assert.Equal(t, "test data", receivedEvents[0].Data)
}

func TestEventBus_UnSubscribe(t *testing.T) {
	eventBus, err := events.NewEventBus()
	require.NoError(t, err)
	defer eventBus.Close()

	subscriber := NewMockSubscriber(constants.EnvironmentService)
	eventBus.Subscribe(events.UserInputEvent, subscriber)

	testEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "test data",
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}

	eventBus.Publish(testEvent)
	time.Sleep(50 * time.Millisecond)
	require.Len(t, subscriber.GetReceivedEvents(), 1)

	eventBus.UnSubscribe(events.UserInputEvent, constants.EnvironmentService)
	eventBus.Publish(testEvent)
	time.Sleep(50 * time.Millisecond)

	assert.Len(t, subscriber.GetReceivedEvents(), 1)
}

func TestEventBus_MultipleSubscribers(t *testing.T) {
	eventBus, err := events.NewEventBus()
	require.NoError(t, err)
	defer eventBus.Close()

	subscriber1 := NewMockSubscriber(constants.EnvironmentService)
	subscriber2 := NewMockSubscriber(constants.MessageService)

	eventBus.Subscribe(events.UserInputEvent, subscriber1)
	eventBus.Subscribe(events.UserInputEvent, subscriber2)

	testEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "broadcast test",
		Timestamp: time.Now(),
		Source:    constants.McpService,
	}

	eventBus.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)

	assert.Len(t, subscriber1.GetReceivedEvents(), 1)
	assert.Len(t, subscriber2.GetReceivedEvents(), 1)

	for _, subscriber := range []*MockSubscriber{subscriber1, subscriber2} {
		receivedEvents := subscriber.GetReceivedEvents()
		assert.Equal(t, events.UserInputEvent, receivedEvents[0].Type)
		assert.Equal(t, "broadcast test", receivedEvents[0].Data)
	}
}

func TestEventBus_SubscriberPanic(t *testing.T) {
	eventBus, err := events.NewEventBus()
	require.NoError(t, err)
	defer eventBus.Close()

	panicSubscriber := &PanicSubscriber{ID: constants.EnvironmentService}
	normalSubscriber := NewMockSubscriber(constants.MessageService)

	eventBus.Subscribe(events.UserInputEvent, panicSubscriber)
	eventBus.Subscribe(events.UserInputEvent, normalSubscriber)

	testEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "panic test",
		Timestamp: time.Now(),
		Source:    constants.McpService,
	}

	eventBus.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)

	receivedEvents := normalSubscriber.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	assert.Equal(t, "panic test", receivedEvents[0].Data)
}

type PanicSubscriber struct {
	ID constants.Source
}

func (p *PanicSubscriber) HandleEvent(event events.Event) {
	panic("test panic")
}

func (p *PanicSubscriber) GetID() constants.Source {
	return p.ID
}
