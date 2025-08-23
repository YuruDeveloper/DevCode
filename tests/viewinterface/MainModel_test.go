package viewinterface_test

import (
	"UniCode/src/events"
	"UniCode/src/types"
	"UniCode/src/viewinterface"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMainModel_ActiveTools_EmptyActiveTools(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)

	// When & Then
	assert.Empty(t, model.ActiveTools, "초기 ActiveTools는 비어있어야 함")
}

func TestMainModel_ActiveTools_WithActiveTools(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	toolCall := uuid.New()
	toolModel := viewinterface.NewToolModel("Read (/test/file.txt)")
	
	// ActiveTools에 수동으로 추가
	model.ActiveTools[toolCall] = toolModel

	// When & Then
	assert.Contains(t, model.ActiveTools, toolCall, "ActiveTools에 도구가 추가되어야 함")
	assert.Equal(t, toolModel, model.ActiveTools[toolCall], "저장된 ToolModel이 일치해야 함")
}

func TestMainModel_HandleEvent_ToolUseReportEvent_Call(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	requestUUID := uuid.New()
	toolCall := uuid.New()
	eventData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Call,
	}
	
	event := events.Event{
		Type:      events.ToolUseReportEvent,
		Data:      eventData,
		Timestamp: time.Now(),
		Source:    types.ToolService,
	}

	// When
	model.HandleEvent(event)

	// Then
	assert.Contains(t, model.ActiveTools, toolCall, "ActiveTools에 도구가 추가되어야 함")
	assert.NotNil(t, model.ActiveTools[toolCall], "저장된 ToolModel이 존재해야 함")
}

func TestMainModel_HandleEvent_ToolUseReportEvent_Success(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	requestUUID := uuid.New()
	toolCall := uuid.New()
	
	// 먼저 Call 상태로 ActiveTools에 추가
	toolModel := viewinterface.NewToolModel("Read (/test/file.txt)")
	model.ActiveTools[toolCall] = toolModel
	
	// Success 이벤트 데이터
	successData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Success,
	}
	
	event := events.Event{
		Type:      events.ToolUseReportEvent,
		Data:      successData,
		Timestamp: time.Now(),
		Source:    types.ToolService,
	}

	// When
	model.HandleEvent(event)

	// Then
	assert.NotContains(t, model.ActiveTools, toolCall, "Success 시 ActiveTools에서 제거되어야 함")
}

func TestMainModel_HandleEvent_ToolUseReportEvent_Error(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	requestUUID := uuid.New()
	toolCall := uuid.New()
	
	// 먼저 Call 상태로 ActiveTools에 추가
	toolModel := viewinterface.NewToolModel("Read (/test/file.txt)")
	model.ActiveTools[toolCall] = toolModel
	
	// Error 이벤트 데이터
	errorData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Error,
	}
	
	event := events.Event{
		Type:      events.ToolUseReportEvent,
		Data:      errorData,
		Timestamp: time.Now(),
		Source:    types.ToolService,
	}

	// When
	model.HandleEvent(event)

	// Then
	assert.NotContains(t, model.ActiveTools, toolCall, "Error 시 ActiveTools에서 제거되어야 함")
}

func TestMainModel_Update_ToolStatusUpdate(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	// ActiveTools에 데이터 추가
	toolCall := uuid.New()
	toolModel := viewinterface.NewToolModel("Read (/test/file.txt)")
	model.ActiveTools[toolCall] = toolModel
	
	updateMsg := viewinterface.UpdateStatus{NewStauts: types.Success}

	// When
	_, cmd := toolModel.Update(updateMsg)

	// Then
	// ToolStatusUpdate 처리가 정상적으로 되는지 확인 (패닉 없이 완료)
	assert.NotNil(t, cmd, "UpdateStatus 처리 시 커맨드를 반환해야 함")
}

func TestMainModel_View_Integration(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	// ActiveTools에 데이터 추가
	toolCall := uuid.New()
	toolModel := viewinterface.NewToolModel("Read (/test/file.txt)")
	model.ActiveTools[toolCall] = toolModel

	// When
	view := model.View()

	// Then
	assert.NotEmpty(t, view, "View는 빈 문자열이 아니어야 함")
	// ActiveTools가 포함된 View가 올바르게 렌더링되는지 확인
	assert.Contains(t, view, "Read (/test/file.txt)", "View에 도구 정보가 포함되어야 함")
}

// Benchmark 테스트 - View 성능 확인
func BenchmarkMainModel_View(b *testing.B) {
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	// 여러 개의 ActiveTools 추가
	for i := 0; i < 10; i++ {
		toolCall := uuid.New()
		toolModel := viewinterface.NewToolModel("Read (/test/file.txt)")
		model.ActiveTools[toolCall] = toolModel
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.View()
	}
}