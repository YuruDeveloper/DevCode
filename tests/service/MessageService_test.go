package tests

import (
	"UniCode/src/types"
	"testing"

	"github.com/google/uuid"
)

func TestNewMessageService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given & When
	messageService, eventBus := SetupMessageService()

	// Then
	if messageService == nil {
		t.Fatal("NewMessageService는 nil을 반환하면 안됩니다")
	}

	if messageService.Bus != eventBus {
		t.Error("MessageService의 Bus가 올바르게 설정되지 않았습니다")
	}
}

func TestMessageService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	messageService, _ := SetupMessageService()
	expectedID := types.MessageService

	// When
	actualID := messageService.GetID()

	// Then
	if int(actualID) != int(expectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(actualID), int(expectedID))
	}
}

func TestMessageService_HandleEvent_ShouldProcessStreamStartEventWithoutPanic(t *testing.T) {
	// Given
	messageService, _ := SetupMessageService()
	streamStartEvent := CreateTestStreamStartEvent()

	// When & Then
	AssertNoPanic(t, "HandleEvent", func() {
		messageService.HandleEvent(streamStartEvent)
	})
}

func TestMessageService_ParsingMessage_ShouldProcessMessageWithoutPanic(t *testing.T) {
	// Given
	messageService, _ := SetupMessageService()
	testData := types.StreamChunkData{
		RequestUUID: uuid.New(),
		Content:     "테스트 메시지",
		IsComplete:  false,
	}

	// When & Then
	AssertNoPanic(t, "ParsingMessage", func() {
		messageService.ParsingMessage(testData)
	})
}