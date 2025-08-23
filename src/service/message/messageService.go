package message

import (
	"UniCode/src/constants"
	"UniCode/src/dto"
	"UniCode/src/events"
	"UniCode/src/service"
	"go.uber.org/zap"
)

func NewMessageService(bus events.Bus, logger *zap.Logger) *MessageService {
	service := &MessageService{bus: bus, logger: logger}
	bus.Subscribe(events.StreamStartEvent, service)
	bus.Subscribe(events.StreamChunkEvent, service)
	bus.Subscribe(events.StreamCompleteEvent, service)
	bus.Subscribe(events.StreamErrorEvent, service)
	return service
}

type MessageService struct {
	bus    events.Bus
	logger *zap.Logger
}

func (instance *MessageService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.StreamStartEvent:
		service.PublishEvent(instance.bus, events.StreamChunkParsedEvent, dto.ParsedChunkData{
			RequestUUID: event.Data.(dto.StreamStartData).RequestUUID,
			Content:     "",
			IsComplete:  false,
		}, constants.MessageService)
	case events.StreamChunkEvent:
		instance.ParsingMessage(event.Data.(dto.StreamChunkData))
	case events.StreamErrorEvent:
		service.PublishEvent(instance.bus, events.StreamChunkParsedErrorEvent, dto.ParsedChunkErrorData{
			RequestUUID: event.Data.(dto.StreamErrorData).RequestUUID,
			Error:       event.Data.(dto.StreamErrorData).Error.Error(),
		}, constants.MessageService)
	case events.StreamCompleteEvent:
		service.PublishEvent(instance.bus, events.StreamChunkParsedEvent, dto.ParsedChunkData{
			RequestUUID: event.Data.(dto.StreamCompleteData).RequestUUID,
			Content:     event.Data.(dto.StreamCompleteData).FinalMessage,
			IsComplete:  event.Data.(dto.StreamCompleteData).IsComplete,
		}, constants.MessageService)
	}
}

func (instance *MessageService) GetID() constants.Source {
	return constants.MessageService
}

func (instance *MessageService) ParsingMessage(data dto.StreamChunkData) {
	service.PublishEvent(instance.bus, events.StreamChunkParsedEvent, dto.ParsedChunkData{
		RequestUUID: data.RequestUUID,
		Content:     data.Content,
		IsComplete:  false,
	}, constants.MessageService)
}
