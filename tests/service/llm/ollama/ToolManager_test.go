package ollama

import (
	"testing"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"

	"DevCode/src/service/llm/ollama"
)

func TestNewToolManager(t *testing.T) {
	manager := ollama.NewToolManager()

	assert.NotNil(t, manager)
	assert.Empty(t, manager.GetToolList())
}

func TestToolManager_RegisterToolList(t *testing.T) {
	manager := ollama.NewToolManager()
	
	tools := []*mcp.Tool{
		{
			Name:        "calculator",
			Description: "계산 도구",
		},
		{
			Name:        "file_reader", 
			Description: "파일 읽기 도구",
		},
	}
	
	manager.RegisterToolList(tools)
	
	toolList := manager.GetToolList()
	assert.Len(t, toolList, 2)
	assert.Equal(t, "calculator", toolList[0].Function.Name)
	assert.Equal(t, "file_reader", toolList[1].Function.Name)
}

func TestToolManager_RegisterToolList_WithNilTool(t *testing.T) {
	manager := ollama.NewToolManager()
	
	tools := []*mcp.Tool{
		{
			Name:        "valid_tool",
			Description: "유효한 도구",
		},
		nil, // nil 도구
		{
			Name:        "another_tool",
			Description: "또 다른 도구", 
		},
	}
	
	manager.RegisterToolList(tools)
	
	toolList := manager.GetToolList()
	assert.Len(t, toolList, 2) // nil 도구는 제외되어야 함
	assert.Equal(t, "valid_tool", toolList[0].Function.Name)
	assert.Equal(t, "another_tool", toolList[1].Function.Name)
}

func TestToolManager_RegisterToolList_EmptyList(t *testing.T) {
	manager := ollama.NewToolManager()
	
	manager.RegisterToolList([]*mcp.Tool{})
	
	toolList := manager.GetToolList()
	assert.Empty(t, toolList)
}

func TestToolManager_RegisterToolCall(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	toolCallUUID := uuid.New()
	toolName := "test_calculator"
	
	manager.RegisterToolCall(requestUUID, toolCallUUID, toolName)
	
	assert.True(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.True(t, manager.HasPendingCalls(requestUUID))
}

func TestToolManager_HasToolCall_NonExistentCall(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	toolCallUUID := uuid.New()
	
	assert.False(t, manager.HasToolCall(requestUUID, toolCallUUID))
}

func TestToolManager_CompleteToolCall(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	toolCallUUID := uuid.New()
	toolName := "test_tool"
	
	// 도구 호출 등록
	manager.RegisterToolCall(requestUUID, toolCallUUID, toolName)
	assert.True(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.True(t, manager.HasPendingCalls(requestUUID))
	
	// 도구 호출 완료
	manager.CompleteToolCall(requestUUID, toolCallUUID)
	assert.False(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.False(t, manager.HasPendingCalls(requestUUID))
}

func TestToolManager_CompleteToolCall_NonExistentCall(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	toolCallUUID := uuid.New()
	
	// 존재하지 않는 호출을 완료해도 오류가 발생하지 않아야 함
	manager.CompleteToolCall(requestUUID, toolCallUUID)
	
	assert.False(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.False(t, manager.HasPendingCalls(requestUUID))
}

func TestToolManager_HasPendingCalls_MultipleCalls(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	toolCall1 := uuid.New()
	toolCall2 := uuid.New()
	
	// 두 개의 도구 호출 등록
	manager.RegisterToolCall(requestUUID, toolCall1, "tool1")
	manager.RegisterToolCall(requestUUID, toolCall2, "tool2")
	
	assert.True(t, manager.HasPendingCalls(requestUUID))
	assert.True(t, manager.HasToolCall(requestUUID, toolCall1))
	assert.True(t, manager.HasToolCall(requestUUID, toolCall2))
	
	// 첫 번째 도구 호출 완료
	manager.CompleteToolCall(requestUUID, toolCall1)
	assert.True(t, manager.HasPendingCalls(requestUUID)) // 아직 두 번째가 남음
	assert.False(t, manager.HasToolCall(requestUUID, toolCall1))
	assert.True(t, manager.HasToolCall(requestUUID, toolCall2))
	
	// 두 번째 도구 호출 완료  
	manager.CompleteToolCall(requestUUID, toolCall2)
	assert.False(t, manager.HasPendingCalls(requestUUID)) // 모든 호출 완료
	assert.False(t, manager.HasToolCall(requestUUID, toolCall1))
	assert.False(t, manager.HasToolCall(requestUUID, toolCall2))
}

func TestToolManager_ClearRequest(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	toolCall1 := uuid.New()
	toolCall2 := uuid.New()
	
	// 두 개의 도구 호출 등록
	manager.RegisterToolCall(requestUUID, toolCall1, "tool1")
	manager.RegisterToolCall(requestUUID, toolCall2, "tool2")
	
	assert.True(t, manager.HasPendingCalls(requestUUID))
	
	// 요청 클리어
	manager.ClearRequest(requestUUID)
	
	assert.False(t, manager.HasPendingCalls(requestUUID))
	assert.False(t, manager.HasToolCall(requestUUID, toolCall1))
	assert.False(t, manager.HasToolCall(requestUUID, toolCall2))
}

func TestToolManager_ClearRequest_NonExistentRequest(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	
	// 존재하지 않는 요청을 클리어해도 오류가 발생하지 않아야 함
	manager.ClearRequest(requestUUID)
	
	assert.False(t, manager.HasPendingCalls(requestUUID))
}

func TestToolManager_MultipleRequests(t *testing.T) {
	manager := ollama.NewToolManager()
	
	request1 := uuid.New()
	request2 := uuid.New()
	toolCall1 := uuid.New()
	toolCall2 := uuid.New()
	
	// 서로 다른 요청에 도구 호출 등록
	manager.RegisterToolCall(request1, toolCall1, "tool1")
	manager.RegisterToolCall(request2, toolCall2, "tool2")
	
	assert.True(t, manager.HasPendingCalls(request1))
	assert.True(t, manager.HasPendingCalls(request2))
	assert.True(t, manager.HasToolCall(request1, toolCall1))
	assert.True(t, manager.HasToolCall(request2, toolCall2))
	
	// 첫 번째 요청의 도구 호출만 완료
	manager.CompleteToolCall(request1, toolCall1)
	
	assert.False(t, manager.HasPendingCalls(request1))
	assert.True(t, manager.HasPendingCalls(request2))
	assert.False(t, manager.HasToolCall(request1, toolCall1))
	assert.True(t, manager.HasToolCall(request2, toolCall2))
}

func TestToolManager_ConcurrentAccess(t *testing.T) {
	manager := ollama.NewToolManager()
	
	done := make(chan bool, 2)
	request1 := uuid.New()
	request2 := uuid.New()
	
	// 동시에 도구 호출 등록
	go func() {
		for i := 0; i < 10; i++ {
			toolCallUUID := uuid.New()
			manager.RegisterToolCall(request1, toolCallUUID, "concurrent_tool_1")
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 10; i++ {
			toolCallUUID := uuid.New()
			manager.RegisterToolCall(request2, toolCallUUID, "concurrent_tool_2")
		}
		done <- true
	}()
	
	// 두 고루틴이 완료될 때까지 대기
	<-done
	<-done
	
	// race condition 없이 정상적으로 등록되었는지 확인
	assert.True(t, manager.HasPendingCalls(request1))
	assert.True(t, manager.HasPendingCalls(request2))
}

func TestToolManager_RegisterToolList_ReplaceExisting(t *testing.T) {
	manager := ollama.NewToolManager()
	
	// 첫 번째 도구 리스트 등록
	firstTools := []*mcp.Tool{
		{Name: "old_tool", Description: "오래된 도구"},
	}
	manager.RegisterToolList(firstTools)
	
	toolList := manager.GetToolList()
	assert.Len(t, toolList, 1)
	assert.Equal(t, "old_tool", toolList[0].Function.Name)
	
	// 새로운 도구 리스트로 교체
	newTools := []*mcp.Tool{
		{Name: "new_tool_1", Description: "새 도구 1"},
		{Name: "new_tool_2", Description: "새 도구 2"},
	}
	manager.RegisterToolList(newTools)
	
	toolList = manager.GetToolList()
	assert.Len(t, toolList, 2) // 이전 도구는 교체됨
	assert.Equal(t, "new_tool_1", toolList[0].Function.Name)
	assert.Equal(t, "new_tool_2", toolList[1].Function.Name)
}

func TestToolManager_ToolCallLifecycle(t *testing.T) {
	manager := ollama.NewToolManager()
	
	requestUUID := uuid.New()
	toolCallUUID := uuid.New()
	toolName := "lifecycle_test_tool"
	
	// 1. 초기 상태 확인
	assert.False(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.False(t, manager.HasPendingCalls(requestUUID))
	
	// 2. 도구 호출 등록
	manager.RegisterToolCall(requestUUID, toolCallUUID, toolName)
	assert.True(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.True(t, manager.HasPendingCalls(requestUUID))
	
	// 3. 도구 호출 완료
	manager.CompleteToolCall(requestUUID, toolCallUUID)
	assert.False(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.False(t, manager.HasPendingCalls(requestUUID))
	
	// 4. 요청 클리어 (이미 완료된 상태에서)
	manager.ClearRequest(requestUUID)
	assert.False(t, manager.HasToolCall(requestUUID, toolCallUUID))
	assert.False(t, manager.HasPendingCalls(requestUUID))
}