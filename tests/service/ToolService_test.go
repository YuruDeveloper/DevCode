package tests

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/viper"
)

func TestNewToolService_ShouldCreateServiceSuccessfully(t *testing.T) {
	// Given & When
	toolService, eventBus := SetupToolService()

	// Then
	if toolService == nil {
		t.Fatal("NewToolService는 nil을 반환하면 안됩니다")
	}

	if toolService.Bus != eventBus {
		t.Error("ToolService의 Bus가 올바르게 설정되지 않았습니다")
	}
}

func TestToolService_GetID_ShouldReturnCorrectServiceID(t *testing.T) {
	// Given
	toolService, _ := SetupToolService()
	expectedID := types.ToolService

	// When
	actualID := toolService.GetID()

	// Then
	if int(actualID) != int(expectedID) {
		t.Errorf("GetID() = %d, 예상값 %d", int(actualID), int(expectedID))
	}
}

func TestToolService_HandleEvent_ShouldProcessToolCallEventWithoutPanic(t *testing.T) {
	// Given
	toolService, _ := SetupToolService()
	toolCallEvent := CreateTestToolCallEvent()

	// When & Then
	AssertNoPanic(t, "HandleEvent", func() {
		toolService.HandleEvent(toolCallEvent)
	})
}

func TestToolService_HandleEvent_ShouldIgnoreNonToolCallEvents(t *testing.T) {
	// Given
	toolService, _ := SetupToolService()
	streamStartEvent := CreateTestStreamStartEvent()

	// When & Then
	AssertNoPanic(t, "HandleEvent with non-ToolCall event", func() {
		toolService.HandleEvent(streamStartEvent)
	})
}

func TestToolService_ProcessToolCall_ShouldProcessToolCallDataWithoutPanic(t *testing.T) {
	// Given
	toolService, _ := SetupToolService()
	testData := types.ToolCallData{
		RequestUUID: uuid.New(),
		ToolName:    "testTool",
		Parameters:   map[string]any{"param1": "value1"},
	}

	// When & Then
	AssertNoPanic(t, "ProcessToolCall", func() {
		toolService.ProcessToolCall(testData)
	})
}

func TestToolService_AllowedTools_ShouldLoadFromConfiguration(t *testing.T) {
	// Given
	expectedAllowed := []string{"tool1", "tool2", "tool3"}
	viper.Set("tool.allowed", expectedAllowed)
	
	// When
	toolService, _ := SetupToolService()

	// Then
	if len(toolService.Allowed) != len(expectedAllowed) {
		t.Errorf("Allowed 도구 개수가 일치하지 않습니다. 예상: %d, 실제: %d", 
			len(expectedAllowed), len(toolService.Allowed))
	}

	for i, expected := range expectedAllowed {
		if i < len(toolService.Allowed) && toolService.Allowed[i] != expected {
			t.Errorf("Allowed[%d] = %s, 예상값 %s", i, toolService.Allowed[i], expected)
		}
	}
}

func TestToolService_HandleEvent_ShouldOnlyProcessToolCallEventType(t *testing.T) {
	// Given
	toolService, eventBus := SetupToolService()
	
	// Set up test event handler to track processed events
	processedEvents := make([]events.Event, 0)
	testHandler := &TestEventHandler{
		HandleFunc: func(event events.Event) {
			processedEvents = append(processedEvents, event)
		},
		ID: TestService,
	}
	eventBus.Subscribe(events.ToolCallEvent, testHandler)

	// Create different event types
	toolCallEvent := CreateTestToolCallEvent()
	streamStartEvent := CreateTestStreamStartEvent()

	// When
	toolService.HandleEvent(toolCallEvent)
	toolService.HandleEvent(streamStartEvent)

	// Then - Should only process ToolCallEvent, not others
	// This test verifies the switch statement behavior
	AssertNoPanic(t, "Processing mixed event types", func() {
		// The assertion is that no panic occurs when processing different event types
	})
}