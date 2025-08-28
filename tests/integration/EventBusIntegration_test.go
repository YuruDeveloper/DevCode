package integration_test

import (
	"DevCode/src/constants"
	"DevCode/src/events"
	"DevCode/src/service/environment"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

func TestEventBusEnvironmentServiceIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	// 실제 EventBus 생성
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	// EnvironmentService 생성 (자동으로 EventBus에 구독됨)
	envService := environment.NewEnvironmentService(eventBus, logger)
	
	// 결과를 수집할 구독자 생성
	resultCollector := NewResultCollector(constants.MessageService)
	eventBus.Subscribe(events.UpdateEnvironmentEvent, resultCollector)
	
	// RequestEnvironmentEvent 발행
	requestEvent := events.Event{
		Type:      events.RequestEnvironmentEvent,
		Data:      nil,
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}
	
	eventBus.Publish(requestEvent)
	
	// 이벤트가 처리될 때까지 대기
	time.Sleep(200 * time.Millisecond)
	
	// 결과 확인
	receivedEvents := resultCollector.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	
	updateEvent := receivedEvents[0]
	assert.Equal(t, events.UpdateEnvironmentEvent, updateEvent.Type)
	assert.Equal(t, constants.EnvironmentService, updateEvent.Source)
	assert.NotNil(t, updateEvent.Data)
	
	// EnvironmentService ID 확인
	assert.Equal(t, constants.EnvironmentService, envService.GetID())
}

func TestMultipleServicesIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	// 여러 서비스 생성
	envService := environment.NewEnvironmentService(eventBus, logger)
	
	// 다양한 이벤트 타입을 수신하는 컬렉터들
	envUpdateCollector := NewResultCollector(constants.ToolService)
	userInputCollector := NewResultCollector(constants.MessageService)
	
	eventBus.Subscribe(events.UpdateEnvironmentEvent, envUpdateCollector)
	eventBus.Subscribe(events.UserInputEvent, userInputCollector)
	
	// 다양한 이벤트 발행
	testEvents := []events.Event{
		{
			Type:      events.RequestEnvironmentEvent,
			Data:      nil,
			Timestamp: time.Now(),
			Source:    constants.MessageService,
		},
		{
			Type:      events.UserInputEvent,
			Data:      "test user input",
			Timestamp: time.Now(),
			Source:    constants.MessageService,
		},
	}
	
	for _, event := range testEvents {
		eventBus.Publish(event)
	}
	
	time.Sleep(300 * time.Millisecond)
	
	// 각 컬렉터가 해당하는 이벤트만 받았는지 확인
	envReceivedEvents := envUpdateCollector.GetReceivedEvents()
	userReceivedEvents := userInputCollector.GetReceivedEvents()
	
	require.Len(t, envReceivedEvents, 1)
	require.Len(t, userReceivedEvents, 1)
	
	assert.Equal(t, events.UpdateEnvironmentEvent, envReceivedEvents[0].Type)
	assert.Equal(t, events.UserInputEvent, userReceivedEvents[0].Type)
	
	_ = envService // 서비스가 올바르게 작동하는지 확인
}

func TestConcurrentEventProcessingIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	defer eventBus.Close()
	
	envService := environment.NewEnvironmentService(eventBus, logger)
	resultCollector := NewResultCollector(constants.HistoryService)
	eventBus.Subscribe(events.UpdateEnvironmentEvent, resultCollector)
	
	numRequests := 10
	var wg sync.WaitGroup
	wg.Add(numRequests)
	
	// 동시에 여러 환경 요청 발행
	for i := 0; i < numRequests; i++ {
		go func(requestID int) {
			defer wg.Done()
			
			requestEvent := events.Event{
				Type:      events.RequestEnvironmentEvent,
				Data:      map[string]int{"request_id": requestID},
				Timestamp: time.Now(),
				Source:    constants.MessageService,
			}
			
			eventBus.Publish(requestEvent)
		}(i)
	}
	
	wg.Wait()
	
	// 모든 요청이 처리될 때까지 대기
	time.Sleep(500 * time.Millisecond)
	
	// 모든 요청에 대한 응답이 수신되었는지 확인
	receivedEvents := resultCollector.GetReceivedEvents()
	assert.Len(t, receivedEvents, numRequests)
	
	// 모든 이벤트가 올바른 타입인지 확인
	for i, event := range receivedEvents {
		assert.Equal(t, events.UpdateEnvironmentEvent, event.Type, "Event %d type mismatch", i)
		assert.Equal(t, constants.EnvironmentService, event.Source, "Event %d source mismatch", i)
	}
	
	_ = envService
}

func TestEventBusShutdownIntegration(t *testing.T) {
	logger := zaptest.NewLogger(t)
	
	eventBus, err := events.NewEventBus(logger)
	require.NoError(t, err)
	
	envService := environment.NewEnvironmentService(eventBus, logger)
	resultCollector := NewResultCollector(constants.LLMService)
	eventBus.Subscribe(events.UpdateEnvironmentEvent, resultCollector)
	
	// 정상 작동 확인
	requestEvent := events.Event{
		Type:      events.RequestEnvironmentEvent,
		Data:      nil,
		Timestamp: time.Now(),
		Source:    constants.MessageService,
	}
	
	eventBus.Publish(requestEvent)
	time.Sleep(100 * time.Millisecond)
	
	receivedEvents := resultCollector.GetReceivedEvents()
	require.Len(t, receivedEvents, 1)
	
	// EventBus 종료
	eventBus.Close()
	
	// 종료 후에는 이벤트가 처리되지 않아야 함 (고루틴 풀이 종료됨)
	eventBus.Publish(requestEvent)
	time.Sleep(100 * time.Millisecond)
	
	// 새로운 이벤트는 처리되지 않아야 함
	finalEvents := resultCollector.GetReceivedEvents()
	assert.Len(t, finalEvents, 1) // 여전히 1개만 있어야 함
	
	_ = envService
}

// 테스트용 결과 수집 구독자
type ResultCollector struct {
	ID             constants.Source
	ReceivedEvents []events.Event
	Mutex          sync.RWMutex
}

func NewResultCollector(id constants.Source) *ResultCollector {
	return &ResultCollector{
		ID:             id,
		ReceivedEvents: make([]events.Event, 0),
	}
}

func (r *ResultCollector) HandleEvent(event events.Event) {
	r.Mutex.Lock()
	defer r.Mutex.Unlock()
	r.ReceivedEvents = append(r.ReceivedEvents, event)
}

func (r *ResultCollector) GetID() constants.Source {
	return r.ID
}

func (r *ResultCollector) GetReceivedEvents() []events.Event {
	r.Mutex.RLock()
	defer r.Mutex.RUnlock()
	return append([]events.Event{}, r.ReceivedEvents...)
}