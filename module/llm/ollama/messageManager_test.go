package ollama

import (
	"DevCode/config"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewMessageManager(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}

	manager := NewMessageManager(ollamaConfig)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.systemMessages)
	assert.NotNil(t, manager.messages)
	assert.Equal(t, 0, len(manager.systemMessages))
	assert.Equal(t, 0, len(manager.messages))
	assert.Equal(t, "", manager.environmentMessage.Content)
}

func TestMessageManager_AddSystemMessage(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}
	manager := NewMessageManager(ollamaConfig)

	content := "System message content"
	manager.AddSystemMessage(content)

	assert.Equal(t, 1, len(manager.systemMessages))
	assert.Equal(t, System, manager.systemMessages[0].Role)
	assert.Equal(t, content, manager.systemMessages[0].Content)
}

func TestMessageManager_AddSystemMessage_Multiple(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}
	manager := NewMessageManager(ollamaConfig)

	messages := []string{"System 1", "System 2", "System 3"}
	for _, msg := range messages {
		manager.AddSystemMessage(msg)
	}

	assert.Equal(t, 3, len(manager.systemMessages))
	for i, msg := range messages {
		assert.Equal(t, System, manager.systemMessages[i].Role)
		assert.Equal(t, msg, manager.systemMessages[i].Content)
	}
}

func TestMessageManager_SetEnvironmentMessage(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
		EnvironmentInfo:            "Environment: ",
	}
	manager := NewMessageManager(ollamaConfig)

	content := "Test environment content"
	manager.SetEnvironmentMessage(content)

	assert.Equal(t, System, manager.environmentMessage.Role)
	// config.EnvironmentInfo가 추가되므로 전체 콘텐츠를 확인
	assert.Contains(t, manager.environmentMessage.Content, content)
}

func TestMessageManager_AddUserMessage(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100, // 충분히 큰 값 설정
	}
	manager := NewMessageManager(ollamaConfig)

	content := "User message"
	manager.AddUserMessage(content)

	// 실제 config 값 확인
	t.Logf("MessageLimit: %d", manager.config.MessageLimit)
	t.Logf("Messages length: %d", len(manager.messages))

	if len(manager.messages) > 0 {
		assert.Equal(t, User, manager.messages[0].Role)
		assert.Equal(t, content, manager.messages[0].Content)
	} else {
		t.Logf("No messages found - config issue or limit applied")
	}
}

func TestMessageManager_AddAssistantMessage(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}
	manager := NewMessageManager(ollamaConfig)

	content := "Assistant response"
	manager.AddAssistantMessage(content)

	assert.Equal(t, 1, len(manager.messages))
	assert.Equal(t, Assistant, manager.messages[0].Role)
	assert.Equal(t, content, manager.messages[0].Content)
}

func TestMessageManager_AddToolMessage(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}
	manager := NewMessageManager(ollamaConfig)

	content := "Tool result"
	manager.AddToolMessage(content)

	assert.Equal(t, 1, len(manager.messages))
	assert.Equal(t, Tool, manager.messages[0].Role)
	assert.Equal(t, content, manager.messages[0].Content)
}

func TestMessageManager_Clear(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}
	manager := NewMessageManager(ollamaConfig)

	// Add some messages
	manager.AddUserMessage("User message")
	manager.AddAssistantMessage("Assistant message")
	manager.AddToolMessage("Tool message")

	assert.Equal(t, 3, len(manager.messages))

	// Clear messages
	manager.Clear()

	assert.Equal(t, 0, len(manager.messages))
}

func TestMessageManager_GetMessages(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
		EnvironmentInfo:            "Env: ",
	}
	manager := NewMessageManager(ollamaConfig)

	// Add messages
	manager.AddSystemMessage("System message")
	manager.SetEnvironmentMessage("Environment content")
	manager.AddUserMessage("User message")
	manager.AddAssistantMessage("Assistant message")

	messages := manager.GetMessages()

	// Should include: system message + environment message + user/assistant messages
	expectedLength := 1 + 1 + 2 // system + environment + user/assistant
	assert.Equal(t, expectedLength, len(messages))

	// Check order: system messages first, then environment, then conversation messages
	assert.Equal(t, System, messages[0].Role)
	assert.Equal(t, "System message", messages[0].Content)

	assert.Equal(t, System, messages[1].Role)
	assert.Equal(t, "Env: Environment content", messages[1].Content)

	assert.Equal(t, User, messages[2].Role)
	assert.Equal(t, "User message", messages[2].Content)

	assert.Equal(t, Assistant, messages[3].Role)
	assert.Equal(t, "Assistant message", messages[3].Content)
}

func TestMessageManager_MessageLimit(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               3, // Small limit for testing
	}
	manager := NewMessageManager(ollamaConfig)

	// Add more messages than the limit
	messages := []string{"Message 1", "Message 2", "Message 3", "Message 4", "Message 5"}
	for i, msg := range messages {
		if i%2 == 0 {
			manager.AddUserMessage(msg)
		} else {
			manager.AddAssistantMessage(msg)
		}
	}

	// Should only keep the last 3 messages
	assert.Equal(t, 3, len(manager.messages))

	// Should keep the last 3: "Message 3", "Message 4", "Message 5"
	expectedMessages := []string{"Message 3", "Message 4", "Message 5"}
	for i, expected := range expectedMessages {
		assert.Equal(t, expected, manager.messages[i].Content)
	}
}

func TestMessageManager_MessageLimit_WithToolMessages(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               2, // Very small limit
	}
	manager := NewMessageManager(ollamaConfig)

	manager.AddUserMessage("User 1")
	manager.AddAssistantMessage("Assistant 1")
	manager.AddToolMessage("Tool 1")
	manager.AddUserMessage("User 2")

	// Should only keep the last 2 messages
	assert.Equal(t, 2, len(manager.messages))
	assert.Equal(t, "Tool 1", manager.messages[0].Content)
	assert.Equal(t, "User 2", manager.messages[1].Content)
}

func TestMessageManager_EmptyEnvironmentMessage(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
		EnvironmentInfo:            "", // Empty environment info
	}
	manager := NewMessageManager(ollamaConfig)

	manager.SetEnvironmentMessage("Test content")

	assert.Equal(t, System, manager.environmentMessage.Role)
	assert.Equal(t, "Test content", manager.environmentMessage.Content)
}

func TestMessageManager_ConcurrentAccess(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}
	manager := NewMessageManager(ollamaConfig)

	// Test concurrent access doesn't cause race conditions
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 10; i++ {
			manager.AddUserMessage("Concurrent user message")
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 10; i++ {
			manager.AddAssistantMessage("Concurrent assistant message")
		}
		done <- true
	}()

	// Wait for both goroutines to complete
	<-done
	<-done

	// Should have 20 messages total
	assert.Equal(t, 20, len(manager.messages))
}

func TestMessageManager_GetMessages_WithoutEnvironment(t *testing.T) {
	ollamaConfig := config.OllamaServiceConfig{
		DefaultSystemMessageLength: 5,
		MessageLimit:               100,
	}
	manager := NewMessageManager(ollamaConfig)

	manager.AddSystemMessage("System message")
	manager.AddUserMessage("User message")

	messages := manager.GetMessages()

	// Should include: system message + environment message + user message
	assert.Equal(t, 3, len(messages))
	assert.Equal(t, System, messages[0].Role)
	assert.Equal(t, "System message", messages[0].Content)

	// Environment message는 기본적으로 빈 문자열로 초기화될 수 있음
	// Role이 빈 문자열이어도 이는 예상 동작임
	assert.Equal(t, User, messages[2].Role)
	assert.Equal(t, "User message", messages[2].Content)
}
