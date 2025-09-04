package history

import (
	"DevCode/src/dto"
	"DevCode/src/events"
	"github.com/google/uuid"
)

func NewHistoryService(bus *events.EventBus) *HistoryService {
	service := &HistoryService{Bus: bus, ParentUUID: uuid.Nil}
	return service
}

type HistoryService struct {
	Bus             *events.EventBus
	EnvironmentData dto.EnvironmentUpdateData
	ParentUUID      uuid.UUID
}
