package tests

import (
	"UniCode/src/events"
	"UniCode/src/service"
	"UniCode/src/types"
	"testing"
	"time"

	"github.com/google/uuid"
)

// SetupHistoryService creates a new HistoryService for testing
func SetupHistoryService() (*service.HistoryService, *events.EventBus) {
	EventBus := events.NewEventBus()
	HistoryService := service.NewHistoryService(EventBus)
	return HistoryService, EventBus
}

// CreateTestEnvironmentData creates test environment data for testing
func CreateTestEnvironmentData() types.EnviromentUpdateData {
	return types.EnviromentUpdateData{
		CreateUUID:           uuid.New(),
		Cwd:                  "/test/path",
		OS:                   "linux",
		OSVersion:            "5.4.0",
		IsDirectoryGitRepo:   true,
		TodayDate:            "2024-01-01",
	}
}

func TestNewHistoryService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given
	EventBus := events.NewEventBus()

	// When
	HistoryService := service.NewHistoryService(EventBus)

	// Then
	if HistoryService == nil {
		t.Fatal("NewHistoryService는 nil을 반환하면 안됩니다")
	}

	if HistoryService.Bus != EventBus {
		t.Error("HistoryService의 Bus가 올바르게 설정되지 않았습니다")
	}

	if HistoryService.ParentUUID != uuid.Nil {
		t.Error("ParentUUID가 uuid.Nil로 초기화되지 않았습니다")
	}
}

func TestHistoryService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	HistoryService, _ := SetupHistoryService()
	ExpectedID := types.HistoryService

	// When
	ActualID := HistoryService.GetID()

	// Then
	if int(ActualID) != int(ExpectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(ActualID), int(ExpectedID))
	}
}

func TestHistoryService_HandleEvent_ShouldUpdateEnvironmentDataCorrectly(t *testing.T) {
	// Given
	HistoryService, _ := SetupHistoryService()
	EnvironmentData := CreateTestEnvironmentData()
	UpdateEnvironmentEvent := events.Event{
		Type:      events.UpdateEnvionmentEvent,
		Data:      EnvironmentData,
		Timestamp: time.Now(),
		Source:    types.EnvironmentService,
	}

	// When
	HistoryService.HandleEvent(UpdateEnvironmentEvent)

	// Then
	AssertEnvironmentDataEquals(t, HistoryService.EnviromentData, EnvironmentData)
}

func TestHistoryService_HandleEvent_ShouldHandleUnknownEventsWithoutPanic(t *testing.T) {
	// Given
	HistoryService, _ := SetupHistoryService()
	UnknownEvent := events.Event{
		Type:      events.StreamStartEvent,
		Data:      types.StreamStartData{RequestUUID: uuid.New()},
		Timestamp: time.Now(),
		Source:    types.LLMService,
	}

	// When & Then
	AssertNoPanic(t, "HandleEvent", func() {
		HistoryService.HandleEvent(UnknownEvent)
	})
}

// AssertEnvironmentDataEquals compares two environment data structures
func AssertEnvironmentDataEquals(t *testing.T, actual, expected types.EnviromentUpdateData) {
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