package viewinterface

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"DevCode/src/viewinterface"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMainModel_ToolErrorDisplaysRedLight(t *testing.T) {
	// Given: MainModel과 EventBus 초기화
	bus, err := events.NewEventBus()
	assert.NoError(t, err, "EventBus 생성 실패")
	mainModel := viewinterface.NewMainModel(bus)
	
	// Tool 호출 UUID 생성
	toolCallID := uuid.New()
	toolInfo := "테스트 툴"
	
	// When: Tool Call 이벤트 발생 (Tool이 활성화됨)
	callReportData := dto.ToolUseReportData{
		ToolCall:   toolCallID,
		ToolStatus: constants.Call,
		ToolInfo:   toolInfo,
	}
	
	// Tool Call 업데이트 처리
	_, _ = mainModel.Update(callReportData)
	
	// Tool이 ActiveTools에 추가되었는지 확인
	assert.Contains(t, mainModel.ActiveTools, toolCallID, "Tool이 ActiveTools에 추가되어야 함")
	
	// When: Tool Error 이벤트 발생
	errorReportData := dto.ToolUseReportData{
		ToolCall:   toolCallID,
		ToolStatus: constants.Error,
		ToolInfo:   toolInfo,
	}
	
	// Tool Error 업데이트 처리
	updatedModel, _ := mainModel.Update(errorReportData)
	mainModel = updatedModel.(*viewinterface.MainModel)
	
	// Then: Tool이 ActiveTools에서 제거되고 AssistantMessage에 추가되어야 함
	assert.NotContains(t, mainModel.ActiveTools, toolCallID, "Tool Error 후 ActiveTools에서 제거되어야 함")
	assert.NotEmpty(t, mainModel.AssistantMessage, "AssistantMessage에 tool 결과가 추가되어야 함")
	
	// 빨간색 StatusLight가 포함되어 있는지 확인
	expectedRedLight := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9)).Render("●")
	assert.Contains(t, mainModel.AssistantMessage, expectedRedLight, "빨간색 StatusLight가 포함되어야 함")
}

func TestMainModel_ToolErrorViaEventBus(t *testing.T) {
	// Given: MainModel과 EventBus 초기화
	bus, err := events.NewEventBus()
	assert.NoError(t, err, "EventBus 생성 실패")
	mainModel := viewinterface.NewMainModel(bus)
	
	toolCallID := uuid.New()
	toolInfo := "이벤트 테스트 툴"
	
	// Given: Tool Call 이벤트 직접 처리
	callReportData := dto.ToolUseReportData{
		ToolCall:   toolCallID,
		ToolStatus: constants.Call,
		ToolInfo:   toolInfo,
	}
	
	// Tool Call 업데이트 처리
	_, _ = mainModel.Update(callReportData)
	
	// Tool이 ActiveTools에 추가되었는지 확인
	assert.Contains(t, mainModel.ActiveTools, toolCallID, "Tool이 ActiveTools에 추가되어야 함")
	
	// When: Tool Error 이벤트 직접 처리
	errorReportData := dto.ToolUseReportData{
		ToolCall:   toolCallID,
		ToolStatus: constants.Error,
		ToolInfo:   toolInfo,
	}
	
	// Tool Error 업데이트 처리
	updatedModel, _ := mainModel.Update(errorReportData)
	mainModel = updatedModel.(*viewinterface.MainModel)
	
	// Then: Tool Error가 처리되어 빨간불이 표시되어야 함
	expectedRedLight := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(9)).Render("●")
	assert.Contains(t, mainModel.AssistantMessage, expectedRedLight, "Tool Error 처리 시 빨간색 불이 표시되어야 함")
}

func TestMainModel_ToolSuccessDisplaysGreenLight(t *testing.T) {
	// Given: MainModel과 EventBus 초기화  
	bus, err := events.NewEventBus()
	assert.NoError(t, err, "EventBus 생성 실패")
	mainModel := viewinterface.NewMainModel(bus)
	
	toolCallID := uuid.New()
	toolInfo := "성공 테스트 툴"
	
	// Tool Call 이벤트 처리
	callReportData := dto.ToolUseReportData{
		ToolCall:   toolCallID,
		ToolStatus: constants.Call,
		ToolInfo:   toolInfo,
	}
	_, _ = mainModel.Update(callReportData)
	
	// When: Tool Success 이벤트 발생
	successReportData := dto.ToolUseReportData{
		ToolCall:   toolCallID,
		ToolStatus: constants.Success,
		ToolInfo:   toolInfo,
	}
	
	updatedModel, _ := mainModel.Update(successReportData)
	mainModel = updatedModel.(*viewinterface.MainModel)
	
	// Then: 초록색 StatusLight가 포함되어 있는지 확인
	expectedGreenLight := lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(10)).Render("●")
	assert.Contains(t, mainModel.AssistantMessage, expectedGreenLight, "초록색 StatusLight가 포함되어야 함")
}