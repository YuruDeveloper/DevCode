package tests

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"testing"
	"time"
)

func TestNewHistoryService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given & When
	historyService, eventBus := SetupHistoryService()

	// Then
	if historyService == nil {
		t.Fatal("NewHistoryService는 nil을 반환하면 안됩니다")
	}

	if historyService.Bus != eventBus {
		t.Error("HistoryService의 Bus가 올바르게 설정되지 않았습니다")
	}

	if historyService.ParentUUID.String() != "00000000-0000-0000-0000-000000000000" {
		t.Error("ParentUUID가 uuid.Nil로 초기화되지 않았습니다")
	}
}

func TestHistoryService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	historyService, _ := SetupHistoryService()
	expectedID := types.HistoryService

	// When
	actualID := historyService.GetID()

	// Then
	if int(actualID) != int(expectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(actualID), int(expectedID))
	}
}

func TestHistoryService_HandleEvent_ShouldUpdateEnvironmentDataCorrectly(t *testing.T) {
	// Given
	historyService, _ := SetupHistoryService()
	environmentData := CreateTestEnvironmentData()
	updateEnvironmentEvent := events.Event{
		Type:      events.UpdateEnvionmentEvent,
		Data:      environmentData,
		Timestamp: time.Now(),
		Source:    types.EnvironmentService,
	}

	// When
	historyService.HandleEvent(updateEnvironmentEvent)

	// Then
	assertEnvironmentDataEquals(t, historyService.EnviromentData, environmentData)
}

func TestHistoryService_HandleEvent_ShouldHandleUnknownEventsWithoutPanic(t *testing.T) {
	// Given
	historyService, _ := SetupHistoryService()
	unknownEvent := CreateTestStreamStartEvent()

	// When & Then
	AssertNoPanic(t, "HandleEvent", func() {
		historyService.HandleEvent(unknownEvent)
	})
}

// assertEnvironmentDataEquals compares two environment data structures
func assertEnvironmentDataEquals(t *testing.T, actual, expected types.EnviromentUpdateData) {
	if actual.CreateUUID != expected.CreateUUID {
		t.Error("CreateUUID가 올바르게 업데이트되지 않았습니다")
	}

	if actual.Cwd != expected.Cwd {
		t.Error("Cwd가 올바르게 업데이트되지 않았습니다")
	}

	if actual.OS != expected.OS {
		t.Error("OS가 올바르게 업데이트되지 않았습니다")
	}

	if actual.OSVersion != expected.OSVersion {
		t.Error("OSVersion이 올바르게 업데이트되지 않았습니다")
	}

	if actual.IsDirectoryGitRepo != expected.IsDirectoryGitRepo {
		t.Error("IsDirectoryGitRepo가 올바르게 업데이트되지 않았습니다")
	}

	if actual.TodayDate != expected.TodayDate {
		t.Error("TodayDate가 올바르게 업데이트되지 않았습니다")
	}
}

