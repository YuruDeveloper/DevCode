package service

import (
	"UniCode/src/events"
	"UniCode/src/types"

	"github.com/ollama/ollama/api"
)

func NewMessageService(bus *events.EventBus) *MessageService{
	service := &MessageService { Bus: bus }
	bus.Subscribe(events.StreamStartEvent,service)
	return service
}

type MessageService struct {
	Bus *events.EventBus
}

func (instance *MessageService) HandleEvent(event events.Event) {
	switch event.Type {
		case events.StreamStartEvent:
			
	}
}

func (instance *MessageService) GetID() types.Source {
	return types.MessageService
}

func (instance *MessageService) ParingMessage(message api.Message) {
}