package message

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"time"

	"go.uber.org/zap"
)

func NewMessageService(bus *events.EventBus, logger *zap.Logger) *MessageService {
	service := &MessageService{bus: bus, logger: logger}
	service.Subscribe()
	return service
}

type MessageService struct {
	bus    *events.EventBus
	logger *zap.Logger
}

func (instance *MessageService) Subscribe() {
	instance.bus.StreamStartEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamStartData]) {
		instance.bus.StreamChunkParsedEvent.Publish(events.Event[dto.ParsedChunkData]{
			Data: dto.ParsedChunkData{
				RequestID:  event.Data.RequestID,
				Content:    "",
				IsComplete: false,
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageService,
		})
	})
	instance.bus.StreamChunkEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamChunkData]) {
		instance.ParsingMessage(event.Data)
	})
	instance.bus.StreamErrorEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamErrorData]) {
		instance.logger.Error("Stream error occurred",
			zap.String("request_uuid", event.Data.RequestID.String()),
			zap.String("error", event.Data.Error.Error()))
		instance.bus.StreamChunkParsedErrorEvent.Publish(events.Event[dto.ParsedChunkErrorData]{
			Data: dto.ParsedChunkErrorData{
				RequestID: event.Data.RequestID,
				Error:     event.Data.Error.Error(),
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageService,
		})
	})
	instance.bus.StreamCompleteEvent.Subscribe(constants.MessageService, func(event events.Event[dto.StreamCompleteData]) {
		instance.bus.StreamChunkParsedEvent.Publish(events.Event[dto.ParsedChunkData]{
			Data: dto.ParsedChunkData{
				RequestID:  event.Data.RequestID,
				Content:    event.Data.FinalMessage,
				IsComplete: event.Data.IsComplete,
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageService,
		})
	})
}

func (instance *MessageService) ParsingMessage(data dto.StreamChunkData) {
	instance.bus.StreamChunkParsedEvent.Publish(events.Event[dto.ParsedChunkData]{
		Data: dto.ParsedChunkData{
			RequestID:  data.RequestID,
			Content:    data.Content,
			IsComplete: false,
		},
		TimeStamp: time.Now(),
		Source:    constants.MessageService,
	})
}
