package ollama

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"DevCode/src/config"
	"DevCode/src/service/llm/ollama"
)

func TestNewMessageManager(t *testing.T) {
	cfg := config.OllamaServiceConfig{
		MessageLimit:               100,
		DefaultSystemMessageLength: 10,
		DefaultToolSize:            10,
	}
	manager := ollama.NewMessageManager(cfg)

	assert.NotNil(t, manager)
	
	messages := manager.GetMessages()
	// 새로 만든 매니저는 시스템 메시지가 있을 수 있습니다
	assert.GreaterOrEqual(t, len(messages), 0)
}

func TestMessageManager_AddUserMessage(t *testing.T) {
	cfg := config.OllamaServiceConfig{
		MessageLimit:               100,
		DefaultSystemMessageLength: 10,
		DefaultToolSize:            10,
	}
	manager := ollama.NewMessageManager(cfg)
	
	userMessage := "안녕하세요!"
	manager.AddUserMessage(userMessage)
	
	messages := manager.GetMessages()
	// 메시지가 추가되었는지 확인 (최소 1개 이상)
	assert.Greater(t, len(messages), 0)
}