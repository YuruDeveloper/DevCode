package service

import (
	"UniCode/src/events"
	"UniCode/src/types"

	"github.com/ollama/ollama/api"
)

func NewMessageService(bus *events.EventBus) *MessageService{
	service := &MessageService { Bus: bus }
	bus.Subscribe(events.LLMResponseEvent,service)
	return service
}

type MessageService struct {
	Bus *events.EventBus
}

func (instance *MessageService) HandleEvent(event events.Event) {
	if event.Type == events.LLMResponseEvent {
		instance.ParingMessage(event.Data.(types.ResponseData).Message)
	}
}

func (instance *MessageService) GetID() types.Source {
	return types.MessageService
}

func (instance *MessageService) ParingMessage(message api.Message) {
}