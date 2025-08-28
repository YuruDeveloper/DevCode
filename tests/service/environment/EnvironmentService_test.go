package environment_test

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/service/environment"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockEnvironmentBus struct {
	PublishedEvents []events.Event
	Subscribers     map[events.EventType][]events.Subscriber
}

func NewMockEnvironmentBus() *MockEnvironmentBus {
	return &MockEnvironmentBus{
		PublishedEvents: make([]events.Event, 0),
		Subscribers:     make(map[events.EventType][]events.Subscriber),
	}
}

func (m *MockEnvironmentBus) Subscribe(eventType events.EventType, subscriber events.Subscriber) {
	m.Subscribers[eventType] = append(m.Subscribers[eventType], subscriber)
}

func (m *MockEnvironmentBus) UnSubscribe(eventType events.EventType, subscriberID constants.Source) {
	subscribers := m.Subscribers[eventType]
	for i, subscriber := range subscribers {
		if subscriber.GetID() == subscriberID {
			m.Subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			return
		}
	}
}

func (m *MockEnvironmentBus) Publish(event events.Event) {
	m.PublishedEvents = append(m.PublishedEvents, event)
}

func TestNewEnvironmentService(t *testing.T) {
	mockBus := NewMockEnvironmentBus()

	service := environment.NewEnvironmentService(mockBus)

	assert.NotNil(t, service)
	assert.Equal(t, constants.EnvironmentService, service.GetID())

	subscribers := mockBus.Subscribers[events.RequestEnvironmentEvent]
	require.Len(t, subscribers, 1)
	assert.Equal(t, constants.EnvironmentService, subscribers[0].GetID())
}

func TestEnvironmentService_HandleEvent_RequestEnvironmentEvent(t *testing.T) {
	mockBus := NewMockEnvironmentBus()
	service := environment.NewEnvironmentService(mockBus)

	requestEvent := events.Event{
		Type:      events.RequestEnvironmentEvent,
		Data:      nil,
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}

	service.HandleEvent(requestEvent)

	require.Len(t, mockBus.PublishedEvents, 1)

	publishedEvent := mockBus.PublishedEvents[0]
	assert.Equal(t, events.UpdateEnvironmentEvent, publishedEvent.Type)
	assert.Equal(t, constants.EnvironmentService, publishedEvent.Source)

	envData, ok := publishedEvent.Data.(dto.EnvironmentUpdateData)
	require.True(t, ok)

	assert.NotEmpty(t, envData.CreateUUID)
	assert.NotEmpty(t, envData.Cwd)
	assert.NotEmpty(t, envData.OS)
	assert.NotEmpty(t, envData.TodayDate)

	_, err := time.Parse("2006-01-02", envData.TodayDate)
	assert.NoError(t, err)
}

func TestEnvironmentService_HandleEvent_IgnoreOtherEvents(t *testing.T) {
	mockBus := NewMockEnvironmentBus()
	service := environment.NewEnvironmentService(mockBus)

	otherEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "test data",
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}

	service.HandleEvent(otherEvent)

	assert.Len(t, mockBus.PublishedEvents, 0)
}

func TestEnvironmentService_EnvironmentData_UniqueUUIDs(t *testing.T) {
	mockBus := NewMockEnvironmentBus()
	service := environment.NewEnvironmentService(mockBus)

	for i := 0; i < 2; i++ {
		requestEvent := events.Event{
			Type:      events.RequestEnvironmentEvent,
			Data:      nil,
			Timestamp: time.Now(),
			Source:    constants.MessageService,
		}
		service.HandleEvent(requestEvent)
	}

	require.Len(t, mockBus.PublishedEvents, 2)

	envData1, ok1 := mockBus.PublishedEvents[0].Data.(dto.EnvironmentUpdateData)
	envData2, ok2 := mockBus.PublishedEvents[1].Data.(dto.EnvironmentUpdateData)

	require.True(t, ok1)
	require.True(t, ok2)

	assert.Equal(t, envData1.Cwd, envData2.Cwd)
	assert.Equal(t, envData1.OS, envData2.OS)
	assert.Equal(t, envData1.TodayDate, envData2.TodayDate)

	assert.NotEqual(t, envData1.CreateUUID, envData2.CreateUUID)
}
