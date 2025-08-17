package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"testing"
	"time"

	"github.com/google/uuid"
)

const TestService types.Source = 999

// TestEventHandler is a test implementation of EventHandler interface
type TestEventHandler struct {
	HandleFunc func(events.Event)
	ID         types.Source
}

func (handler *TestEventHandler) HandleEvent(event events.Event) {
	if handler.HandleFunc != nil {
		handler.HandleFunc(event)
	}
}

func (handler *TestEventHandler) GetID() types.Source {
	return handler.ID
}


// SetupEnvironmentService creates a new EnvironmentService for testing
func SetupEnvironmentService() (*service.EnvironmentService, *events.EventBus) {
	EventBus := events.NewEventBus()
	EnvironmentService := service.NewEnvironmentService(EventBus)
	return EnvironmentService, EventBus
}

// SetupTestEventHandler creates a test event handler
func SetupTestEventHandler(EventBus *events.EventBus, EventType events.EventType) (*TestEventHandler, *events.Event) {
	var ReceivedEvent *events.Event
	TestHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			if event.Type == EventType {
				ReceivedEvent = &event
			}
		},
		ID: TestService,
	}
	EventBus.Subscribe(EventType, TestHandler)
	return TestHandler, ReceivedEvent
}

func TestNewEnvironmentService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given
	EventBus := events.NewEventBus()

	// When
	EnvironmentService := service.NewEnvironmentService(EventBus)

	// Then
	if EnvironmentService == nil {
		t.Fatal("NewEnvironmentService는 nil을 반환하면 안됩니다")
	}

	if EnvironmentService.Bus != EventBus {
		t.Error("EnvironmentService의 Bus가 올바르게 설정되지 않았습니다")
	}
}

func TestEnvironmentService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	EnvironmentService, _ := SetupEnvironmentService()
	ExpectedID := types.EnvironmentService

	// When
	ActualID := EnvironmentService.GetID()

	// Then
	if int(ActualID) != int(ExpectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(ActualID), int(ExpectedID))
	}
}

func TestEnvironmentService_HandleEvent_ShouldPublishEnvironmentUpdateEvent(t *testing.T) {
	// Given
	EnvironmentService, EventBus := SetupEnvironmentService()
	var ReceivedEvent *events.Event
	
	TestHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			if event.Type == events.UpdateEnvionmentEvent {
				ReceivedEvent = &event
			}
		},
		ID: TestService,
	}
	EventBus.Subscribe(events.UpdateEnvionmentEvent, TestHandler)

	RequestEvent := events.Event{
		Type: events.RequestEnvionmentvent,
		Data: types.EnviromentRequestData{
			CreateUUID: uuid.New(),
		},
		Timestamp: time.Now(),
		Source:    TestService,
	}

	// When
	EnvironmentService.HandleEvent(RequestEvent)
	time.Sleep(50 * time.Millisecond) // Wait for async processing

	// Then
	if ReceivedEvent == nil {
		t.Error("UpdateEnvionmentEvent가 발행되지 않았습니다")
		return
	}

	AssertEnvironmentUpdateEventIsValid(t, *ReceivedEvent)
}

// AssertEnvironmentUpdateEventIsValid validates environment update event data
func AssertEnvironmentUpdateEventIsValid(t *testing.T, event events.Event) {
	EnvironmentData, ok := event.Data.(types.EnviromentUpdateData)
	if !ok {
		t.Error("이벤트 데이터 타입이 올바르지 않습니다")
		return
	}

	if EnvironmentData.CreateUUID == uuid.Nil {
		t.Error("CreateUUID가 설정되지 않았습니다")
	}

	if EnvironmentData.Cwd == "" {
		t.Error("현재 작업 디렉토리가 설정되지 않았습니다")
	}

	if EnvironmentData.OS == "" {
		t.Error("OS 정보가 설정되지 않았습니다")
	}

	if EnvironmentData.TodayDate == "" {
		t.Error("오늘 날짜가 설정되지 않았습니다")
	}
}

