package ollama

import (
	"sync"

	"github.com/ollama/ollama/api"
)

const (
	MessageLimit = 100
	DefaultSystemMessageLength = 10
)

const (
	Assistant = "assistant"
	Tool = "tool"
	System = "system"
	User = "User"
	EnvironmentInfo = "Here is useful information about the environment you are running in:\n"
)

func NewMessageManager() *MessageManager{
	return &MessageManager{
		systemMessages: make([]api.Message,0,DefaultSystemMessageLength),
		environmentMessage: api.Message{},
		messages: make([]api.Message, 0,MessageLimit + 1),
	}
}

type MessageManager struct {
	systemMessages []api.Message
	environmentMessage api.Message
	messages []api.Message
	messageMutex sync.RWMutex
}

func (instance *MessageManager) AddSystemMessage(content string) {
	instance.messageMutex.Lock()
	defer instance.messageMutex.Unlock()
	instance.systemMessages = append(instance.systemMessages, api.Message{
		Role: System,
		Content: content,
	})

}

func (instance *MessageManager) SetEnvironmentMessage(content string) {
	instance.messageMutex.Lock()
	defer instance.messageMutex.Unlock()
	instance.environmentMessage = api.Message{
		Role: System,
		Content: EnvironmentInfo + content,
	}
}

func (instance *MessageManager) AddUserMessage(content string) {
	instance.messageMutex.Lock()
	defer instance.messageMutex.Unlock()
	instance.messages = append(instance.messages, api.Message{
		Role: User,
		Content: content,
	})
	instance.checkMessageLimit()
}

func (instance *MessageManager) AddAssistantMessage(content string) {
	instance.messageMutex.Lock()
	defer instance.messageMutex.Unlock()
	instance.messages = append(instance.messages, api.Message{
		Role: Assistant,
		Content: content,
	})
	instance.checkMessageLimit()
}

func (instance *MessageManager) AddToolMessage(content string) {
	instance.messageMutex.Lock()
	defer instance.messageMutex.Unlock()
	instance.messages = append(instance.messages, api.Message{
		Role: Tool,
		Content: content,
	})
	instance.checkMessageLimit()
}

func (instance *MessageManager) Clear() {
	instance.messageMutex.Lock()
	defer instance.messageMutex.Unlock()
	instance.messages = instance.messages[:0]
}

func (instance *MessageManager) GetMessages() []api.Message {
	instance.messageMutex.RLock()
	defer instance.messageMutex.RUnlock()
	return append(instance.systemMessages,append([]api.Message{instance.environmentMessage},instance.messages...)...)
}

func (instance *MessageManager) checkMessageLimit() {
	if len(instance.messages) > MessageLimit {
		instance.messages = instance.messages[:MessageLimit]
	}
}