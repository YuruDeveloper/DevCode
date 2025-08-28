package events_test

import (
	"DevCode/src/constants"
	"DevCode/src/events"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

type MockSubscriber struct {
	ID           constants.Source
	ReceivedEvents []events.Event
	Mutex         sync.Mutex
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
	logger := zaptest.NewLogger(t)
	
	eventBus, err := events.NewEventBus(logger)
	
	require.NoError(t, err)
	assert.NotNil(t, eventBus)
}

func TestEventBus_Subscribe(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	subscriber := NewMockSubscriber(constants.EnvironmentService)
	
	eventBus.Subscribe(events.UserInputEvent, subscriber)
	
	// 구독이 성공했는지 이벤트 발행으로 확인
	testEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "test data",
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}
	
	eventBus.Publish(testEvent)
	
	// 짧은 시간 대기 후 이벤트 수신 확인
	time.Sleep(100 * time.Millisecond)
	
	receivedEvents := subscriber.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	assert.Equal(t, events.UserInputEvent, receivedEvents[0].Type)
	assert.Equal(t, "test data", receivedEvents[0].Data)
}

func TestEventBus_UnSubscribe(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	subscriber := NewMockSubscriber(constants.EnvironmentService)
	
	// 구독
	eventBus.Subscribe(events.UserInputEvent, subscriber)
	
	// 이벤트 발행하여 구독 확인
	testEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "test data",
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}
	
	eventBus.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)
	
	receivedEvents := subscriber.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	
	// 구독 해제
	eventBus.UnSubscribe(events.UserInputEvent, constants.EnvironmentService)
	
	// 다시 이벤트 발행
	eventBus.Publish(testEvent)
	time.Sleep(100 * time.Millisecond)
	
	// 이벤트가 더 이상 수신되지 않아야 함
	receivedEvents = subscriber.GetReceivedEvents()
	assert.Len(t, receivedEvents, 1) // 여전히 1개만 있어야 함
}

func TestEventBus_Publish_MultipleSubscribers(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	subscriber1 := NewMockSubscriber(constants.EnvironmentService)
	subscriber2 := NewMockSubscriber(constants.MessageService)
	subscriber3 := NewMockSubscriber(constants.ToolService)
	
	// 모든 구독자를 같은 이벤트 타입에 구독
	eventBus.Subscribe(events.UserInputEvent, subscriber1)
	eventBus.Subscribe(events.UserInputEvent, subscriber2)
	eventBus.Subscribe(events.UserInputEvent, subscriber3)
	
	testEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "broadcast test",
		Timestamp: time.Now(),
		Source:    constants.McpService,
	}
	
	eventBus.Publish(testEvent)
	
	// 모든 구독자가 이벤트를 받을 때까지 대기
	time.Sleep(200 * time.Millisecond)
	
	// 모든 구독자가 이벤트를 받았는지 확인
	assert.Len(t, subscriber1.GetReceivedEvents(), 1)
	assert.Len(t, subscriber2.GetReceivedEvents(), 1)
	assert.Len(t, subscriber3.GetReceivedEvents(), 1)
	
	for _, subscriber := range []*MockSubscriber{subscriber1, subscriber2, subscriber3} {
		receivedEvents := subscriber.GetReceivedEvents()
		assert.Equal(t, events.UserInputEvent, receivedEvents[0].Type)
		assert.Equal(t, "broadcast test", receivedEvents[0].Data)
	}
}

func TestEventBus_Publish_DifferentEventTypes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	subscriber1 := NewMockSubscriber(constants.EnvironmentService)
	subscriber2 := NewMockSubscriber(constants.MessageService)
	
	// 다른 이벤트 타입에 구독
	eventBus.Subscribe(events.UserInputEvent, subscriber1)
	eventBus.Subscribe(events.ToolCallEvent, subscriber2)
	
	userInputEvent := events.Event{
		Type:      events.UserInputEvent,
		Data:      "user input",
		Timestamp: time.Now(),
		Source:    constants.McpService,
	}
	
	toolCallEvent := events.Event{
		Type:      events.ToolCallEvent,
		Data:      "tool call",
		Timestamp: time.Now(),
		Source:    constants.McpService,
	}
	
	eventBus.Publish(userInputEvent)
	eventBus.Publish(toolCallEvent)
	
	time.Sleep(100 * time.Millisecond)
	
	// subscriber1은 UserInputEvent만 받아야 함
	events1 := subscriber1.GetReceivedEvents()
	require.Len(t, events1, 1)
	assert.Equal(t, events.UserInputEvent, events1[0].Type)
	
	// subscriber2는 ToolCallEvent만 받아야 함
	events2 := subscriber2.GetReceivedEvents()
	require.Len(t, events2, 1)
	assert.Equal(t, events.ToolCallEvent, events2[0].Type)
}

func TestEventBus_ConcurrentPublish(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	subscriber := NewMockSubscriber(constants.EnvironmentService)
	eventBus.Subscribe(events.UserInputEvent, subscriber)
	
	numGoroutines := 100
	numEventsPerGoroutine := 10
	
	var wg sync.WaitGroup
	wg.Add(numGoroutines)
	
	// 동시에 여러 고루틴에서 이벤트 발행
	for i := 0; i < numGoroutines; i++ {
		go func(routineID int) {
			defer wg.Done()
			for j := 0; j < numEventsPerGoroutine; j++ {
				testEvent := events.Event{
					Type:      events.UserInputEvent,
					Data:      map[string]int{"routine": routineID, "event": j},
					Timestamp: time.Now(),
					Source:    constants.MessageService,
				}
				eventBus.Publish(testEvent)
			}
		}(i)
	}
	
	wg.Wait()
	
	// 모든 이벤트가 처리될 때까지 대기
	time.Sleep(500 * time.Millisecond)
	
	receivedEvents := subscriber.GetReceivedEvents()
	expectedCount := numGoroutines * numEventsPerGoroutine
	assert.Len(t, receivedEvents, expectedCount)
}

func TestEventBus_SubscriberPanic(t *testing.T) {
	logger := zaptest.NewLogger(t)
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	// 패닉을 발생시키는 구독자
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
	
	// 패닉이 발생해도 다른 구독자는 정상 동작해야 함
	eventBus.Publish(testEvent)
	
	time.Sleep(200 * time.Millisecond)
	
	// 정상 구독자는 이벤트를 받았어야 함
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