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

func TestMainModel_ToolCallView_EmptyActiveTools(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)

	// When
	result := model.ToolCallView()

	// Then
	assert.Empty(t, result, "빈 ActiveTools일 때 빈 문자열을 반환해야 함")
}

func TestMainModel_ToolCallView_WithActiveTools(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	requestUUID := uuid.New()
	toolCall := uuid.New()
	toolData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Call,
	}
	
	// ActiveTools에 수동으로 추가
	key := requestUUID.String() + toolCall.String()
	model.ActiveTools[key] = toolData

	// When
	result := model.ToolCallView()

	// Then
	assert.Contains(t, result, "Read (/test/file.txt)", "ToolInfo가 결과에 포함되어야 함")
	assert.Contains(t, result, "\n", "줄바꿈이 포함되어야 함")
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
	key := requestUUID.String() + toolCall.String()
	assert.Contains(t, model.ActiveTools, key, "ActiveTools에 도구가 추가되어야 함")
	assert.Equal(t, eventData, model.ActiveTools[key], "저장된 데이터가 일치해야 함")
}

func TestMainModel_HandleEvent_ToolUseReportEvent_Success(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	requestUUID := uuid.New()
	toolCall := uuid.New()
	
	// 먼저 Call 상태로 ActiveTools에 추가
	callData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Call,
	}
	key := requestUUID.String() + toolCall.String()
	model.ActiveTools[key] = callData
	
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
	assert.NotContains(t, model.ActiveTools, key, "Success 시 ActiveTools에서 제거되어야 함")
}

func TestMainModel_HandleEvent_ToolUseReportEvent_Error(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	requestUUID := uuid.New()
	toolCall := uuid.New()
	
	// 먼저 Call 상태로 ActiveTools에 추가
	callData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Call,
	}
	key := requestUUID.String() + toolCall.String()
	model.ActiveTools[key] = callData
	
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
	assert.NotContains(t, model.ActiveTools, key, "Error 시 ActiveTools에서 제거되어야 함")
}

func TestMainModel_Update_ToolStatusUpdate(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	// ActiveTools에 데이터 추가
	requestUUID := uuid.New()
	toolCall := uuid.New()
	toolData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Call,
	}
	key := requestUUID.String() + toolCall.String()
	model.ActiveTools[key] = toolData
	
	msg := viewinterface.ToolStatusUpdate{}

	// When
	_, cmd := model.Update(msg)

	// Then
	// ToolStatusUpdate 처리가 정상적으로 되는지 확인 (패닉 없이 완료)
	assert.Nil(t, cmd, "ToolStatusUpdate는 특별한 명령을 반환하지 않아야 함")
}

func TestMainModel_View_Integration(t *testing.T) {
	// Given
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	// ActiveTools에 데이터 추가
	requestUUID := uuid.New()
	toolCall := uuid.New()
	toolData := types.ToolUseReportData{
		RequestUUID: requestUUID,
		ToolCall:    toolCall,
		ToolInfo:    "Read (/test/file.txt)",
		ToolStatus:  types.Call,
	}
	key := requestUUID.String() + toolCall.String()
	model.ActiveTools[key] = toolData

	// When
	view := model.View()

	// Then
	assert.NotEmpty(t, view, "View는 빈 문자열이 아니어야 함")
	// ToolCallView가 포함되어 있는지는 직접 확인하기 어려우므로
	// ToolCallView를 별도로 테스트하여 통합 확인
	toolCallView := model.ToolCallView()
	assert.Contains(t, toolCallView, "Read (/test/file.txt)", "ToolCallView에 도구 정보가 포함되어야 함")
}

// Benchmark 테스트 - ToolCallView 성능 확인
func BenchmarkMainModel_ToolCallView(b *testing.B) {
	bus := events.NewEventBus()
	model := viewinterface.NewMainModel(bus)
	
	// 여러 개의 ActiveTools 추가
	for i := 0; i < 10; i++ {
		requestUUID := uuid.New()
		toolCall := uuid.New()
		toolData := types.ToolUseReportData{
			RequestUUID: requestUUID,
			ToolCall:    toolCall,
			ToolInfo:    "Read (/test/file.txt)",
			ToolStatus:  types.Call,
		}
		key := requestUUID.String() + toolCall.String()
		model.ActiveTools[key] = toolData
	}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		model.ToolCallView()
	}
}