package history

import (
	"DevCode/src/constants"
	"DevCode/src/dto"
	"DevCode/src/events"
	"github.com/google/uuid"
)

func NewHistoryService(bus *events.EventBus) *HistoryService {
	service := &HistoryService{Bus: bus, ParentUUID: uuid.Nil}
	bus.Subscribe(events.UserInputEvent, service)
	bus.Subscribe(events.UpdateEnvironmentEvent, service)
	bus.Subscribe(events.StreamChunkParsedEvent, service)
	return service
}

type HistoryService struct {
	Bus             *events.EventBus
	EnvironmentData dto.EnvironmentUpdateData
	ParentUUID      uuid.UUID
}

func (instance *HistoryService) HandleEvent(event events.Event) {
	switch event.Type {
	case events.UpdateEnvironmentEvent:
		instance.EnvironmentData = event.Data.(dto.EnvironmentUpdateData)

	}
}

func (instance *HistoryService) GetID() constants.Source {
	return constants.HistoryService
}
