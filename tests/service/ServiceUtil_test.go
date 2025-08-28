package service_test

import (
	"DevCode/src/constants"
	"DevCode/src/events"
	"DevCode/src/service"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MockEventBus struct {
	PublishedEvents []events.Event
}

func NewMockEventBus() *MockEventBus {
	return &MockEventBus{
		PublishedEvents: make([]events.Event, 0),
	}
}

func (m *MockEventBus) Subscribe(eventType events.EventType, subscriber events.Subscriber) {
	// Mock implementation - not needed for this test
}

func (m *MockEventBus) UnSubscribe(eventType events.EventType, subscriberID constants.Source) {
	// Mock implementation - not needed for this test
}

func (m *MockEventBus) Publish(event events.Event) {
	m.PublishedEvents = append(m.PublishedEvents, event)
}

func TestPublishEvent(t *testing.T) {
	mockBus := NewMockEventBus()

	testData := "test data"
	testEventType := events.UserInputEvent
	testSource := constants.MessageService

	// 이벤트 발행 전 시간 기록
	beforePublish := time.Now()

	service.PublishEvent(mockBus, testEventType, testData, testSource)

	// 이벤트 발행 후 시간 기록
	afterPublish := time.Now()

	// 이벤트가 발행되었는지 확인
	require.Len(t, mockBus.PublishedEvents, 1)

	publishedEvent := mockBus.PublishedEvents[0]

	// 이벤트 필드들 확인
	assert.Equal(t, testEventType, publishedEvent.Type)
	assert.Equal(t, testData, publishedEvent.Data)
	assert.Equal(t, testSource, publishedEvent.Source)

	// 타임스탬프가 올바른 범위에 있는지 확인
	assert.True(t, publishedEvent.Timestamp.After(beforePublish) || publishedEvent.Timestamp.Equal(beforePublish))
	assert.True(t, publishedEvent.Timestamp.Before(afterPublish) || publishedEvent.Timestamp.Equal(afterPublish))
}

func TestPublishEvent_MultipleEvents(t *testing.T) {
	mockBus := NewMockEventBus()

	// 여러 이벤트 발행
	events := []struct {
		eventType events.EventType
		data      interface{}
		source    constants.Source
	}{
		{events.UserInputEvent, "user input", constants.MessageService},
		{events.ToolCallEvent, "tool call", constants.ToolService},
		{events.StreamStartEvent, "stream start", constants.LLMService},
	}

	for _, eventData := range events {
		service.PublishEvent(mockBus, eventData.eventType, eventData.data, eventData.source)
	}

	// 모든 이벤트가 발행되었는지 확인
	require.Len(t, mockBus.PublishedEvents, len(events))

	// 각 이벤트가 올바르게 발행되었는지 확인
	for i, eventData := range events {
		publishedEvent := mockBus.PublishedEvents[i]
		assert.Equal(t, eventData.eventType, publishedEvent.Type)
		assert.Equal(t, eventData.data, publishedEvent.Data)
		assert.Equal(t, eventData.source, publishedEvent.Source)
	}
}

func TestPublishEvent_WithComplexData(t *testing.T) {
	mockBus := NewMockEventBus()

	// 복잡한 데이터 구조 테스트
	complexData := map[string]interface{}{
		"user_id":   123,
		"message":   "hello world",
		"metadata":  []string{"tag1", "tag2"},
		"timestamp": time.Now(),
		"is_active": true,
	}

	service.PublishEvent(mockBus, events.UserInputEvent, complexData, constants.MessageService)

	require.Len(t, mockBus.PublishedEvents, 1)

	publishedEvent := mockBus.PublishedEvents[0]
	assert.Equal(t, events.UserInputEvent, publishedEvent.Type)
	assert.Equal(t, complexData, publishedEvent.Data)
	assert.Equal(t, constants.MessageService, publishedEvent.Source)
}

func TestPublishEvent_WithNilData(t *testing.T) {
	mockBus := NewMockEventBus()

	// nil 데이터로 이벤트 발행
	service.PublishEvent(mockBus, events.StreamCompleteEvent, nil, constants.LLMService)

	require.Len(t, mockBus.PublishedEvents, 1)

	publishedEvent := mockBus.PublishedEvents[0]
	assert.Equal(t, events.StreamCompleteEvent, publishedEvent.Type)
	assert.Nil(t, publishedEvent.Data)
	assert.Equal(t, constants.LLMService, publishedEvent.Source)
}

func TestPublishEvent_TimestampPrecision(t *testing.T) {
	mockBus := NewMockEventBus()

	// 연속으로 이벤트 발행
	service.PublishEvent(mockBus, events.UserInputEvent, "first", constants.MessageService)
	service.PublishEvent(mockBus, events.UserInputEvent, "second", constants.MessageService)

	require.Len(t, mockBus.PublishedEvents, 2)

	firstEvent := mockBus.PublishedEvents[0]
	secondEvent := mockBus.PublishedEvents[1]

	// 두 번째 이벤트의 타임스탬프가 첫 번째보다 같거나 늦어야 함
	assert.True(t,
		secondEvent.Timestamp.After(firstEvent.Timestamp) ||
			secondEvent.Timestamp.Equal(firstEvent.Timestamp),
	)
}
