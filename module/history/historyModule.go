package history

import (
	"DevCode/dto"
	"DevCode/events"
	"github.com/google/uuid"
)

func NewHistoryModule(bus *events.EventBus) *HistoryModule {
	module := &HistoryModule{Bus: bus, ParentUUID: uuid.Nil}
	return module
}

type HistoryModule struct {
	Bus             *events.EventBus
	EnvironmentData dto.EnvironmentUpdateData
	ParentUUID      uuid.UUID
}
