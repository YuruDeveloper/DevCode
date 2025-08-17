package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"testing"

	"github.com/ollama/ollama/api"
)

// SetupMessageService creates a new MessageService for testing
func SetupMessageService() (*service.MessageService, *events.EventBus) {
	EventBus := events.NewEventBus()
	MessageService := service.NewMessageService(EventBus)
	return MessageService, EventBus
}

// AssertNoPanic ensures that a function does not panic during execution
func AssertNoPanic(t *testing.T, functionName string, fn func()) {
	defer func() {
		if Recovery := recover(); Recovery != nil {
			t.Errorf("%s에서 패닉 발생: %v", functionName, Recovery)
		}
	}()
	fn()
}

func TestNewMessageService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given
	EventBus := events.NewEventBus()

	// When
	MessageService := service.NewMessageService(EventBus)

	// Then
	if MessageService == nil {
		t.Fatal("NewMessageService는 nil을 반환하면 안됩니다")
	}

	if MessageService.Bus != EventBus {
		t.Error("MessageService의 Bus가 올바르게 설정되지 않았습니다")
	}
}

func TestMessageService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	MessageService, _ := SetupMessageService()
	ExpectedID := types.MessageService

	// When
	ActualID := MessageService.GetID()

	// Then
	if int(ActualID) != int(ExpectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(ActualID), int(ExpectedID))
	}
}

func TestMessageService_HandleEvent_ShouldProcessStreamStartEventWithoutPanic(t *testing.T) {
	// Given
	MessageService, _ := SetupMessageService()
	StreamStartEvent := events.Event{
		Type: events.StreamStartEvent,
		Data: types.StreamStartData{},
	}

	// When & Then
	AssertNoPanic(t, "HandleEvent", func() {
		MessageService.HandleEvent(StreamStartEvent)
	})
}

func TestMessageService_ParsingMessage_ShouldProcessMessageWithoutPanic(t *testing.T) {
	// Given
	MessageService, _ := SetupMessageService()
	TestMessage := api.Message{
		Role:    "user",
		Content: "테스트 메시지",
	}

	// When & Then
	AssertNoPanic(t, "ParsingMessage", func() {
		MessageService.ParingMessage(TestMessage)
	})
}