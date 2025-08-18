package tests

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewEnvironmentService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given & When
	environmentService, eventBus := SetupEnvironmentService()

	// Then
	if environmentService == nil {
		t.Fatal("NewEnvironmentService는 nil을 반환하면 안됩니다")
	}

	if environmentService.Bus != eventBus {
		t.Error("EnvironmentService의 Bus가 올바르게 설정되지 않았습니다")
	}
}

func TestEnvironmentService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	environmentService, _ := SetupEnvironmentService()
	expectedID := types.EnvironmentService

	// When
	actualID := environmentService.GetID()

	// Then
	if int(actualID) != int(expectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(actualID), int(expectedID))
	}
}

func TestEnvironmentService_HandleEvent_ShouldPublishEnvironmentUpdateEvent(t *testing.T) {
	// Given
	environmentService, eventBus := SetupEnvironmentService()
	var receivedEvent *events.Event
	
	testHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			if event.Type == events.UpdateEnvionmentEvent {
				receivedEvent = &event
			}
		},
		ID: TestService,
	}
	eventBus.Subscribe(events.UpdateEnvionmentEvent, testHandler)

	requestEvent := events.Event{
		Type: events.RequestEnvionmentvent,
		Data: types.EnviromentRequestData{
			CreateUUID: uuid.New(),
		},
		Timestamp: time.Now(),
		Source:    TestService,
	}

	// When
	environmentService.HandleEvent(requestEvent)
	time.Sleep(AsyncWaitTime)

	// Then
	if receivedEvent == nil {
		t.Error("UpdateEnvionmentEvent가 발행되지 않았습니다")
		return
	}

	assertEnvironmentUpdateEventIsValid(t, *receivedEvent)
}

// assertEnvironmentUpdateEventIsValid validates environment update event data
func assertEnvironmentUpdateEventIsValid(t *testing.T, event events.Event) {
	environmentData, ok := event.Data.(types.EnviromentUpdateData)
	if !ok {
		t.Error("이벤트 데이터 타입이 올바르지 않습니다")
		return
	}

	if environmentData.CreateUUID.String() == "00000000-0000-0000-0000-000000000000" {
		t.Error("CreateUUID가 설정되지 않았습니다")
	}

	if environmentData.Cwd == "" {
		t.Error("현재 작업 디렉토리가 설정되지 않았습니다")
	}

	if environmentData.OS == "" {
		t.Error("OS 정보가 설정되지 않았습니다")
	}

	if environmentData.TodayDate == "" {
		t.Error("오늘 날짜가 설정되지 않았습니다")
	}
}


