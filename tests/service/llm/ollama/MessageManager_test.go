package ollama

import (
	"testing"

	"github.com/ollama/ollama/api"
	"github.com/stretchr/testify/assert"

	"DevCode/src/service/llm/ollama"
)

func TestNewMessageManager(t *testing.T) {
	manager := ollama.NewMessageManager()

	assert.NotNil(t, manager)
	
	messages := manager.GetMessages()
	// 새로 만든 매니저는 빈 환경 메시지가 하나 있을 수 있습니다
	assert.LessOrEqual(t, len(messages), 1)
}

func TestMessageManager_AddSystemMessage(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	systemContent := "시스템 프롬프트입니다."
	manager.AddSystemMessage(systemContent)
	
	messages := manager.GetMessages()
	assert.Len(t, messages, 2) // system message + empty environment message
	assert.Equal(t, "system", messages[0].Role)
	assert.Equal(t, systemContent, messages[0].Content)
}

func TestMessageManager_SetEnvironmentMessage(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	envContent := "Working directory: /home/test"
	manager.SetEnvironmentMessage(envContent)
	
	messages := manager.GetMessages()
	assert.Len(t, messages, 1) // only environment message (system messages empty)
	assert.Equal(t, "system", messages[0].Role)
	assert.Contains(t, messages[0].Content, "Here is useful information about the environment")
	assert.Contains(t, messages[0].Content, envContent)
}

func TestMessageManager_AddUserMessage(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	userMessage := "안녕하세요!"
	manager.AddUserMessage(userMessage)
	
	messages := manager.GetMessages()
	assert.Len(t, messages, 2) // environment message + user message
	
	// Find user message
	var userMsg *api.Message
	for _, msg := range messages {
		if msg.Role == "User" {
			userMsg = &msg
			break
		}
	}
	
	assert.NotNil(t, userMsg)
	assert.Equal(t, userMessage, userMsg.Content)
}

func TestMessageManager_AddAssistantMessage(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	assistantMessage := "안녕하세요! 도움이 필요하신가요?"
	manager.AddAssistantMessage(assistantMessage)
	
	messages := manager.GetMessages()
	
	// Find assistant message
	var assistantMsg *api.Message
	for _, msg := range messages {
		if msg.Role == "assistant" {
			assistantMsg = &msg
			break
		}
	}
	
	assert.NotNil(t, assistantMsg)
	assert.Equal(t, assistantMessage, assistantMsg.Content)
}

func TestMessageManager_AddToolMessage(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	toolMessage := "도구 실행 완료: 파일을 성공적으로 읽었습니다."
	manager.AddToolMessage(toolMessage)
	
	messages := manager.GetMessages()
	
	// Find tool message
	var toolMsg *api.Message
	for _, msg := range messages {
		if msg.Role == "tool" {
			toolMsg = &msg
			break
		}
	}
	
	assert.NotNil(t, toolMsg)
	assert.Equal(t, toolMessage, toolMsg.Content)
}

func TestMessageManager_Clear(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	manager.AddUserMessage("테스트 메시지")
	manager.AddAssistantMessage("응답 메시지")
	
	// Clear 전에 메시지가 있는지 확인
	messages := manager.GetMessages()
	assert.Greater(t, len(messages), 1)
	
	manager.Clear()
	
	// Clear 후에는 시스템 메시지와 환경 메시지만 남아야 함
	messages = manager.GetMessages()
	assert.LessOrEqual(t, len(messages), 1) // environment message가 있을 수 있음
	
	// 환경 메시지가 있다면 시스템 메시지여야 함
	for _, msg := range messages {
		if msg.Content != "" { // 빈 메시지가 아닌 경우에만 확인
			assert.Equal(t, "system", msg.Role)
		}
	}
}

func TestMessageManager_MessageOrder(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	manager.AddSystemMessage("시스템 메시지")
	manager.SetEnvironmentMessage("환경 정보")
	manager.AddUserMessage("사용자 메시지")
	manager.AddAssistantMessage("어시스턴트 메시지")
	manager.AddToolMessage("도구 메시지")
	
	messages := manager.GetMessages()
	
	// 메시지 순서 확인: System -> Environment -> User -> Assistant -> Tool
	assert.Equal(t, "system", messages[0].Role) // System message
	assert.Equal(t, "system", messages[1].Role) // Environment message
	assert.Equal(t, "User", messages[2].Role)   // User message
	assert.Equal(t, "assistant", messages[3].Role) // Assistant message
	assert.Equal(t, "tool", messages[4].Role)   // Tool message
}

func TestMessageManager_MessageLimit(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	// MessageLimit을 초과하는 메시지 추가
	for i := 0; i <= 105; i++ { // MessageLimit = 100
		manager.AddUserMessage("메시지 " + string(rune('0'+i%10)))
	}
	
	messages := manager.GetMessages()
	
	// 시스템 메시지 + 환경 메시지 + 최대 100개의 일반 메시지
	assert.LessOrEqual(t, len(messages), 102) // system + environment + 100 messages
}

func TestMessageManager_ConcurrentAccess(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	done := make(chan bool, 2)
	
	// 동시에 메시지 추가
	go func() {
		for i := 0; i < 10; i++ {
			manager.AddUserMessage("사용자 메시지")
		}
		done <- true
	}()
	
	go func() {
		for i := 0; i < 10; i++ {
			manager.AddAssistantMessage("어시스턴트 메시지")
		}
		done <- true
	}()
	
	// 두 고루틴이 완료될 때까지 대기
	<-done
	<-done
	
	messages := manager.GetMessages()
	
	// 메시지가 정상적으로 추가되었는지 확인 (race condition 없이)
	assert.GreaterOrEqual(t, len(messages), 20) // 최소 20개의 메시지
}

func TestMessageManager_EmptyMessages(t *testing.T) {
	manager := ollama.NewMessageManager()
	
	manager.AddSystemMessage("")
	manager.AddUserMessage("")
	manager.AddAssistantMessage("")
	manager.AddToolMessage("")
	
	messages := manager.GetMessages()
	
	// 빈 메시지도 정상적으로 추가되는지 확인
	assert.GreaterOrEqual(t, len(messages), 4)
	
	// 각 역할의 메시지가 존재하는지 확인
	roles := make(map[string]bool)
	for _, msg := range messages {
		roles[msg.Role] = true
	}
	
	assert.True(t, roles["system"])
	assert.True(t, roles["User"])
	assert.True(t, roles["assistant"])
	assert.True(t, roles["tool"])
}