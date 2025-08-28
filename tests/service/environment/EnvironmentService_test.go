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
	"go.uber.org/zap/zaptest"
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
	logger := zaptest.NewLogger(t)
	mockBus := NewMockEnvironmentBus()
	
	service := environment.NewEnvironmentService(mockBus, logger)
	
	assert.NotNil(t, service)
	
	// RequestEnvironmentEvent에 구독했는지 확인
	subscribers := mockBus.Subscribers[events.RequestEnvironmentEvent]
	require.Len(t, subscribers, 1)
	assert.Equal(t, constants.EnvironmentService, subscribers[0].GetID())
}

func TestEnvironmentService_GetID(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockBus := NewMockEnvironmentBus()
	
	service := environment.NewEnvironmentService(mockBus, logger)
	
	assert.Equal(t, constants.EnvironmentService, service.GetID())
}

func TestEnvironmentService_HandleEvent_RequestEnvironmentEvent(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockBus := NewMockEnvironmentBus()
	
	service := environment.NewEnvironmentService(mockBus, logger)
	
	// RequestEnvironmentEvent 이벤트 생성
	requestEvent := events.Event{
		Type:      events.RequestEnvironmentEvent,
		Data:      nil,
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}
	
	// 이벤트 처리
	service.HandleEvent(requestEvent)
	
	// UpdateEnvironmentEvent가 발행되었는지 확인
	require.Len(t, mockBus.PublishedEvents, 1)
	
	publishedEvent := mockBus.PublishedEvents[0]
	assert.Equal(t, events.UpdateEnvironmentEvent, publishedEvent.Type)
	assert.Equal(t, constants.EnvironmentService, publishedEvent.Source)
	
	// 환경 데이터가 올바르게 설정되었는지 확인
	envData, ok := publishedEvent.Data.(dto.EnvironmentUpdateData)
	require.True(t, ok)
	
	// 기본적인 환경 정보 검증
	assert.NotEmpty(t, envData.CreateUUID)
	assert.NotEmpty(t, envData.Cwd)
	assert.NotEmpty(t, envData.OS)
	assert.NotEmpty(t, envData.TodayDate)
	
	// 날짜 형식 검증 (YYYY-MM-DD)
	_, err := time.Parse("2006-01-02", envData.TodayDate)
	assert.NoError(t, err)
}

func TestEnvironmentService_HandleEvent_IgnoreOtherEvents(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockBus := NewMockEnvironmentBus()
	
	service := environment.NewEnvironmentService(mockBus, logger)
	
	// 다른 타입의 이벤트 생성
	otherEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "test data",
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}
	
	// 이벤트 처리
	service.HandleEvent(otherEvent)
	
	// 아무 이벤트도 발행되지 않아야 함
	assert.Len(t, mockBus.PublishedEvents, 0)
}

func TestEnvironmentService_HandleEvent_MultipleRequests(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockBus := NewMockEnvironmentBus()
	
	service := environment.NewEnvironmentService(mockBus, logger)
	
	// 여러 RequestEnvironmentEvent 처리
	for i := 0; i < 3; i++ {
		requestEvent := events.Event{
			Type:      events.RequestEnvironmentEvent,
			Data:      nil,
			Timestamp: time.Now(),
			Source:    constants.MessageService,
		}
		service.HandleEvent(requestEvent)
	}
	
	// 3개의 UpdateEnvironmentEvent가 발행되었는지 확인
	require.Len(t, mockBus.PublishedEvents, 3)
	
	for i, publishedEvent := range mockBus.PublishedEvents {
		assert.Equal(t, events.UpdateEnvironmentEvent, publishedEvent.Type, "Event %d type mismatch", i)
		assert.Equal(t, constants.EnvironmentService, publishedEvent.Source, "Event %d source mismatch", i)
		
		envData, ok := publishedEvent.Data.(dto.EnvironmentUpdateData)
		require.True(t, ok, "Event %d data type mismatch", i)
		assert.NotEmpty(t, envData.CreateUUID, "Event %d UUID empty", i)
	}
}

func TestEnvironmentService_EnvironmentData_Consistency(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockBus := NewMockEnvironmentBus()
	
	service := environment.NewEnvironmentService(mockBus, logger)
	
	// 두 번의 요청 처리
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
	
	// 일관성 있는 데이터는 같아야 함
	assert.Equal(t, envData1.Cwd, envData2.Cwd)
	assert.Equal(t, envData1.OS, envData2.OS)
	assert.Equal(t, envData1.IsDirectoryGitRepo, envData2.IsDirectoryGitRepo)
	assert.Equal(t, envData1.TodayDate, envData2.TodayDate)
	
	// UUID는 매번 달라야 함
	assert.NotEqual(t, envData1.CreateUUID, envData2.CreateUUID)
}

func TestEnvironmentService_EventTimestamp(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockBus := NewMockEnvironmentBus()
	
	service := environment.NewEnvironmentService(mockBus, logger)
	
	beforeRequest := time.Now()
	
	requestEvent := events.Event{
		Type:      events.RequestEnvironmentEvent,
		Data:      nil,
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}
	
	service.HandleEvent(requestEvent)
	
	afterRequest := time.Now()
	
	require.Len(t, mockBus.PublishedEvents, 1)
	
	publishedEvent := mockBus.PublishedEvents[0]
	
	// 발행된 이벤트의 타임스탬프가 올바른 범위에 있는지 확인
	assert.True(t, 
		publishedEvent.Timestamp.After(beforeRequest) || 
		publishedEvent.Timestamp.Equal(beforeRequest),
	)
	assert.True(t, 
		publishedEvent.Timestamp.Before(afterRequest) || 
		publishedEvent.Timestamp.Equal(afterRequest),
	)
}