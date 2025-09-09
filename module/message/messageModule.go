package message

import (
	"DevCode/constants"
	"DevCode/dto"
	"DevCode/events"
	"time"

	"go.uber.org/zap"
)

func NewMessageModule(bus *events.EventBus, logger *zap.Logger) *MessageModule {
	module := &MessageModule{bus: bus, logger: logger}
	module.Subscribe()
	return module
}

type MessageModule struct {
	bus    *events.EventBus
	logger *zap.Logger
}

func (instance *MessageModule) Subscribe() {
	events.Subscribe(instance.bus, instance.bus.StreamStartEvent, constants.MessageModule, func(event events.Event[dto.StreamStartData]) {
		events.Publish(instance.bus, instance.bus.StreamChunkParsedEvent, events.Event[dto.ParsedChunkData]{
			Data: dto.ParsedChunkData{
				RequestID:  event.Data.RequestID,
				Content:    "",
				IsComplete: false,
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageModule,
		})
	})
	events.Subscribe(instance.bus, instance.bus.StreamChunkEvent, constants.MessageModule, func(event events.Event[dto.StreamChunkData]) {
		instance.ParsingMessage(event.Data)
	})
	events.Subscribe(instance.bus, instance.bus.StreamErrorEvent, constants.MessageModule, func(event events.Event[dto.StreamErrorData]) {
		instance.logger.Error("Stream error occurred",
			zap.String("request_uuid", event.Data.RequestID.String()),
			zap.String("error", event.Data.Error.Error()))
		events.Publish(instance.bus, instance.bus.StreamChunkParsedErrorEvent, events.Event[dto.ParsedChunkErrorData]{
			Data: dto.ParsedChunkErrorData{
				RequestID: event.Data.RequestID,
				Error:     event.Data.Error.Error(),
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageModule,
		})
	})
	events.Subscribe(instance.bus, instance.bus.StreamCompleteEvent, constants.MessageModule, func(event events.Event[dto.StreamCompleteData]) {
		events.Publish(instance.bus, instance.bus.StreamChunkParsedEvent, events.Event[dto.ParsedChunkData]{
			Data: dto.ParsedChunkData{
				RequestID:  event.Data.RequestID,
				Content:    event.Data.FinalMessage,
				IsComplete: event.Data.IsComplete,
			},
			TimeStamp: time.Now(),
			Source:    constants.MessageModule,
		})
	})
}

func (instance *MessageModule) ParsingMessage(data dto.StreamChunkData) {
	events.Publish(instance.bus, instance.bus.StreamChunkParsedEvent, events.Event[dto.ParsedChunkData]{
		Data: dto.ParsedChunkData{
			RequestID:  data.RequestID,
			Content:    data.Content,
			IsComplete: false,
		},
		TimeStamp: time.Now(),
		Source:    constants.MessageModule,
	})
}
