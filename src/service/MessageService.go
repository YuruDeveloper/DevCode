package service

import (
	"UniCode/src/events"
	"UniCode/src/types"
)

func NewMessageService(bus *events.EventBus) *MessageService {
	service := &MessageService{Bus: bus}
	bus.Subscribe(events.StreamStartEvent, service)
	bus.Subscribe(events.StreamChunkEvent, service)
	bus.Subscribe(events.StreamCompleteEvent, service)
	bus.Subscribe(events.StreamErrorEvent, service)
	return service
}

type MessageService struct {
	Bus *events.EventBus
}

func (instance *MessageService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.StreamStartEvent:
		PublishEvent(instance.Bus, events.StreamChunkParsedEvent, types.ParsedChunkData{
			RequestUUID: event.Data.(types.StreamStartData).RequestUUID,
			Content:     "",
			IsComplete:  false,
		}, types.MessageService)
	case events.StreamChunkEvent:
		instance.ParsingMessage(event.Data.(types.StreamChunkData))
	case events.StreamErrorEvent:
		PublishEvent(instance.Bus, events.StreamChunkParsedErrorEvent, types.ParsedChunkErrorData{
			RequestUUID: event.Data.(types.StreamErrorData).RequestUUID,
			Error:       event.Data.(types.StreamErrorData).Error.Error(),
		}, types.MessageService)
	case events.StreamCompleteEvent:
		PublishEvent(instance.Bus, events.StreamChunkParsedEvent, types.ParsedChunkData{
			RequestUUID: event.Data.(types.StreamCompleteData).RequestUUID,
			Content:     event.Data.(types.StreamCompleteData).FinalMessage.Content,
			IsComplete:  event.Data.(types.StreamCompleteData).IsComplete,
		}, types.MessageService)
	}
}

func (instance *MessageService) GetID() types.Source {
	return types.MessageService
}

func (instance *MessageService) ParsingMessage(data types.StreamChunkData) {
	PublishEvent(instance.Bus, events.StreamChunkParsedEvent, types.ParsedChunkData{
		RequestUUID: data.RequestUUID,
		Content:     data.Content,
		IsComplete:  false,
	}, types.MessageService)
}
