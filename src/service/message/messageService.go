package message

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"time"
)

func NewMessageService(bus *events.EventBus) *MessageService {
	service := &MessageService{bus: bus}
	service.Subscribe()
	return service
}

type MessageService struct {
	bus *events.EventBus
}

func (instance *MessageService) Subscribe() {
	instance.bus.StreamStartEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamStartData]) {
		instance.bus.StreamChunkParsedEvent.Publish(events.Event[dto.ParsedChunkData]{
			Data: dto.ParsedChunkData{
				RequestUUID: event.Data.RequestUUID,
				Content:     "",
				IsComplete:  false,
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageService,
		})
	})
	instance.bus.StreamChunkEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamChunkData]) {
		instance.ParsingMessage(event.Data)
	})
	instance.bus.StreamErrorEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamErrorData]) {
		instance.bus.StreamChunkParsedErrorEvent.Publish(events.Event[dto.ParsedChunkErrorData]{
			Data: dto.ParsedChunkErrorData{
				RequestUUID: event.Data.RequestUUID,
				Error:       event.Data.Error.Error(),
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageService,
		})
	})
	instance.bus.StreamCompleteEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamCompleteData]) {
		instance.bus.StreamChunkParsedEvent.Publish(events.Event[dto.ParsedChunkData]{
			Data: dto.ParsedChunkData{
				RequestUUID: event.Data.RequestUUID,
				Content:     event.Data.FinalMessage,
				IsComplete:  event.Data.IsComplete,
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageService,
		})
	})
}

func (instance *MessageService) GetID() constants.Source {
	return constants.MessageService
}

func (instance *MessageService) ParsingMessage(data dto.StreamChunkData) {
	instance.bus.StreamChunkParsedEvent.Publish(events.Event[dto.ParsedChunkData]{
		Data: dto.ParsedChunkData{
			RequestUUID: data.RequestUUID,
			Content:     data.Content,
			IsComplete:  false,
		},
		TimeStamp: time.Now(),
		Source:    constants.MessageService,
	})
}
